package podiumbundle

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
	"thermetrix_backend/app/core"
	"time"
)

// GetMyConsents swagger:route GET /me/consents consent getMyConsents
//
// retrieves all Consents
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
//			data: []DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyConsentsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	consents := &DoctorPatientRelations{}
	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Find(&consents)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ?", doctor.ID).Find(&consents)
	}

	c.SendJSON(w, &consents, http.StatusOK)
}

// GetMyPendingConsents swagger:route GET /me/consents/pending consent getMyPendingConsents
//
// retrieves all pending Consents
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
//			data: []DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyPendingConsentsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	consents := &DoctorPatientRelations{}
	if c.isPatient(user) {
		c.ormDB.Set("gorm:auto_preload", true).Where("").Where("patient_id = ?", user.ID).Where("consent_status = 1").Where("consent_type = 2").Find(&consents)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("").Where("doctor_id = ?", doctor.ID).Where("consent_status = 1").Where("consent_type = 1").Find(&consents)
	}

	c.SendJSON(w, &consents, http.StatusOK)
}

// SaveConsent swagger:route POST /me/consents consent saveConsent
//
// Saves a consent
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
//			data: DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) SaveConsentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	consent := &DoctorPatientRelation{}
	if err := c.GetContent(&consent, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if consent.ID == 0 {
		consent.ConsentStatus = 1
		consent.ConsentDate.Time = time.Now()
		consent.ConsentDate.Valid = true
	}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		consent.PatientId = patient.ID
		consent.Patient.ID = consent.PatientId
		consent.ConsentType = 1
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		consent.DoctorId = doctor.ID
		consent.Doctor.ID = consent.DoctorId
		consent.ConsentType = 2
	}

	oldConsent := &DoctorPatientRelation{}

	c.ormDB.Where("patient_id = ?", consent.Patient.ID).Where("doctor_id = ?", consent.Doctor.ID).Order("consent_status desc").First(&oldConsent)

	if oldConsent.ConsentStatus > 0 {
		c.HandleError(errors.New("Already have consent"), w)
		return
	}

	c.ormDB.Set("gorm:save_associations", false).Save(&consent)

	c.ormDB.Model(&Doctor{}).Where("id = ?", consent.DoctorId).Update("has_home_users", true)

	c.SendJSON(w, &consent, http.StatusOK)
	//go systembundle.SendWebsocketDataInfoMessage("Update message", systembundle.Websocket_Add, systembundle.Websocket_Messages, message.ID, 0, nil)
}

// AcceptConsent swagger:route PATCH /me/consents/{consentId}/accept consent acceptConsent
//
// Accepts a consent
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
//	+ name: consentId
//    in: path
//    description: the ID of the consent
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) AcceptConsentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	consent := &DoctorPatientRelation{}
	vars := mux.Vars(r)
	consentId, _ := strconv.ParseInt(vars["consentId"], 10, 64)

	c.ormDB.Set("gorm:auto_preload", true).Find(&consent, consentId)

	if consent.ConsentStatus == 1 {
		doctor := &Doctor{}
		if consent.ConsentType == 1 {
			doctor = c.getDoctor(user)
			if doctor == nil || doctor.ID != consent.DoctorId {
				return
			}

		} else if consent.ConsentType == 2 {
			patient := c.getPatient(user)
			if patient == nil || patient.ID != consent.PatientId {
				return
			}
			c.ormDB.First(&doctor, &consent.Doctor.ID)
		}
		c.ormDB.Model(&DoctorPatientRelation{}).Where("id = ?", consentId).Update("consent_status", 2)
		consent.ConsentStatus = 2

		welcomeMessage := doctor.StandardWelcomeMessage
		if strings.TrimSpace(welcomeMessage) == "" {
			welcomeMessage = "Hello"
		}
		c.ormDB.Set("gorm:auto_preload", true).First(&consent.Patient)

		message := Message{
			DoctorId:    consent.Doctor.ID,
			IsUnread:    true,
			MessageText: doctor.StandardWelcomeMessage,
			SenderId:    user.ID,
			Sender:      *user,
			RecipientId: consent.Patient.User.ID,
			Recipient:   consent.Patient.User,
			MessageTime: core.NullTime{Time: time.Now(), Valid: true},
		}
		c.ormDB.Set("gorm:save_associations", false).Create(&message)
		c.CreateNotification(message.Recipient.ID, 1, fmt.Sprintf("New Message from %s", message.Sender.Username), message.MessageText, message.ID, fmt.Sprintf("/me/conversations/%d", message.Sender.ID), nil)

	}

	c.ormDB.Model(&Patient{}).Where("id=?", consent.Patient.ID).Update("has_paired_podiatrist", true)

	c.SendJSON(w, &consent, http.StatusOK)

}

// DeclineConsent swagger:route PATCH /me/consents/{consentId}/decline consent declineConsent
//
// Declines a consent
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
//	+ name: consentId
//    in: path
//    description: the ID of the consent
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) DeclineConsentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	consent := &DoctorPatientRelation{}

	vars := mux.Vars(r)
	consentId, _ := strconv.ParseInt(vars["consentId"], 10, 64)

	c.ormDB.Set("gorm:auto_preload", true).Find(&consent, consentId)

	if consent.ConsentStatus == 1 {
		if consent.ConsentType == 1 {
			doctor := c.getDoctor(user)
			if doctor == nil || doctor.ID != consent.DoctorId {
				return
			}
		} else if consent.ConsentType == 2 {
			patient := c.getPatient(user)
			if patient == nil || patient.ID != consent.PatientId {
				return
			}
		}
		c.ormDB.Model(&DoctorPatientRelation{}).Where("id = ?", consentId).Update("consent_status", 3)
		consent.ConsentStatus = 3
	}
	c.SendJSON(w, &consent, http.StatusOK)
}

// DeleteConsent swagger:route DELETE /me/consents/{consentId} consent deleteConsent
//
// Deletes a consent
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
//	+ name: consentId
//    in: path
//    description: the ID of the consent
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) DeleteConsentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	consent := &DoctorPatientRelation{}

	vars := mux.Vars(r)
	consentId, _ := strconv.ParseInt(vars["consentId"], 10, 64)

	c.ormDB.Delete(&DoctorPatientRelation{}, consentId)

	c.SendJSON(w, &consent, http.StatusOK)
}

// SaveConsent swagger:route POST /me/consents consent saveConsent
//
// Saves a consent
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
//			data: DoctorPatientRelation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) SaveDoctorApproachHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	approach := &DoctorApproach{}
	if err := c.GetContent(&approach, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.isPatient(user) {
		approach.Patient = *c.getPatient(user)
		approach.PatientId = approach.Patient.ID
	} else {
		c.HandleError(errors.New("Only Patients can approach Doctors"), w)
		return
	}

	c.ormDB.Set("gorm:save_associations", false).Create(&approach)

	if approach.ApproachType == "INVITE" {
		doctor := Doctor{}
		c.ormDB.Find(&doctor, approach.Doctor.ID)
		core.SendMail("info@podium.care", []string{doctor.Email}, []string{}, []string{}, "A Patient would like to see you",
			fmt.Sprintf(`
Dear %s,<br/><br/>

A Podium home user has requested to pair with you on their Podium system, following a search of local clinicians (powered by podsfixfeet).  Our database shows that you have not yet purchased a Podium or have access to the accompanying portal.
If you would like more information of how to become a Podium partner, please contact us on info@podium.care, 01443 805769 or visit our website www.podium.care.
We have passed on your contact details for them to contact you directly.<br/>
Kind Regards<br/><br/>
The Podium Care Team
`, doctor.Name), []string{})

		now := time.Now()
		countApproaches := 0
		c.ormDB.Model(&DoctorApproach{}).Where("doctor_id = ? AND approach_type = 'INVITE'", doctor.ID).Count(&countApproaches)
		core.SendMail("info@podium.care", []string{"portal@thermetrix.com "}, []string{}, []string{}, "Podium home user pairing report",
			fmt.Sprintf(`
The following patient:
%s<br/>
Tried to pair with a non-podium podiatrist:
%s, %s<br/>
On %s<br/>
This clinician has been requested %d times.`, user.Username, doctor.Name, doctor.Email, now.Format("01.02.2006 15:04"), countApproaches), []string{})
	}

	c.SendJSON(w, &approach, http.StatusOK)
}
