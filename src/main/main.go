package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"

	"astraltech.xyz/accountmanager/src/components"
	"astraltech.xyz/accountmanager/src/email"
	"astraltech.xyz/accountmanager/src/helpers"
	"astraltech.xyz/accountmanager/src/ldap"
	"astraltech.xyz/accountmanager/src/logging"
	"astraltech.xyz/accountmanager/src/session"
)

var (
	ldapServer     ldap.LDAPServer
	serverConfig   *ServerConfig
	sessionManager *session.SessionManager
	noReplyEmail   email.EmailAccount
)

type UserData struct {
	isAuth      bool
	DisplayName string
	Email       string
}

var (
	userData      = make(map[string]*UserData)
	userDataMutex sync.RWMutex
)

func authenticateUser(username, password string) (*UserData, error) {
	logging.Event(logging.AuthenticateUser, username)
	userDN := fmt.Sprintf("uid=%s,cn=users,cn=accounts,%s", username, serverConfig.LDAPConfig.BaseDN)

	connected, err := ldapServer.AuthenticateUser(userDN, password)
	if err != nil {
		return nil, err
	}
	if connected == false {
		logging.Debug("Failed to authenticate user")
		return nil, fmt.Errorf("Failed to authenticate user %s", username)
	}
	logging.Info("User authenticated successfully")

	userSearch, err := ldapServer.SerchServer(
		userDN, password,
		serverConfig.LDAPConfig.BaseDN,
		fmt.Sprintf("(&(objectClass=inetOrgPerson)(uid=%s))", ldap.LDAPEscapeFilter(username)),
		[]string{"displayName", "mail", "jpegphoto"},
	)
	if err != nil {
		return nil, err
	}

	entry := userSearch.GetEntry(0)
	user := UserData{
		isAuth:      true,
		DisplayName: entry.GetAttributeValue("displayName"),
		Email:       entry.GetAttributeValue("mail"),
	}

	photoData := entry.GetRawAttributeValue("jpegphoto")
	if len(photoData) > 0 {
		components.CreateUserPhoto(username, photoData)
	}
	return &user, nil
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
			logging.Error(err.Error())
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

	err = ldapServer.ChangePassword(userDN, oldPassword, newPassword)
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
	sessionManager = session.GetSessionManager()
	sessionManager.SetStoreType(session.InMemory)

	var err error
	serverConfig, err = loadServerConfig("./data/config.json")
	if err != nil {
		log.Fatal("Could not load server config")
	}

	noReplyEmail = email.CreateEmailAccount(email.EmailAccountData{
		Username: serverConfig.EmailConfig.Username,
		Password: serverConfig.EmailConfig.Password,
		Email:    serverConfig.EmailConfig.Email,
	}, serverConfig.EmailConfig.SMTPURL, serverConfig.EmailConfig.SMTPPort)

	ldapServer = ldap.LDAPServer{
		URL:                serverConfig.LDAPConfig.LDAPURL,
		StartTLS:           serverConfig.LDAPConfig.Security == "tls",
		IgnoreInsecureCert: serverConfig.LDAPConfig.IgnoreInvalidCert,
	}

	components.LDAPServer = &ldapServer
	components.BaseDN = serverConfig.LDAPConfig.BaseDN
	components.ServiceUserBindDN = serverConfig.LDAPConfig.BindDN
	components.ServiceUserPassword = serverConfig.LDAPConfig.BindPassword

	connected, err := ldapServer.TestConnection()
	if connected != true || err != nil {
		if err != nil {
			logging.Error(err.Error())
		}
		logging.Fatal("Failed to connect to LDAP server")
	}

	InitPasswordExpiry()

	helpers.HandleFunc("/favicon.ico", faviconHandler)
	helpers.HandleFunc("/logo", logoHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	helpers.HandleFunc("/login", loginHandler)
	helpers.HandleFunc("/profile", profileHandler)
	helpers.HandleFunc("/logout", logoutHandler)

	helpers.HandleFunc("/avatar", components.AvatarHandler)
	helpers.HandleFunc("/change-photo", components.UploadPhotoHandler)
	helpers.HandleFunc("/change-password", changePasswordHandler)

	helpers.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/profile", http.StatusFound) // 302 redirect
	})

	serverAddress := fmt.Sprintf(":%d", serverConfig.WebserverConfig.Port)
	logging.Fatal(http.ListenAndServe(serverAddress, nil).Error())
}
