package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"astraltech.xyz/accountmanager/src/logging"
	"astraltech.xyz/accountmanager/src/worker"
)

var (
	ldapServer      *LDAPServer
	ldapServerMutex sync.Mutex
	serverConfig    *ServerConfig
)

type UserData struct {
	isAuth      bool
	Username    string
	DisplayName string
	Email       string
}

var (
	photoCreatedTimestamp = make(map[string]time.Time)
	photoCreatedMutex     sync.Mutex
	blankPhotoData        []byte
)

func createUserPhoto(username string, photoData []byte) error {
	Mkdir("./avatars", os.ModePerm)

	path := fmt.Sprintf("./avatars/%s.jpeg", username)
	cleaned := filepath.Clean(path)
	dst, err := CreateFile(cleaned)

	if err != nil {
		return fmt.Errorf("Could not save file")
	}
	photoCreatedMutex.Lock()
	photoCreatedTimestamp[username] = time.Now()
	photoCreatedMutex.Unlock()
	defer dst.Close()
	logging.Info("Writing to avarar file")
	_, err = dst.Write(photoData)
	if err != nil {
		return err
	}
	return nil
}

func authenticateUser(username, password string) (UserData, error) {
	logging.Event(logging.AuthenticateUser, username)
	ldapServerMutex.Lock()
	defer ldapServerMutex.Unlock()
	if ldapServer.Connection == nil {
		return UserData{isAuth: false}, fmt.Errorf("LDAP server not connected")
	}
	userDN := fmt.Sprintf("uid=%s,cn=users,cn=accounts,%s", username, serverConfig.LDAPConfig.BaseDN)
	connected := connectAsLDAPUser(ldapServer, userDN, password)
	if connected != nil {
		return UserData{isAuth: false}, connected
	}

	userSearch := searchLDAPServer(
		ldapServer,
		serverConfig.LDAPConfig.BaseDN,
		fmt.Sprintf("(&(objectClass=inetOrgPerson)(uid=%s))", ldapEscapeFilter(username)),
		[]string{"displayName", "mail", "jpegphoto"},
	)
	if !userSearch.Succeeded {
		return UserData{isAuth: false}, fmt.Errorf("user metadata not found")
	}

	entry := userSearch.LDAPSearch.Entries[0]
	user := UserData{
		isAuth:      true,
		Username:    username,
		DisplayName: entry.GetAttributeValue("displayName"),
		Email:       entry.GetAttributeValue("mail"),
	}

	photoData := entry.GetRawAttributeValue("jpegphoto")
	if len(photoData) > 0 {
		createUserPhoto(user.Username, photoData)
	}
	return user, nil
}

type LoginPageData struct {
	IsHiddenClassList string
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	logging.Info("Handing login page")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFiles("src/pages/login_page.html"))
	if r.Method == http.MethodGet {
		logging.Info("Rending login page")
		tmpl.Execute(w, LoginPageData{IsHiddenClassList: "hidden"})
		return
	}

	// 2. Logic for processing the form
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		if strings.Contains(username, "/") {
			tmpl.Execute(w, LoginPageData{IsHiddenClassList: ""})
		}
		password := r.FormValue("password")

		logging.Infof("New Login request for %s\n", username)
		userData, err := authenticateUser(username, password)
		if err != nil {
			log.Print(err)
			tmpl.Execute(w, LoginPageData{IsHiddenClassList: ""})
		} else {
			if userData.isAuth == true {
				cookie := createSession(&userData)
				if cookie == nil {
					http.Error(w, "Session error", 500)
					return
				}
				http.SetCookie(w, cookie)
				http.Redirect(w, r, "/profile", http.StatusFound)
			} else {
				tmpl.Execute(w, LoginPageData{IsHiddenClassList: ""})
			}
		}
	}
}

type ProfileData struct {
	Username    string
	Email       string
	DisplayName string
	CSRFToken   string
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	exist, sessionData := validateSession(r)
	if !exist {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("src/pages/profile_page.html"))
		tmpl.Execute(w, ProfileData{
			Username:    sessionData.data.Username,
			Email:       sessionData.data.Email,
			DisplayName: sessionData.data.DisplayName,
			CSRFToken:   sessionData.CSRFToken,
		})
		return
	}
}

func avatarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	username := r.URL.Query().Get("user")
	if strings.Contains(username, "/") {
		w.Write(blankPhotoData)
		return
	}

	filePath := fmt.Sprintf("./avatars/%s.jpeg", username)
	cleaned := filepath.Clean(filePath)
	value, err := ReadFile(cleaned)

	if err == nil {
		photoCreatedMutex.Lock()
		if time.Since(photoCreatedTimestamp[username]) <= 5*time.Minute {
			photoCreatedMutex.Unlock()
			w.Write(value)
			return
		}
		photoCreatedMutex.Unlock()
	}

	ldapServerMutex.Lock()
	defer ldapServerMutex.Unlock()
	connected := connectAsLDAPUser(ldapServer, serverConfig.LDAPConfig.BindDN, serverConfig.LDAPConfig.BindPassword)
	if connected != nil {
		w.Write(blankPhotoData)
		return
	}

	userSearch := searchLDAPServer(
		ldapServer,
		serverConfig.LDAPConfig.BaseDN,
		fmt.Sprintf("(&(objectClass=inetOrgPerson)(uid=%s))", ldapEscapeFilter(username)),
		[]string{"jpegphoto"},
	)
	if !userSearch.Succeeded || len(userSearch.LDAPSearch.Entries) == 0 {
		w.Write(blankPhotoData)
		return
	}
	entry := userSearch.LDAPSearch.Entries[0]
	bytes := entry.GetRawAttributeValue("jpegphoto")
	if len(bytes) == 0 {
		w.Write(blankPhotoData)
		return
	} else {
		w.Write(bytes)
		createUserPhoto(username, bytes)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	token := cookie.Value

	exist, sessionData := validateSession(r)
	if exist {
		if r.FormValue("csrf_token") != sessionData.CSRFToken {
			http.Error(w, "Unable to log user out", http.StatusForbidden)
			logging.Debugf("%s attempted to logout with invalid csrf token", sessionData.data.Username)
			return
		}
	}
	logging.Infof("handling logout event for %s", sessionData.data.Username)

	deleteSession(hashSession(token))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func uploadPhotoHandler(w http.ResponseWriter, r *http.Request) {
	exist, sessionData := validateSession(r)
	if !exist {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if r.FormValue("csrf_token") != sessionData.CSRFToken {
		http.Error(w, "CSRF Forbidden", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "File not found", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if header.Size > (10 * 1024 * 1024) {
		http.Error(w, "File is to large (limit is 10 MB)", http.StatusBadRequest)
		return
	}

	// 3. Read file into memory
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	userDN := fmt.Sprintf("uid=%s,cn=users,cn=accounts,%s", sessionData.data.Username, serverConfig.LDAPConfig.BaseDN)
	ldapServerMutex.Lock()
	defer ldapServerMutex.Unlock()
	modifyLDAPAttribute(ldapServer, userDN, "jpegphoto", []string{string(data)})
	createUserPhoto(sessionData.data.Username, data)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	logging.Info("Requesting Favicon")
	http.ServeFile(w, r, serverConfig.StyleConfig.FaviconPath)
}

func logoHandler(w http.ResponseWriter, r *http.Request) {
	logging.Info("Requesting Logo")
	http.ServeFile(w, r, serverConfig.StyleConfig.LogoPath)
}

func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	exist, sessionData := validateSession(r)
	if !exist {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success": false, "error": "Not authenticated"}`))
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success": false, "error": "Bad request"}`))
		return
	}

	if r.FormValue("csrf_token") != sessionData.CSRFToken {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success": false, "error": "CSRF Forbidden"}`))
		return
	}

	oldPassword := r.FormValue("old_password")
	newPassword := r.FormValue("new_password")
	newPasswordRepeat := r.FormValue("new_password_repeat")

	if newPassword != newPasswordRepeat {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success": false, "error": "Passwords do not match"}`))
		return
	}

	userDN := fmt.Sprintf(
		"uid=%s,cn=users,cn=accounts,%s",
		sessionData.data.Username,
		serverConfig.LDAPConfig.BaseDN,
	)

	err = changeLDAPPassword(ldapServer, userDN, oldPassword, newPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		if strings.Contains(err.Error(), "Invalid Credentials") {
			w.Write([]byte(`{"success": false, "error": "Current password incorrect"}`))
		} else if strings.Contains(err.Error(), "Too soon to change password") {
			w.Write([]byte(`{"success": false, "error": "Too soon to change password"}`))
		} else {
			w.Write([]byte(`{"success": false, "error": "Internal error"}`))
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

func main() {
	logging.Info("Starting the server")

	var err error = nil
	blankPhotoData, err = ReadFile("static/blank_profile.jpg")
	if err != nil {
		logging.Fatal("Could not load blank profile image")
	}
	serverConfig, err = loadServerConfig("./data/config.json")
	if err != nil {
		log.Fatal("Could not load server config")
	}

	ldapServerMutex.Lock()
	server := connectToLDAPServer(serverConfig.LDAPConfig.LDAPURL, serverConfig.LDAPConfig.Security == "tls", serverConfig.LDAPConfig.IgnoreInvalidCert)
	ldapServer = server
	ldapServerMutex.Unlock()
	defer closeLDAPServer(ldapServer)

	worker.CreateWorker(time.Minute*5, cleanupSessions)
	HandleFunc("/favicon.ico", faviconHandler)
	HandleFunc("/logo", logoHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	HandleFunc("/login", loginHandler)
	HandleFunc("/profile", profileHandler)
	HandleFunc("/logout", logoutHandler)

	HandleFunc("/avatar", avatarHandler)
	HandleFunc("/change-photo", uploadPhotoHandler)
	HandleFunc("/change-password", changePasswordHandler)

	HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/profile", http.StatusFound) // 302 redirect
	})

	serverAddress := fmt.Sprintf(":%d", serverConfig.WebserverConfig.Port)
	logging.Fatal(http.ListenAndServe(serverAddress, nil).Error())
}
