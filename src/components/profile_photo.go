package components

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"astraltech.xyz/accountmanager/src/helpers"
	"astraltech.xyz/accountmanager/src/ldap"
	"astraltech.xyz/accountmanager/src/logging"
	"astraltech.xyz/accountmanager/src/session"
)

var (
	photoCreatedTimestamp = make(map[string]time.Time)
	photoCreatedMutex     sync.Mutex
	blankPhotoData        []byte
)

var (
	sessionManager = session.GetSessionManager()
	LDAPServer     *ldap.LDAPServer

	BaseDN string

	ServiceUserBindDN   string
	ServiceUserPassword string
)

func ReadBlankPhoto() {
	blank, err := helpers.ReadFile("static/images/blank_profile.jpg")
	if err != nil {
		logging.Fatal("Could not load blank profile image")
	}
	blankPhotoData = blank
}

func CreateUserPhoto(username string, photoData []byte) error {
	helpers.Mkdir("./avatars", os.ModePerm)

	path := fmt.Sprintf("./avatars/%s.jpeg", username)
	cleaned := filepath.Clean(path)
	dst, err := helpers.CreateFile(cleaned)

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

func UploadPhotoHandler(w http.ResponseWriter, r *http.Request) {
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
	userDN := fmt.Sprintf("uid=%s,cn=users,cn=accounts,%s", sessionData.UserID, BaseDN)
	err = LDAPServer.ModifyAttribute(ServiceUserBindDN, ServiceUserPassword, userDN, "jpegphoto", []string{string(data)})
	if err != nil {
		logging.Error(err.Error())
		return
	}
	CreateUserPhoto(sessionData.UserID, data)
}

func AvatarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/jpeg")
	username := r.URL.Query().Get("user")
	if strings.Contains(username, "/") {
		w.Write(blankPhotoData)
		return
	}

	filePath := fmt.Sprintf("./avatars/%s.jpeg", username)
	cleaned := filepath.Clean(filePath)
	fileExist, err := helpers.DoesFileExist(cleaned)
	if err != nil {
		w.Write(blankPhotoData)
		logging.Error(err.Error())
		return
	}
	photoCreatedMutex.Lock()
	if fileExist && time.Since(photoCreatedTimestamp[username]) <= 5*time.Minute {
		photoCreatedMutex.Unlock()
		val, err := helpers.ReadFile(cleaned)
		if err != nil {
			logging.Error(err.Error())
			w.Write(blankPhotoData)
			return
		}
		w.Write(val)
		return
	}
	photoCreatedMutex.Unlock()

	userSearch, err := LDAPServer.SerchServer(
		ServiceUserBindDN, ServiceUserPassword,
		BaseDN,
		fmt.Sprintf("(&(objectClass=inetOrgPerson)(uid=%s))", ldap.LDAPEscapeFilter(username)),
		[]string{"jpegphoto"},
	)
	if err != nil {
		logging.Error(err.Error())
		w.Write(blankPhotoData)
		return
	}
	if userSearch.EntryCount() == 0 {
		w.Write(blankPhotoData)
		return
	}

	entry := userSearch.GetEntry(0)
	bytes := entry.GetRawAttributeValue("jpegphoto")
	if len(bytes) == 0 {
		w.Write(blankPhotoData)
		return
	} else {
		w.Write(bytes)
		CreateUserPhoto(username, bytes)
		return
	}
}
