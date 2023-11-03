package systembundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"thermetrix_backend/app/core"
	"thermetrix_backend/app/podiumbundle"
	"thermetrix_backend/app/websocket"
	"time"
)

// login swagger:route GET /system/login system login
//
// Logs you in
//
// produces:
// - application/json
// parameters:
//	+ name: User
//    type: core.User
//    required: true
//    in: body
//    description: An User Object
// Responses:
//    default: HandleErrorData
//        200:
//	       data: core.User
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) Login(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var user core.User
	err := decoder.Decode(&user)
	if err != nil {
		// panic()
	}

	practiceLogin := false
	adminLogin := false

	if len(r.URL.Query()) > 0 {
		values := r.URL.Query()
		if val, ok := values["practice"]; ok && len(val) > 0 {
			if val[0] != "" {
				practiceLogin, _ = strconv.ParseBool(val[0])
			}
		}

		if val, ok := values["admin"]; ok && len(val) > 0 {
			if val[0] != "" {
				adminLogin, _ = strconv.ParseBool(val[0])
			}
		}
	}

	if len(user.PasswordX) == 0 {
		loginError := make(map[string]string)
		loginError["login"] = "failed"
		c.SendJSON(w, loginError, http.StatusUnauthorized)
		return
	}

	clientId := r.Header.Get("Client")

	if practiceLogin {
		c.ormDB.Where("username=? AND password=? AND user_type = 3", user.Username, c.GetMD5Hash(user.PasswordX)).First(&user)
	} else {
		c.ormDB.Where("username=? AND password=? AND user_type < 3", user.Username, c.GetMD5Hash(user.PasswordX)).First(&user)
		if user.ID == 0 && clientId == core.Client_Portal {
			//CAR-0077 Practice and doctor can no longer have the same username. We can now check the new accounts ourselves to see whether a doctor or practice is logging in.
			c.ormDB.Where("username=? AND password=? AND user_type = 3", user.Username, c.GetMD5Hash(user.PasswordX)).First(&user)
		}
	}
	c.ormDB.First(&user.Setting, user.SettingId)

	if user.ID == 0 {
		log.Printf("ERROR %v", err)
		loginError := make(map[string]string)
		loginError["login"] = "failed"
		c.SendJSON(w, loginError, http.StatusUnauthorized)
		return
	} else if !user.IsActive {
		/*loginError := make(map[string]string)
		loginError["login"] = "account locked"
		c.SendJSON(w, loginError, http.StatusUnauthorized)*/
		c.HandleAccountLockedError(errors.New("Account locked"), w)
		return
	}

	accountsSession := SystemAccountsSession{}

	c.ormDB.Where("account_id=?", user.ID).First(&accountsSession)

	firstLogin := accountsSession.ID == 0
	token, _ := core.NewV4()
	if accountsSession.SessionToken == "" {
		accountsSession.SessionToken = token.String()
		accountsSession.AccountId = user.ID
		accountsSession.LoginTime = core.NullTime{Time: time.Now(), Valid: true}
		c.ormDB.Set("gorm:save_associations", false).Create(&accountsSession)
	} else {
		accountsSession.ID = 0
		accountsSession.SessionToken = token.String()
		accountsSession.AccountId = user.ID
		accountsSession.LoginTime = core.NullTime{Time: time.Now(), Valid: true}
		c.ormDB.Set("gorm:save_associations", false).Create(&accountsSession)
	}

	user.Token = accountsSession.SessionToken
	user.PasswordX = ""
	(*c.Controller.Users)[user.Token] = user

	if adminLogin {
		if clientId != core.Client_Admin {
			loginError := make(map[string]string)
			loginError["login"] = "Not a valid client"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}

		if !user.IsSysadmin {
			loginError := make(map[string]string)
			loginError["login"] = "Not a admin account"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}
		c.SendJSON(w, &user, http.StatusOK)
		return
	}

	if c.isPatient(&user) {

		if clientId != core.Client_APK_Home && clientId != core.CLIENT_APK_Remote && clientId != core.Client_Admin {
			loginError := make(map[string]string)
			loginError["login"] = "Not a valid client"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}

		helperUser := podiumbundle.HelperUser{}
		copier.Copy(&helperUser, &user)
		helperUser.Patient = c.getPatient(&user)

		reward, countScans, countConsecutiveScans := podiumbundle.CalculateRewardForPatient(c.ormDB, helperUser.Patient.ID, false)
		helperUser.Patient.CurrentReward = *reward
		helperUser.Patient.TotalScans = int64(countScans)
		helperUser.Patient.ConsecutiveScans = int64(countConsecutiveScans)
		c.ormDB.DB().QueryRow("SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =? AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1", helperUser.Patient.ID).Scan(&helperUser.Patient.LastAppointmentDate)
		c.ormDB.DB().QueryRow("SELECT measurement_date FROM measurements m WHERE m.patient_id =? ORDER BY measurement_date DESC LIMIT 1", helperUser.Patient.ID).Scan(&helperUser.Patient.LastMeasurementDate)
		helperUser.Patient.LastQuestionnaire = &podiumbundle.PatientQuestionnaire{}
		helperUser.Patient.LastScanQuestionnaire = &podiumbundle.PatientQuestionnaire{}

		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = 0").Where("patient_id = ?", helperUser.Patient.ID).Last(&(helperUser.Patient.LastQuestionnaire))
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id > 0").Where("patient_id = ?", helperUser.Patient.ID).Last(&(helperUser.Patient.LastScanQuestionnaire))
		//Set all linked Doctors to have home users
		c.ormDB.Debug().Model(&podiumbundle.Doctor{}).Where("has_home_users = 0 OR has_home_users IS NULL").Where("id IN (SELECT doctor_id FROM doctor_patient_relations WHERE patient_id = ?)", helperUser.Patient.ID).Update("has_home_users", true)
		if helperUser.Patient.ID == 0 {
			loginError := make(map[string]string)
			loginError["login"] = "failed"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}
		if firstLogin {
			c.CreateNotification(user.ID, 4, "Welcome", "Welcome to Podium’s online services.  You can now pair, share scans, arrange appointments and chat with a podiatrist.  If you have not yet chosen a podiatrist, you can find (& pair) with a clinician in the settings menu.", 0, "", nil)
		}
		c.SendJSON(w, &helperUser, http.StatusOK)
	} else if c.isDoctor(&user) {
		if clientId != core.Client_Portal && clientId != core.Client_APK_Pro && clientId != core.Client_Admin { // clientId != "060a4e73-dcf5-4d6d-920a-6bee885806c9" ||
			loginError := make(map[string]string)
			loginError["login"] = "Not a valid client"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}
		helperUser := podiumbundle.HelperUser{}
		copier.Copy(&helperUser, &user)
		helperUser.Doctor = c.getDoctor(&user)
		if helperUser.Doctor.ID == 0 {
			loginError := make(map[string]string)
			loginError["login"] = "failed"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}
		c.SendJSON(w, &helperUser, http.StatusOK)
	} else if c.isPractice(&user) {
		if clientId != core.Client_Portal && clientId != core.Client_APK_Pro && false && clientId != core.Client_Admin { // clientId != "060a4e73-dcf5-4d6d-920a-6bee885806c9" ||
			loginError := make(map[string]string)
			loginError["login"] = "Not a valid client"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}

		helperUser := podiumbundle.HelperUser{}
		copier.Copy(&helperUser, &user)
		helperUser.Practice = c.getPractice(&user)
		if helperUser.Practice.ID == 0 {
			loginError := make(map[string]string)
			loginError["login"] = "failed"
			c.SendJSON(w, loginError, http.StatusUnauthorized)
			return
		}
		c.SendJSON(w, &helperUser, http.StatusOK)
	} else {
		c.SendJSON(w, &user, http.StatusOK)
	}

}

func (c *SystemController) isPatient(user *core.User) bool {
	if user.UserType == 1 {
		return true
	}
	return false
}

func (c *SystemController) isDoctor(user *core.User) bool {
	if user.UserType == 2 {
		return true
	}
	return false
}

func (c *SystemController) isPractice(user *core.User) bool {
	if user.UserType == 3 {
		return true
	}
	return false
}
func (c *SystemController) getPatient(user *core.User) *podiumbundle.Patient {
	if user.UserType == 1 {
		patient := &podiumbundle.Patient{}
		c.ormDB.Set("gorm:auto_preload", true).Where("user_id=?", user.ID).First(&patient)
		return patient
	}
	return nil
}

func (c *SystemController) getDoctor(user *core.User) *podiumbundle.Doctor {
	if user.UserType == 2 {
		doctor := &podiumbundle.Doctor{}
		c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM doctor_users WHERE status > 0 AND user_id=?) ", user.ID).First(&doctor)
		doctor.Users = nil
		return doctor
	}
	return nil
}

func (c *SystemController) getPractice(user *core.User) *podiumbundle.Practice {
	if user.UserType == 3 {
		practice := &podiumbundle.Practice{}
		c.ormDB.Set("gorm:auto_preload", true).Where("user_id=?", user.ID).First(&practice)
		return practice
	}
	return nil
}

func (c *SystemController) Logout(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")

	if len(auth) != len("Bearer 9871b73e-df71-4780-5ed6-b2cbee85f3b5") {
		c.HandleError(errors.New("Not Auhtorized"), w)
		return
	} else {
		tmp := strings.Split(auth, " ")
		if user, ok := (*c.Users)[tmp[1]]; ok {
			c.ormDB.Where("session_token=? AND account_id=?", tmp[1], user.ID).Delete(&SystemAccountsSession{})
		}
	}
	c.SendJSON(w, core.User{}, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) RegisterPatientHandler(w http.ResponseWriter, r *http.Request) {
	var patient podiumbundle.Patient

	_, user := c.Controller.TryGetUser(w, r)

	if err := c.GetContent(&patient, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.ormDB.NewRecord(&patient) {
		// check Username
		patient.User.Username = strings.TrimSpace(patient.User.Username)
		patient.User.Email = strings.TrimSpace(patient.User.Email)
		if len(patient.User.Username) < 4 {
			err := errors.New("Username to short, you need minimum 4 characters")
			c.HandleError(err, w)
			return
		}
		patient.User.Email = strings.TrimSpace(patient.User.Email)
		err := core.ValidateFormat(patient.User.Email)
		if err != nil && patient.User.Email != "" { // empty mail is ok!
			err = errors.New("E-Mail: " + err.Error())
			c.HandleError(err, w)
			return
		}

		//TODO SM PASSWORD
		if !c.isPractice(user) && !c.isDoctor(user) {
			err = core.ValidatePassword(patient.User.PasswordX)
			if err != nil {
				c.HandleError(err, w)
				return
			}
		} else {

		}

		userDB := core.User{}
		if patient.User.Email == "" { // empty mail -> do not check email
			c.ormDB.Set("gorm:auto_preload", true).Where("username=?", patient.User.Username).First(&userDB)
		} else {
			c.ormDB.Set("gorm:auto_preload", true).Where("username=? OR email=?", patient.User.Username, patient.User.Email).First(&userDB)
		}

		if userDB.ID > 0 {
			err := errors.New("Username or E-Mail already exists")
			c.HandleError(err, w)
			return
		}

		createdBy := uint(0)
		if user != nil {
			createdBy = user.ID
		}

		if c.ormDB.NewRecord(&patient.User) {
			patient.User.RegisteredAt.Time = time.Now()
			patient.User.RegisteredAt.Valid = true
			patient.User.UserType = 1

			if user != nil && patient.User.CreatedBy == 0 {
				patient.User.CreatedBy = createdBy
			}
			patient.User.IsActive = true
			//patient.User.IsPasswordExpired = true
			// patient.User.IsPasswordExpired = false

			if len(patient.User.PasswordX) > 0 {
				patient.User.PasswordX = core.GetMD5Hash(patient.User.PasswordX)
				patient.User.Password = patient.User.PasswordX
			}
			//c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", patient.User.PasswordX, patient.User.ID)

			c.ormDB.Set("gorm:save_associations", false).Save(&patient.User)
			patient.User.PasswordX = ""
			accountsSession := SystemAccountsSession{}

			token, _ := core.NewV4()
			accountsSession.ID = 0
			accountsSession.SessionToken = token.String()
			accountsSession.AccountId = patient.User.ID
			accountsSession.LoginTime = core.NullTime{Time: time.Now(), Valid: true}
			c.ormDB.Set("gorm:save_associations", false).Create(&accountsSession)
			patient.User.Token = accountsSession.SessionToken

			(*c.Controller.Users)[patient.User.Token] = patient.User

		}
		c.ormDB.Set("gorm:save_associations", false).Save(&patient)
		c.ormDB.Exec("UPDATE patients SET user_id = ? WHERE id = ?", patient.User.ID, patient.ID)
		c.CreateNotification(patient.User.ID, 4, "Welcome", "Welcome to Podium’s online services.  You can now pair, arrange appointments and chat with a podiatrist", 0, "", nil)
		podiumbundle.CreatePatientQuestionnaire(*c.ormDB, patient.ID, []int64{1}, []string{"SETUP"}, 0)
	}
	c.SendJSON(w, &patient, http.StatusOK)

	wsIds := []uint{user.ID}
	go web3socket.SendWebsocketDataInfoMessage("Register Patient", web3socket.Websocket_Update, web3socket.Websocket_Patients, uint(patient.ID), wsIds, nil)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) RegisterDoctorHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	var newUser core.User
	newDoctor := &podiumbundle.Doctor{}

	if err := c.GetContent(&newUser, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	practice := c.getPractice(user)

	if c.ormDB.NewRecord(&newUser) {
		// check Username
		newUser.Username = strings.TrimSpace(newUser.Username)
		newUser.Email = strings.TrimSpace(newUser.Email)
		if len(newUser.Username) < 1 {
			err := errors.New("Username to short, you need minimum 1 characters")
			c.HandleError(err, w)
			return
		}
		newUser.Email = strings.TrimSpace(newUser.Email)
		err := core.ValidateFormat(newUser.Email)
		if err != nil && newUser.Email != "" { // empty mail is ok!
			err = errors.New("E-Mail: " + err.Error())
			c.HandleError(err, w)
			return
		}
		err = core.ValidatePassword(newUser.PasswordX)
		if err != nil {
			c.HandleError(err, w)
			return
		}
		userDB := core.User{}
		if newUser.Email == "" { // empty mail -> do not check email
			c.ormDB.Set("gorm:auto_preload", true).Where("username=?", newUser.Username).First(&userDB)
		} else {
			c.ormDB.Set("gorm:auto_preload", true).Where("username=? OR email=?", newUser.Username, newUser.Email).First(&userDB)
		}

		if userDB.ID > 0 {
			err := errors.New("Username or E-Mail already exists")
			c.HandleError(err, w)
			return
		}

		if c.ormDB.NewRecord(newUser) {
			newUser.RegisteredAt.Time = time.Now()
			newUser.RegisteredAt.Valid = true
			newUser.UserType = 2
			newUser.CreatedBy = user.ID
			newUser.IsActive = true
			//patient.User.IsPasswordExpired = true
			// patient.User.IsPasswordExpired = false

			newUser.PasswordX = core.GetMD5Hash(newUser.PasswordX)
			newUser.Password = newUser.PasswordX
			c.ormDB.Set("gorm:save_associations", false).Save(&newUser)
			newUser.PasswordX = ""

			accountsSession := SystemAccountsSession{}

			token, _ := core.NewV4()
			accountsSession.ID = 0
			accountsSession.SessionToken = token.String()
			accountsSession.AccountId = newUser.ID
			accountsSession.LoginTime = core.NullTime{Time: time.Now(), Valid: true}
			c.ormDB.Set("gorm:save_associations", false).Create(&accountsSession)
			newUser.Token = accountsSession.SessionToken

			(*c.Controller.Users)[newUser.Token] = newUser

			newDoctor.IsRegistered = true
			newDoctor.Postcode = practice.Postcode
			if newDoctor.Postcode != "" {
				newDoctor.Latitude, newDoctor.Longitude = core.GetOSMLatLon("", newDoctor.Postcode)
			}
			c.ormDB.Set("gorm:save_associations", false).Create(&newDoctor)
			doctorUser := &podiumbundle.DoctorUser{}
			doctorUser.DoctorId = newDoctor.ID
			doctorUser.UserId = newUser.ID
			doctorUser.User = newUser
			doctorUser.Status = 1
			c.ormDB.Set("gorm:save_associations", false).Create(&doctorUser)
			practiceDoctor := podiumbundle.PracticeDoctor{}
			practiceDoctor.Doctor = *newDoctor
			practiceDoctor.DoctorId = newDoctor.ID
			practiceDoctor.PracticeId = practice.ID
			c.ormDB.Set("gorm:save_associations", false).Create(&practiceDoctor)

			newDoctor.Users = append(newDoctor.Users, *doctorUser)
		}
	}
	c.SendJSON(w, &newDoctor, http.StatusOK)
	//Only practice
	go web3socket.SendWebsocketDataInfoMessage("Register new doctor", web3socket.Websocket_Add, web3socket.Websocket_Podiatrist, newDoctor.ID, []uint{user.ID}, nil)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) RegisterPracticeHandler(w http.ResponseWriter, r *http.Request) {
	var practice podiumbundle.Practice

	if err := c.GetContent(&practice, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.ormDB.NewRecord(&practice) {
		// check Username
		practice.User.Username = strings.TrimSpace(practice.User.Username)
		practice.User.Email = strings.TrimSpace(practice.User.Email)
		if len(practice.User.Username) < 4 {
			err := errors.New("Username to short, you need minimum 4 characters")
			c.HandleError(err, w)
			return
		}
		err := core.ValidateFormat(practice.User.Email)
		if err != nil && practice.User.Email != "" { // empty mail is ok!
			err = errors.New("E-Mail: " + err.Error())
			c.HandleError(err, w)
			return
		}
		err = core.ValidatePassword(practice.User.PasswordX)
		if err != nil {
			c.HandleError(err, w)
			return
		}
		userDB := core.User{}
		if practice.User.Email == "" { // empty mail -> do not check email
			c.ormDB.Set("gorm:auto_preload", true).Where("username=?", practice.User.Username).First(&userDB)
		} else {
			c.ormDB.Set("gorm:auto_preload", true).Where("username=? OR email=?", practice.User.Username, practice.User.Email).First(&userDB)
		}

		if userDB.ID > 0 {
			err := errors.New("Username or E-Mail already exists")
			c.HandleError(err, w)
			return
		}

		if c.ormDB.NewRecord(&practice.User) {
			practice.User.RegisteredAt.Time = time.Now()
			practice.User.RegisteredAt.Valid = true
			practice.User.UserType = 3
			practice.User.IsActive = true
			//patient.User.IsPasswordExpired = true
			// patient.User.IsPasswordExpired = false
			practice.User.Password = core.GetMD5Hash(practice.User.PasswordX)
			c.ormDB.Set("gorm:save_associations", false).Save(&practice.User)

			accountsSession := SystemAccountsSession{}

			token, _ := core.NewV4()
			accountsSession.ID = 0
			accountsSession.SessionToken = token.String()
			accountsSession.AccountId = practice.User.ID
			accountsSession.LoginTime = core.NullTime{Time: time.Now(), Valid: true}
			c.ormDB.Set("gorm:save_associations", false).Create(&accountsSession)
			practice.User.Token = accountsSession.SessionToken

			(*c.Controller.Users)[practice.User.Token] = practice.User

		}
		c.ormDB.Set("gorm:save_associations", false).Save(&practice)

		if practice.HasSameDoctor {
			newUser := practice.User
			newUser.ID = 0
			newUser.UserType = 2
			newUser.CreatedBy = practice.User.ID
			newUser.Password = core.GetMD5Hash(practice.User.PasswordX)
			c.ormDB.Set("gorm:save_associations", false).Create(&newUser)
			//c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", newUser.PasswordX, newUser.ID)

			doctor := &podiumbundle.Doctor{}
			doctor.Postcode = practice.Postcode
			if doctor.Postcode != "" {
				doctor.Latitude, doctor.Longitude = core.GetOSMLatLon("", doctor.Postcode)
			}
			doctor.IsRegistered = true
			c.ormDB.Set("gorm:save_associations", false).Create(&doctor)
			doctorUser := &podiumbundle.DoctorUser{}
			doctorUser.DoctorId = doctor.ID
			doctorUser.UserId = newUser.ID
			doctorUser.Status = 1
			c.ormDB.Set("gorm:save_associations", false).Create(&doctorUser)
			practiceDoctor := podiumbundle.PracticeDoctor{}
			practiceDoctor.Doctor = *doctor
			practiceDoctor.DoctorId = doctor.ID
			practiceDoctor.PracticeId = practice.ID
			c.ormDB.Set("gorm:save_associations", false).Create(&practiceDoctor)
		}
	}
	practice.User.PasswordX = ""
	c.SendJSON(w, &practice, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) LockUserHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	var lockedUser core.User

	if err := c.GetContent(&lockedUser, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	fullLockedUser := core.User{}
	c.ormDB.First(&fullLockedUser, lockedUser.ID)

	if c.isPractice(user) {
		c.ormDB.Exec("UPDATE system_accounts SET is_active = 0 WHERE id = ? AND created_by = ?", lockedUser.ID, user.ID)
	}

	for key, user := range *c.Users {
		if user.ID == lockedUser.ID {
			user.IsActive = false
			(*c.Users)[key] = user
		}
	}

	c.SendJSON(w, &lockedUser, http.StatusOK)

	ids := []uint{lockedUser.ID, user.ID}

	switch fullLockedUser.UserType {

	case core.UserTypeDoctor:
		break
	case core.UserTypeSystem:
		break
	case core.UserTypePatient:
		go web3socket.SendWebsocketDataInfoMessage("Patient locked", web3socket.Websocket_Patients, web3socket.Websocket_Patients, lockedUser.ID, ids, nil)
		break
	case core.UserTypePractice:
		break
	}

	go web3socket.SendWebsocketDataInfoMessage("Account locked", web3socket.Websocket_Update, web3socket.Websocket_Account_Locked, lockedUser.ID, ids, nil)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) UnlockUserHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	var lockedUser core.User

	if err := c.GetContent(&lockedUser, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	fullLockedUser := core.User{}
	c.ormDB.First(&fullLockedUser, lockedUser.ID)

	if c.isPractice(user) {
		c.ormDB.Exec("UPDATE system_accounts SET is_active = 1 WHERE id = ? AND created_by = ?", lockedUser.ID, user.ID)
	}

	c.SendJSON(w, &lockedUser, http.StatusOK)

	ids := []uint{lockedUser.ID, user.ID}

	switch fullLockedUser.UserType {

	case core.UserTypeDoctor:
		break
	case core.UserTypeSystem:
		break
	case core.UserTypePatient:
		go web3socket.SendWebsocketDataInfoMessage("Patient unlocked", web3socket.Websocket_Patients, web3socket.Websocket_Patients, lockedUser.ID, ids, nil)
		break
	case core.UserTypePractice:
		break
	}

	go web3socket.SendWebsocketDataInfoMessage("Account unlocked", web3socket.Websocket_Update, web3socket.Websocket_Account_Unlocked, lockedUser.ID, ids, nil)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) RequestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {
	helperUser := &ResetPasswordHelper{}
	response := &core.ResponseData{}

	if err := c.GetContent(&helperUser, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	users := &core.Users{}

	c.ormDB.Where("username = ?", helperUser.Username).Find(&users)

	if len(*users) == 0 {
		response.Message = "Username not found"
		response.Status = 999
	} else {
		for _, user := range *users {
			if user.ID == 0 {
				response.Message = "Username not found"
				response.Status = 999
			} else if user.Email != "" {
				resetData := &SystemAccountPasswordReset{}
				resetData.User = user
				resetData.UserId = user.ID
				resetData.IsValid = true
				resetData.Token = core.GetMD5Hash(time.Now().String())

				c.ormDB.Model(&SystemAccountPasswordReset{}).Where("user_id = ?", user.ID).Update("is_valid", false)
				c.ormDB.Create(&resetData)

				to := []string{user.Email}
				portalAddress := "https://podiumfrontend.z33.web.core.windows.net"
				if core.Config.Portal.Address != "" {
					portalAddress = core.Config.Portal.Address
				}

				fullText := ""
				if c.isPatient(&user) {
					fullText = fmt.Sprintf(`Click <a href="%s/password-reset/%s">here</a> to reset password`, portalAddress, resetData.Token)
				} else {
					tmp := "clinician"
					if c.isPractice(&user) {
						tmp = "practice"
					}
					fullText = fmt.Sprintf(`This is an automatically generated email. You have requested to reset your password for the <b  style="font-size:13px">%s portal, login = %s </b> that accompanies the podium professional.<br><a href="%s/password-reset/%s">Click here</a> to be taken to a webform to complete the process and enter a new password.`, tmp, user.Username, portalAddress, resetData.Token)
				}

				core.SendMail("info@podium.care", to, []string{}, []string{}, "Password Reset", fullText, []string{})

				response.Message = "Link to reset password sent via e-mail."
				response.Status = 100
			} else {
				response.Message = "No e-mail address linked to account, please contact your Practice"
				response.Status = 999
			}
		}
	}

	c.SendJSON(w, &response, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {

	response := &core.ResponseData{}
	validPasswordHours := 0.0
	var resetPasswordHelper ResetPasswordHelper
	if err := c.GetContent(&resetPasswordHelper, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	resetData := &SystemAccountPasswordReset{}
	c.ormDB.Where("token = ?", resetPasswordHelper.Token).Find(&resetData)
	if resetData.ID == 0 {

		response.Status = 999
		response.Message = "Not a valid Token"
		c.SendJSON(w, &response, http.StatusUnauthorized)
		return
	}
	if resetData.IsValid == false || (validPasswordHours > 0 && time.Since(resetData.CreatedAt).Hours() > validPasswordHours) {
		response.Status = 999
		response.Message = "Not a valid Token"
		c.SendJSON(w, &response, http.StatusUnauthorized)
		return
	}

	if resetPasswordHelper.Password != "" {
		err := core.ValidatePassword(resetPasswordHelper.Password)
		if err != nil {
			c.HandleError(err, w)
			return
		}
		resetPasswordHelper.Password = core.GetMD5Hash(resetPasswordHelper.Password)
		c.ormDB.Exec("UPDATE system_accounts SET password = ?, is_password_expired = ? WHERE id = ?", resetPasswordHelper.Password, false, resetData.UserId)
		c.ormDB.Model(&SystemAccountPasswordReset{}).Where("user_id = ?", resetData.UserId).Update("is_valid", false)
	}

	response.Status = 100
	response.Message = "Success"
	c.SendJSON(w, &response, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) TosAcceptedHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	user.TosAccepted = true
	c.ormDB.Debug().Model(&user).Where("id = ?", user.ID).Update("tos_accepted", true)

	c.SendJSON(w, &user, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) GetTosPdfHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/pdf")
	w.Header().Add("Access-Control-Allow-Origin", "*")

	filePath := core.GetUploadFilepath() + "system/tos.pdf"
	http.ServeFile(w, r, filePath)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) GetFrontendTranslationsHandler(w http.ResponseWriter, r *http.Request) {
	//var user *core.User

	db := c.ormDB.Debug()

	urlQuery := r.URL.Query()
	if len(urlQuery) > 0 {
		values := urlQuery
		if val, ok := values["language_code"]; ok && len(val) > 0 {
			db = db.Where("system_frontend_translation_language_id IN (SELECT id FROM system_frontend_translation_languages WHERE language_code = ?)", val)
		}
	}

	keys := SystemFrontendTranslationKeyTranslations{}

	db.Set("gorm:auto_preload", true).Find(&keys)

	c.SendJSON(w, &keys, http.StatusOK)
}

func (c *SystemController) GetServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	//var user *core.User

	config := SystemServerConfig{}
	config.CustomerServerKey = core.Config.Server.CustomerServerKey

	c.SendJSON(w, &config, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: Patient
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) GetTutroialPdfHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/pdf")
	w.Header().Add("Access-Control-Allow-Origin", "*")

	filePath := core.GetUploadFilepath() + "system/tutorial_podiatrist.pdf"
	urlQuery := r.URL.Query()
	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["isPodiatrist"]; ok && len(val) > 0 {
			filePath = core.GetUploadFilepath() + "system/tutorial_podiatrist.pdf"
		}

		if val, ok := values["isPractice"]; ok && len(val) > 0 {
			filePath = core.GetUploadFilepath() + "system/tutorial_practice.pdf"
		}
	}

	http.ServeFile(w, r, filePath)
}

func (c *SystemController) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/pdf")
	w.Header().Add("Access-Control-Allow-Origin", "*")

	filePath := core.GetUploadFilepath() + "system/tos.pdf"
	http.ServeFile(w, r, filePath)
}

func (c *SystemController) CreateNotification(userId uint, notificationType int64, title string, message string, foreignId uint, sourceUrl string, actions []podiumbundle.NotificationAction) {

	notification := podiumbundle.Notification{
		Title:            title,
		Message:          message,
		UserId:           userId,
		NotificationDate: core.NullTime{Time: time.Now(), Valid: true},
		ForeignId:        foreignId,
		SourceUrl:        sourceUrl,
		NotificationType: notificationType,
		Actions:          actions,
		Visible:          true,
	}

	c.ormDB.Set("gorm:save_associations", true).Create(&notification)

	// Idea: sent now a push notification

}

func (c *SystemController) ImportWholePracticeHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isSysadmin(user) {
		err := errors.New("only admins can upload versions")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	// If an error occurs during file save, send error and return
	filePath, err := c.saveImportFile(r)
	if err != nil {
		c.HandleError(err, w)
		return
	}

	newPractice, importError, err := c.importWholePracticeFromExcel(filePath)
	if importError == nil {
		if err != nil {
			c.HandleError(err, w)
			return
		}
		/*
			msg := core.ResponseData{
				Status:  http.StatusOK,
				Message: "Import completed."
			}*/

		c.SendJSON(w, &newPractice, http.StatusOK)
	} else {
		c.SendJSON(w, importError, http.StatusInternalServerError)
	}

	//c.ormDB.Set("gorm:save_associations", false).Create(&newVersion)

	//c.SendJSON(w, newVersion, http.StatusOK)
}

func (c *SystemController) ImportDoctorsForAdminHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isSysadmin(user) {
		err := errors.New("only admins can upload versions")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	vars := mux.Vars(r)
	practiceId, err := strconv.ParseInt(vars["practiceId"], 10, 64)
	if c.HandleError(err, w) {
		return
	}
	practiceUser := core.User{}

	c.ormDB.Where("id IN (SELECT user_id FROM practices WHERE id = ?)", practiceId).First(&practiceUser)

	c.ImportDoctors(w, r, &practiceUser)
}

func (c *SystemController) ImportDoctorsForPracticeHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isPractice(user) {
		err := errors.New("please login as practice")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	c.ImportDoctors(w, r, user)
}

func (c *SystemController) ImportPatientsForAdminHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isSysadmin(user) {
		err := errors.New("only admins can upload versions")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	vars := mux.Vars(r)
	practiceId, err := strconv.ParseInt(vars["practiceId"], 10, 64)
	if c.HandleError(err, w) {
		return
	}
	practiceUser := core.User{}

	c.ormDB.Where("id IN (SELECT user_id FROM practices WHERE id = ?)", practiceId).First(&practiceUser)

	c.ImportPatients(w, r, &practiceUser)
}

func (c *SystemController) ImportPatientsForPracticeHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isPractice(user) {
		err := errors.New("please login as practice")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	c.ImportPatients(w, r, user)
} //919e69274f8ba7f03f0a62593b8d0fe1

func (c *SystemController) ImportDoctorsPatientsForAdminHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isSysadmin(user) {
		err := errors.New("only admins can upload versions")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	vars := mux.Vars(r)
	practiceId, err := strconv.ParseInt(vars["practiceId"], 10, 64)
	if c.HandleError(err, w) {
		return
	}
	practiceUser := core.User{}

	c.ormDB.Where("id IN (SELECT user_id FROM practices WHERE id = ?)", practiceId).First(&practiceUser)

	c.ImportDoctorsPatients(w, r, &practiceUser)
}

func (c *SystemController) ImportDoctorsPatientsForPracticeHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isPractice(user) {
		err := errors.New("please login as practice")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	c.ImportDoctorsPatients(w, r, user)
}

func (c *SystemController) ImportDoctors(w http.ResponseWriter, r *http.Request, user *core.User) {
	// If an error occurs during file save, send error and return
	filePath, err := c.saveImportFile(r)
	if err != nil {
		c.HandleError(err, w)
		return
	}

	importErrors, err := c.importDoctorsFromExcel(filePath, user)
	if importErrors == nil {
		if err != nil {
			c.HandleError(err, w)
			return
		}
		msg := core.ResponseData{
			Status:  http.StatusOK,
			Message: "Import completed.",
		}
		c.SendJSON(w, &msg, http.StatusOK)
		return
	} else {
		c.SendJSON(w, importErrors, http.StatusInternalServerError)
	}
}

func (c *SystemController) ImportPatients(w http.ResponseWriter, r *http.Request, user *core.User) {
	// If an error occurs during file save, send error and return
	filePath, err := c.saveImportFile(r)
	if err != nil {
		c.HandleError(err, w)
		return
	}

	importErrors, err := c.importPatientsFromExcel(filePath, user)
	if importErrors == nil {
		if err != nil {
			c.HandleError(err, w)
			return
		}
		msg := core.ResponseData{
			Status:  http.StatusOK,
			Message: "Import completed.",
		}
		c.SendJSON(w, &msg, http.StatusOK)
		return
	} else {
		c.SendJSON(w, importErrors, http.StatusInternalServerError)
	}
}

func (c *SystemController) ImportDoctorsPatients(w http.ResponseWriter, r *http.Request, user *core.User) {
	// If an error occurs during file save, send error and return
	filePath, err := c.saveImportFile(r)
	if err != nil {
		c.HandleError(err, w)
		return
	}

	importErrors, err := c.importDoctorsAndPatientsFromExcel(filePath, user)
	if importErrors == nil {
		if err != nil {
			c.HandleError(err, w)
			return
		}
		msg := core.ResponseData{
			Status:  http.StatusOK,
			Message: "Import completed.",
		}
		c.SendJSON(w, &msg, http.StatusOK)
		return
	} else {
		c.SendJSON(w, importErrors, http.StatusInternalServerError)
	}
}

func (c *SystemController) saveImportFile(r *http.Request) (string, error) {
	//deviceType := r.FormValue("device_type")
	//version := r.FormValue("version")

	//X tmpPath := "tmp/imports/" + core.RandomString(10) + "/"
	tmpPath := c.GetTmpUploadPath()

	err := os.MkdirAll(tmpPath, 0777)

	if err != nil {
		log.Println(err)
	}
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		log.Println(err)
	}
	filePath := ""
	//var err error
	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); nil != err {
				//status = http.StatusInternalServerError
				_ = infile
				log.Println(err)
				return "", err
			}
			// open destination
			var outfile *os.File
			log.Println(hdr.Filename)
			pos := strings.LastIndex(hdr.Filename, "/")
			filename := hdr.Filename[pos+1:]
			filePath = fmt.Sprintf("%s%s", tmpPath, filename)
			if outfile, err = os.Create(filePath); nil != err {
				//status = http.StatusInternalServerError
				log.Println(err)
			}
			// 32K buffer copy
			var written int64
			if written, err = io.Copy(outfile, infile); nil != err {
				//status = http.StatusInternalServerError
				log.Println(err)
				return "", err
			}
			log.Println("uploaded file:" + hdr.Filename + ";length:" + strconv.Itoa(int(written)))
		}
	}

	return filePath, nil
}

func (c *SystemController) isSysadmin(user *core.User) bool {
	if user.UserType == 0 && user.IsSysadmin {
		return true
	}
	return false
}

func (c *SystemController) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	err, user := c.Controller.TryGetUser(w, r)

	if !err {
		err := errors.New("User not found")
		c.HandleError(err, w)
		return
	}

	if !user.IsSysadmin {
		err := errors.New("No permission to call this route")
		c.HandleError(err, w)
		return
	}

	users := core.Users{}
	paging := c.GetPaging(r.URL.Query())

	db, dbTotalCount := c.CreateWhereConditionsUsers(r.URL.Query(), r, user)
	db.Set("gorm:auto_preload", true).Limit(paging.Limit).Offset(paging.Offset).Find(&users)
	dbTotalCount.Model(&core.Users{}).Count(&paging.TotalCount)

	c.SendJSONPaging(w, r, paging, &users, http.StatusOK)
}

func (c *SystemController) CreateWhereConditionsUsers(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("username LIKE ?", search)
				dbTotalCount = dbTotalCount.Where("username LIKE ?", search)
			}
		}

		if val, ok := values["filter"]; ok && len(val) > 0 {
			tmpFilters := make(map[string][]string)
			for _, filter := range val {
				data := strings.Split(filter, ",")
				if len(data) > 1 {
					switch strings.ToLower(data[0]) {
					case "user_type":
						tmpFilters[data[0]] = append(tmpFilters[data[0]], data[1])
					}
				}
				log.Println(filter)
			}
			for key, filterData := range tmpFilters {
				switch key {
				case "user_type":
					db = db.Where("user_type IN (?)", filterData)
					dbTotalCount = dbTotalCount.Where("user_type IN (?)", filterData)
				}
			}
		}

		if val, ok := values["order"]; ok && len(val) > 0 {
			if val[0] != "" {
				if strings.Contains(val[0], ",") {
					sortSplit := strings.Split(val[0], ",")
					sortKey := sortSplit[0]
					sortDirection := sortSplit[1]
					switch sortKey {
					default:
						db = db.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						break
					}
				}
			}
		}
	}

	return db, dbTotalCount
}

func (c *SystemController) LogoHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Add("Access-Control-Allow-Origin", "*")

	vars := mux.Vars(r)
	logoType, _ := vars["logo_type"]

	filePath := core.GetUploadFilepath()

	switch logoType {
	case LogoType_MainLogo:
		filePath += "system/mainLogo.png"
	case LogoType_MainLogoWhite:
		filePath += "system/mainLogoWhite.png"
	case LogoType_PartnerLogo:
		filePath += "system/partnerLogo.png"
	case LogoType_LoginBackground:
		filePath += "system/loginBackground.svg"
	case LogoType_SmallLoginBackground:
		filePath += "system/small-background-login.png"
	}

	http.ServeFile(w, r, filePath)
}

func (c *SystemController) ExtraLogoHandler(w http.ResponseWriter, r *http.Request) {

	//w.Header().Add("Content-Type", "application/pdf")
	//w.Header().Add("Access-Control-Allow-Origin", "*")
	//w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	//w.Header().Add("Access-Control-Allow-Origin", "*")
	//w.Header().Add("Access-Control-Allow-Origin", "*")

	//w.Header().Set("Content-Disposition", `inline; filename="measurement`+strconv.Itoa(int(measurement.ID))+`_`+file.MeasurementType+`.jpg"`)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Add("Access-Control-Allow-Origin", "*")
	filePath := "uploads/system/extra_logo.png"
	http.ServeFile(w, r, filePath)
	return

	//filePath := "uploads/system/extra_logo.png"
	//http.ServeFile(w, r, filePath)
	//c.SendFile(w, r, filePath)
}
