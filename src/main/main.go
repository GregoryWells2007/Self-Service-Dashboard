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
	os.Mkdir("./avatars", os.ModePerm)

	path := fmt.Sprintf("./avatars/%s.jpeg", username)
	cleaned := filepath.Clean(path)
	dst, err := os.Create(cleaned)

	if err != nil {
		fmt.Printf("Not saving file\n")
		return fmt.Errorf("Could not save file")
	}
	photoCreatedMutex.Lock()
	photoCreatedTimestamp[username] = time.Now()
	photoCreatedMutex.Unlock()
	defer dst.Close()
	_, err = dst.Write(photoData)
	if err != nil {
		return err
	}
	return nil
}

func authenticateUser(username, password string) (UserData, error) {
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFiles("src/pages/login_page.html"))
	if r.Method == http.MethodGet {
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

		log.Printf("New Login request for %s\n", username)
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
		fmt.Println("Returned blank avatar because couldnt connect as user")
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
		fmt.Println("Returned blank avatar because we couldnt find the user")
		return
	}
	entry := userSearch.LDAPSearch.Entries[0]
	bytes := entry.GetRawAttributeValue("jpegphoto")
	if len(bytes) == 0 {
		fmt.Println("Returned blank avatar because we just don't have an avatar")
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
			log.Printf("%s attempted to logout with invalid csrf token", sessionData.data.Username)
			return
		}
	}

	sessionMutex.Lock()
	delete(sessions, token)
	sessionMutex.Unlock()
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
	http.ServeFile(w, r, serverConfig.StyleConfig.FaviconPath)
}

func logoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, serverConfig.StyleConfig.LogoPath)
}

func cleanupSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	sessions_to_delete := []string{}
	for session_token, session_data := range sessions {
		timeUntilRemoval := time.Minute * 5
		if session_data.loggedIn {
			timeUntilRemoval = time.Hour
		}
		if time.Since(session_data.timeCreated) > timeUntilRemoval {
			sessions_to_delete = append(sessions_to_delete, session_token)
		}
	}
	for _, session_id := range sessions_to_delete {
		delete(sessions, session_id)
	}
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

	createWorker(time.Minute*5, cleanupSessions)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/logo", logoHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/profile", profileHandler)
	http.HandleFunc("/logout", logoutHandler)

	http.HandleFunc("/avatar", avatarHandler)
	http.HandleFunc("/change-photo", uploadPhotoHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/profile", http.StatusFound) // 302 redirect
	})

	serverAddress := fmt.Sprintf(":%d", serverConfig.WebserverConfig.Port)
	log.Fatal(http.ListenAndServe(serverAddress, nil))
}
