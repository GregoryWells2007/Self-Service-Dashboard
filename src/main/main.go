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
	"astraltech.xyz/accountmanager/src/session"
)

var (
	ldapServer      *LDAPServer
	ldapServerMutex sync.Mutex
	serverConfig    *ServerConfig
	sessionManager  *session.SessionManager
)

type UserData struct {
	isAuth      bool
	DisplayName string
	Email       string
}

var (
	userData      = make(map[string]UserData)
	userDataMutex sync.RWMutex
)

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
		DisplayName: entry.GetAttributeValue("displayName"),
		Email:       entry.GetAttributeValue("mail"),
	}

	photoData := entry.GetRawAttributeValue("jpegphoto")
	if len(photoData) > 0 {
		createUserPhoto(username, photoData)
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
		newUserData, err := authenticateUser(username, password)
		userDataMutex.Lock()
		userData[username] = newUserData
		userDataMutex.Unlock()
		if err != nil {
			log.Print(err)
			tmpl.Execute(w, LoginPageData{IsHiddenClassList: ""})
		} else {
			if newUserData.isAuth == true {
				cookie, err := sessionManager.CreateSession(username)
				if err != nil {
					logging.Error(err.Error())
					http.Error(w, "Session error", http.StatusInternalServerError)
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
	sessionData, err := sessionManager.GetSession(r)
	if err != nil {
		logging.Error(err.Error())
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		tmpl := template.Must(template.ParseFiles("src/pages/profile_page.html"))
		userDataMutex.RLock()
		tmpl.Execute(w, ProfileData{
			Username:    sessionData.UserID,
			Email:       userData[sessionData.UserID].Email,
			DisplayName: userData[sessionData.UserID].DisplayName,
			CSRFToken:   sessionData.CSRFToken,
		})
		userDataMutex.RUnlock()
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

	sessionData, err := sessionManager.GetSession(r)
	if err != nil {
		logging.Error(err.Error())
	}
	if r.FormValue("csrf_token") != sessionData.CSRFToken {
		http.Error(w, "Unable to log user out", http.StatusForbidden)
		logging.Debugf("%s attempted to logout with invalid csrf token", sessionData.UserID)
		return
	}
	logging.Infof("handling logout event for %s", sessionData.UserID)

	sessionManager.DeleteSession(token)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func uploadPhotoHandler(w http.ResponseWriter, r *http.Request) {
	sessionData, err := sessionManager.GetSession(r)
	if err != nil {
		logging.Error(err.Error())
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	err = r.ParseMultipartForm(10 << 20) // 10MB limit
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
	userDN := fmt.Sprintf("uid=%s,cn=users,cn=accounts,%s", sessionData.UserID, serverConfig.LDAPConfig.BaseDN)
	ldapServerMutex.Lock()
	defer ldapServerMutex.Unlock()
	modifyLDAPAttribute(ldapServer, userDN, "jpegphoto", []string{string(data)})
	createUserPhoto(sessionData.UserID, data)
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

	sessionData, err := sessionManager.GetSession(r)
	if err != nil {
		logging.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success": false, "error": "Not authenticated"}`))
		return
	}

	err = r.ParseMultipartForm(10 << 20)
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
		sessionData.UserID,
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
	sessionManager = session.CreateSessionManager(session.InMemory)

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
