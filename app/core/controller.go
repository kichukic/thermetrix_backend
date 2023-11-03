package core

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// Controller handle all base methods
type Controller struct {
	Users *map[string]User
}

const (
	NeededPermission_Read = iota
	NeededPermission_Add
	NeededPermission_Edit
)

const (
	Account_Locked = 1000
)

const component_contacts = "CONTACT"

var Config Configuration

func (c *Controller) SendNotification() {

}

func (c *Controller) SendJSON(w http.ResponseWriter, v interface{}, code int) {
	c.SendJSONExtra(w, nil, v, code, nil)
}

func (c *Controller) SendJSONPaging(w http.ResponseWriter, r *http.Request, paging *Paging, v interface{}, code int) {
	c.SendJSONExtra(w, paging, v, code, nil)
}

// SendJSON marshals v to a json struct and sends appropriate headers to w
func (c *Controller) SendJSONExtra(w http.ResponseWriter, paging *Paging, v interface{}, code int, extras ...interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Access-Control-Allow-Origin", "*")

	var t1 string = reflect.TypeOf(v).String()
	var t2 string = reflect.TypeOf(ResponseData{}).String()
	//log.Println(t1)
	//log.Println(t2)
	var tmp interface{}
	//if reflect.TypeOf(v) == reflect.TypeOf((*ResponseData)(nil)).Elem() || reflect.TypeOf(v) == reflect.TypeOf(ResponseData{}) {
	if t1 == t2 || t1 == "*"+t2 {
		//log.Println("Equal")
		tmp = v
	} else {
		if extras != nil && len(extras) > 0 && extras[0] != nil {
			sections := extras[0].(*[]Section)
			tmp = ResponseData{
				Status:   1,
				Data:     v,
				Sections: *sections,
				Paging:   paging,
			}
		} else {
			tmp = ResponseData{
				Status: 1,
				Data:   v,
				Paging: paging,
			}
		}

	}

	b, err := json.Marshal(tmp)

	//log.Println(string(b))
	//log.Println(code)

	if err != nil {
		log.Print(fmt.Sprintf("Error while encoding JSON: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error": "Internal server error"}`)
	} else {
		w.WriteHeader(code)
		io.WriteString(w, string(b))
	}
}

// GetContent of the request inside given struct
func (c *Controller) GetContent(v interface{}, r *http.Request) error {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
	}

	//log.Println(string(body))

	err = json.Unmarshal(body, v)
	if err != nil {
		log.Println(err)
		stringBody := string(body)
		stringBody = strings.Replace(stringBody, `"vat_rate":"",`, `"vat_rate":0.19,`, -1)
		err = json.Unmarshal([]byte(stringBody), v)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil

	/*
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(v)
		if err != nil {
			return err
		}

		return nil
	*/
}

// HandleError write error on response and return false if there is no error
//
// swagger:response HandleError
func (c *Controller) HandleError(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	msg := ResponseData{
		Status:  999,
		Message: "An error occured",
		Detail:  err.Error(),
	}

	c.SendJSON(w, &msg, http.StatusInternalServerError)
	return true
}

// HandleError write error on response and return false if there is no error
//
// swagger:response HandleError
func (c *Controller) HandleErrorWithStatus(err error, w http.ResponseWriter, statusCode int) bool {
	if err == nil {
		return false
	}

	msg := ResponseData{
		Status:  999,
		Message: "An error occured",
		Detail:  err.Error(),
	}

	c.SendJSON(w, &msg, statusCode)
	return true
}

func (c *Controller) SendErrors(w http.ResponseWriter, v map[string]string, statusCode int) {

	errorText := ""
	for _, val := range v {
		errorText += val + " \n"
	}

	msg := ResponseData{
		Status:  999,
		Message: "An error occured",
		Detail:  errorText,
	}

	c.SendJSON(w, &msg, statusCode)
}

// HandleError write error on response and return false if there is no error
//
// swagger:response HandlePermissionError
func (c *Controller) HandlePermissionError(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	msg := ResponseData{
		Status:  998,
		Message: "You are not allowed to access these data",
		Detail:  err.Error(),
	}

	c.SendJSON(w, &msg, http.StatusForbidden)
	return true
}

// HandleError write error on response and return false if there is no error
//
// swagger:response HandleUnauthorizedError
func (c *Controller) HandleUnauthorizedError(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	msg := ResponseData{
		Status:  997,
		Message: "You are not authorized, please login",
		Detail:  err.Error(),
	}

	c.SendJSON(w, &msg, http.StatusUnauthorized)
	return true
}

// HandleError write error on response and return false if there is no error
//
// swagger:response HandleUnauthorizedError
func (c *Controller) HandleAccountLockedError(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	msg := ResponseData{
		Status:  Account_Locked,
		Message: "Account locked",
		Detail:  err.Error(),
	}

	c.SendJSON(w, &msg, http.StatusUnauthorized)
	return true
}

func (c *Controller) OptionsHandler(w http.ResponseWriter, r *http.Request) {

	//log.Println("OPTIONS-Handler")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Authorization")
	w.Header().Add("Access-Control-Allow-Headers", "Client")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Access-Control-Allow-Headers", "X-Timezone-Offset")
	w.Header().Add("Access-Control-Allow-Headers", "X-Timezone")
	w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")

	w.WriteHeader(http.StatusOK)
}

// https://play.golang.org/p/nt-sRBtMIT
func (c *Controller) GetUser(w http.ResponseWriter, r *http.Request) (bool, *User) {

	auth := r.Header.Get("Authorization")
	//log.Println("auth: ", auth)
	//log.Printf("USERS: %v", c.Users)

	if len(auth) != len("Bearer 9871b73e-df71-4780-5ed6-b2cbee85f3b5") {
		//log.Println("----------cfvtgbzhujnk")
		c.HandleUnauthorizedError(errors.New("Not Auhtorized"), w)
		return false, nil
	} else {
		tmp := strings.Split(auth, " ")

		if user, ok := (*c.Users)[tmp[1]]; ok {
			return true, &user
		} else {
			c.HandleUnauthorizedError(errors.New("Session invalid"), w)
			return false, nil
		}
	}
}
func (c *Controller) TryGetUser(w http.ResponseWriter, r *http.Request) (bool, *User) {

	auth := r.Header.Get("Authorization")
	//log.Println("auth: ", auth)
	//log.Printf("USERS: %v", c.Users)

	if len(auth) != len("Bearer 9871b73e-df71-4780-5ed6-b2cbee85f3b5") {
		return false, nil
	} else {
		tmp := strings.Split(auth, " ")
		//log.Println(tmp[1])

		if user, ok := (*c.Users)[tmp[1]]; !ok {
			return false, nil
		} else {
			return true, &user
		}
	}

}

func (c *Controller) GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

const letterBytes = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPRSTUVWXYZ123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func (c *Controller) RandomString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func RandomString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (c *Controller) GetPaging(values url.Values) *Paging {
	paging := Paging{
		Page:   -1,
		Offset: -1,
		Limit:  -1,
	}
	if len(values) > 0 {
		if val, ok := values["page"]; ok && len(val) > 0 {
			if val[0] != "" {
				paging.Page, _ = strconv.Atoi(val[0])
			}
		}
		if val, ok := values["per_page"]; ok && len(val) > 0 {
			if val[0] != "" {
				paging.PerPage, _ = strconv.Atoi(val[0])
			}
		}

		if val, ok := values["limit"]; ok && len(val) > 0 {
			if val[0] != "" {
				paging.Limit, _ = strconv.Atoi(val[0])
			}
		}

		if val, ok := values["offset"]; ok && len(val) > 0 {
			if val[0] != "" {
				paging.Offset, _ = strconv.Atoi(val[0])
			}
		} /*
			if val, ok := values["page_count"]; ok && len(val) > 0 {
				if val[0] != "" {
					paging.PageCount, _ = strconv.Atoi(val[0])
				}
			}
			if val, ok := values["total_count"]; ok && len(val) > 0 {
				if val[0] != "" {
					paging.TotalCount, _ = strconv.Atoi(val[0])
				}
			}*/
	}

	if paging.Limit > 0 || paging.Offset > 0 {
		return &paging
	}
	if paging.Page >= 0 {
		if paging.PerPage > 0 {
			paging.Limit = paging.PerPage
		} else {
			//paging.Limit = 200
			//paging.PerPage = 200
		}

		paging.Offset = paging.Page * paging.PerPage

	} else {
		paging.Page = 0
		paging.PerPage = 200
	}

	return &paging
}

func ZipFiles(fileName string, fileNames []string) (string, error) {
	//X tmpPath := "./tmp/" + RandomString(10)
	//tmpFilename := tmpPath + "/" + fileName
	tmpPath := GetTmpUploadPath()
	tmpFilename := tmpPath + fileName
	if err := os.MkdirAll(tmpPath, os.ModePerm); err != nil {
		log.Println(err)
		return "", err
	}
	file, err := os.Create(tmpFilename)
	if err != nil {
		log.Printf("Failed to open zip for writing: %s", err)
	}
	defer file.Close()
	zipw := zip.NewWriter(file)
	defer zipw.Close()
	for _, filename := range fileNames {
		file, err := os.Open(filename)
		if err != nil {
			log.Println(fmt.Errorf("Failed to open %s: %s", filename, err))
			continue
		}
		pos := strings.LastIndex(filename, "/")
		log.Println(filename[pos+1:])
		wr, err := zipw.Create(filename[pos+1:])
		if err != nil {
			msg := "Failed to create entry for %s in zip file: %s"
			log.Println(fmt.Errorf(msg, filename, err))
		}

		if _, err := io.Copy(wr, file); err != nil {
			log.Println(err)
			return "", err
		}
		file.Close()
	}
	return tmpFilename, nil
}

func (c *Controller) SendFile(w http.ResponseWriter, r *http.Request, filepath string) {
	pos := strings.LastIndex(filepath, "/")
	log.Println(filepath, pos)
	tmp := filepath
	if len(tmp) > pos+1 {
		tmp = filepath[pos+1:]
	}
	log.Println(tmp, pos)
	c.SendFileWithName(w, r, filepath, tmp)
}

func (c *Controller) SendFileWithName(w http.ResponseWriter, r *http.Request, filepath, filename string) {
	w.Header().Add("Content-Disposition", filename)
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Expose-Headers", "Content-Disposition,Access-Control-Allow-Origin,")
	http.ServeFile(w, r, filepath)
}

func (c *Controller) GetTmpUploadPath() string {
	return c.GetTmpUploadPathWithRandomStringCount(10)
}

func (c *Controller) GetTmpUploadPathWithRandomStringCount(randomStringCount int) string {
	return GetTmpUploadPathWithRandomStringCount(randomStringCount)
}

func GetTmpUploadPath() string {
	return GetTmpUploadPathWithRandomStringCount(10)
}

func GetTmpUploadPathWithRandomStringCount(randomStringCount int) string {
	tmpPath := Config.Server.TmpPath
	if tmpPath == "" {
		tmpPath = "./tmp"
	}
	if !strings.HasSuffix(tmpPath, "/") {
		tmpPath += "/"
	}
	tmpPath += RandomString(randomStringCount) + "/"
	err := os.MkdirAll(tmpPath, 0700)
	if err != nil {
		log.Println(err)
	}
	return tmpPath
}
