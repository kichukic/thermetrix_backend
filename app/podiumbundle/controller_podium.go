package podiumbundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	tools "github.com/kirillDanshin/nulltime"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"thermetrix_backend/app/core"
	web3socket "thermetrix_backend/app/websocket"
	"time"
)

// PodiumController struct
type PodiumController struct {
	core.Controller
	ormDB *gorm.DB
}

// NewPodiumController instance
func NewPodiumController(ormDB *gorm.DB, users *map[string]core.User) *PodiumController {

	c := &PodiumController{
		Controller: core.Controller{Users: users},
		//hm:         hm,
		ormDB: ormDB,
	}

	if core.Config.Database.DoAutoMigrate {
		ormDB.AutoMigrate(&DoctorPatientRelation{}, &Doctor{}, &DoctorUser{}, &RewardStarRating{}, &RewardMonetaryDiscount{}, &RewardUserLevel{}, &Patient{}, &PatientReward{}, &SpecialistField{}, &SpecialistField{}, &Appointment{}, &Annotation{}, &Measurement{}, &MeasurementFile{}, &MeasurementFavorite{}, &Device{}, &Sensor{}, &DeviceType{}, &DeviceTypeVersion{}, &SensorType{}, &Message{}, &AppointmentStatus{}, &AppointmentStatusDef{})
		ormDB.AutoMigrate(&MeasurementDoctorRisk{})
		ormDB.AutoMigrate(&QuestionTemplate{}, &QuestionTemplateAnswer{}, &PatientQuestionnaire{}, &PatientQuestionnaireQuestion{})
		ormDB.AutoMigrate(&RiskCalculation{}, &Notification{}, &NotificationAction{}, &RiskDefinition{})
		ormDB.AutoMigrate(&Podiatrist{}, &MeasurementShared{}, &DoctorApproach{})
		ormDB.AutoMigrate(&Practice{}, &PracticeDevice{}, &PracticeContract{}, &PracticeDoctor{})
		ormDB.AutoMigrate(&DoctorDevice{}, &PatientDevice{}, &DeviceStatus{})
		ormDB.AutoMigrate(&DeviceSystemVersionType{}, &DeviceSystemVersion{})
		ormDB.AutoMigrate(&PatientImages{})
		ormDB.AutoMigrate(&CoverLetter{})
		ormDB.AutoMigrate(&notes{})

		c.insertRiskDefinitions()
		c.insertAppointmentStatusDefs()
		c.insertDeviceTypes()
		c.insertRewardMonetaryDiscounts()
		c.insertRewardStarRatings()
		c.insertStandardQuestionTemplates()
		c.insertStandardQuestionTemplateAnswers()
		c.insertRewardUserLevels()

		//Nächsten Update löschen
		if core.Config.Database.InitSetIsTenMinsApart {
			c.InitSetIsTenMinsApart()
		}

		//test command
	}

	return c
}

// GetDoctors swagger:route GET /doctors doctor GetDoctors
//
// retrieves all doctors
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDoctors4AppHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	doctors := Doctors{}
	db := c.ormDB.Set("gorm:auto_preload", true)

	// read filters from get paranms

	if len(r.URL.Query()) > 0 {
		values := r.URL.Query()
		if values["search"][0] != "" {
			addresses, err := core.GetOSMObjects("", values["search"][0])
			if err == nil && len(addresses) > 0 {
				maxDistance := 50 // in KM
				for len(doctors) == 0 && maxDistance < 150 {
					db.Select("*, (ACOS(SIN(PI()*latitude/180.0)*SIN(PI()*?/180.0)+COS(PI()*latitude/180.0)*COS(PI()*?/180.0)*COS(PI()*?/180.0-PI()*longitude/180.0))*6371) as distance", addresses[0].Lat, addresses[0].Lat, addresses[0].Lon).Where("name LIKE concat('%', ?, '%') OR postcode LIKE concat('%', ?, '%') OR town LIKE concat('%', ?, '%') OR address_line1 LIKE concat('%', ?, '%') OR address_line2 LIKE concat('%', ?, '%')", values["search"][0], values["search"][0], values["search"][0], values["search"][0], values["search"][0]).Or("id IN (SELECT du.doctor_id FROM doctor_users du LEFT JOIN system_accounts sa ON du.user_id = sa.id WHERE username LIKE concat('%', ?, '%') )", values["search"][0]).Or("(ACOS(SIN(PI()*latitude/180.0)*SIN(PI()*?/180.0)+COS(PI()*latitude/180.0)*COS(PI()*?/180.0)*COS(PI()*?/180.0-PI()*longitude/180.0))*6371) < ?", addresses[0].Lat, addresses[0].Lat, addresses[0].Lon, maxDistance).Order("distance ASC").Find(&doctors)
					maxDistance += 50
				}
			} else {
				db.Where("name LIKE concat('%', ?, '%') OR postcode LIKE concat('%', ?, '%') OR town LIKE concat('%', ?, '%') OR address_line1 LIKE concat('%', ?, '%') OR address_line2 LIKE concat('%', ?, '%')", values["search"][0], values["search"][0], values["search"][0], values["search"][0], values["search"][0]).Or("id IN (SELECT du.doctor_id FROM doctor_users du LEFT JOIN system_accounts sa ON du.user_id = sa.id WHERE username LIKE concat('%', ?, '%') )", values["search"][0]).Find(&doctors)
			}
		}
	} else {
		db.Find(&doctors)

	}
	c.SendJSON(w, &doctors, http.StatusOK)
}

// GetDoctors swagger:route GET /doctors doctor GetDoctors
//
// retrieves all doctors
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDoctorsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	doctors := Doctors{}
	paging := c.GetPaging(r.URL.Query())

	db, dbTotalCount := c.CreateWhereConditionsDoctors(r.URL.Query(), r, user)
	db.Set("gorm:auto_preload", true).Limit(paging.Limit).Offset(paging.Offset).Find(&doctors)
	dbTotalCount.Model(&Doctors{}).Count(&paging.TotalCount)

	for key, doctor := range doctors {
		doctor.Practice = doctor.GetPractice(c.ormDB)
		doctors[key] = doctor
	}

	c.SendJSONPaging(w, r, paging, &doctors, http.StatusOK)
}

func (c *PodiumController) CreateWhereConditionsDoctors(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("CONCAT(last_name, first_name) LIKE ? OR `doctors`.name LIKE ? OR `doctors`.postcode LIKE ? OR town LIKE ? OR address_line1 LIKE ? OR address_line2 LIKE ?", search, search, search, search, search, search).Or("`doctors`.id IN (SELECT doctor_id FROM doctor_users WHERE user_id IN (SELECT id FROM system_accounts WHERE username LIKE ? ))", search).Or("`doctors`.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (SELECT id FROM practices WHERE `practices`.name LIKE ?))", search)
				dbTotalCount = dbTotalCount.Where("CONCAT(last_name, first_name) LIKE ? OR `doctors`.name LIKE ? OR `doctors`.postcode LIKE ? OR town LIKE ? OR address_line1 LIKE ? OR address_line2 LIKE ?", search, search, search, search, search, search).Or("`doctors`.id IN (SELECT doctor_id FROM doctor_users WHERE user_id IN (SELECT id FROM system_accounts WHERE username LIKE ? ))", search).Or("`doctors`.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (SELECT id FROM practices WHERE `practices`.name LIKE ?))", search)
			}
		}

		if val, ok := values["filter"]; ok && len(val) > 0 {
			var practiceIds []int
			for _, filter := range val {
				filterSplit := strings.Split(filter, ",")
				filterKey := filterSplit[0]
				filterValue := filterSplit[1]
				switch filterKey {
				case "practices":
					if practiceId, err := strconv.Atoi(filterValue); err != nil {
						log.Println(err)
					} else {
						practiceIds = append(practiceIds, practiceId)
					}
					break
				default:
					break
				}
			}
			if len(practiceIds) > 0 {
				db = db.Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (?))", practiceIds)
				dbTotalCount = dbTotalCount.Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (?))", practiceIds)
			}
		}

		if val, ok := values["order"]; ok && len(val) > 0 {
			if val[0] != "" {
				if strings.Contains(val[0], ",") {
					sortSplit := strings.Split(val[0], ",")
					sortKey := sortSplit[0]
					sortDirection := sortSplit[1]
					switch sortKey {
					case "username":
						db = db.Joins("LEFT JOIN doctor_users AS du ON du.doctor_id = `doctors`.id").Joins("LEFT JOIN system_accounts AS sa ON sa.id = du.user_id")
						db = db.Order(fmt.Sprintf("sa.username %s", sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("sa.username %s", sortDirection))
						break
					case "practice":
						db = db.Joins("LEFT JOIN practice_doctors AS pd ON pd.doctor_id = `doctors`.id").Joins("LEFT JOIN practices AS p ON p.id = pd.practice_id")
						db = db.Order(fmt.Sprintf("p.name %s", sortDirection))
						break
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

func (c *PodiumController) CreateWhereConditionsMyDoctors(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("doctors.email LIKE ? OR CONCAT(last_name, first_name) LIKE ? OR name LIKE ? OR postcode LIKE ? OR town LIKE ? OR address_line1 LIKE  ? OR address_line2 LIKE  ? OR doctors.id IN (SELECT du.doctor_id FROM doctor_users du LEFT JOIN system_accounts sa ON du.user_id = sa.id WHERE username LIKE ? ) OR (SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = doctors.id AND ten_mins_apart AND deleted_at IS NULL))) LIKE ? OR (SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = doctors.id AND deleted_at IS NULL))) LIKE ?", search, search, search, search, search, search, search, search, search, search)
				//db = db.Where("CONCAT(last_name, first_name) LIKE ? OR name LIKE ? OR postcode LIKE ? OR town LIKE ? OR address_line1 LIKE ? OR address_line2 LIKE ?  OR doctors.id IN (SELECT du.doctor_id FROM doctor_users du LEFT JOIN system_accounts sa ON du.user_id = sa.id WHERE username LIKE ? )", search, search, search, search, search, search, search,search)
				//db = db.Or("", search)

				dbTotalCount = dbTotalCount.Where("doctors.email LIKE ? OR CONCAT(last_name, first_name) LIKE ? OR name LIKE ? OR postcode LIKE ? OR town LIKE ? OR address_line1 LIKE  ? OR address_line2 LIKE  ? OR doctors.id IN (SELECT du.doctor_id FROM doctor_users du LEFT JOIN system_accounts sa ON du.user_id = sa.id WHERE username LIKE ? ) OR (SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = doctors.id AND ten_mins_apart AND deleted_at IS NULL))) LIKE ? OR (SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = doctors.id AND deleted_at IS NULL))) LIKE ?", search, search, search, search, search, search, search, search, search, search)
				//dbTotalCount = dbTotalCount.Or("", search)
			}
		}

		if val, ok := values["order"]; ok && len(val) > 0 {
			if val[0] != "" {
				if strings.Contains(val[0], ",") {
					sortSplit := strings.Split(val[0], ",")
					sortKey := sortSplit[0]
					sortDirection := sortSplit[1]
					switch sortKey {
					case "count_successful_scans":
						if c.isPractice(user) {
							query := fmt.Sprintf("(SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND user_id = %v AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = doctors.id AND ten_mins_apart AND deleted_at IS NULL))) %s", user.ID, sortDirection)
							db = db.Order(query)
						}

					//	query :=  fmt.Sprintf("(SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND user_id = %v  AND (doctor_id > 0 AND measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND deleted_at IS NULL AND measurements.patient_id = patient.id))) %s", user.ID, sortDirection)

					//c.ormDB.Where("user_id = ? AND patient_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND deleted_at IS NULL))", user.ID, patient.ID, doctor.ID).Order("measurement_date ASC").Find(&measurements)
					case "count_scans":
						//db = db.Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						//
						//db = db.Joins("LEFT JOIN doctor_users ON doctors.id = doctor_users.doctor_id").Joins("Left JOIN system_accounts ON doctor_users.user_id = system_accounts.id")
						if c.isPractice(user) {
							query := fmt.Sprintf("(SELECT COUNT(*) FROM measurements WHERE deleted_at IS NULL AND user_id = %v AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = doctors.id AND deleted_at IS NULL))) %s", user.ID, sortDirection)
							db = db.Order(query)
						}

						break
					case "username":
						//db = db.Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						db = db.Joins("LEFT JOIN doctor_users ON doctors.id = doctor_users.doctor_id").Joins("Left JOIN system_accounts ON doctor_users.user_id = system_accounts.id")
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						break
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

// getMyDoctors swagger:route GET /me/doctors doctor getMyDoctors
//
// retrieves all linked doctors
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyDoctorsHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isPatient(user) && !c.isPractice(user) {
		return
	}

	doctors := Doctors{}
	if c.isPatient(user) {
		patient := c.getPatient(user)

		// read filters from get paranms

		if len(r.URL.Query()) > 0 {
			values := r.URL.Query()
			//Für was ist das??
			if values["search"][0] != "" {
				c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id =?)", patient.ID).Where("name LIKE concat('%', ?, '%') OR postcode LIKE concat('%', ?, '%') OR town LIKE concat('%', ?, '%') OR address_line1 LIKE concat('%', ?, '%') OR address_line2 LIKE concat('%', ?, '%') ", values["search"][0], values["search"][0], values["search"][0], values["search"][0], values["search"][0]).Find(&doctors)
			} else {
				c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id =?)", patient.ID).Find(&doctors)
			}
		} else {
			c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id =?)", patient.ID).Find(&doctors)
		}

		c.SendJSON(w, &doctors, http.StatusOK)
		return
	} else if c.isPractice(user) {
		practice := c.getPractice(user)

		paging := c.GetPaging(r.URL.Query())
		db, dbTotalCount := c.CreateWhereConditionsMyDoctors(r.URL.Query(), r, user)

		db = db.Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID)
		dbTotalCount = dbTotalCount.Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID)

		/*if len(r.URL.Query()) > 0 {
			values := r.URL.Query()
			if values["search"][0] != "" {
			//	db = db.Where("name LIKE concat('%', ?, '%') OR postcode LIKE concat('%', ?, '%') OR town LIKE concat('%', ?, '%') OR address_line1 LIKE concat('%', ?, '%') OR address_line2 LIKE concat('%', ?, '%') ", values["search"][0], values["search"][0], values["search"][0], values["search"][0], values["search"][0])
				//dbTotalCount= dbTotalCount.Where("name LIKE concat('%', ?, '%') OR postcode LIKE concat('%', ?, '%') OR town LIKE concat('%', ?, '%') OR address_line1 LIKE concat('%', ?, '%') OR address_line2 LIKE concat('%', ?, '%') ", values["search"][0], values["search"][0], values["search"][0], values["search"][0], values["search"][0])
			}
		}*/

		db.Debug().Set("gorm:auto_preload", true).Limit(paging.Limit).Offset(paging.Offset).Find(&doctors)
		dbTotalCount.Model(&Doctor{}).Count(&paging.TotalCount)

		clientId := r.Header.Get("Client")
		getSpecialData := false

		if len(r.URL.Query()) > 0 {
			values := r.URL.Query()
			//Für was ist das??
			if val, ok := values["special_data"]; ok && len(val) > 0 {
				getSpecialData, _ = strconv.ParseBool(val[0])
			}
		}
		for i, doctor := range doctors { //USER_ID = 545,  Doctor = 385
			c.ormDB.Debug().Model(&Measurement{}).Where("user_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND deleted_at IS NULL))", user.ID, doctor.ID).Count(&doctor.CountScans)
			c.ormDB.Debug().Model(&Measurement{}).Where("user_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND ten_mins_apart AND deleted_at IS NULL))", user.ID, doctor.ID).Count(&doctor.CountSuccessfulScans)

			if clientId == core.Client_APK_Pro && getSpecialData {
				for usersKey, _ := range doctor.Users {
					doctor.Users[usersKey].User.PasswordX = core.GetMD5Hash(doctor.Users[usersKey].User.Password)
					if false {

					}
				}
			}

			//c.ormDB.Model(&Measurement{}).Where("user_id = ? AND doctor_id = ?", user.ID, doctor.ID).Group("patient_id").Count(&doctor.CountSuccessfulScans)

			/*
				patients := Patients{}
				//Gib mir alle Patienten von der Praxis, die einen Scan haben
				c.ormDB.Debug().Where("id IN (SELECT patient_id FROM measurements WHERE user_id = ?)", user.ID).Find(&patients)
				tmp := "("
				//count:= 0
				for _, patient := range patients {
					measurements := Measurements{}
					lastGoodMeasurement := Measurement{}
					//Gib mir die Scans, wo die Praxis und Patient übereinstimmen und der doctor zugriff haben
					c.ormDB.Debug().Where("user_id = ? AND patient_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND deleted_at IS NULL))", user.ID, patient.ID, doctor.ID).Order("measurement_date ASC").Find(&measurements)
					/ *for _, measurement := range measurements {
						if  measurement.MeasurementDate.Time.Unix()-lastGoodMeasurement.MeasurementDate.Time.Unix() > 600 {

							tmp += fmt.Sprintf("%v, ", measurement.ID)
							doctor.CountSuccessfulScans++
							lastGoodMeasurement = measurement
							log.Println(doctor.CountSuccessfulScans)
							log.Println(measurement.ID)
						}
					}
				* /
				tmp += ")"
				log.Println(tmp)

				}*/

			doctors[i] = doctor
		}
		c.SendJSONPaging(w, r, paging, &doctors, http.StatusOK)
		return
		//c.SendJSON(w, &doctors, http.StatusOK)
	} //2020-12-18 16:38:50, patient: 340
}
func (c *PodiumController) GetDoctorsSpecificHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isPractice(user) {

		return
	}

	practice := c.getPractice(user)

	var users core.Users
	_ = users

	db := c.ormDB
	db = db.Preload("Users").Preload("Users.User")
	doctors := practice.getSmallDoctors(db)

	for key, _ := range doctors {
		for usersKey, _ := range doctors[key].Users {
			doctors[key].Users[usersKey].User.PasswordX = core.GetMD5Hash(doctors[key].Users[usersKey].User.Password)
		}
	}

	c.SendJSON(w, &doctors, http.StatusOK)

	return

	//c.SendJSON(w, &doctors, http.StatusOK)
} 

//2020-12-18 16:38:50, patient: 340

// getMyPatients swagger:route GET /me/patients patient getMyPatients
//
// retrieves your patients
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyDoctorPatientsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	doctorId, _ := strconv.ParseInt(vars["doctorId"], 10, 64)

	patients := Patients{}
	if c.isPractice(user) {
		c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ?)", doctorId).Find(&patients) // nur  angenommene
	}

	c.SendJSON(w, &patients, http.StatusOK)
}

// getDoctor swagger:route GET /doctors/{doctorId} doctor getDoctor
//
// retrieves informations about a doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: doctorId
//     in: path
//     description: the ID for the doctor
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDoctorHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	doctorId, err := strconv.ParseInt(vars["doctorId"], 10, 64)
	doctor := Doctor{}

	c.ormDB.Set("gorm:auto_preload", true).First(&doctor, doctorId)
	if c.HandleError(err, w) {
		return
	}

	doctor.Practice = doctor.GetPractice(c.ormDB.Preload("User"))

	c.SendJSON(w, &doctor, http.StatusOK)
}

// getDoctor swagger:route GET /doctors/{doctorId} doctor getDoctor
//
// retrieves informations about a doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: doctorId
//     in: path
//     description: the ID for the doctor
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveDoctorHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok || (c.isPractice(user) && c.isDoctor(user)) {
		_ = user
		return
	}

	doctor := Doctor{}
	if err := c.GetContent(&doctor, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if !doctor.Validate(c.ormDB) {
		log.Println(doctor.Errors)
		c.SendErrors(w, doctor.Errors, http.StatusBadRequest)
		return
	}

	if doctor.Postcode != "" {
		doctor.Latitude, doctor.Longitude = core.GetOSMLatLon("", doctor.Postcode)
	}
	userIds := []uint{}

	for i, doctorUser := range doctor.Users {
		if c.ormDB.NewRecord(&doctorUser.User) {
			// As requested by TMX, usernames should only be unique within each practice
			doctorUser.User.Username = fmt.Sprintf("%s_%s", doctor.Practice.AccountsAbbreviation, doctorUser.User.Username)
			doctor.Users[i] = doctorUser
		}
	}

	/*
		c.ormDB.Set("gorm:save_associations", false).Save(&doctor)

		for i, doctorUser := range doctor.Users {
			userIds = append(userIds, doctorUser.User.ID)
			if doctorUser.User.PasswordX != "" {
				err := core.ValidatePassword(doctorUser.User.PasswordX)
				if err != nil {
					c.HandleError(err, w)
					return
				}
				doctorUser.User.PasswordX = core.GetMD5Hash(doctorUser.User.PasswordX)
				c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", doctorUser.User.PasswordX, doctorUser.User.ID)
				doctorUser.User.IsPasswordExpired = false
			}
			doctor.Users[i] = doctorUser
		}*/

	if _, err := doctor.Save(c.ormDB); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	c.SendJSON(w, &doctor, http.StatusOK)

	if c.isPractice(user) {
		userIds = append(userIds, user.ID)
	}
	//Der Doctor und die Praxis
	go web3socket.SendWebsocketDataInfoMessage("Update doctor", web3socket.Websocket_Update, web3socket.Websocket_Podiatrist, doctor.ID, userIds, nil)
}

func (c *PodiumController) SaveAccountForDoctorHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	doctorId, err := strconv.ParseInt(vars["doctorId"], 10, 64)
	log.Println(err)
	if c.HandleError(err, w) {
		return
	}

	userPayload := core.User{}
	if err = c.GetContent(&userPayload, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	doctorDb := Doctor{}
	c.ormDB.First(&doctorDb, doctorId)

	if doctorDb.ID == 0 {
		err = errors.New("doctor not found")
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	userPayload.Save(c.ormDB)
	if _, err = userPayload.Save(c.ormDB); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	doctorUser := DoctorUser{}
	c.ormDB.Where("doctor_id=? AND user_id=?", doctorId, userPayload.ID).First(&doctorUser)
	if doctorUser.DoctorId == 0 {
		doctorUser.DoctorId = uint(doctorId)
		doctorUser.UserId = userPayload.ID
		if doctorUser.Status == 0 {
			doctorUser.Status = DoctorUserStatus_Doctor
		}
		c.ormDB.Set("gorm:save_associations", false).Create(&doctorUser)
	} else {
		c.ormDB.Set("gorm:save_associations", false).Save(&doctorUser)
	}

	c.SendJSON(w, &userPayload, http.StatusOK)
	go web3socket.SendBroadCastWebsocketDataInfoMessage("Updated doctor account", web3socket.Websocket_Update, web3socket.Websocket_UserAccount, userPayload.ID, nil)
}

func (c *PodiumController) getPracticeOfDoctor(doctorId uint) Practice {
	practice := Practice{}
	c.ormDB.Debug().
		Preload("User").
		Where("id IN (SELECT practice_doctors.practice_id FROM practice_doctors WHERE practice_doctors.doctor_id = ? AND ISNULL(practice_doctors.deleted_at))", doctorId).
		First(&practice)
	return practice
}

// getDoctor swagger:route GET /doctors/{doctorId} doctor getDoctor
//
// retrieves informations about a doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: doctorId
//     in: path
//     description: the ID for the doctor
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) DeleteDoctorHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok || !c.isPractice(user) {
		_ = user
		return
	}

	vars := mux.Vars(r)
	doctorId, _ := strconv.ParseInt(vars["doctorId"], 10, 64)
	doctor := Doctor{}
	practiceDoctor := PracticeDoctor{}
	practice := c.getPractice(user)

	c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?)", practice.ID).Delete(&doctor, doctorId)

	c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? AND practice_id = ?", doctorId, practice.ID).Delete(&practiceDoctor)

	c.ormDB.Model(&core.User{}).Set("gorm:auto_preload", true).Where("id IN (SELECT user_id FROM doctor_users WHERE doctor_id = ? AND  doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", doctorId, practice.ID).Update("is_active", false)
	c.SendJSON(w, &doctor, http.StatusOK)

	//Nur Praxis, der Doctor
	userIds := []uint{user.ID, uint(doctorId)}
	go web3socket.SendWebsocketDataInfoMessage("Delete doctor", web3socket.Websocket_Delete, web3socket.Websocket_Podiatrist, doctor.ID, userIds, nil)
}

// getDoctor swagger:route GET /doctors/{doctorId} doctor getDoctor
//
// retrieves informations about a doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: doctorId
//     in: path
//     description: the ID for the doctor
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) TransferDoctorDataHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok || (c.isPractice(user) && c.isDoctor(user)) {
		_ = user
		return
	}

	if !c.isPractice(user) {
		err := errors.New("Only Practices can transfer doctors")
		c.HandleError(err, w)
		return
	}
	doctorTransfer := DoctorTransferHelper{}
	if err := c.GetContent(&doctorTransfer, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	newDocUser := core.User{}
	c.ormDB.Find(&doctorTransfer.NewDoctor, doctorTransfer.NewDoctor.ID)
	c.ormDB.Find(&newDocUser, c.getMainUserIdFromDoctorId(doctorTransfer.NewDoctor.ID))
	for _, patient := range doctorTransfer.Patients {
		c.ormDB.Find(&patient, patient.ID)
		patientRelations := DoctorPatientRelations{}
		c.ormDB.Set("gorm:auto_preload", false).Where("patient_id = ? AND doctor_id = ?", patient.ID, doctorTransfer.OldDoctor.ID).Find(&patientRelations)
		for _, patientRelation := range patientRelations {
			patientRelation.ID = 0
			patientRelation.DoctorId = doctorTransfer.NewDoctor.ID
			c.ormDB.Set("gorm:save_associations", false).Create(&patientRelation)
			message := Message{
				DoctorId:    doctorTransfer.NewDoctor.ID,
				IsUnread:    true,
				MessageText: doctorTransfer.NewDoctor.StandardWelcomeMessage,
				SenderId:    newDocUser.ID,
				Sender:      newDocUser,
				RecipientId: patient.UserId,
				Recipient:   patient.User,
				MessageTime: core.NullTime{Time: time.Now(), Valid: true},
			}
			c.ormDB.Set("gorm:save_associations", false).Create(&message)
			c.CreateNotification(message.Recipient.ID, 1, fmt.Sprintf("New Message from %s", message.Sender.Username), message.MessageText, message.ID, fmt.Sprintf("/me/conversations/%d", message.Sender.ID), nil)
		}
	}

	measurementsShared := MeasurementsShared{}
	c.ormDB.Set("gorm:auto_preload", false).Where("doctor_id = ?", doctorTransfer.OldDoctor.ID).Find(&measurementsShared)
	for _, measurementShared := range measurementsShared {
		measurementShared.ID = 0
		measurementShared.DoctorId = doctorTransfer.NewDoctor.ID
		c.ormDB.Set("gorm:save_associations", false).Create(&measurementShared)
	}

	c.SendJSON(w, &doctorTransfer, http.StatusOK)
	//TODO SM ASK JC
	//go systembundle.SendWebsocketDataInfoMessage("Transfer doctor", systembundle.Websocket_Update, systembundle.Websocket_Podiatrist, 0, 0, nil)
}

// getPatients swagger:route GET /patients patient getPatients
//
// retrieves all patients
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPatientsHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	patients := Patients{}
	paging := c.GetPaging(r.URL.Query())

	db, dbTotalCount := c.CreateWhereConditionsPatients(r.URL.Query(), r, user)

	db.Debug().Set("gorm:auto_preload", true).Limit(paging.Limit).Offset(paging.Offset).Find(&patients)
	dbTotalCount.Model(&Patients{}).Count(&paging.TotalCount)

	for i, patient := range patients {
		c.ormDB.DB().QueryRow("SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =? AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1", patient.ID).Scan(&patient.LastAppointmentDate)
		c.ormDB.DB().QueryRow("SELECT measurement_date FROM measurements m WHERE m.patient_id =? ORDER BY measurement_date DESC LIMIT 1", patient.ID).Scan(&patient.LastMeasurementDate)

		patient.Practice = patient.GetPractice(c.ormDB.Preload("User"))

		patients[i] = patient
	}

	c.SendJSONPaging(w, r, paging, &patients, http.StatusOK)
}

func (c *PodiumController) CreateWhereConditionsPatients(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("CONCAT(last_name, first_name) LIKE ? OR `patients`.user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR `patients`.user_id IN (SELECT id FROM system_accounts WHERE created_by IN (SELECT id FROM practices WHERE name = ?))", search, search, search)
				dbTotalCount = dbTotalCount.Where("CONCAT(last_name, first_name) LIKE ? OR `patients`.user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR `patients`.user_id IN (SELECT id FROM system_accounts WHERE created_by IN (SELECT id FROM practices WHERE name = ?))", search, search, search)
			}
		}

		if val, ok := values["filter"]; ok && len(val) > 0 {
			var practiceIds []int
			for _, filter := range val {
				filterSplit := strings.Split(filter, ",")
				filterKey := filterSplit[0]
				filterValue := filterSplit[1]
				switch filterKey {
				case "practice":
					if practiceId, err := strconv.Atoi(filterValue); err != nil {
						log.Println(err)
					} else {
						practiceIds = append(practiceIds, practiceId)
					}
					break
				default:
					break
				}
			}
			if len(practiceIds) > 0 {
				db = db.Where("patients.user_id IN (SELECT id FROM system_accounts WHERE created_by IN (SELECT user_id FROM practices WHERE id IN (?))) OR patients.id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (?)) AND consent_status IN (2))", practiceIds, practiceIds)
				dbTotalCount = dbTotalCount.Where("patients.user_id IN (SELECT id FROM system_accounts WHERE created_by IN (SELECT user_id FROM practices WHERE id IN (?))) OR patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (?)) AND consent_status IN (2))", practiceIds, practiceIds)
			}
		}

		if val, ok := values["order"]; ok && len(val) > 0 {
			if val[0] != "" {
				if strings.Contains(val[0], ",") {
					sortSplit := strings.Split(val[0], ",")
					sortKey := sortSplit[0]
					sortDirection := sortSplit[1]
					switch sortKey {
					case "name":
						db = db.Order(fmt.Sprintf("last_name %s, first_name %s", sortDirection, sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						break
					case "username":
						db = db.Joins("LEFT JOIN system_accounts AS a ON a.id = `patients`.user_id")
						db = db.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						break
					case "practice":
						db = db.Joins("LEFT JOIN system_accounts AS a ON a.id = `patients`.user_id").Joins("LEFT JOIN practices AS p ON p.id = a.created_by").Joins("LEFT JOIN system_accounts AS ap ON ap.id = p.user_id")
						db = db.Order(fmt.Sprintf("ap.username %s", sortDirection))
						//dbTotalCount = dbTotalCount.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						break
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

// getPatients swagger:route GET /patients patient getPatients
//
// retrieves all patients
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPracticesHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	practices := Practices{}
	paging := c.GetPaging(r.URL.Query())

	db, dbTotalCount := c.CreateWhereConditionsPractices(r.URL.Query(), r, user)

	preloadData := true
	if len(r.URL.Query()) > 0 {
		values := r.URL.Query()

		if val, ok := values["preload"]; ok && len(val) > 0 {
			if val[0] == "false" {
				preloadData = false
			}
		}
	}

	if preloadData {
		db = db.Set("gorm:auto_preload", true)
	} else {
		db = db.Preload("User")
	}

	db.Limit(paging.Limit).Offset(paging.Offset).Find(&practices)
	dbTotalCount.Model(&Practices{}).Count(&paging.TotalCount)

	c.SendJSONPaging(w, r, paging, &practices, http.StatusOK)
}

func (c *PodiumController) GetPracticeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	practiceId, _ := strconv.ParseInt(vars["practiceId"], 10, 64)

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	smallPractice := false

	if len(r.URL.Query()) > 0 {

		values := r.URL.Query()
		if values["small_practice"] != nil && values["small_practice"][0] != "" {
			if values["small_practice"][0] == "true" {
				smallPractice = true
			}
		}
	}

	if practiceId == 0 {
		c.HandleError(errors.New("practice not found"), w)
	} else {
		practice := Practice{}
		db := c.ormDB
		if !smallPractice {
			db = db.Preload("User").Preload("PracticeContract")
		}
		db.First(&practice, practiceId)

		//patients := Patients{}
		//c.ormDB.Preload("User").Where("user_id IN (SELECT id FROM system_accounts WHERE created_by = ?)", practiceId).Find(&patients)
		//practice.Patients = patients

		c.SendJSON(w, &practice, http.StatusOK)
	}
}

func (c *PodiumController) GetPracticeDoctorsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	practiceId, _ := strconv.ParseInt(vars["practiceId"], 10, 64)

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if practiceId == 0 {
		c.HandleError(errors.New("practice not found"), w)
	} else {
		doctors := Doctors{}

		paging := c.GetPaging(r.URL.Query())
		db, dbTotalCount := c.CreateWhereConditionsDoctors(r.URL.Query(), r, user)

		db = db.Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practiceId)
		dbTotalCount = dbTotalCount.Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practiceId)

		db.Preload("Users.User").Limit(paging.Limit).Offset(paging.Offset).Find(&doctors)
		dbTotalCount.Model(&Doctor{}).Count(&paging.TotalCount)

		c.SendJSONPaging(w, r, paging, &doctors, http.StatusOK)
	}
}

func (c *PodiumController) GetPracticePatientsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	practiceId, _ := strconv.ParseInt(vars["practiceId"], 10, 64)

	practiceUser := c.getPracticeUserFromPracticeId(uint(practiceId))

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if practiceId == 0 {
		c.HandleError(errors.New("practice not found"), w)
	} else {
		patients := Patients{}

		paging := c.GetPaging(r.URL.Query())
		db, dbTotalCount := c.CreateWhereConditionsPatients(r.URL.Query(), r, user)

		db = db.Where("user_id IN (SELECT id FROM system_accounts WHERE created_by = ?) OR patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status IN (2))", practiceUser.ID, practiceId)
		dbTotalCount = dbTotalCount.Where("user_id IN (SELECT id FROM system_accounts WHERE created_by = ?) OR patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status IN (2))", practiceUser.ID, practiceId)

		db.Preload("User").Limit(paging.Limit).Offset(paging.Offset).Find(&patients)
		dbTotalCount.Model(&Patient{}).Count(&paging.TotalCount)

		c.SendJSONPaging(w, r, paging, &patients, http.StatusOK)
	}
}

func (c *PodiumController) GetPracticeDevicesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	practiceId, _ := strconv.ParseInt(vars["practiceId"], 10, 64)

	//practiceUser := c.getPracticeUserFromPracticeId(uint(practiceId))

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if practiceId == 0 {
		c.HandleError(errors.New("practice not found"), w)
	} else {

		devices := Devices{}

		paging := c.GetPaging(r.URL.Query())
		db, dbTotalCount := c.CreateWhereConditionsDevices(r.URL.Query(), r)

		db = db.Where("devices.id IN (SELECT device_id FROM practice_devices WHERE practice_id = ?)", practiceId)
		dbTotalCount = dbTotalCount.Where("devices.id IN (SELECT device_id FROM practice_devices WHERE practice_id = ?)", practiceId)

		db.Debug().Limit(paging.Limit).Offset(paging.Offset).Find(&devices) // nur  angenommene
		dbTotalCount.Model(&Devices{}).Count(&paging.TotalCount)

		c.SendJSONPaging(w, r, paging, &devices, http.StatusOK)
	}
}


type EmailRequest struct {
	To          string   `json:"to"`
	Subject     string   `json:"subject"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
}

// ispl_KTH_14/2/2024
func (c *PodiumController) SendMail1(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	to := r.FormValue("to")
	subject := r.FormValue("subject")
	body := r.FormValue("body")
	patient_username := r.FormValue("patient_username")
	fileHeaders := r.MultipartForm.File["attachments"]
	if len(fileHeaders) == 0 {
		http.Error(w, "No attachments provided", http.StatusBadRequest)
		return
	}

	attachmentDir := "/root/hijack/thermetrix_backend/PodiumFiles" 
	err = os.MkdirAll(attachmentDir, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	attachmentPaths := []string{}
	for _, fileHeader := range fileHeaders {
		attachmentFileName := filepath.Join(attachmentDir, patient_username+"_"+fileHeader.Filename)
		if _, err := os.Stat(attachmentFileName); err == nil {
			count := 1
			for {
				originalFilename := fileHeader.Filename
				extension := filepath.Ext(originalFilename)
				baseName := originalFilename[:len(originalFilename)-len(extension)]
				newFilename := fmt.Sprintf("%s (%d)%s", baseName, count, extension)
				attachmentFileName = filepath.Join(attachmentDir, patient_username+"_"+newFilename)
				_, err := os.Stat(attachmentFileName)
				if err != nil {
					break
				}
				count++
			}
		}

		attachmentFile, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer attachmentFile.Close()

		attachmentOutput, err := os.Create(attachmentFileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer attachmentOutput.Close()

		_, err = io.Copy(attachmentOutput, attachmentFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		attachmentPaths = append(attachmentPaths, attachmentFileName)
	}

	from := "info@podium.care"
	config := core.EmailConfig{
		SMTPHost:           "smtp.ionos.co.uk",
		SMTPPort:           587,
		SMTPUsername:       "info@podium.care",
		SMTPPassword:       "probably+All+Junk_#1",
		InsecureSkipVerify: false,
		ServerName:         "smtp.ionos.co.uk",
	}

	attachmentPathsString := make([]string, len(attachmentPaths))
	for i, path := range attachmentPaths {
		attachmentPathsString[i] = path
	}

	err = core.SendMail1(from, to, subject, body, attachmentPathsString, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := "Email sent successfully"
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(response))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//ispl_KTH_14/2/2024

// ispl_KTH_14/2/2024
func (c *PodiumController) SaveImagesForPatient(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	patientID := r.FormValue("patient_id")
	measurementID := r.FormValue("measurement_id")
	isDFAComplete := false
	fileHeaders := map[string][]*multipart.FileHeader{
		"DFA": r.MultipartForm.File["DFA"],
	}
	currentDir, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//this one has to change to core.uploadpath
	imageDir := filepath.Join(currentDir, "patients_files")
	for headerType, headers := range fileHeaders {
		var imagePaths []string

		for _, fileHeader := range headers {
			uploadedFile, err := fileHeader.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer uploadedFile.Close()
			completeImagePath := filepath.Join(imageDir, fileHeader.Filename)
			imageFile, err := os.Create(completeImagePath)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer imageFile.Close()
			_, err = io.Copy(imageFile, uploadedFile)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			imagePaths = append(imagePaths, completeImagePath)
		}

		if headerType == "DFA" && len(imagePaths) > 0 {
			isDFAComplete = true
		}

		if err := c.savePatientImagesToDB(measurementID, patientID, imagePaths, headerType); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if isDFAComplete {
		if err := c.ormDB.Model(&PatientImages{}).Where("measurement_id = ? AND patient_id = ?", measurementID, patientID).Update("IsDFA_Complete", true).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Files saved successfully"))
}

//ispl_KTH_14/2/2024

// ispl_KTH_14/2/2024
func (c *PodiumController) savePatientImagesToDB(measurementID string, patientID string, imagePaths []string, headerType string) error {
	var patientImages PatientImages
	exists := c.ormDB.First(&patientImages, "measurement_id = ? AND patient_id = ?", measurementID, patientID).Error == nil

	if len(imagePaths) == 0 {
		patientImages = PatientImages{
			Measurement_id: measurementID,
			PatientID:      patientID,
			ImagePath:      "", 
			Images:         "", 
			ImageKey:       headerType,
		}
	} else {
		dirPath, _ := filepath.Split(imagePaths[0])
		_, filename := filepath.Split(imagePaths[0])

		patientImages = PatientImages{
			Measurement_id: measurementID,
			PatientID:      patientID,
			ImagePath:      dirPath,
			Images:         filename, 
			ImageKey:       headerType,
		}
	}

	if exists {
		return c.ormDB.Model(&PatientImages{}).Where("measurement_id = ? AND patient_id = ?", measurementID, patientID).Updates(patientImages).Error
	} else {
		return c.ormDB.Create(&patientImages).Error
	}
}

//ispl_KTH_14/2/2024

// ispl_KTH_14/2/2024
func (c *PodiumController) getPatientImagesFromDB(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	patientID := params["MeasurementID"]

	patientImages, err := c.fetchPatientImagesFromDB(patientID)
	if err != nil {
		http.Error(w, "Failed to fetch patient images", http.StatusInternalServerError)
		return
	}

	if patientImages == nil || len(patientImages) == 0 {
		http.Error(w, "No patient images found", http.StatusNotFound)
		return
	}

	isDFA_Complete := false
	if len(patientImages) > 0 {
		isDFA_Complete = patientImages[0].IsDFA_Complete
	}

	imageUrls := make([]string, 0)

	for _, image := range patientImages {
		if image.Images != "" {
			imageURL := fmt.Sprintf("http://localhost:4001/api/v1/serve-image/%s", strings.TrimPrefix(image.Images, "/"))
			imageUrls = append(imageUrls, imageURL)
		}
	}

	response := map[string]interface{}{
		"isDFA_Complete": isDFA_Complete,
		"patientId":      patientID,
		"url":            imageUrls,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonData)
}

//ispl_KTH_14/2/2024

// ispl_KTH_14/2/2024
func (c *PodiumController) fetchPatientImagesFromDB(patientID string) ([]PatientImage, error) {
	var patientImages []PatientImage

	if err := c.ormDB.Where("measurement_id = ?", patientID).Find(&patientImages).Error; err != nil {
		return nil, err
	}
	return patientImages, nil
}

//ispl_KTH_14/2/2024

// ispl_KTH_14/2/2024
func (c *PodiumController) serveImageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Print("call received inside the funcrtions>>>>>>>>>")
	params := mux.Vars(r)
	filePath := params["filepath"]
	print("the path ?>>>>>>>>>", filePath)
	currentDir, err := os.Getwd()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	//this has to be changed to core.uploadpath
	basePath := path.Join(currentDir, "patients_files")
	absFilePath := path.Join(basePath, filePath)

	file, err := os.Open(absFilePath)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	defer file.Close()
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	http.ServeContent(w, r, filePath, time.Now(), file)
}

//ispl_KTH_14/2/2024

// ispl_KTH_14/2/2024
func (c *PodiumController) ScanHistory(w http.ResponseWriter, r *http.Request) {
	var patientImages []PatientImage
	//replace base url with server's base url
	baseURL := "http://localhost:4001/api/v1/serve-image/"
	if err := c.ormDB.Find(&patientImages).Error; err != nil {
		http.Error(w, "Failed to fetch patient images", http.StatusInternalServerError)
		return
	}

	allPatientsData := make(map[string]map[string]interface{})

	for _, image := range patientImages {
		_, filename := filepath.Split(image.Images)  
		fileURL := baseURL + url.QueryEscape(filename) 

		if _, exists := allPatientsData[image.PatientID]; !exists {
			allPatientsData[image.PatientID] = map[string]interface{}{
				"isDFA_Complete": image.IsDFA_Complete,
				"urls":           []string{},
			}
		}

		allPatientsData[image.PatientID]["urls"] = append(allPatientsData[image.PatientID]["urls"].([]string), fileURL)
	}

	// Convert the map to a slice of patient data
	var allPatientsSlice []map[string]interface{}
	for patientID, data := range allPatientsData {
		patientData := map[string]interface{}{
			"patientId":      patientID,
			"isDFA_Complete": data["isDFA_Complete"],
			"url":            data["urls"],
		}
		allPatientsSlice = append(allPatientsSlice, patientData)
	}

	jsonData, err := json.Marshal(allPatientsSlice)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonData)
}
//ispl_KTH_14/2/2024


//ispl_KTH_03/3/2024
func (c *PodiumController) GetNotesByMeasurementIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
    measurementID := vars["MeasurementID"]
    notes, err := c.getNotesByMeasurementID(measurementID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    jsonResponse, err := json.Marshal(notes)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonResponse)
}

func (c *PodiumController) getNotesByMeasurementID(measurementID string) ([]notes, error) {

    var notes []notes

    if err := c.ormDB.Where("measurement_id = ?", measurementID).Find(&notes).Error; err != nil {
        return nil, err
    }
	fmt.Print(">>>>>>>>>>>>",measurementID)
    return notes, nil
}





func (c *PodiumController) SaveNotes(w http.ResponseWriter, r *http.Request) {
    var requestBody notes
    err := json.NewDecoder(r.Body).Decode(&requestBody)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := c.saveDataToDB(requestBody.MeasurementID, requestBody.PatientID, requestBody.Notes); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Data saved successfully"))
}

func (c *PodiumController) saveDataToDB(measurementID string, patientID string, notes string) error {

	type note struct {
		MeasurementID string `gorm:"column:measurement_id"`
		PatientID     string `gorm:"column:patient_id"`
		Notes         string `gorm:"column:notes"`
	}

	existingData := note{}
	err := c.ormDB.Where("measurement_id = ?", measurementID).First(&existingData).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	newData := note{
		MeasurementID: measurementID,
		PatientID:     patientID,
		Notes:         notes,
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := c.ormDB.Create(&newData).Error; err != nil {
			return err
		}
	} else {
		if err := c.ormDB.Model(&newData).Where("measurement_id = ?", measurementID).Updates(&newData).Error; err != nil {
			return err
		}
	}

	return nil
}

//ispl_KTH_03/3/2024



//ispl_KTH_03/3/2024
func (c *PodiumController) SaveCoverLetter(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	patientID := r.FormValue("patient_id")
	measurementID := r.FormValue("measurement_id")
	coverLetterFile, coverLetterHeader, err := r.FormFile("cover_letter")
	if err != nil && err != http.ErrMissingFile {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	coverLetterDir := filepath.Join(currentDir, "cover_letters")

	var coverLetterPath string
	if coverLetterFile != nil {
		defer coverLetterFile.Close()
		timestamp := time.Now().Format("20060102150405") 
		filename := fmt.Sprintf("%s_%s", timestamp, coverLetterHeader.Filename)

		coverLetterPath = filepath.Join(coverLetterDir, filename)
		coverLetter, err := os.Create(coverLetterPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer coverLetter.Close()

		_, err = io.Copy(coverLetter, coverLetterFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	isReferred := coverLetterFile != nil

	if err := c.saveCoverLetterToDB(measurementID, patientID, coverLetterPath, isReferred); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Cover letter saved successfully"))
}

func (c *PodiumController) saveCoverLetterToDB(measurementID string, patientID string, coverLetterPath string, isReferred bool) error {
	existingCoverLetter := CoverLetter{}
	err := c.ormDB.Where("measurement_id = ?", measurementID).First(&existingCoverLetter).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	coverLetter := CoverLetter{
		MeasurementID: measurementID,
		PatientID:     patientID,
		CoverLetter:   coverLetterPath,
		IsReferred:    isReferred,
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := c.ormDB.Create(&coverLetter).Error; err != nil {
			return err
		}
	} else {
		if err := c.ormDB.Model(&coverLetter).Where("measurement_id = ?", measurementID).Updates(&coverLetter).Error; err != nil {
			return err
		}
	}

	return nil
}
//ispl_KTH_03/3/2024

//ispl_KTH_03/3/2024
type CoverLetterResponse struct {
	MeasurementID string `json:"measurement_id"`
	CoverLetter   string `json:"cover_letter"`
	IsReferred    bool   `json:"is_referred"`
}

func (c *PodiumController) GetCoverLetter(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    measurementID := vars["MeasurementID"]
    var coverLetter CoverLetter
    if measurementID == "" {
        response := map[string]interface{}{
            "cover_letter_url": "",
            "is_referred":      false,
            "measurement_id":   "",
        }
        jsonData, err := json.Marshal(response)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(jsonData)
        return
    }

    if err := c.ormDB.Where("measurement_id = ?", measurementID).First(&coverLetter).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            response := map[string]interface{}{
                "cover_letter_url": "",
                "is_referred":      false,
                "measurement_id":   measurementID,
            }
            jsonData, err := json.Marshal(response)
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            w.Write(jsonData)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    coverLetterURL := ""
    if coverLetter.CoverLetter != "" {
        coverLetterURL = fmt.Sprintf("http://localhost:4001/api/v1/download-cover-letter/%s", filepath.Base(coverLetter.CoverLetter))
    }

    response := map[string]interface{}{
        "cover_letter_url": coverLetterURL,
        "is_referred":      coverLetter.IsReferred,
        "measurement_id":   measurementID,
    }

    jsonData, err := json.Marshal(response)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(jsonData)
}

func (c *PodiumController) serveCoverLetterHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	filePath := params["filepath"]
	currentDir, err := os.Getwd()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	absFilePath := filepath.Join(currentDir, "cover_letters", filePath)
	file, err := os.Open(absFilePath)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	defer file.Close()
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, filePath, time.Now(), file)
}
//ispl_KTH_03/3/2024

func (c *PodiumController) SavePracticeHandler(w http.ResponseWriter, r *http.Request) {
	practice := Practice{}
	if err := c.GetContent(&practice, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if !practice.Validate(c.ormDB) {
		log.Println(practice.Errors)
		c.SendErrors(w, practice.Errors, http.StatusBadRequest)
		return
	}

	if practice.User.ID == 0 {
		practice.User.Username = fmt.Sprintf("%s_%s", practice.AccountsAbbreviation, practice.User.Username)
	}

	if _, err := practice.Save(c.ormDB); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	c.SendJSON(w, &practice, http.StatusOK)
}

func (c *PodiumController) CreateWhereConditionsDevices(urlQuery url.Values, r *http.Request) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("device_identifier LIKE ? OR device_mac LIKE ? OR device_serial LIKE ?", search, search, search)
				dbTotalCount = dbTotalCount.Where("device_identifier LIKE ? OR device_mac LIKE ? OR device_serial LIKE ?", search, search, search)
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

func (c *PodiumController) CreateWhereConditionsPractices(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("name LIKE ? OR postcode LIKE ? OR user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?)", search, search, search)
				dbTotalCount = dbTotalCount.Where("name LIKE ? OR postcode LIKE ? OR user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?)", search, search, search)
			}
		}

		if val, ok := values["order"]; ok && len(val) > 0 {
			if val[0] != "" {
				if strings.Contains(val[0], ",") {
					sortSplit := strings.Split(val[0], ",")
					sortKey := sortSplit[0]
					sortDirection := sortSplit[1]
					switch sortKey {
					case "username":
						//db = db.Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						db = db.Joins("LEFT JOIN system_accounts ON practices.user_id = system_accounts.id")
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						break
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

// getPatient swagger:route GET /patients/{patientId} patient getPatient
//
// retrieves informations about a patient
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: patientId
//     in: path
//     description: the ID for the patient
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPatientHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	patientId, err := strconv.ParseInt(vars["patientId"], 10, 64)
	patient := Patient{}

	c.ormDB.Set("gorm:auto_preload", true).First(&patient, patientId)
	if c.HandleError(err, w) {
		return
	}

	patient.Practice = patient.GetPractice(c.ormDB)

	c.SendJSON(w, &patient, http.StatusOK)
}

// savePatient swagger:route POST /patients patient savePatient
//
// # Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SavePatientHandler(w http.ResponseWriter, r *http.Request) {

	var patient Patient

	isNewPatient := false
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isDoctor(user) && !c.isSysadmin(user) && !c.isPractice(user) {
		err := errors.New("You are not allowed to create new patients")
		c.HandleError(err, w)
		return
	}

	if err := c.GetContent(&patient, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if !patient.Validate(c.ormDB) {
		log.Println(patient.Errors)
		c.SendErrors(w, patient.Errors, http.StatusBadRequest)
		return
	}

	if c.ormDB.NewRecord(&patient) {
		isNewPatient = true
		patient.HasPairedPodiatrist = true

		// check Username
		patient.User.Username = strings.TrimSpace(patient.User.Username)
		patient.User.Email = strings.TrimSpace(patient.User.Email)
		if len(patient.User.Username) < 4 {
			err := errors.New("Username to short, you need minimum 4 characters")
			c.HandleError(err, w)
			return
		}

		if c.ormDB.NewRecord(&patient.User) {
			// As requested by TMX, usernames should only be unique within each practice
			patient.User.Username = fmt.Sprintf("%s_%s", patient.Practice.AccountsAbbreviation, patient.User.Username)
		}

		if len(patient.User.Email) > 0 {
			err := core.ValidateFormat(patient.User.Email)
			if err != nil {
				err = errors.New("E-Mail: " + err.Error())
				c.HandleError(err, w)
				return
			}
		}

		/*
			userDB := core.User{}
			c.ormDB.Set("gorm:auto_preload", true).Where("username=? OR email=?", patient.User.Username, patient.User.Email).First(&userDB)

			if userDB.ID > 0 {
				err := errors.New("Username or E-Mail already exists")
				c.HandleError(err, w)
				return
			}
		*/

		//if c.ormDB.NewRecord(&patient.User) {
		/*
			if c.isPatient(user) {
					err := core.ValidatePassword(patient.User.PasswordX)
					if err != nil {
						c.HandleError(err, w)
						return
					}
				}

		*/
		patient.User.RegisteredAt.Time = time.Now()
		patient.User.RegisteredAt.Valid = true
		patient.User.UserType = 1
		user.IsPasswordExpired = false
		//patient.User.IsPasswordExpired = true

		if c.isSysadmin(user) && patient.Practice.ID > 0 {
			practiceUser := core.User{}
			c.ormDB.Where("id IN (SELECT user_id FROM practices WHERE id = ?)", patient.Practice.ID).First(&practiceUser)
			patient.User.CreatedBy = practiceUser.ID
		} else if user != nil {
			patient.User.CreatedBy = user.ID
		}

		patient.User.IsActive = true

		if len(patient.User.PasswordX) > 0 {
			patient.User.PasswordX = core.GetMD5Hash(patient.User.PasswordX)
			patient.User.Password = patient.User.PasswordX
		}
		//c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", patient.User.PasswordX, patient.User.ID)

		c.ormDB.Debug().Set("gorm:save_associations", false).Save(&patient.User)
		patient.User.PasswordX = ""
		//}
		c.ormDB.Debug().Set("gorm:save_associations", false).Save(&patient)
		c.ormDB.Exec("UPDATE patients SET user_id = ? WHERE id = ?", patient.User.ID, patient.ID)

		tempDoctor := c.getDoctor(user)
		doctor := Doctor{}
		if tempDoctor != nil {
			doctor = *tempDoctor
		}
		consent := &DoctorPatientRelation{}
		consent.Patient = patient
		consent.PatientId = patient.ID
		consent.Doctor = doctor
		consent.DoctorId = consent.Doctor.ID
		consent.ConsentStatus = 2
		consent.ConsentType = 2
		consent.ConsentDate.Time = time.Now()
		consent.ConsentDate.Valid = true
		c.ormDB.Set("gorm:save_associations", false).Save(&consent)

		if strings.TrimSpace(doctor.StandardWelcomeMessage) != "" {
			c.ormDB.Set("gorm:auto_preload", true).Where("user_id=?", user.ID).First(&consent.Patient)

			message := Message{
				DoctorId:    consent.Doctor.ID,
				IsUnread:    true,
				MessageText: doctor.StandardWelcomeMessage,
				SenderId:    user.ID,
				RecipientId: consent.Patient.User.ID,
				MessageTime: core.NullTime{Time: time.Now(), Valid: true},
			}
			c.ormDB.Create(&message)
		}
	} else {
		isNewPatient = false
		if patient.User.PasswordX != "" {
			err := core.ValidatePassword(patient.User.PasswordX)
			if err != nil {
				c.HandleError(err, w)
				return
			}
			patient.User.PasswordX = core.GetMD5Hash(patient.User.PasswordX)
			c.ormDB.Exec("UPDATE system_accounts SET password = ?, email = ? WHERE id = ?", patient.User.PasswordX, patient.User.Email, patient.User.ID)
			patient.User.PasswordX = ""
		} else {
			c.ormDB.Exec("UPDATE system_accounts SET email = ? WHERE id = ?", patient.User.Email, patient.User.ID)
		}
		c.ormDB.Set("gorm:save_associations", false).Save(&patient)
		c.ormDB.Exec("UPDATE patients SET user_id = ? WHERE id = ?", patient.User.ID, patient.ID)
	}

	c.SendJSON(w, &patient, http.StatusOK)

	if isNewPatient {
		c.SendWebsocketToUsersConnectedWithPatient(c.ormDB, patient, "Add Patients", web3socket.Websocket_Add)
		//go websocket.SendWebsocketDataInfoMessage("Add Patients", websocket.Websocket_Add, websocket.Websocket_Patients, uint(patient.ID), wsIds, nil)
		//go websocket.SendWebsocketDataInfoMessage("Add patient", websocket.Websocket_Add, websocket.Websocket_Patients, patient.ID, 0, nil)
	} else {
		c.SendWebsocketToUsersConnectedWithPatient(c.ormDB, patient, "Update Patients", web3socket.Websocket_Update)
		//go websocket.SendWebsocketDataInfoMessage("Update Patients", websocket.Websocket_Update, websocket.Websocket_Patients, uint(patient.ID), wsIds, nil)
		/* TODO SM
		ids := []uint{}

		//doctorIds := patient.GetPairedDoctors(c.ormDB)
		if c.isPractice(user){
			practice := c.GetPractice(user)
			doctorIds := practice.GetDoctors(c.ormDB)
			ids = append(doctorIds, user.ID)
		}else{

		}

		go websocket.SendWebsocketDataInfoMessage("Update patient", websocket.Websocket_Update, websocket.Websocket_Patients, patient.ID, 0, nil)*/
	}

}

func (c *PodiumController) SendWebsocketToUsersConnectedWithPatient(ormDB *gorm.DB, patient Patient, webMessage string, websocketType string) {

	patientUser := core.User{}
	ormDB.First(&patientUser, patient.UserId)

	wsIds := []uint{}
	wsIds = append(wsIds, patientUser.ID)

	doctors := patient.GetPairedDoctors(ormDB)

	doctorIds := []uint{}
	for _, item := range doctors {
		if item.ID > 0 {
			doctorIds = append(doctorIds, item.ID)
		}
	}

	userDoctorIds := c.getMainUserIdsFromDoctorIds(doctorIds)
	userPracticeIds := c.getPracticeUserIdsFromDoctorIds(doctorIds)

	wsIds = append(wsIds, userDoctorIds...)
	wsIds = append(wsIds, userPracticeIds...)

	createdByUserId := patientUser.CreatedBy

	if createdByUserId > 0 {
		createdBy := core.User{}
		ormDB.First(&createdBy, createdByUserId)
		wsIds = append(wsIds, createdBy.ID)

		switch createdBy.UserType {
		case core.UserTypePractice:
			p := Practice{}
			ormDB.First(&p, createdBy.ID)
			pDoctorIds := p.GetDoctorIds(ormDB)
			pUserDoctorIds := c.getMainUserIdsFromDoctorIds(pDoctorIds)
			wsIds = append(wsIds, pUserDoctorIds...)

		case core.UserTypeDoctor:

			d := Doctor{}
			ormDB.First(&d, createdBy.ID)

			pId := d.GetPracticeUserId(ormDB)
			wsIds = append(wsIds, pId)
		}
	}

	wsIds = removeDuplicateInt(wsIds)

	go web3socket.SendWebsocketDataInfoMessage(webMessage, websocketType, web3socket.Websocket_Patients, uint(patient.ID), wsIds, nil)

}

func removeDuplicateInt(intSlice []uint) []uint {
	allKeys := make(map[uint]bool)
	list := []uint{}
	for _, item := range intSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func (c *PodiumController) getDoctorAndPracticeId(user *core.User) ([]uint, bool) {

	if !c.isPractice(user) && !c.isDoctor(user) {
		return nil, false
	}

	ids := []uint{}
	ids = append(ids, user.ID)

	if c.isPractice(user) {

	} else {

	}
	return ids, true
}

// savePatient swagger:route POST /patients patient savePatient
//
// # Save a patient, its user and adds a consentrequest
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) RegisterPatientHandler(w http.ResponseWriter, r *http.Request) {
	var patient Patient

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
		err := core.ValidateFormat(patient.User.Email)
		if err != nil {
			err = errors.New("E-Mail: " + err.Error())
			c.HandleError(err, w)
			return
		}

		err = core.ValidatePassword(patient.User.PasswordX)
		if err != nil {
			c.HandleError(err, w)
			return
		}
		userDB := core.User{}

		c.ormDB.Set("gorm:auto_preload", true).Where("username=? OR email=?", patient.User.Username, patient.User.Email).First(&userDB)

		if userDB.ID > 0 {
			err := errors.New("Username or E-Mail already exists")
			c.HandleError(err, w)
			return
		}

		if c.ormDB.NewRecord(&patient.User) {
			patient.User.RegisteredAt.Time = time.Now()
			patient.User.RegisteredAt.Valid = true
			patient.User.UserType = 1
			//patient.User.IsPasswordExpired = true
			// patient.User.IsPasswordExpired = false
			//c.ormDB.Set("save_associations", false).Save(&patient.User)

			patient.User.IsActive = true
			patient.User.PasswordX = core.GetMD5Hash(patient.User.PasswordX)
			patient.User.Password = patient.User.PasswordX
			//c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", patient.User.PasswordX, patient.User.ID)
			c.ormDB.Set("gorm:save_associations", false).Save(&patient.User)

			accountsSession := HelperSystemAccountsSession{}

			token, _ := core.NewV4()
			accountsSession.ID = 0
			accountsSession.SessionToken = token.String()
			accountsSession.AccountId = patient.User.ID
			accountsSession.LoginTime = core.NullTime{Time: time.Now(), Valid: true}
			c.ormDB.Set("gorm:save_associations", false).Create(&accountsSession)
			patient.User.Token = accountsSession.SessionToken

			(*c.Controller.Users)[patient.User.Token] = patient.User

		}
		//c.ormDB.Set("save_associations", false).Save(&patient)
		c.ormDB.Set("gorm:save_associations", false).Save(&patient)
		c.ormDB.Exec("UPDATE patients SET user_id = ? WHERE id = ?", patient.User.ID, patient.ID)
	}
	c.SendJSON(w, &patient, http.StatusOK)
}

// getMyAppointments swagger:route GET /me/appointments appointments getMyAppointments
//
// retrieves your appointments
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyAppointmentsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	appointments := Appointments{}
	tempDB := c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef")

	if len(r.URL.Query()) > 0 {

		values := r.URL.Query()
		if values["status"] != nil && values["status"][0] != "" {
			splitValues := strings.Split(values["status"][0], ",")
			tempDB = tempDB.Where("appointment_status_id IN (SELECT id FROM appointment_statuses WHERE status_def_id IN (?))", splitValues)
		}

		if values["date_from"] != nil && values["date_from"][0] != "" {
			tempDB = tempDB.Where("date_time_from > ?", values["date_from"][0])
		}

		if values["date_to"] != nil && values["date_to"][0] != "" {
			tempDB = tempDB.Where("date_time_from > ?", values["date_to"][0])
		}
	}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		tempDB.Where("patient_id = ?", patient.ID).Find(&appointments)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		tempDB.Where("doctor_id = ?", doctor.ID).Find(&appointments)
	} else {
		err := errors.New("You are dont have appointments")
		c.HandleError(err, w)
	}

	c.SendJSON(w, &appointments, http.StatusOK)
}

// getAppointments swagger:route GET /me/appointments appointments getAppointments
//
// retrieves appointments for user, expected if user is system user, he get all appointments
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetAppointmentsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	appointments := Appointments{}

	tempDB := c.ormDB.Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef")

	if len(r.URL.Query()) > 0 {

		values := r.URL.Query()
		if values["status"] != nil && values["status"][0] != "" {
			splitValues := strings.Split(values["status"][0], ",")
			tempDB = tempDB.Where("appointment_status_id IN (SELECT id FROM appointment_statuses WHERE status_def_id IN (?))", splitValues)
		}

		if values["date_from"] != nil && values["date_from"][0] != "" {
			tempDB = tempDB.Where("date_time_from > ?", values["date_from"][0])
		}

		if values["date_to"] != nil && values["date_to"][0] != "" {
			tempDB = tempDB.Where("date_time_from > ?", values["date_to"][0])
		}
	}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		//c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Find(&appointments)
		tempDB.Where("patient_id = ?", patient.ID).Find(&appointments)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		tempDB.Where("doctor_id = ?", doctor.ID).Find(&appointments)
	} else if user.UserType == 0 {
		tempDB.Find(&appointments)
	}

	c.SendJSON(w, &appointments, http.StatusOK)
}

// getOpenAppointments swagger:route GET /appointments/requests appointments getOpenAppointments
//
// retrieves appointments for user, expected if user is system user, he get all appointments
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetOpenAppointmentsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	appointments := Appointments{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Where("patient_id = ?", patient.ID).Where("appointment_status_id IN (SELECT id FROM appointment_statuses WHERE status_def_id = 2)").Find(&appointments)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Where("doctor_id = ?", doctor.ID).Where("appointment_status_id IN (SELECT id FROM appointment_statuses WHERE status_def_id IN (1,2))").Find(&appointments)
	} else if user.UserType == 0 {
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Find(&appointments)
	}

	c.SendJSON(w, &appointments, http.StatusOK)
}

// getAppointment swagger:route GET /appointments/{appointmentId} appointments getAppointment
//
// retrieves appointments for user, expected if user is system user, he get all appointments
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: appointmentId
//     in: path
//     description: the ID for the appointment
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	appointmentId, _ := strconv.ParseInt(vars["appointmentId"], 10, 64)

	appointment := Appointment{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Where("patient_id = ?", patient.ID).First(&appointment, appointmentId)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Where("doctor_id = ?", doctor.ID).First(&appointment, appointmentId)
	} else if user.UserType == 0 {
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").First(&appointment, appointmentId)
	}

	c.SendJSON(w, &appointment, http.StatusOK)
}

// saveAppointment swagger:route POST /appointments appointments saveAppointment
//
// # Create Appointments
//
// produces:
// - application/json
// parameters:
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: Appointment
//     type: Appointment
//     required: true
//     in: body
//     description: An Appointment object
//
// Responses:
//
//	   default: HandleErrorData
//	       200:
//				data: Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	var appointment Appointment

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if err := c.GetContent(&appointment, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.ormDB.NewRecord(&appointment) {
		if c.isPatient(user) {
			appointment.Patient = *c.getPatient(user)
			appointment.PatientId = appointment.Patient.ID
			/*if !c.hasConsent(appointment.Patient, appointment.Doctor) {

			}*/

			//check if patient already has an appointmentRequest
			appointmentsDB := Appointments{}
			c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Where("patient_id = ?", appointment.Patient.ID).Where("doctor_id=?", appointment.Doctor.ID).Find(&appointmentsDB)
			for _, app := range appointmentsDB {
				if app.AppointmentStatus.StatusDefId == 1 {
					err := errors.New("You already have an open inquiry with this doctor.")
					c.HandleError(err, w)
					return
				}
			}

			appointment.RequestDateTime = core.NullTime{Time: time.Now(), Valid: true}
			c.ormDB.Set("gorm:save_associations", false).Create(&appointment)

			appointment.AppointmentStatus = AppointmentStatus{StatusDefId: 1, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointment.ID, CreatedById: user.ID}
			c.ormDB.Set("gorm:save_associations", false).Create(&appointment.AppointmentStatus)
			c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.AppointmentStatus, appointment.AppointmentStatus.ID)
			c.ormDB.Set("gorm:save_associations", false).Save(&appointment)

			mainUserId := c.getMainUserIdFromDoctorId(appointment.Doctor.ID)
			c.CreateNotification(mainUserId, 2, fmt.Sprintf("Appointment request %s %s", appointment.Patient.FirstName, appointment.Patient.LastName), "", appointment.ID, fmt.Sprintf("/appointments/%d", appointment.ID), nil)

		} else if c.isDoctor(user) {
			appointment.Doctor = *c.getDoctor(user)
			appointment.DoctorId = appointment.Doctor.ID
			/*if !c.hasConsent(appointment.Patient, appointment.Doctor) {

			}*/
			appointment.RequestDateTime = core.NullTime{Time: time.Now(), Valid: true}
			c.ormDB.Set("gorm:save_associations", false).Create(&appointment)

			appointment.AppointmentStatus = AppointmentStatus{StatusDefId: 2, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointment.ID, CreatedById: user.ID}
			c.ormDB.Set("gorm:save_associations", false).Create(&appointment.AppointmentStatus)
			c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.AppointmentStatus, appointment.AppointmentStatus.ID)
			c.ormDB.Set("gorm:save_associations", false).Save(&appointment)

			c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.Patient, appointment.Patient.ID)
			c.CreateNotification(appointment.Patient.User.ID, 2, fmt.Sprintf("Appointment request declined %s %s", appointment.Doctor.FirstName, appointment.Doctor.LastName), "", appointment.ID, fmt.Sprintf("/appointments/%d", appointment.ID), nil)
		}

	} else {
		appointmentDB := Appointment{}
		c.ormDB.Set("gorm:auto_preload", true).First(&appointmentDB, appointment.ID)
		if c.isDoctor(user) {
			doctor := c.getDoctor(user)
			if appointmentDB.Doctor.ID == doctor.ID { //If correct Doctor
				if appointmentDB.AppointmentStatus.StatusDefId == 1 { //If requested an Appointment by Patient
					if appointmentDB.Patient.ID == appointment.Patient.ID { //If correct Patient
						appointment.AppointmentStatus = AppointmentStatus{StatusDefId: 2, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointment.ID, CreatedById: user.ID}
						c.ormDB.Set("gorm:save_associations", false).Create(&appointment.AppointmentStatus)
						c.ormDB.Set("gorm:save_associations", false).Save(&appointment)

						c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.Patient, appointment.Patient.ID)
						c.CreateNotification(appointment.Patient.User.ID, 2, fmt.Sprintf("Appointment date suggestion %s %s", appointment.Doctor.FirstName, appointment.Doctor.LastName), "", appointment.ID, fmt.Sprintf("/appointments/%d", appointment.ID), nil)
					}
				} else if appointmentDB.AppointmentStatus.StatusDefId == 2 || appointmentDB.AppointmentStatus.StatusDefId == 3 { //If date is suggested or patient accepted date of appointment
					appointmentDB.AppointmentStatus = AppointmentStatus{StatusDefId: 7, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointmentDB.ID, CreatedById: user.ID}
					c.ormDB.Set("gorm:save_associations", false).Create(&appointmentDB.AppointmentStatus)
					c.ormDB.Set("gorm:auto_preload", true).Find(&appointmentDB.AppointmentStatus, appointmentDB.AppointmentStatus.ID)
					c.ormDB.Set("gorm:save_associations", false).Save(&appointmentDB)

					//Now create new Appointment
					appointment.Doctor = appointmentDB.Doctor
					appointment.Patient = appointmentDB.Patient
					appointment.RequestDateTime = core.NullTime{Time: time.Now(), Valid: true}
					appointment.OriginAppointmentId = appointmentDB.ID
					appointment.OriginAppointment = &appointmentDB
					c.ormDB.Set("gorm:save_associations", false).Create(&appointment)

					appointment.AppointmentStatus = AppointmentStatus{StatusDefId: 2, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointment.ID, CreatedById: user.ID}
					c.ormDB.Set("gorm:save_associations", false).Create(&appointment.AppointmentStatus)
					c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.AppointmentStatus, appointment.AppointmentStatus.ID)
					c.ormDB.Set("gorm:save_associations", false).Save(&appointment)

					appointmentDB = appointment

					c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.Patient, appointment.Patient.ID)
					c.CreateNotification(appointment.Patient.User.ID, 2, fmt.Sprintf("Appointment reschedule request %s %s", appointment.Doctor.FirstName, appointment.Doctor.LastName), "", appointment.ID, fmt.Sprintf("/appointments/%d", appointment.ID), nil)
				}
			}
		}
	}
	c.SendJSON(w, &appointment, http.StatusOK)
}

// acceptAppointment swagger:route PATCH /appointments/{appointmentId}/accept appointments acceptAppointment
//
// # Accept an Appointment specified by ID
//
// produces:
// - application/json
// parameters:
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: appointmentId
//     in: path
//     description: the ID for the Appointment
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//	       200:
//				data: Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) AcceptAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	appointmentId, _ := strconv.ParseInt(vars["appointmentId"], 10, 64)

	appointmentDB := Appointment{}
	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Find(&appointmentDB, appointmentId)
		if appointmentDB.Patient.ID == patient.ID { //If correct Patient
			if appointmentDB.AppointmentStatus.StatusDefId == 2 { //If appointment requested by doctor
				appointmentDB.AppointmentStatus = AppointmentStatus{StatusDefId: 3, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointmentDB.ID, CreatedById: user.ID}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointmentDB.AppointmentStatus)
				c.ormDB.Set("gorm:auto_preload", true).Find(&appointmentDB.AppointmentStatus, appointmentDB.AppointmentStatus.ID)
				c.ormDB.Set("gorm:save_associations", false).Save(&appointmentDB)
			}
		}
	}

	c.SendJSON(w, &appointmentDB, http.StatusOK)
}

// declineAppointment swagger:route PATCH /appointments/{appointmentId}/decline appointments declineAppointment
//
// # Decline an Appointment specified by ID
//
// produces:
// - application/json
// parameters:
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: appointmentId
//     in: path
//     description: the ID for the Appointment
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//	       200:
//				data: Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) DeclineAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	appointmentId, _ := strconv.ParseInt(vars["appointmentId"], 10, 64)

	appointmentDB := Appointment{}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Find(&appointmentDB, appointmentId)
		if appointmentDB.Doctor.ID == doctor.ID { //If correct Doctor
			if appointmentDB.AppointmentStatus.StatusDefId == 1 || appointmentDB.AppointmentStatus.StatusDefId == 2 { //If requested an Appointment by Patient
				appointmentDB.AppointmentStatus = AppointmentStatus{StatusDefId: 5, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointmentDB.ID, CreatedById: user.ID}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointmentDB.AppointmentStatus)
				c.ormDB.Set("gorm:save_associations", false).Save(&appointmentDB)
			}

			c.CreateNotification(appointmentDB.Patient.User.ID, 2, fmt.Sprintf("Appointment request declined %s %s", doctor.FirstName, doctor.LastName), "", appointmentDB.ID, fmt.Sprintf("/appointments/%d", appointmentDB.ID), nil)
		}
	} else if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Find(&appointmentDB, appointmentId)
		if appointmentDB.Patient.ID == patient.ID { //If correct Patient
			if appointmentDB.AppointmentStatus.StatusDefId == 2 || appointmentDB.AppointmentStatus.StatusDefId == 3 { //If suggestion by a Podiatrist
				appointmentDB.AppointmentStatus = AppointmentStatus{StatusDefId: 4, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointmentDB.ID, CreatedById: user.ID}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointmentDB.AppointmentStatus)
				c.ormDB.Set("gorm:auto_preload", true).Find(&appointmentDB.AppointmentStatus, appointmentDB.AppointmentStatus.ID)
				c.ormDB.Set("gorm:save_associations", false).Save(&appointmentDB)
			}

			// no message to doctor
		}
	}

	c.SendJSON(w, &appointmentDB, http.StatusOK)
}

// rescheduleAppointment swagger:route PATCH /appointments/{appointmentId}/reschedule appointments rescheduleAppointment
//
// # Reschedule an Appointment specified by ID
//
// produces:
// - application/json
// parameters:
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: appointmentId
//     in: path
//     description: the ID for the Appointment
//     required: true
//     type: integer
//   - name: appointment
//     in: body
//     description: the new Appointment
//     required: false
//     type: Appointment
//
// Responses:
//
//	   default: HandleErrorData
//	       200:
//				data: Appointment
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) RescheduleAppointmentHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	var appointment Appointment
	if err := c.GetContent(&appointment, r); err != nil {
		log.Println(err)
	}

	vars := mux.Vars(r)
	appointmentId, _ := strconv.ParseInt(vars["appointmentId"], 10, 64)

	appointmentDB := Appointment{}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Find(&appointmentDB, appointmentId)
		if appointmentDB.Doctor.ID == doctor.ID { //If correct Doctor
			if appointmentDB.AppointmentStatus.StatusDefId == 2 || appointmentDB.AppointmentStatus.StatusDefId == 3 { //If date is suggested or patient accepted date of appointment
				appointmentDB.AppointmentStatus = AppointmentStatus{StatusDefId: 5, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointmentDB.ID, CreatedById: user.ID}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointmentDB.AppointmentStatus)
				c.ormDB.Set("gorm:save_associations", false).Save(&appointmentDB)

				if &appointment != nil {
					//Now create new Appointment
					appointment.Doctor = appointmentDB.Doctor
					appointment.Patient = appointmentDB.Patient
					appointment.RequestDateTime = core.NullTime{Time: time.Now(), Valid: true}
					appointment.OriginAppointmentId = appointmentDB.ID
					appointment.OriginAppointment = &appointmentDB
					c.ormDB.Set("gorm:save_associations", false).Create(&appointment)

					appointment.AppointmentStatus = AppointmentStatus{StatusDefId: 1, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointment.ID, CreatedById: user.ID}
					c.ormDB.Set("gorm:save_associations", false).Create(&appointment.AppointmentStatus)
					c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.AppointmentStatus, appointment.AppointmentStatus.ID)
					c.ormDB.Set("gorm:save_associations", false).Save(&appointment)

				}
			}
		}
	} else if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Preload("Doctor").Preload("Patient").Preload("Patient.User").Preload("AppointmentStatus").Preload("AppointmentStatus.StatusDef").Find(&appointmentDB, appointmentId)
		if appointmentDB.Patient.ID == patient.ID { //If correct Patient
			if appointmentDB.AppointmentStatus.StatusDefId == 2 || appointmentDB.AppointmentStatus.StatusDefId == 3 { //If date is suggested or patient accepted date of appointment
				appointmentDB.AppointmentStatus = AppointmentStatus{StatusDefId: 6, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointmentDB.ID, CreatedById: user.ID}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointmentDB.AppointmentStatus)
				c.ormDB.Set("gorm:auto_preload", true).Find(&appointmentDB.AppointmentStatus, appointmentDB.AppointmentStatus.ID)
				c.ormDB.Set("gorm:save_associations", false).Save(&appointmentDB)

				//Now create new Appointment

				var appointment Appointment
				appointment.Doctor = appointmentDB.Doctor
				appointment.Patient = appointmentDB.Patient
				appointment.RequestDateTime = core.NullTime{Time: time.Now(), Valid: true}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointment)

				appointment.AppointmentStatus = AppointmentStatus{StatusDefId: 1, StatusDate: core.NullTime{Time: time.Now(), Valid: true}, SourceAppointmentId: appointment.ID, CreatedById: user.ID}
				c.ormDB.Set("gorm:save_associations", false).Create(&appointment.AppointmentStatus)
				c.ormDB.Set("gorm:auto_preload", true).Find(&appointment.AppointmentStatus, appointment.AppointmentStatus.ID)
				c.ormDB.Set("gorm:save_associations", false).Save(&appointment)

				appointmentDB = appointment
			}
		}
	}

	c.SendJSON(w, &appointmentDB, http.StatusOK)
}

func (c *PodiumController) hasConsent(doctor *Doctor, patient *Patient) bool {
	//if patient.UserType == 1 && doctor.UserType == 2 {
	consent := DoctorPatientRelation{}
	c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id=? AND patient_id=? AND consent_status=2", doctor.ID, patient.ID).First(&consent)
	if consent.ID > 0 {
		return true
	}
	return false
	//}
	//return false
}

func (c *PodiumController) isPatient(user *core.User) bool {
	if user.UserType == 1 {
		return true
	}
	return false
}

func (c *PodiumController) isSystemAdmin(user *core.User) bool {
	if user.IsSysadmin == true {
		return true
	}
	return false
}

func (c *PodiumController) getPatient(user *core.User) *Patient {
	if user.UserType == 1 {
		patient := &Patient{}
		c.ormDB.Set("gorm:auto_preload", true).Where("user_id=?", user.ID).First(&patient)
		return patient
	}
	return nil
}

func (c *PodiumController) isDoctor(user *core.User) bool {
	if user.UserType == 2 {
		return true
	}
	return false
}
func (c *PodiumController) getDoctor(user *core.User) *Doctor {
	if user.UserType == 2 {
		doctor := &Doctor{}
		c.ormDB.Debug().Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM doctor_users WHERE status > 0 AND user_id=?) ", user.ID).First(&doctor)
		return doctor
	}
	return nil
}

func (c *PodiumController) isPractice(user *core.User) bool {
	if user.UserType == 3 {
		return true
	}
	return false
}
func (c *PodiumController) getPractice(user *core.User) *Practice {
	if user.UserType == 3 {
		practice := &Practice{}
		c.ormDB.Set("gorm:auto_preload", true).Where("user_id = ?", user.ID).First(&practice)
		return practice
	}
	return nil
}

func (c *PodiumController) getPracticeUserFromPracticeId(practiceId uint) *core.User {
	practiceUser := core.User{}
	c.ormDB.Where("id IN (SELECT user_id FROM practices WHERE id = ?)", practiceId).First(&practiceUser)

	return &practiceUser
}

func (c *PodiumController) isSysadmin(user *core.User) bool {
	if user.UserType == 0 && user.IsSysadmin {
		return true
	}
	return false
}

func (c *PodiumController) getMainUserIdFromDoctorId(doctorId uint) uint {
	doctorUser := DoctorUser{}
	c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ?", doctorId).First(&doctorUser)
	return doctorUser.UserId
}

func (c *PodiumController) getMainUserFromDoctorId(doctorId uint) core.User {
	doctorUser := DoctorUser{}
	c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ?", doctorId).First(&doctorUser)
	return doctorUser.User
}

func (c *PodiumController) getMainUserIdsFromDoctors(doctors Doctors) []uint {
	doctorIds := []uint{}
	for _, item := range doctors {
		doctorIds = append(doctorIds, item.ID)
	}
	return c.getMainUserIdsFromDoctorIds(doctorIds)
}

func (c *PodiumController) Find(a []uint, x uint) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

func (c *PodiumController) getMainUserIdsFromDoctorIds(doctorIds []uint) []uint {
	doctorUsers := []DoctorUser{}
	c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id IN (?)", doctorIds).Find(&doctorUsers)

	ids := []uint{}

	for _, item := range doctorUsers {
		if item.UserId > 0 {
			ids = append(ids, item.UserId)
		}
	}
	return ids
}

func (c *PodiumController) getPracticeUserIdsFromDoctorIds(doctorIds []uint) []uint {
	users := core.Users{}
	//c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id IN (?)", doctorIds).Find(&practiceDoctor)
	c.ormDB.Debug().Set("gorm:auto_preload", true).Where("system_accounts.id IN (Select practices.user_id FROM practices Where practices.id IN (SELECT practice_doctors.practice_id FROM practice_doctors WHERE practice_doctors.doctor_id IN (?)))", doctorIds).Find(&users)

	ids := []uint{}

	for _, item := range users {
		if item.ID > 0 {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

// getMyMeasurements swagger:route GET /me/measurements measurements getMyMeasurements
//
// retrieves your measurements
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Measurement
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyMeasurementsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	measurements := Measurements{}

	c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", user.ID).Find(&measurements)

	c.SendJSON(w, &measurements, http.StatusOK)
}

// getMyChats swagger:route GET /me/conversations chats getMyChats
//
// retrieves your chats
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []HelperConversations
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyChatsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	searchValue := ""
	urlQuery := r.URL.Query()
	if len(urlQuery) > 0 {
		values := urlQuery
		if val, ok := values["search"]; ok && len(val) > 0 {
			searchValue = "%" + val[0] + "%"
		}
	}

	chats := HelperConversations{}

	if c.isDoctor(user) {
		//No chats are displayed from Pro users who are not Home or Remote users.
		doctor := c.getDoctor(user)
		if searchValue != "" {
			c.ormDB.Debug().Raw(`
SELECT
    sum(unread_messages) as unread_messages,
    message_time,
    interlocutor_id,
    id as last_message_id
FROM
    (
        (
            SELECT
                *
            FROM
                (
                    SELECT
                        id,
                        sum(is_unread) as unread_messages,
                        message_time,
                        sender_id as interlocutor_id,
                        doctor_id
                    from
                        messages
                    WHERE
                        deleted_at IS NULL
                        AND doctor_id = ?
                        AND sender_id NOT IN (
                            SELECT
                                user_id
                            FROM
                                doctor_users
                            WHERE
                                doctor_id = ?
                        )
                        AND sender_id NOT IN (
                            SELECT
                                user_id
                            FROM
                                practices
                            WHERE
                                id IN (
                                    SELECT
                                        practice_id
                                    FROM
                                        practice_doctors
                                    WHERE
                                        doctor_id = ?
                                )
                        )
                        AND (
                            sender_id IN (
                                SELECT
                                    sa.id
                                FROM
                                    patients p
                                    LEFT JOIN system_accounts sa ON p.user_id = sa.id
                                WHERE
                                    (LOWER(p.first_name) LIKE LOWER(?)
                                    OR LOWER(p.last_name) LIKE LOWER(?)
                                    OR LOWER(sa.username) LIKE LOWER(?)) 
									AND (home_user = 1 OR  remote_user = 1) 
                            ))
                            GROUP BY
                                id
                            ORDER BY
                                message_time DESC
                        ) A
                    GROUP BY
                        interlocutor_id DESC
                )
            UNION
                (
                    SELECT
                        *
                    FROM
                        (
                            SELECT
                                id,
                                0 as unread_messages,
                                message_time,
                                recipient_id as interlocutor_id,
                                doctor_id
                            from
                                messages
                            WHERE
                                deleted_at IS NULL
                                AND doctor_id = ?
                                AND recipient_id NOT IN (
                                    SELECT
                                        user_id
                                    FROM
                                        doctor_users
                                    WHERE
                                        doctor_id = ?
                                )
                                AND recipient_id NOT IN (
                                    SELECT
                                        user_id
                                    FROM
                                        practices
                                    WHERE
                                        id IN (
                                            SELECT
                                                practice_id
                                            FROM
                                                practice_doctors
                                            WHERE
                                                doctor_id = ?
                                        )
                                )
                                AND (
                            recipient_id IN (
                                SELECT
                                    sa.id
                                FROM
                                    patients p
                                    LEFT JOIN system_accounts sa ON p.user_id = sa.id
                                WHERE
									(LOWER(p.first_name) LIKE LOWER(?)
                                    OR LOWER(p.last_name) LIKE LOWER(?)
                                    OR LOWER(sa.username) LIKE LOWER(?)) 
									AND (home_user = 1 OR  remote_user = 1) 
                            ))
                            GROUP BY
                                id
                            ORDER BY
                                message_time DESC
                        ) B
                    GROUP BY
                        interlocutor_id
                )
            ORDER BY
                message_time
        ) t
        GROUP BY
            interlocutor_id
        ORDER BY
            message_time`, doctor.ID, doctor.ID, doctor.ID, searchValue, searchValue, searchValue, doctor.ID, doctor.ID, doctor.ID, searchValue, searchValue, searchValue).Scan(&chats)
		} else {
			c.ormDB.Debug().Raw(`
SELECT
    sum(unread_messages) as unread_messages,
    message_time,
    interlocutor_id,
    id as last_message_id
FROM
    (
        (
            SELECT
                *
            FROM
                (
                    SELECT
                        id,
                        sum(is_unread) as unread_messages,
                        message_time,
                        sender_id as interlocutor_id,
                        doctor_id
                    from
                        messages
                    WHERE
                        deleted_at IS NULL
                        AND doctor_id = ?
                        AND sender_id NOT IN (
                            SELECT
                                user_id
                            FROM
                                doctor_users
                            WHERE
                                doctor_id = ?
                        )
                        AND sender_id NOT IN (
                            SELECT
                                user_id
                            FROM
                                practices
                            WHERE
                                id IN (
                                    SELECT
                                        practice_id
                                    FROM
                                        practice_doctors
                                    WHERE
                                        doctor_id = ?
                                )
                        )
                        AND (
                            sender_id IN (
                                SELECT
                                    sa.id
                                FROM
                                    patients p
                                    LEFT JOIN system_accounts sa ON p.user_id = sa.id
                                WHERE home_user = 1 OR  remote_user = 1 
                            ))
                            GROUP BY
                                id
                            ORDER BY
                                message_time DESC
                        ) A
                    GROUP BY
                        interlocutor_id DESC
                )
            UNION
                (
                    SELECT
                        *
                    FROM
                        (
                            SELECT
                                id,
                                0 as unread_messages,
                                message_time,
                                recipient_id as interlocutor_id,
                                doctor_id
                            from
                                messages
                            WHERE
                                deleted_at IS NULL
                                AND doctor_id = ?
                                AND recipient_id NOT IN (
                                    SELECT
                                        user_id
                                    FROM
                                        doctor_users
                                    WHERE
                                        doctor_id = ?
                                )
                                AND recipient_id NOT IN (
                                    SELECT
                                        user_id
                                    FROM
                                        practices
                                    WHERE
                                        id IN (
                                            SELECT
                                                practice_id
                                            FROM
                                                practice_doctors
                                            WHERE
                                                doctor_id = ?
                                        )
                                )
                                AND (
                            recipient_id IN (
                                SELECT
                                    sa.id
                                FROM
                                    patients p
                                    LEFT JOIN system_accounts sa ON p.user_id = sa.id
                                WHERE
									home_user = 1 OR  remote_user = 1 
                            ))
                            GROUP BY
                                id
                            ORDER BY
                                message_time DESC
                        ) B
                    GROUP BY
                        interlocutor_id
                )
            ORDER BY
                message_time
        ) t
        GROUP BY
            interlocutor_id
        ORDER BY
            message_time`, doctor.ID, doctor.ID, doctor.ID, doctor.ID, doctor.ID, doctor.ID).Scan(&chats)

			/*c.ormDB.Debug().Raw(`SELECT sum(unread_messages) as unread_messages, message_time, interlocutor_id, id as last_message_id FROM ((SELECT * FROM (SELECT id, sum(is_unread) as unread_messages, message_time, sender_id as interlocutor_id, doctor_id from messages WHERE deleted_at IS NULL AND doctor_id =? AND sender_id NOT IN (SELECT user_id FROM doctor_users WHERE doctor_id = ?) AND sender_id NOT IN (SELECT user_id FROM practices WHERE id IN (SELECT practice_id FROM practice_doctors WHERE doctor_id = ?)) GROUP BY id ORDER BY message_time DESC) A GROUP BY interlocutor_id DESC)
			UNION
			(SELECT * FROM (SELECT id, 0 as unread_messages, message_time, recipient_id as interlocutor_id, doctor_id from messages WHERE deleted_at IS NULL AND doctor_id =? AND recipient_id NOT IN (SELECT user_id FROM doctor_users WHERE doctor_id = ?) AND recipient_id NOT IN (SELECT user_id FROM practices WHERE id IN (SELECT practice_id FROM practice_doctors WHERE doctor_id = ?)) GROUP BY id ORDER BY message_time DESC) B GROUP BY interlocutor_id) ORDER BY message_time DESC) t GROUP BY interlocutor_id ORDER BY message_time;`, doctor.ID, doctor.ID, doctor.ID, doctor.ID, doctor.ID, doctor.ID).Scan(&chats)*/
		}
	} else if c.isPatient(user) {
		if searchValue != "" {
			c.ormDB.Debug().Raw(`
SELECT
    sum(unread_messages) as unread_messages,
    message_time,
    interlocutor_id,
    id as last_message_id,
    doctor_id
FROM
    (
        (
            SELECT
                *
            FROM
                (
                    SELECT
                        id,
                        sum(is_unread) as unread_messages,
                        message_time,
                        sender_id as interlocutor_id,
                        doctor_id
                    from
                        messages
                    WHERE
                        deleted_at IS NULL
                        AND recipient_id = ?
                        AND sender_id IN (SELECT sa.id FROM system_accounts sa LEFT JOIN doctor_users du ON sa.id = du.user_id LEFT JOIN doctors d ON du.doctor_id = d.id WHERE LOWER(sa.username) LIKE LOWER(?) OR LOWER(d.name) LIKE LOWER(?))
                    GROUP BY
                        id
                    ORDER BY
                        message_time DESC
                ) A
            GROUP BY
                interlocutor_id DESC
        )
        UNION
            (
                SELECT
                    *
                FROM
                    (
                        SELECT
                            id,
                            0 as unread_messages,
                            message_time,
                            recipient_id as interlocutor_id,
                            doctor_id
                        from
                            messages
                        WHERE
                            deleted_at IS NULL
                            AND sender_id = ?
                            AND recipient_id IN (SELECT sa.id FROM system_accounts sa LEFT JOIN doctor_users du ON sa.id = du.user_id LEFT JOIN doctors d ON du.doctor_id = d.id WHERE LOWER(sa.username) LIKE LOWER(?) OR LOWER(d.name) LIKE LOWER(?))
                        GROUP BY
                            id
                        ORDER BY
                            message_time DESC
                    ) B
                GROUP BY
                    interlocutor_id
            )
        ORDER BY
            message_time DESC
    ) t
GROUP BY
    doctor_id
ORDER BY
    message_time;`, user.ID, searchValue, searchValue, user.ID, searchValue, searchValue).Scan(&chats)
		} else {
			c.ormDB.Debug().Raw(`SELECT sum(unread_messages) as unread_messages, message_time, interlocutor_id, id as last_message_id, doctor_id FROM ((SELECT * FROM (SELECT id, sum(is_unread) as unread_messages, message_time, sender_id as interlocutor_id, doctor_id from messages WHERE deleted_at IS NULL AND recipient_id =? GROUP BY id ORDER BY message_time DESC) A GROUP BY interlocutor_id DESC)
UNION
(SELECT * FROM (SELECT id, 0 as unread_messages, message_time, recipient_id as interlocutor_id, doctor_id from messages WHERE deleted_at IS NULL AND sender_id =? GROUP BY id ORDER BY message_time DESC) B GROUP BY interlocutor_id) ORDER BY message_time DESC) t GROUP BY doctor_id ORDER BY message_time;`, user.ID, user.ID).Scan(&chats)
		}
	}

	for key, chat := range chats {
		c.ormDB.Find(&chat.Interlocutor, chat.InterlocutorId)
		chat.InterlocutorHelper.User = chat.Interlocutor
		chat.InterlocutorHelper.PasswordX = ""
		if c.isPatient(&chat.Interlocutor) {
			chat.InterlocutorHelper.Patient = c.getPatient(&chat.Interlocutor)
			//c.ormDB.Set("gorm:auto_preload", true).Where("sender_id = ? or recipient_id =?", user.ID, user.ID).Where("sender_id = ? or recipient_id =?", chat.InterlocutorHelper.ID, chat.InterlocutorHelper.ID).Last(&chat.LastMessage)
		} else if c.isDoctor(&chat.Interlocutor) {
			chat.InterlocutorHelper.Doctor = c.getDoctor(&chat.Interlocutor)
			//c.ormDB.Debug().Set("gorm:auto_preload", true).Where("sender_id = ? or recipient_id =?", user.ID, user.ID).Where("sender_id = ? or recipient_id =?", chat.InterlocutorHelper.ID, chat.InterlocutorHelper.ID).Last(&chat.LastMessage)
		} else if c.isPractice(&chat.Interlocutor) {
			chat.InterlocutorHelper.User = c.getMainUserFromDoctorId(chat.DoctorId)
			chat.InterlocutorHelper.Doctor = c.getDoctor(&chat.InterlocutorHelper.User)

		}
		c.ormDB.Debug().Set("gorm:auto_preload", true).Find(&chat.LastMessage, chat.LastMessageId)
		log.Printf("%d - %d - %s\n", key, chat.InterlocutorId, chat.LastMessage.MessageTime.Time.Format("2006-01-02 15:04:05"))
		//chats[key] = chat
		chats[key] = chat
	}

	sort.Sort(&chats)

	c.SendJSON(w, &chats, http.StatusOK)
}

// GetChatMessages swagger:route GET /me/conversations/{userId} chats getMyChats
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	userId, _ := strconv.ParseInt(vars["userId"], 10, 64)

	interlocutor := core.User{}

	c.ormDB.Find(&interlocutor, userId)

	messages := &Messages{}

	searchValue := ""
	urlQuery := r.URL.Query()
	if len(urlQuery) > 0 {
		values := urlQuery
		if val, ok := values["search"]; ok && len(val) > 0 {
			searchValue = "%" + val[0] + "%"
		}
	}

	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		db := c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? ", doctor.ID).Where("sender_id = ? OR recipient_id = ?", interlocutor.ID, interlocutor.ID)
		if searchValue != "" {
			db = db.Where("message_text LIKE ?", searchValue)
		}
		db.Find(&messages)
		c.ormDB.Model(&Message{}).Where("doctor_id = ? ", doctor.ID).Where("sender_id = ?", interlocutor.ID).Update("is_unread", false)
	} else if c.isDoctor(&interlocutor) {
		doctor := c.getDoctor(&interlocutor)
		db := c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? ", doctor.ID).Where("sender_id = ? OR recipient_id = ?", user.ID, user.ID)
		if searchValue != "" {
			db = db.Where("message_text LIKE ?", searchValue)
		}
		db.Find(&messages)
		c.ormDB.Model(&Message{}).Where("doctor_id = ? ", doctor.ID).Where("recipient_id = ?", user.ID).Update("is_unread", false)
	}

	c.SendJSON(w, &messages, http.StatusOK)
}

// GetChatMessages swagger:route GET /me/conversations/{userId} chats getMyChats
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetNewChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	userId, _ := strconv.ParseInt(vars["userId"], 10, 64)
	messageId, _ := strconv.ParseInt(vars["messageId"], 10, 64)

	messages := &Messages{}

	interlocutor := core.User{}

	c.ormDB.Find(&interlocutor, userId)

	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? ", doctor.ID).Where("sender_id = ? OR recipient_id = ?", interlocutor.ID, interlocutor.ID).Where("id > ?", messageId).Find(&messages)
		c.ormDB.Model(&Message{}).Where("doctor_id = ? ", doctor.ID).Where("sender_id = ?", interlocutor.ID).Where("id > ?", messageId).Update("is_unread", false)
	} else if c.isDoctor(&interlocutor) {
		doctor := c.getDoctor(&interlocutor)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? ", doctor.ID).Where("sender_id = ? OR recipient_id = ?", user.ID, user.ID).Where("id > ?", messageId).Find(&messages)
		c.ormDB.Model(&Message{}).Where("doctor_id = ? ", doctor.ID).Where("recipient_id = ?", user.ID).Where("id > ?", messageId).Update("is_unread", false)
		if c.isPatient(user) {
			c.ormDB.Debug().Set("gorm:save_associations", false).Model(&Notification{}).Where("user_id = ? AND foreign_id <= ? AND notification_type = 1", user.ID, messageId).Where("foreign_id = (Select messages.id From messages Where doctor_id = ? AND messages.id = foreign_id)", doctor.ID).Update("visible", false)
		}
	}

	c.SendJSON(w, &messages, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) DeleteChatMessagesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	userId, _ := strconv.ParseInt(vars["userId"], 10, 64)
	messageId, _ := strconv.ParseInt(vars["messageId"], 10, 64)

	message := &Message{}

	interlocutor := core.User{}

	c.ormDB.Find(&interlocutor, userId)

	if c.isDoctor(user) {
		doctor := c.getDoctor(user)

		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? ", doctor.ID).Delete(&message, messageId)
	} else if c.isPatient(user) {
		c.ormDB.Set("gorm:auto_preload", true).Where("sender_id=?", user.ID).Delete(&message, messageId)
	}

	c.SendJSON(w, &message, http.StatusOK)
	//go systembundle.SendWebsocketDataInfoMessage("Delete message", systembundle.Websocket_Delete, systembundle.Websocket_Messages, uint(messageId), 0, nil)
}

// SaveChatMessage swagger:route POST /me/conversations chats saveMyChats
//
// saves a Message of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveChatMessageHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	message := &Message{}
	if err := c.GetContent(&message, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if message.ID == 0 {
		message.SenderId = user.ID
		message.Sender = *user
	} else if message.Sender.ID != user.ID {
		return
	}

	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Find(&message.Recipient, message.Recipient.ID)
		if !c.isPatient(&message.Recipient) {
			err := errors.New("You can only chat with patients")
			c.HandleError(err, w)
			return
		}
		// check if relation/consent exists
		patient := c.getPatient(&message.Recipient)
		if !c.hasConsent(doctor, patient) {
			err := errors.New("You can only chat with your patients")
			c.HandleError(err, w)
			return
		}
		message.DoctorId = doctor.ID
	} else if c.isPatient(user) {
		patient := c.getPatient(user)

		c.ormDB.Set("gorm:auto_preload", true).Find(&message.Recipient, message.Recipient.ID)
		if !c.isDoctor(&message.Recipient) {
			err := errors.New("You can only chat with clinicians")
			c.HandleError(err, w)
			return
		}
		doctor := c.getDoctor(&message.Recipient)
		if !c.hasConsent(doctor, patient) {
			err := errors.New("You can only chat with your podiatrist")
			c.HandleError(err, w)
			return
		}
		message.DoctorId = doctor.ID
	}

	message.MessageTime.Time = time.Now()
	message.MessageTime.Valid = true

	message.IsUnread = true

	c.ormDB.Set("gorm:save_associations", false).Save(&message)

	for i, measure := range message.Attachments {
		measureShared := &MeasurementShared{}
		c.ormDB.Set("gorm:auto_preload", false).Where("measurement_id = ?", measure.ID).Where("doctor_id = ?", message.DoctorId).First(&measureShared)
		if measureShared.ID == 0 {
			measureShared.DoctorId = message.DoctorId
			measureShared.MeasurementId = measure.ID
			c.ormDB.Set("gorm:save_associations", false).Create(measureShared)
		}
		c.ormDB.Set("gorm:auto_preload", false).Find(&measure, measure.ID)
		message.Attachments[i] = measure
	}

	c.CreateNotification(message.Recipient.ID, 1, fmt.Sprintf("New Message from %s", message.Sender.Username), message.MessageText, message.ID, fmt.Sprintf("/me/conversations/%d", message.Sender.ID), nil)

	c.SendJSON(w, &message, http.StatusOK)
	//go systembundle.SendWebsocketDataInfoMessage("Update message", systembundle.Websocket_Add, systembundle.Websocket_Messages, message.ID, 0, nil)
}

// getMyPatients swagger:route GET /me/patients patient getMyPatients
//
// retrieves your patients
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Patient
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyPatientsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok { ///&& user.UserType == 0
		_ = user
		return
	}

	paging := c.GetPaging(r.URL.Query())
	db, dbTotalCount := c.CreateWhereConditionsMyPatients(r.URL.Query(), r, user)

	offset := paging.Offset

	patients := Patients{}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		practiceUserId := doctor.GetPracticeUserId(c.ormDB)
		db.Debug().Set("gorm:auto_preload", true).Where("patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ? AND consent_status IN (2))", doctor.ID).Limit(paging.Limit).Offset(offset).Find(&patients) // nur  angenommeneWhere("patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ? AND consent_status IN (2))", doctor.ID).Find(&patients) // nur  angenommene
		dbTotalCount = dbTotalCount.Where("patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ? AND consent_status IN (2))", doctor.ID)
		for i, patient := range patients {
			c.ormDB.DB().QueryRow("SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =? AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1", patient.ID).Scan(&patient.LastAppointmentDate)
			c.ormDB.DB().QueryRow("SELECT measurement_date FROM measurements m WHERE m.patient_id =? ORDER BY measurement_date DESC LIMIT 1", patient.ID).Scan(&patient.LastMeasurementDate)

			c.ormDB.Model(&Measurement{}).Where("patient_id = ? AND user_id = ?", patient.ID, practiceUserId).Count(&patient.CountScans)
			questions := QuestionTemplates{}
			c.ormDB.Set("gorm:auto_preload", false).Where("doctor_id = ?", doctor.ID).Where("patient_id = ?", patient.ID).Where("question_type = 3").Find(&questions)
			if len(questions) > 0 {
				patient.AddedOwnQuestions = true
			}
			patients[i] = patient
		}
		fmt.Println("Number of patients for doctor >>>>>>>>>>>>>>>>:", len(patients))
	} else if c.isPractice(user) {

		practice := c.getPractice(user)
		//TODO SM RS SIMON FULL DOCTOR MERGE
		//doctors := practice.getFullDoctors(c.ormDB)
		doctors := practice.GetDoctors(c.ormDB)
		userIdsOfDoctors := c.getMainUserIdsFromDoctors(doctors)

		db.Set("gorm:auto_preload", true).Where("user_id IN (SELECT id FROM system_accounts WHERE created_by = ?) OR patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status IN (2))", user.ID, practice.ID).Limit(paging.Limit).Offset(offset).Find(&patients) // nur  angenommene
		dbTotalCount = dbTotalCount.Where("user_id IN (SELECT id FROM system_accounts WHERE created_by = ?) OR patients.id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status IN (2))", user.ID, practice.ID)
		//c.ormDB.Set("gorm:auto_preload", true).Where("user_id IN (SELECT id FROM system_accounts WHERE created_by = ?) OR id IN ( SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status IN (2))", user.ID, practice.ID).Find(&patients) // nur  angenommene
		fmt.Print(userIdsOfDoctors)
		for i, patient := range patients {
			c.ormDB.DB().QueryRow("SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =? AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1", patient.ID).Scan(&patient.LastAppointmentDate)
			c.ormDB.DB().QueryRow("SELECT measurement_date FROM measurements m WHERE m.patient_id =? ORDER BY measurement_date DESC LIMIT 1", patient.ID).Scan(&patient.LastMeasurementDate)
			c.ormDB.Model(&Measurement{}).Where("patient_id = ? AND user_id = ?", patient.ID, user.ID).Count(&patient.CountScans)

			if -1 < c.Find(userIdsOfDoctors, patient.User.CreatedBy) || user.ID == patient.User.CreatedBy {
				patient.PracticeCanEdit = true
			}

			patients[i] = patient
		}
		fmt.Println("Number of patients for practice ++++++++++++++++++++++:", len(patients))
	}

	dbTotalCount.Model(&Patient{}).Count(&paging.TotalCount)

	c.SendJSONPaging(w, r, paging, &patients, http.StatusOK)
}

func (c *PodiumController) CreateWhereConditionsMyPatients(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	specificForPractice := false

	switch user.UserType {
	case core.UserTypePractice:
		{
			specificForPractice = true
		}
	}

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()

	if len(urlQuery) > 0 {
		values := urlQuery

		hTime := r.Header["X-Timezone"]

		if val, ok := values["search"]; ok && len(val) > 0 {

			search := "%" + val[0] + "%"

			queryHasTimeOrDate := false
			var date core.NullTime
			tmp := val[0]
			date.FromString(tmp)

			if len(tmp) >= 4 {
				if date.Valid {
					queryHasTimeOrDate = true

					if specificForPractice {
						db = db.Where("(SELECT patient_id FROM measurements WHERE patients.id = patient_id AND user_id = ? AND measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND DATE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE DATE(?)) OR DATE(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE DATE(?) OR (SELECT date_time_from  FROM appointments WHERE appointments.patient_id = patients.id AND date_time_from = (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1) AND DATE(COALESCE(CONVERT_TZ(date_time_from, 'UTC', ?), date_time_from)) LIKE DATE(?))", user.ID, hTime, date, hTime, date, hTime, date)
						dbTotalCount = dbTotalCount.Where("(SELECT patient_id FROM measurements WHERE patients.id = patient_id AND user_id = ? AND measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND DATE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE DATE(?)) OR DATE(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE DATE(?) OR (SELECT date_time_from  FROM appointments WHERE appointments.patient_id = patients.id AND date_time_from = (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1) AND DATE(COALESCE(CONVERT_TZ(date_time_from, 'UTC', ?), date_time_from)) LIKE DATE(?))", user.ID, hTime, date, hTime, date, hTime, date)
					} else {
						db = db.Where("(SELECT patient_id FROM measurements WHERE patients.id = patient_id AND measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND DATE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE DATE(?)) OR DATE(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE DATE(?) OR     DATE(COALESCE(CONVERT_TZ( (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1), 'UTC', ?),  (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1)))  LIKE DATE(?) ", hTime, date, hTime, date, hTime, date)
						dbTotalCount = dbTotalCount.Where("(SELECT patient_id FROM measurements WHERE patients.id = patient_id AND measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND DATE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE DATE(?)) OR DATE(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE DATE(?) OR     DATE(COALESCE(CONVERT_TZ( (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1), 'UTC', ?),  (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1)))  LIKE DATE(?) ", hTime, date, hTime, date, hTime, date)
					}
					//db = db.Where("(SELECT patient_id FROM measurements WHERE patients.id = patient_id AND  measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR  YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE YEAR(?) AND MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE MONTH(?) AND DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE DAYOFMONTH(?) LIMIT 1)", hTime, date, hTime, date, hTime, date, hTime, date)
					//dbTotalCount = dbTotalCount.Where("(SELECT patient_id FROM measurements WHERE patients.id = patient_id AND measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR  YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE YEAR(?) AND MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE MONTH(?) AND DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE DAYOFMONTH(?) LIMIT 1)", hTime, date, hTime, date, hTime, date, hTime, date)
					//dbTotalCount = dbTotalCount.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR date(measurements.measurement_date) = date(?) OR time(measurements.measurement_date) = time(?))", hTime, search, search, search, search, date, date)
				}
			}
			//}

			if !queryHasTimeOrDate {
				if specificForPractice {
					db = db.Where("(SELECT COUNT(patient_id) FROM measurements WHERE deleted_at IS NULL AND patient_id = patients.id AND user_id = ?) LIKE ? OR CONCAT(last_name, first_name) LIKE ? OR town Like ? OR  user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR patients.id = (SELECT patient_id FROM measurements WHERE measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND (YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ?)) OR YEAR(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ?   OR   (SELECT date_time_from  FROM appointments WHERE appointments.patient_id = patients.id AND date_time_from = (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1) AND COALESCE(CONVERT_TZ(date_time_from, 'UTC', ?), date_time_from) LIKE ?) ", user.ID, search, search, search, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search) // OR  user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR id = (SELECT id FROM measurements m WHERE (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND (YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ?))
					dbTotalCount = dbTotalCount.Where("(SELECT COUNT(patient_id) FROM measurements WHERE deleted_at IS NULL AND patient_id = patients.id AND user_id = ?) LIKE ? OR CONCAT(last_name, first_name) LIKE ? OR town Like ? OR  user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR patients.id = (SELECT patient_id FROM measurements WHERE measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND (YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ?)) OR YEAR(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR   (SELECT date_time_from  FROM appointments WHERE appointments.patient_id = patients.id AND date_time_from = (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1) AND COALESCE(CONVERT_TZ(date_time_from, 'UTC', ?), date_time_from) LIKE ?)", user.ID, search, search, search, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search)
				} else {
					db = db.Where("(SELECT COUNT(patient_id) FROM measurements WHERE deleted_at IS NULL AND patient_id = patients.id ) LIKE ? OR CONCAT(last_name, first_name) LIKE ? OR town Like ? OR  user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR patients.id = (SELECT patient_id FROM measurements WHERE measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND (YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ?)) OR YEAR(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR   (SELECT date_time_from  FROM appointments WHERE appointments.patient_id = patients.id AND date_time_from = (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1) AND COALESCE(CONVERT_TZ(date_time_from, 'UTC', ?), date_time_from) LIKE ?)", search, search, search, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search) // OR  user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR id = (SELECT id FROM measurements m WHERE (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND (YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ?))
					dbTotalCount = dbTotalCount.Where("(SELECT COUNT(patient_id) FROM measurements WHERE deleted_at IS NULL AND patient_id = patients.id ) LIKE ? OR CONCAT(last_name, first_name) LIKE ? OR town Like ? OR  user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?) OR patients.id = (SELECT patient_id FROM measurements WHERE measurement_date = (SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1) AND (YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ?)) OR YEAR(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(birth_date, 'UTC', ?),birth_date)) LIKE ? OR   (SELECT date_time_from  FROM appointments WHERE appointments.patient_id = patients.id AND date_time_from = (SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1) AND COALESCE(CONVERT_TZ(date_time_from, 'UTC', ?), date_time_from) LIKE ?)", search, search, search, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search)
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
					case "last_appointment_date":

						query := fmt.Sprintf("(SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id WHERE aps.status_def_id = 3 AND a.patient_id = patients.id AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1)")
						db = db.Order(fmt.Sprintf("%s %s", query, sortDirection))
						break
					case "count_scans":
						if specificForPractice {
							query := fmt.Sprintf("(SELECT COUNT(patient_id) FROM measurements WHERE deleted_at IS NULL AND patient_id = patients.id AND user_id = %v)", user.ID)
							db = db.Order(fmt.Sprintf("%s %s", query, sortDirection))
						} else {
							query := fmt.Sprintf("(SELECT COUNT(patient_id) FROM measurements WHERE deleted_at IS NULL AND patient_id = patients.id )")
							db = db.Order(fmt.Sprintf("%s %s", query, sortDirection))
						}
						break
					case "last_measurement_date":
						/*if specificForPracticeOrDoctor {
							query := fmt.Sprintf("(SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id AND user_id = %v ORDER BY measurement_date DESC LIMIT 1)", user.ID)
							db = db.Order(fmt.Sprintf("%s %s", query, sortDirection))
						}else {*/
						query := fmt.Sprintf("(SELECT measurement_date FROM measurements m WHERE m.patient_id = patients.id ORDER BY measurement_date DESC LIMIT 1)")
						db = db.Order(fmt.Sprintf("%s %s", query, sortDirection))
						//}
						break
					case "username":
						db = db.Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						break
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

// getMyStatistics swagger:route GET /me/statistics patient getMyStatistics
//
// retrieves your patients
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Statistic
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isDoctor(user) && !c.isPractice(user) {
		c.HandleError(nil, w)
		return
	}

	if c.isDoctor(user) {

		statistic := HelperStatistic{}

		doctor := c.getDoctor(user)

		c.ormDB.Model(&Appointment{}).Where("doctor_id = ?", doctor.ID).Where("MONTH(NOW())=MONTH(date_time_from) AND YEAR(NOW())=YEAR(date_time_from)").Where("id IN (SELECT source_appointment_id FROM appointment_statuses WHERE id IN (SELECT appointment_status_id FROM appointments) AND status_def_id = 3)").Count(&statistic.AppointmentsThisMonth)
		c.ormDB.Model(&DoctorPatientRelation{}).Where("doctor_id = ? AND consent_status = 2", doctor.ID).Count(&statistic.NumberOfPatients)
		c.ormDB.Model(&DoctorPatientRelation{}).Where("doctor_id = ? AND consent_status = 2 AND MONTH(NOW())=MONTH(consent_date) AND YEAR(NOW())=YEAR(consent_date)", doctor.ID).Count(&statistic.NewPatientsThisMonth)

		c.SendJSON(w, &statistic, http.StatusOK)
	} else if c.isPractice(user) {

		statistic := PracticeStatistic{}
		c.ormDB.Model(&core.User{}).Where("user_type = 1 AND created_by = ?", user.ID).Count(&statistic.CountCustomers)
		c.ormDB.Model(&core.User{}).Where("user_type = 2 AND created_by = ?", user.ID).Count(&statistic.CountDoctors)
		c.ormDB.Model(&Measurement{}).Where("user_id = ?", user.ID).Where("YEAR(measurement_date) = YEAR(NOW())").Count(&statistic.CountScansThisYear)
		c.ormDB.Model(&Measurement{}).Where("user_id = ?", user.ID).Where("YEAR(measurement_date) = YEAR(NOW()) AND MONTH(measurement_date) = MONTH(NOW())").Count(&statistic.CountScansThisMonth)
		c.ormDB.Model(&Measurement{}).Where("user_id = ?", user.ID).Where("YEARWEEK(measurement_date) = YEARWEEK(NOW())").Count(&statistic.CountScansThisWeek)
		c.ormDB.Model(&Measurement{}).Where("user_id = ?", user.ID).Where("DATE(measurement_date) = DATE(NOW())").Count(&statistic.CountScansToday)
		c.ormDB.Model(&Measurement{}).Where("user_id = ?", user.ID).Count(&statistic.CountScansTotal)
		statistic.CountSuccessfulScans = 0
		//		practiceDoctors := Doctors{}

		practice := c.getPractice(user)
		doctors := Doctors{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("doctors.id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(&doctors)

		for _, doctor := range doctors { //USER_ID = 545,  Doctor = 385
			count := int64(0)
			c.ormDB.Debug().Model(&Measurement{}).Where("user_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND ten_mins_apart AND deleted_at IS NULL))", user.ID, doctor.ID).Count(&count)

			statistic.CountSuccessfulScans += count
		}

		/*patients := Patients{}
		c.ormDB.Where("id IN (SELECT patient_id FROM measurements WHERE deleted_at IS NULL AND user_id = ?)", user.ID).Find(&patients)
		//		c.ormDB.Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(&practiceDoctors)
		//		for _, doctor := range practiceDoctors {
		for _, patient := range patients {
			measurements := Measurements{}
			lastGoodMeasurement := Measurement{}
			c.ormDB.Where("user_id = ? AND patient_id = ?", user.ID, patient.ID).Order("measurement_date ASC").Find(&measurements)
			for _, measurement := range measurements {
				if math.Abs(float64(measurement.MeasurementDate.Time.Unix()-lastGoodMeasurement.MeasurementDate.Time.Unix())) > 600 {
					statistic.CountSuccessfulScans++
					lastGoodMeasurement = measurement
				}
			}
		}*/

		//		}
		c.ormDB.Raw("SELECT YEAR(measurement_date) as year, MONTH(measurement_date) as month, COUNT(id) as count_scans FROM measurements WHERE user_id =? GROUP BY YEAR(measurement_date), MONTH(measurement_date)", user.ID).Find(&statistic.CountScansPerMonth)
		c.ormDB.Raw("SELECT YEAR(measurement_date) as year, MONTH(measurement_date) as month, WEEK(measurement_date) as week, COUNT(id) as count_scans FROM measurements WHERE user_id =? GROUP BY YEAR(measurement_date), MONTH(measurement_date), WEEK(measurement_date)", user.ID).Find(&statistic.CountScansPerWeek)

		c.SendJSON(w, &statistic, http.StatusOK)
	}
}

// setTutorialSeen swagger:route PUT /me/tutorial/seen patient setTutorialSeen
//
// set if the user do not wantg to see tutorial again
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: User
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SetTutorialSeenHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	c.ormDB.Model(&core.User{}).Where("id=?", user.ID).Update("is_tutorial_seen", true)

	c.SendJSON(w, user, http.StatusOK)
}

// GetQuestionTemplates swagger:route GET /questions/templates questionnaire getQuestionTemplates
//
// retrieves all template questions
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []QuestionTemplate
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetQuestionTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	questions := &QuestionTemplates{}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id = ?", doctor.ID).Find(&questions)
	} else if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id IN ( SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id = ?)", patient.ID).Where("patient_id = 0 OR patient_id = ?", patient.ID).Find(&questions)
	} else {
		c.ormDB.Set("gorm:auto_preload", true).Find(&questions)
	}

	c.SendJSON(w, &questions, http.StatusOK)
}

// GetPatientQuestionTemplates swagger:route GET /questions/templates/patient/{patientId} questionnaire getQuestionTemplates
//
// retrieves all template questions
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []QuestionTemplate
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPatientQuestionTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	patientId, _ := strconv.ParseInt(vars["patientId"], 10, 64)
	questions := QuestionTemplates{}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id = ?", doctor.ID).Where("patient_id = 0 OR patient_id =?", patientId).Where("question_type != 1").Find(&questions)

		// add scan rating (red, blue, bluered, green
		question := QuestionTemplate{}
		question.QuestionText = "Scan ratings"
		question.QuestionKeyword = "Scan ratings"
		question.Answers = []QuestionTemplateAnswer{
			QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "redblue"},
			QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "blue"},
			QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "green"},
			QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "red"},
		}

		questions = append(QuestionTemplates{question}, questions...)

		question2 := QuestionTemplate{}
		question2.QuestionText = "Risk ratings"
		question2.QuestionKeyword = "Risk ratings"
		question2.Answers = []QuestionTemplateAnswer{
			QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "LR"},
			QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "MR"},
			QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "HR"},
			QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "AP"},
		}

		questions = append(QuestionTemplates{question2}, questions...)

		question3 := QuestionTemplate{}
		question3.QuestionText = "Doctor Risk ratings"
		question3.QuestionKeyword = "Doctor Risk ratings"
		question3.Answers = []QuestionTemplateAnswer{
			QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "LR"},
			QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "MR"},
			QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "HR"},
			QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "AP"},
		}

		questions = append(QuestionTemplates{question3}, questions...)

	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?)", practice.ID).Where("patient_id = 0 OR patient_id =?", patientId).Where("question_type != 1").Find(&questions)

		// add scan rating (red, blue, bluered, green
		/*	question := QuestionTemplate{}
			question.QuestionText = "Scan ratings"
			question.QuestionKeyword = "Scan ratings"
			question.Answers = []QuestionTemplateAnswer{
				QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "redblue"},
				QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "blue"},
				QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "green"},
				QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "red"},
			}

			questions = append(QuestionTemplates{question}, questions...)

			question2 := QuestionTemplate{}
			question2.QuestionText = "Risk ratings"
			question2.QuestionKeyword = "Risk ratings"
			question2.Answers = []QuestionTemplateAnswer{
				QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "LR"},
				QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "MR"},
				QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "HR"},
				QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "AP"},
			}

			questions = append(QuestionTemplates{question2}, questions...)*/
		question3 := QuestionTemplate{}
		question3.QuestionText = "Doctor Risk ratings"
		question3.QuestionKeyword = "Doctor Risk ratings"
		question3.Answers = []QuestionTemplateAnswer{
			QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "LR"},
			QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "MR"},
			QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "HR"},
			QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "AP"},
		}

		questions = append(QuestionTemplates{question3}, questions...)

	} else if c.isPatient(user) {
		/*patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id = ?", doctor.ID).Find(&questionnaires)*/
	} else {
		c.ormDB.Set("gorm:auto_preload", true).Find(&questions)
	}

	c.SendJSON(w, &questions, http.StatusOK)
}

// GetPatientQuestionTemplates swagger:route GET /questions/templates/patient/{patientId} questionnaire getQuestionTemplates
//
// retrieves all template questions
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []QuestionTemplate
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyQuestionTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		questions := QuestionTemplates{}

		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = 0 OR patient_id =?", patient.ID).Where("question_type != 1").Find(&questions)

		// add scan rating (red, blue, bluered, green
		question := QuestionTemplate{}
		question.QuestionText = "Scan ratings"
		question.QuestionKeyword = "Scan ratings"
		question.Answers = []QuestionTemplateAnswer{
			QuestionTemplateAnswer{AnswerValue: 1, AnswerText: "redblue"},
			QuestionTemplateAnswer{AnswerValue: 2, AnswerText: "blue"},
			QuestionTemplateAnswer{AnswerValue: 3, AnswerText: "green"},
			QuestionTemplateAnswer{AnswerValue: 4, AnswerText: "red"},
		}

		questions = append(QuestionTemplates{question}, questions...)

		c.SendJSON(w, &questions, http.StatusOK)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		questions := QuestionTemplates{}

		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ?", doctor.ID).Where("question_type != 1").Find(&questions)

		patient := Patient{}
		time := core.NullTime{}
		for key, item := range questions {
			item.Patient = &patient
			item.DateTo = time
			item.DateFrom = time
			questions[key] = item
		}

		c.SendJSON(w, &questions, http.StatusOK)
	}
}

// GetQuestionTemplate swagger:route GET /questions/template/{questionTemplateId} questionnaire getQuestionTemplate
//
// retrieves a QuestionTemplate specified by ID
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: QuestionTemplate
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetQuestionTemplateHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	questionTemplateId, _ := strconv.ParseInt(vars["questionTemplateId"], 10, 64)

	question := &QuestionTemplate{}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id = ?", doctor.ID).Find(&question, questionTemplateId)
	} else if c.isPatient(user) {
		/*patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id = ?", doctor.ID).Find(&questionnaires)*/
	} else {
		c.ormDB.Set("gorm:auto_preload", true).Find(&question, questionTemplateId)
	}

	c.SendJSON(w, &question, http.StatusOK)
}

// SaveQuestionTemplate swagger:route POST /questions/template questionnaire SaveQuestionTemplate
//
// # Saves a QuestionTemplate
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: QuestionTemplate
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveQuestionTemplateHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	question := &QuestionTemplate{}
	if err := c.GetContent(&question, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		question.DoctorId = doctor.ID
		question.QuestionType = 3
		c.ormDB.Set("gorm:save_associations", false).Save(&question)
		for i, answer := range question.Answers {
			c.ormDB.Set("gorm:save_associations", false).Save(&answer)
			question.Answers[i] = answer
		}
	}

	c.SendJSON(w, &question, http.StatusOK)
}

// GetPatientQuestionnaires swagger:route GET /questionnaires questionnaire getPatientQuestionnaires
//
// retrieves all patient questionaires
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPatientQuestionnairesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	questionnaires := &PatientQuestionnaires{}

	c.ormDB.Set("gorm:auto_preload", true).Find(&questionnaires)

	c.SendJSON(w, &questionnaires, http.StatusOK)
}

// GetPatientQuestionnaires swagger:route GET /questionnaires/unanswered questionnaire getPatientQuestionnaires
//
// retrieves all patient questionaires
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetUnansweredPatientQuestionnairesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	questionnaires := PatientQuestionnaires{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Where("id IN (SELECT pqq.patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id=qt.id WHERE pqq.answer_id = 0 AND qt.question_type = 3)").Find(&questionnaires)
	}
	for i, questionnaire := range questionnaires {
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_questionnaire_id = ?", questionnaire.ID).Where("answer_id = 0").Where("template_question_id IN (SELECT id FROM question_templates WHERE doctor_id > 0)").Find(&questionnaire.Questions)
		questionnaires[i] = questionnaire
	}
	c.SendJSON(w, &questionnaires, http.StatusOK)
}

// GetPatientQuestionnaires swagger:route GET /questionnaires/unanswered questionnaire getPatientQuestionnaires
//
// retrieves all patient questionaires
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPodiatristQuestionsFromPatientQuestionnairesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	questionnaires := PatientQuestionnaires{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		db := c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Where("id IN (SELECT pqq.patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id=qt.id WHERE  qt.question_type = 3)")
		if len(r.URL.Query()) > 0 {

			values := r.URL.Query()

			if values["questionnaire_date_from"] != nil && values["questionnaire_date_from"][0] != "" {
				db = db.Where("questionnaire_date > ?", values["questionnaire_date_from"][0])
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
							break
						}
					}
				}
			}
		}
		db.Find(&questionnaires) //pqq.answer_id = 0 AND
	}

	for i, questionnaire := range questionnaires {
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_questionnaire_id = ?", questionnaire.ID).Where("template_question_id IN (SELECT id FROM question_templates WHERE doctor_id > 0)").Find(&questionnaire.Questions) //.Where("answer_id = 0")
		questionnaires[i] = questionnaire
	}

	c.SendJSON(w, &questionnaires, http.StatusOK)
}

// GetPatientQuestionnaire swagger:route GET /questionnaires/{questionnaireId} questionnaire getPatientQuestionnaire
//
// retrieves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPatientQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	questionnaireId, _ := strconv.ParseInt(vars["questionnaireId"], 10, 64)

	questionnaire := &PatientQuestionnaire{}

	c.ormDB.Set("gorm:auto_preload", true).Find(&questionnaire, questionnaireId)

	c.SendJSON(w, &questionnaire, http.StatusOK)
}

// GetMeasurementQuestionnaire swagger:route GET /measurements/{measurementId}/questionnaire questionnaire getMeasurementQuestionnaire
//
// retrieves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	questionnaire := &PatientQuestionnaire{}
	//2029

	c.ormDB.Debug().Set("gorm:auto_preload", true).Where("measurement_id =?", measurementId).Find(&questionnaire)

	c.SendJSON(w, &questionnaire, http.StatusOK)
}

// GetMeasurementQuestionnaire swagger:route GET /measurements/{measurementId}/questionnaire questionnaire getMeasurementQuestionnaire
//
// retrieves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementSetupQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)
	measurement := Measurement{}
	c.ormDB.Unscoped().Set("gorm:auto_preload", false).First(&measurement, measurementId)

	questionnaire := &PatientQuestionnaire{}
	db := c.ormDB.Debug().Set("gorm:auto_preload", true).Select("patient_questionnaires.*, ABS(TIMESTAMPDIFF(SECOND, ?, questionnaire_date)) as SecondsBetweenDates", measurement.MeasurementDate)
	db = db.Where("questionnaire_date <= ?", measurement.MeasurementDate).Where("patient_id = ?", measurement.PatientId)
	db = db.Where("id IN (SELECT patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE qt.question_type = 1 AND pqq.answer_id > 0)")

	db.Order("SecondsBetweenDates ASC").First(&questionnaire)

	c.SendJSON(w, &questionnaire, http.StatusOK)
}

// GetPatientQuestionnaire swagger:route GET /questionnaires/{questionnaireId} questionnaire getPatientQuestionnaire
//
// retrieves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetLatestPatientQuestionnaireQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	if !c.isPatient(user) {
		return
	}
	patient := c.getPatient(user)
	questionnaireQuestions := PatientQuestionnaireQuestions{}

	//c.ormDB.Set("gorm:auto_preload", true).Where("patient_questionnaire_id IN (SELECT id IN patient_questionnaire WHERE patient_id = ?)", patient.ID).Group("template_question_id").Order().Find(&questionnaireQuestions)
	c.ormDB.Debug().Set("gorm:auto_preload", true).Raw("SELECT * FROM (SELECT * FROM patient_questionnaire_questions WHERE patient_questionnaire_id IN (SELECT id FROM patient_questionnaires WHERE patient_id = ?) ORDER BY question_date DESC LIMIT 18446744073709551615) AS a GROUP BY template_question_id ", patient.ID).Scan(&questionnaireQuestions)
	for i, question := range questionnaireQuestions {
		c.ormDB.Set("gorm:auto_preload", true).Find(&question, question.ID)
		questionnaireQuestions[i] = question
	}
	c.SendJSON(w, &questionnaireQuestions, http.StatusOK)
}

// SavePatientQuestionnaire swagger:route POST /questionnaire questionnaire savePatientQuestionnaire
//
// # Saves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SavePatientQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	patientQuestionnaire := &PatientQuestionnaire{}
	if err := c.GetContent(&patientQuestionnaire, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	patient := &Patient{}
	if c.isPatient(user) {
		patient = c.getPatient(user)
	} else if c.isPractice(user) {
		c.ormDB.Set("gorm:auto_preload", false).Find(&patient, patientQuestionnaire.Patient.ID)
	} else {
		return
	}

	patientQuestionnaireDB := &PatientQuestionnaire{}
	if patientQuestionnaire.ID == 0 /*&& patientQuestionnaire.MeasurementId > 0*/ {
		if patientQuestionnaire.MeasurementId > 0 {
			c.ormDB.Set("gorm:auto_preload", true).Where("questionnaire_date = ?", patientQuestionnaire.QuestionnaireDate).Where("patient_id = ?", patient.ID).Where("measurement_id = ?", patientQuestionnaire.MeasurementId).First(&patientQuestionnaireDB)
			if patientQuestionnaireDB.ID > 0 {
				for i, question := range patientQuestionnaireDB.Questions {
					for _, newQuestion := range patientQuestionnaire.Questions {
						if question.TemplateQuestion.ID == newQuestion.TemplateQuestion.ID { //Only copy answers of generated questionnaire
							question.Answer = newQuestion.Answer
							question.AnswerId = question.Answer.ID
						}
					}
					patientQuestionnaireDB.Questions[i] = question
				}
				if patientQuestionnaire.QuestionnaireDate.Valid {
					patientQuestionnaireDB.QuestionnaireDate = patientQuestionnaire.QuestionnaireDate
				}
				patientQuestionnaire = patientQuestionnaireDB
			}
		}
	} else {
		c.ormDB.Set("gorm:auto_preload", true).First(&patientQuestionnaireDB, patientQuestionnaire.ID)
		if patient.ID != patientQuestionnaireDB.Patient.ID {
			return
		}
		if patientQuestionnaireDB.ID > 0 {
			for i, question := range patientQuestionnaireDB.Questions {
				for _, newQuestion := range patientQuestionnaire.Questions {
					if question.TemplateQuestion.ID == newQuestion.TemplateQuestion.ID { //Only copy answers of generated questionnaire
						question.Answer = newQuestion.Answer
						question.AnswerId = question.Answer.ID
					}
				}
				if question.TemplateQuestion.QuestionType == 3 {
					c.ormDB.Model(&Notifications{}).Where("foreign_id = ?", question.ID).Where("notification_type = 3").Update("visible", false)
				}
				patientQuestionnaireDB.Questions[i] = question
			}
			if patientQuestionnaire.QuestionnaireDate.Valid {
				patientQuestionnaireDB.QuestionnaireDate = patientQuestionnaire.QuestionnaireDate
			}
			patientQuestionnaire = patientQuestionnaireDB
		}
	}

	var riskRating int64 = 0
	isInitialRiskCalculation := false
	c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaire)
	for i, question := range patientQuestionnaire.Questions {
		c.ormDB.Set("gorm:save_associations", false).Save(&question)
		patientQuestionnaire.Questions[i] = question

		answer := QuestionTemplateAnswer{}
		c.ormDB.Set("gorm:auto_preload", true).First(&question, question.ID)
		c.ormDB.Set("gorm:auto_preload", true).First(&answer, question.Answer.ID)
		if question.TemplateQuestion.QuestionType == 1 {
			isInitialRiskCalculation = true
			// calculate initial risk
		}
		if riskRating < answer.RiskRating {
			riskRating = answer.RiskRating
		}
	}

	riskCalculation := RiskCalculation{
		PatientId:                patient.ID,
		CalculationDate:          core.NullTime{Time: time.Now(), Valid: true},
		IsInitialRiskCalculation: isInitialRiskCalculation,
		RiskRatingId:             riskRating,
		PatientQuestionnaireId:   patientQuestionnaire.ID,
	}
	c.ormDB.Set("gorm:save_associations", false).Create(&riskCalculation)
	// UPDATE USER RISK
	// First geht initial risk (is this was not the initial risk calculation)
	if !isInitialRiskCalculation {
		initialRiskCalculation := RiskCalculation{}
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id=? AND is_initial_risk_calculation=1", patient.ID).Last(&initialRiskCalculation)
		if riskRating < initialRiskCalculation.RiskRatingId {
			riskCalculation = initialRiskCalculation
		}
	} else {
		c.ormDB.Model(&Patient{}).Where("id=?", patient.ID).Update("setup_complete", true)
	}

	c.ormDB.Model(&Patient{}).Where("id=?", patient.ID).Update("risk_rating", riskCalculation)

	c.SendJSON(w, &patientQuestionnaire, http.StatusOK)

	wsIds := []uint{}

	if patientQuestionnaire.MeasurementId > 0 {
		wsMeasurementShareds := []MeasurementShared{}
		db := c.ormDB.Debug().Where("measurement_id = ?", patientQuestionnaire.MeasurementId)
		if c.isPatient(user) {
			db.Find(&wsMeasurementShareds)
			doctorIds := []uint{}
			for _, item := range wsMeasurementShareds {
				if item.DoctorId > 0 {
					doctorIds = append(doctorIds, item.DoctorId)
				}
			}
			userDoctorIds := c.getMainUserIdsFromDoctorIds(doctorIds)
			userPracticeIds := c.getPracticeUserIdsFromDoctorIds(userDoctorIds)
			wsIds = append(wsIds, userDoctorIds...)
			wsIds = append(wsIds, userPracticeIds...)
		} else {
			//Practice
			practice := c.getPractice(user)

			c.ormDB.Debug().Where("doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?)", practice.ID).Where("measurement_id = ?", patientQuestionnaire.MeasurementId).Find(&wsMeasurementShareds)

			wsIds = append(wsIds, user.ID)
			doctorIds := []uint{}
			for _, item := range wsMeasurementShareds {
				if item.DoctorId > 0 {
					doctorIds = append(doctorIds, item.DoctorId)
				}
			}
			userDoctorIds := c.getMainUserIdsFromDoctorIds(doctorIds)
			wsIds = append(wsIds, userDoctorIds...)
		}

		if len(wsIds) > 0 {
			go web3socket.SendWebsocketDataInfoMessage("Update questionnaire of measurement", web3socket.Websocket_Update, web3socket.Websocket_Measurements, uint(patientQuestionnaire.MeasurementId), wsIds, nil)
		}
	}

}

// SaveMeasurementQuestionnaire swagger:route POST /measurements/{measurementId}/questionnaire questionnaire saveMeasurementQuestionnaire
//
// # Saves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveMeasurementQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	//TODO ADD WS if Home/Remote version
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	patientQuestionnaire := &PatientQuestionnaire{}
	if err := c.GetContent(&patientQuestionnaire, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	patientQuestionnaireDB := &PatientQuestionnaire{}

	c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id =?", measurementId).Find(&patientQuestionnaireDB)
	var riskRating int64 = 0

	if patientQuestionnaire.ID != patientQuestionnaireDB.ID {
		for i, question := range patientQuestionnaireDB.Questions {
			for _, newQuestion := range patientQuestionnaire.Questions {
				if question.TemplateQuestion.ID == newQuestion.TemplateQuestion.ID { //Only copy answers of generated questionnaire
					question.Answer = newQuestion.Answer
					question.AnswerId = question.Answer.ID
				}
			}
			patientQuestionnaireDB.Questions[i] = question
		}
		patientQuestionnaire = patientQuestionnaireDB
	} else {
		if patientQuestionnaireDB.ID > 0 {
			for _, question := range patientQuestionnaireDB.Questions {
				for _, newQuestion := range patientQuestionnaire.Questions {
					if question.TemplateQuestion.ID == newQuestion.TemplateQuestion.ID { //Only copy answers of generated questionnaire
						if question.TemplateQuestion.QuestionType == 3 {
							c.ormDB.Model(&Notification{}).Where("foreign_id = ?", question.ID).Where("notification_type = 3").Update("visible", false)
						}
					}
				}
			}
		}
	}

	if !c.isPatient(user) {
		// only patients can save their questions
		err := errors.New("only patients can save answers")
		c.HandleError(err, w)
		return

	}

	patient := c.getPatient(user)
	if patient.ID != patientQuestionnaire.Patient.ID {
		err := errors.New("patients can only save their own answers")
		c.HandleError(err, w)
		return
	}

	patientQuestionnaire.MeasurementId = uint(measurementId)
	c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaire)
	for i, question := range patientQuestionnaire.Questions {
		c.ormDB.Set("gorm:save_associations", false).Save(&question)
		patientQuestionnaire.Questions[i] = question

		answer := QuestionTemplateAnswer{}
		c.ormDB.Set("gorm:auto_preload", true).First(&question, question.ID)
		c.ormDB.Set("gorm:auto_preload", true).First(&answer, question.Answer.ID)

		if riskRating < answer.RiskRating {
			riskRating = answer.RiskRating
		}
	}

	riskCalculation := RiskCalculation{
		PatientId:                patient.ID,
		CalculationDate:          core.NullTime{Time: time.Now(), Valid: true},
		IsInitialRiskCalculation: false,
		RiskRatingId:             riskRating,
		PatientQuestionnaireId:   patientQuestionnaire.ID,
	}
	c.ormDB.Set("gorm:save_associations", false).Create(&riskCalculation)

	initialRiskCalculation := RiskCalculation{}
	c.ormDB.Set("gorm:auto_preload", true).Where("patient_id=? AND is_initial_risk_calculation=1", patient.ID).First(&initialRiskCalculation)
	if riskRating < initialRiskCalculation.RiskRatingId {
		riskCalculation = initialRiskCalculation
	}

	c.ormDB.Model(&Patient{}).Update("risk_rating", riskCalculation).Where("id=?", patient.ID)

	c.SendJSON(w, &patientQuestionnaire, http.StatusOK)

	measurementsShareds := []MeasurementShared{}
	c.ormDB.Where("measurement_id = ?", measurementId).Find(&measurementsShareds)

	wsIds := []uint{}
	doctorIds := []uint{}
	if measurementsShareds != nil {
		for _, item := range measurementsShareds {
			doctorIds = append(doctorIds, item.DoctorId)
		}
	}

	userIdsOfDoctor := c.getMainUserIdsFromDoctorIds(doctorIds)
	userIdsOfPractice := c.getPracticeUserIdsFromDoctorIds(doctorIds)
	wsIds = append(wsIds, userIdsOfDoctor...)
	wsIds = append(wsIds, userIdsOfPractice...)

	go web3socket.SendWebsocketDataInfoMessage("Save Questionnaire of measurement", web3socket.Websocket_Add, web3socket.Websocket_Measurements, uint(measurementId), wsIds, nil)
}

// SavePatientQuestionnaireQuestion swagger:route POST /questionnaire/{questionnaireId}/question questionnaire savePatientQuestionnaireQuestion
//
// # Saves a patient questionnaire question
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaireQuestion
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SavePatientQuestionnaireQuestionHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	questionnaireId, _ := strconv.ParseInt(vars["questionnaireId"], 10, 64)

	patientQuestionnaireQuestion := &PatientQuestionnaireQuestion{}
	if err := c.GetContent(&patientQuestionnaireQuestion, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	patientQuestionnaireQuestion.QuestionDate = core.NullTime{Time: time.Now(), Valid: true}
	if c.isPatient(user) {
		/*patient := c.getPatient(user)

		if(patient.ID != patientQuestionnaireQuestion.Patient.ID){
			return
		}*/
		patientQuestionnaireQuestion.PatientQuestionnaireId = uint(questionnaireId)
		c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaireQuestion)

	}
	if c.isDoctor(user) {
		//newQuestion := false
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Find(&patientQuestionnaireQuestion.TemplateQuestion.Patient, patientQuestionnaireQuestion.TemplateQuestion.Patient.ID)
		patientQuestionnaireQuestion.PatientQuestionnaireId = uint(questionnaireId)
		if c.ormDB.NewRecord(&patientQuestionnaireQuestion.TemplateQuestion) {
			//	newQuestion = true
			patientQuestionnaireQuestion.TemplateQuestion.QuestionType = 3
			patientQuestionnaireQuestion.TemplateQuestion.DoctorId = doctor.ID
			c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaireQuestion.TemplateQuestion)
			for i, answer := range patientQuestionnaireQuestion.TemplateQuestion.Answers {
				c.ormDB.Set("gorm:save_associations", false).Save(&answer)
				patientQuestionnaireQuestion.TemplateQuestion.Answers[i] = answer
			}
		} else {
			patientQuestionnaireQuestion.TemplateQuestion.QuestionType = 3
			patientQuestionnaireQuestion.TemplateQuestion.DoctorId = doctor.ID
			c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaireQuestion.TemplateQuestion)
			for i, answer := range patientQuestionnaireQuestion.TemplateQuestion.Answers {
				if c.ormDB.NewRecord(&patientQuestionnaireQuestion.TemplateQuestion.Answers[i]) {
					c.ormDB.Set("gorm:save_associations", false).Save(&answer)
					patientQuestionnaireQuestion.TemplateQuestion.Answers[i] = answer
				}
			}

			if patientQuestionnaireQuestion.TemplateQuestion.DateFrom.Time.Unix() <= time.Now().Unix() && patientQuestionnaireQuestion.PatientQuestionnaireId > 0 {
				c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaireQuestion)
			}
		}
		/*if patientQuestionnaireQuestion.TemplateQuestion.DateFrom.Time.Unix() <= time.Now().Unix() {
			if patientQuestionnaireQuestion.PatientQuestionnaireId == 0 {
				currentQuestionnaire := PatientQuestionnaire{}
				db := c.ormDB.Set("gorm:auto_preload", false).Where("patient_id = ?", patientQuestionnaireQuestion.TemplateQuestion.PatientId).Where("DATE(questionnaire_date) = DATE(?)", time.Now())
				//db = db.Where("id IN (SELECT pqq.patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE qt.recurring_rule = ?)", patientQuestionnaireQuestion.TemplateQuestion.RecurringRule)
				db = db.Where("measurement_id > 0")
				db.Order("questionnaire_date DESC").First(&currentQuestionnaire)
				patientQuestionnaireQuestion.PatientQuestionnaireId = currentQuestionnaire.ID
			}

			if patientQuestionnaireQuestion.PatientQuestionnaireId > 0 {
				c.ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaireQuestion)

				if newQuestion && patientQuestionnaireQuestion.TemplateQuestion.PatientId > 0 && patientQuestionnaireQuestion.TemplateQuestion.DoctorId > 0 {
					doctorName := doctor.FirstName + " " + doctor.LastName
					if doctorName == "" {
						doctorName = doctor.Users[0].User.Username
					}
					c.CreateNotification(patientQuestionnaireQuestion.TemplateQuestion.Patient.User.ID, 3, fmt.Sprintf("New Question from %s", doctorName), "You have a new Question", patientQuestionnaireQuestion.ID, fmt.Sprintf("/questionnaires/%d", patientQuestionnaireQuestion.PatientQuestionnaireId), nil)
				}
			}
		}*/

	}

	c.SendJSON(w, &patientQuestionnaireQuestion, http.StatusOK)
}

// GetPatientQuestionAnswers swagger:route GET /questions/{questionId}/patient/{patientId} questios getPatientQuestionAnswers
//
// retrieves all answers to a QuestionTemplate by a patient
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []PatientQuestionnaireQuestion
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetPatientQuestionnaireQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	questionId, _ := strconv.ParseInt(vars["questionTemplateId"], 10, 64)
	patientId, _ := strconv.ParseInt(vars["patientId"], 10, 64)

	questions := PatientQuestionnaireQuestions{}

	localDB := c.ormDB.Set("gorm:auto_preload", true)
	questionnaireType := ""
	doctorID := ""
	if len(r.URL.Query()) > 0 {
		values := r.URL.Query()
		if val, ok := values["date_from"]; ok && len(val) > 0 {
			if val[0] != "" {
				dateFrom := &core.NullTime{}
				dateFrom.FromString(val[0])
				if questionId > 0 {
					localDB = localDB.Where("DATE(questionnaire_date) >= ?", dateFrom)
				} else {
					localDB = localDB.Where("DATE(measurement_date) >= ?", dateFrom)
				}
			}
		}
		if val, ok := values["date_to"]; ok && len(val) > 0 {
			if val[0] != "" {
				dateTo := &core.NullTime{}
				dateTo.FromString(val[0])
				if questionId > 0 {
					localDB = localDB.Where("DATE(questionnaire_date) <= ?", dateTo)
				} else {
					localDB = localDB.Where("DATE(measurement_date) <= ?", dateTo)
				}
			}
		}
		if val, ok := values["type"]; ok && len(val) > 0 {
			if val[0] != "" {
				questionnaireType = val[0]
			}
		}
		if val, ok := values["doctor_id"]; ok && len(val) > 0 {
			if val[0] != "" {
				doctorID = val[0]
			}
		}
	}

	if questionId > 0 {
		localDB.Where("template_question_id = ?", questionId).Where("patient_questionnaire_id IN (SELECT id FROM patient_questionnaires WHERE patient_id  = ?)", patientId).Order("id").Find(&questions)
	} else {
		switch questionnaireType {
		case "risk":
			riskCaluations := RiskCalculations{}
			localDB.Debug().Unscoped().Where("patient_id = ?", patientId).Where("risk_rating_id > 0").Order("calculation_date ASC").Find(&riskCaluations)
			c.SendJSON(w, &riskCaluations, http.StatusOK)
			return
		case "scan":
		case "measurement":
		case "doctor_risk":
			isPractice := c.isPractice(user)
			currentDoctorID := ""

			if isPractice && len(doctorID) > 0 {
				currentDoctorID = doctorID
			} else {
				doctor := c.getDoctor(user)
				currentDoctorID = fmt.Sprintf("%v", doctor.ID)
			}

			measurementDoctorRisks := MeasurementDoctorRisks{}

			db := c.ormDB.Debug().Set("gorm:auto_preload", true).Where("measurement_doctor_risks.doctor_id = ?", currentDoctorID)
			db = db.Joins("LEFT JOIN measurements ON measurements.id = measurement_doctor_risks.measurement_id")
			db = db.Where("measurements.patient_id = ?", patientId)

			db.Debug().Order("measurements.measurement_date DESC").Find(&measurementDoctorRisks)

			for i, risk := range measurementDoctorRisks {
				measurement := Measurement{}
				c.ormDB.Set("gorm:auto_preload", false).First(&measurement, risk.MeasurementId)
				risk.MeasurementDate = measurement.MeasurementDate
				measurementDoctorRisks[i] = risk
			}

			c.SendJSON(w, &measurementDoctorRisks, http.StatusOK)
			return
		default:

			measurements := Measurements{}
			localDB.Unscoped().Where("patient_id = ?", patientId).Order("measurement_date").Find(&measurements)

			for _, scan := range measurements {

				var scanRating uint = 3
				answer := "green"
				if scan.ColdspotDetected != "NONE" && scan.HotspotDetected != "NONE" {
					scanRating = 1
					answer = "redblue"
				} else if scan.ColdspotDetected != "NONE" {
					scanRating = 2
					answer = "blue"
				} else if scan.HotspotDetected != "NONE" {
					scanRating = 4
					answer = "red"
				}

				question := PatientQuestionnaireQuestion{
					QuestionDate: scan.MeasurementDate,
					Answer:       QuestionTemplateAnswer{AnswerValue: scanRating, AnswerText: answer},
				}
				question.DeletedAtPublic = core.NullTime{}
				if scan.DeletedAt != nil && scan.DeletedAt.Unix() > 0 {
					question.DeletedAtPublic = core.NullTime{Time: *scan.DeletedAt, Valid: true}
				}

				questions = append(questions, question)
			}
		}

	}

	c.SendJSON(w, &questions, http.StatusOK)
}

// GetPatientQuestionAnswers swagger:route GET /questions/{questionId}/patient/{patientId} questios getPatientQuestionAnswers
//
// retrieves all answers to a QuestionTemplate by a patient
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []PatientQuestionnaireQuestion
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyQuestionnaireQuestionsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	questionId, _ := strconv.ParseInt(vars["questionTemplateId"], 10, 64)

	patient := c.getPatient(user)
	patientId := patient.ID

	questions := PatientQuestionnaireQuestions{}

	if questionId > 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("template_question_id = ?", questionId).Where("patient_questionnaire_id IN (SELECT id FROM patient_questionnaires WHERE patient_id  = ?)", patientId).Order("id").Find(&questions)
	} else {
		measurements := Measurements{}
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patientId).Order("measurement_date").Find(&measurements)

		for _, scan := range measurements {

			var scanRating uint = 3
			answer := "green"
			if scan.ColdspotDetected != "NONE" && scan.HotspotDetected != "NONE" {
				scanRating = 1
				answer = "redblue"
			} else if scan.ColdspotDetected != "NONE" {
				scanRating = 2
				answer = "blue"
			} else if scan.HotspotDetected != "NONE" {
				scanRating = 4
				answer = "red"
			}

			question := PatientQuestionnaireQuestion{
				QuestionDate: scan.MeasurementDate,
				Answer:       QuestionTemplateAnswer{AnswerValue: scanRating, AnswerText: answer},
			}

			questions = append(questions, question)
		}

	}

	c.SendJSON(w, &questions, http.StatusOK)
}

// CreatePatientQuestionnaire swagger:route POST /questionnaire/patient/{patientId} questionnaire savePatientQuestionnaire
//
// # Saves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) CreatePatientQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)

	patientId, _ := strconv.ParseInt(vars["patientId"], 10, 64)

	patientQuestionnaire, _ := CreatePatientQuestionnaire(*c.ormDB, uint(patientId), []int64{2, 3}, []string{"DAILY-SCAN", "WEEKLY-SCAN", "MONTHLY-SCAN"}, 0)

	c.SendJSON(w, &patientQuestionnaire, http.StatusOK)
}

// CreatePatientQuestionnaire swagger:route POST /questionnaire/patient/{patientId} questionnaire savePatientQuestionnaire
//
// # Saves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) CreateDailyPatientQuestionnairesHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}*/

	patients := &Patients{}
	patientQuestionnaires := PatientQuestionnaires{}
	c.ormDB.Find(&patients)
	for _, patient := range *patients {
		patientQuestionnaire, _ := CreatePatientQuestionnaire(*c.ormDB, uint(patient.ID), []int64{2, 3}, []string{"DAILY", "WEEKLY", "MONTHLY"}, 0)
		patientQuestionnaires = append(patientQuestionnaires, *patientQuestionnaire)
	}

	c.SendJSON(w, &patientQuestionnaires, http.StatusOK)
}

// CreatePatientQuestionnaire swagger:route POST /questionnaire/patient/{patientId} questionnaire savePatientQuestionnaire
//
// # Saves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) CreateDailyPatientQuestionnaireTestHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}*/

	patient := &Patient{}
	patientQuestionnaires := PatientQuestionnaires{}
	c.ormDB.Find(&patient, 268)
	patientQuestionnaire, _ := CreatePatientQuestionnaire(*c.ormDB, uint(patient.ID), []int64{2, 3}, []string{"DAILY", "WEEKLY", "MONTHLY"}, 0)
	patientQuestionnaires = append(patientQuestionnaires, *patientQuestionnaire)

	c.SendJSON(w, &patientQuestionnaires, http.StatusOK)
}

// GetSetupPatientQuestionnaire swagger:route POST /me/questionnaire questionnaire getSetupPatientQuestionnaire
//
// # Saves a patient questionnaire
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: PatientQuestionnaire
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetSetupPatientQuestionnaireHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	createQuestionnaire := false

	if c.isPatient(user) {
		patient := c.getPatient(user)
		patientQuestionnaire := &PatientQuestionnaire{}
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id=? AND measurement_id=0", patient.ID).Order("questionnaire_date DESC").First(&patientQuestionnaire)

		if len(r.URL.Query()) > 0 {

			values := r.URL.Query()
			if val, ok := values["create_questionnaire"]; ok && len(val) > 0 {
				createQuestionnaire, _ = strconv.ParseBool(val[0])
			}
		}
		newPatientQuestionnaire, _ := CreatePatientQuestionnaireHelper(*c.ormDB, patient.ID, []int64{1}, []string{"SETUP"}, 0, createQuestionnaire)
		if patientQuestionnaire.ID == 0 {
			for _, question := range patientQuestionnaire.Questions {
				for j, newQuestion := range newPatientQuestionnaire.Questions {
					if question.TemplateQuestion.ID == newQuestion.TemplateQuestion.ID {
						newQuestion.Answer = question.Answer
					}

					newPatientQuestionnaire.Questions[j] = newQuestion
				}
			}
		}

		c.SendJSON(w, &newPatientQuestionnaire, http.StatusOK)
	}

}
func CreatePatientQuestionnaire(ormDB gorm.DB, patientId uint, questionTypes []int64, recurringRules []string, measurementId int64) (*PatientQuestionnaire, error) {
	return CreatePatientQuestionnaireHelper(ormDB, patientId, questionTypes, recurringRules, measurementId, true)
}
func CreatePatientQuestionnaireHelper(ormDB gorm.DB, patientId uint, questionTypes []int64, recurringRules []string, measurementId int64, createQuestionnaire bool) (*PatientQuestionnaire, error) {

	forceCreate := false
	_ = forceCreate

	patientQuestionnaire := &PatientQuestionnaire{}

	now := core.NullTime{Time: time.Now(), Valid: true}
	patientQuestionnaire.QuestionnaireDate = now
	ormDB.Set("gorm:auto_preload", true).Find(&patientQuestionnaire.Patient, patientId)
	patientQuestionnaire.PatientId = patientQuestionnaire.Patient.ID
	if uint(patientId) != patientQuestionnaire.Patient.ID {
		return nil, errors.New("No valid Patient")
	}

	patientQuestionnaire.MeasurementId = uint(measurementId)

	templateQuestions := &QuestionTemplates{}

	ormDB.Debug().Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id IN (SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id = ?)", patientQuestionnaire.Patient.ID).Where("patient_id = 0 OR patient_id = ?", patientQuestionnaire.Patient.ID).Where("question_type IN (?)", questionTypes).Where("recurring_rule IN (?)", recurringRules).Where("NOT (date_from > ? OR (DATE(date_to) + INTERVAL 1 DAY) < ?) OR date_to IS NULL", now, now).Order("id").Find(&templateQuestions)
	//ormDB.Set("gorm:auto_preload", true).Where("doctor_id = 0 OR doctor_id IN (SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id = ?)", patientQuestionnaire.Patient.ID).Where("patient_id = 0 OR patient_id = ?", patientQuestionnaire.Patient.ID).Where("question_type IN (?)", questionTypes).Where("recurring_rule IN (?)", recurringRules).Where("(date_from IS NULL AND date_to IS NULL) OR (DATE(date_from) <= DATE(NOW()) AND date_to IS NULL) OR (DATE(date_from) <= DATE(NOW()) AND DATE(date_to) >= DATE(NOW()))").Order("id").Find(&templateQuestions)
	if createQuestionnaire {
		ormDB.Set("gorm:save_associations", false).Save(&patientQuestionnaire)
	}

	loadSetupAnswers := false
	for _, templateQuestion := range *templateQuestions {

		addQuestion := true
		oldQuestion := PatientQuestionnaireQuestion{}
		log.Println(templateQuestion.ID)
		log.Println(templateQuestion.RecurringRule)
		switch templateQuestion.RecurringRule {
		case "SETUP":
			loadSetupAnswers = true
			break
		case "DAILY":
			if templateQuestion.DoctorId > 0 {
				ormDB.Where("DATE(question_date) > (DATE(?) - INTERVAL 1 DAY)", now).Where("template_question_id = ?", templateQuestion.ID).Where("? IN (SELECT patient_id FROM patient_questionnaires WHERE id IN(SELECT patient_questionnaire_id FROM patient_questionnaire_questions WHERE template_question_id = ?))", patientId, templateQuestion.ID).First(&oldQuestion)
				if oldQuestion.ID > 0 {
					addQuestion = false
				}
			}
			break
		case "DAILY-SCAN":
			if templateQuestion.DoctorId > 0 {
				ormDB.Where("DATE(question_date) > (DATE(?) - INTERVAL 1 DAY)", now).Where("template_question_id = ?", templateQuestion.ID).Where("? IN (SELECT patient_id FROM patient_questionnaires WHERE id IN(SELECT patient_questionnaire_id FROM patient_questionnaire_questions WHERE template_question_id = ?))", patientId, templateQuestion.ID).First(&oldQuestion)
				if oldQuestion.ID > 0 {
					addQuestion = false
				}
			}
			break
		case "WEEKLY":
			ormDB.Where("DATE(question_date) > (DATE(?) - INTERVAL 7 DAY)", now).Where("template_question_id = ?", templateQuestion.ID).Where("? IN (SELECT patient_id FROM patient_questionnaires WHERE id IN(SELECT patient_questionnaire_id FROM patient_questionnaire_questions WHERE template_question_id = ?))", patientId, templateQuestion.ID).First(&oldQuestion)
			if oldQuestion.ID > 0 {
				addQuestion = false
			}
			break
		case "WEEKLY-SCAN":
			ormDB.Where("DATE(question_date) > (DATE(?) - INTERVAL 7 DAY)", now).Where("template_question_id = ?", templateQuestion.ID).Where("? IN (SELECT patient_id FROM patient_questionnaires WHERE id IN(SELECT patient_questionnaire_id FROM patient_questionnaire_questions WHERE template_question_id = ?))", patientId, templateQuestion.ID).First(&oldQuestion)
			if oldQuestion.ID > 0 {
				addQuestion = false
			}
			break
		case "MONTHLY":
			ormDB.Where("DATE(question_date) > (DATE(?) - INTERVAL 1 MONTH)", now).Where("template_question_id = ?", templateQuestion.ID).Where("? IN (SELECT patient_id FROM patient_questionnaires WHERE id IN(SELECT patient_questionnaire_id FROM patient_questionnaire_questions WHERE template_question_id = ?))", patientId, templateQuestion.ID).First(&oldQuestion)
			if oldQuestion.ID > 0 {
				addQuestion = false
			}
			break
		case "MONTHLY-SCAN":
			ormDB.Where("DATE(question_date) > (DATE(?) - INTERVAL 1 MONTH)", now).Where("template_question_id = ?", templateQuestion.ID).Where("? IN (SELECT patient_id FROM patient_questionnaires WHERE id IN(SELECT patient_questionnaire_id FROM patient_questionnaire_questions WHERE template_question_id = ?))", patientId, templateQuestion.ID).First(&oldQuestion)
			if oldQuestion.ID > 0 {
				addQuestion = false
			}
			break
		}
		log.Println(addQuestion)
		if addQuestion {
			question := PatientQuestionnaireQuestion{}
			question.TemplateQuestion = templateQuestion
			question.TemplateQuestionId = templateQuestion.ID
			question.PatientQuestionnaireId = patientQuestionnaire.ID
			question.QuestionDate = now
			log.Println(question.QuestionDate)
			if createQuestionnaire {
				ormDB.Set("gorm:save_associations", false).Save(&question)
			}
			log.Println(question.QuestionDate)
			log.Println(question.CreatedAt)
			patientQuestionnaire.Questions = append(patientQuestionnaire.Questions, question)
		}
	}

	//SM X
	if loadSetupAnswers {
		lastQuestionnaire := &PatientQuestionnaire{}
		db := ormDB.Debug().Set("gorm:auto_preload", true).Select("patient_questionnaires.*, ABS(TIMESTAMPDIFF(SECOND, ?, questionnaire_date)) as SecondsBetweenDates", patientQuestionnaire.QuestionnaireDate)
		db = db.Where("questionnaire_date <= ?", patientQuestionnaire.QuestionnaireDate).Where("patient_id = ?", patientQuestionnaire.Patient.ID)
		db = db.Debug().Where("id IN (SELECT patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE qt.question_type = 1 AND pqq.answer_id > 0)")
		db.Order("SecondsBetweenDates ASC").First(&lastQuestionnaire)
		for i, question := range patientQuestionnaire.Questions {
			for _, lastQuestion := range lastQuestionnaire.Questions {
				if question.TemplateQuestion.ID == lastQuestion.TemplateQuestion.ID {
					question.Answer = lastQuestion.Answer
					break
				}
			}
			patientQuestionnaire.Questions[i] = question
		}
	} else {
		// Im Auge behalten.
		for i, question := range patientQuestionnaire.Questions {
			ormDB.Debug().Where("id IN (SELECT answer_id FROM patient_questionnaire_questions WHERE template_question_id = ? AND question_date <= ? AND patient_questionnaire_id IN (SELECT id FROM patient_questionnaires WHERE patient_id = ?))", question.TemplateQuestion.ID, question.QuestionDate, patientQuestionnaire.Patient.ID).Find(&question.Answer)
			patientQuestionnaire.Questions[i] = question
		}
	}

	return patientQuestionnaire, nil
}

// GetMyProfile swagger:route GET /me/user system getMyProfile
//
// retrieves your Profile, either with Patient or Doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []HelperUser
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyProfileHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	helperUser := HelperUser{}

	helperUser.User = *user

	c.ormDB.Set("gorm:auto_preload", true).Find(&helperUser.User, user.ID)
	if c.isPatient(user) {
		helperUser.Patient = c.getPatient(user)

		reward, countScans, countConsecutiveScans := CalculateRewardForPatient(c.ormDB, helperUser.Patient.ID, false)
		helperUser.Patient.CurrentReward = *reward
		helperUser.Patient.TotalScans = int64(countScans)
		helperUser.Patient.ConsecutiveScans = int64(countConsecutiveScans)
		c.ormDB.DB().QueryRow("SELECT date_time_from  FROM appointments a LEFT JOIN appointment_statuses aps ON a.appointment_status_id = aps.id  WHERE aps.status_def_id = 3 AND a.patient_id =? AND a.date_time_from < NOW() ORDER BY date_time_from DESC LIMIT 1", helperUser.Patient.ID).Scan(&helperUser.Patient.LastAppointmentDate)
		c.ormDB.DB().QueryRow("SELECT measurement_date FROM measurements m WHERE m.patient_id =? ORDER BY measurement_date DESC LIMIT 1", helperUser.Patient.ID).Scan(&helperUser.Patient.LastMeasurementDate)
		//&helperUser.Patient.User = nil
	} else if c.isDoctor(user) {
		helperUser.Doctor = c.getDoctor(user)

		for i, device := range helperUser.Doctor.Devices {
			c.ormDB.Debug().Where("device_id = ?", device.Device.ID).Where("(SELECT measurements.id FROM measurements WHERE measurements.doctor_id = ?) = device_statuses.measurement_id", helperUser.Doctor.ID).Order("status_date DESC").Last(&device.Device.LastDeviceStatus)
			c.ormDB.Debug().Where("device_id = ?", device.Device.ID).Where("measurements.doctor_id = ?", helperUser.Doctor.ID).Order("measurement_date DESC").Last(&device.Device.LastMeasurement)
			helperUser.Doctor.Devices[i] = device
		}
	} else if c.isPractice(user) {
		helperUser.Practice = c.getPractice(user)

		for i, device := range helperUser.Practice.Devices {
			c.ormDB.Where("device_id = ?", device.Device.ID).Order("status_date DESC").Last(&device.Device.LastDeviceStatus)
			c.ormDB.Where("device_id = ?", device.Device.ID).Order("measurement_date DESC").Last(&device.Device.LastMeasurement)
			helperUser.Practice.Devices[i] = device
		}
	}

	helperUser.PasswordX = ""
	helperUser.PasswordRepeat = ""
	c.SendJSON(w, &helperUser, http.StatusOK)
}

func (c *PodiumController) SaveAccountSettingHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	userId, _ := strconv.ParseInt(vars["userId"], 10, 64)

	if userId == 0 {
		return
	}

	setting := core.SystemAccountSetting{}
	if err := c.GetContent(&setting, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.ormDB.NewRecord(&setting) {
		c.ormDB.Set("gorm:save_associations", false).Create(&setting)
		c.ormDB.Set("gorm:save_associations", false).Model(&core.User{}).Where("id = ?", userId).Update("setting_id", setting.ID)
	} else {
		c.ormDB.Set("gorm:save_associations", false).Save(&setting)
	}

	c.SendJSON(w, &setting, http.StatusOK)
}

// SaveMyProfile swagger:route GET /me/user system saveMyProfile
//
// retrieves your Profile, either with Patient or Doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []HelperUser
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveMyProfileHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	helperUser := HelperUser{}

	if err := c.GetContent(&helperUser, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if helperUser.ID != user.ID {
		err := errors.New("You can only change your own account")
		c.HandleError(err, w)
		return
	}

	//c.ormDB.Set("gorm:save_associations", false).Save(&helperUser)
	if helperUser.PasswordX != "" {
		err := core.ValidatePassword(helperUser.PasswordX)
		if err != nil {
			c.HandleError(err, w)
			return
		}
		helperUser.PasswordX = core.GetMD5Hash(helperUser.PasswordX)
		helperUser.Password = helperUser.PasswordX
		//c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", helperUser.PasswordX, helperUser.ID)
		user.IsPasswordExpired = false
	} else {
		helperUser.Password = user.Password
	}
	helperUser.Username = user.Username
	helperUser.UserType = user.UserType
	helperUser.IsPasswordExpired = user.IsPasswordExpired
	helperUser.RegisteredAt = user.RegisteredAt

	c.ormDB.Set("gorm:save_associations", false).Save(&helperUser)

	if c.isPatient(user) && helperUser.Patient != nil {
		patient := c.getPatient(user)
		helperUser.Patient.ID = patient.ID
		helperUser.Patient.User = *user
		helperUser.Patient.UserId = user.ID
		log.Println(helperUser.Patient.ID)
		c.ormDB.Set("gorm:save_associations", false).Save(&helperUser.Patient)
		c.ormDB.Set("gorm:save_associations", false).Save(&helperUser.Patient)
	} else if c.isDoctor(user) && helperUser.Doctor != nil {
		doctor := c.getDoctor(user)
		helperUser.Doctor.ID = doctor.ID
		if helperUser.Doctor.Postcode != "" {
			helperUser.Doctor.Latitude, helperUser.Doctor.Longitude = core.GetOSMLatLon("", helperUser.Doctor.Postcode)
		}
		c.ormDB.Set("gorm:save_associations", false).Save(&helperUser.Doctor)

		(*c.Controller.Users)[user.Token] = helperUser.User
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		helperUser.Practice.ID = practice.ID
		helperUser.Practice.User = *user
		helperUser.Practice.UserId = user.ID
		//helperUser.Practice.Name = practice.Name
		c.ormDB.Set("gorm:save_associations", false).Save(&helperUser.Practice)
		for i, device := range helperUser.Practice.Devices {
			c.ormDB.Set("gorm:save_associations", false).Save(&device)
			helperUser.Practice.Devices[i] = device
		}
	}

	helperUser.PasswordX = ""
	helperUser.PasswordRepeat = ""

	c.SendJSON(w, &helperUser, http.StatusOK)
}

// saveAccount swagger:route POST /system/accounts system saveAccount
//
// Save an Account
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
//			data: User
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
/*func (c *PodiumController) SaveAccountHandler(w http.ResponseWriter, r *http.Request) {
	var newUser core.User

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isDoctor(user) && !c.isSysadmin(user) {
		return
	}

	if err := c.GetContent(&newUser, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.ormDB.NewRecord(&newUser) {
		c.ormDB.Set("gorm:save_associations", false).Create(&newUser)
		newUser.PasswordX = core.GetMD5Hash(newUser.PasswordX)
		c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", newUser.PasswordX, newUser.ID)

		if c.isDoctor(user) {
			doctorUser := &DoctorUser{}
			doctorUser.DoctorId = c.getDoctor(user).ID
			doctorUser.UserId = newUser.ID
			c.ormDB.Set("gorm:save_associations", false).Create(&doctorUser)
		}

	} else {
		if newUser.PasswordX != "" {
			newUser.PasswordX = core.GetMD5Hash(newUser.PasswordX)
			c.ormDB.Exec("UPDATE system_accounts SET password = ? WHERE id = ?", newUser.PasswordX, newUser.ID)

		}
		c.ormDB.Set("gorm:save_associations", false).Save(&newUser)
	}

	c.SendJSON(w, &newUser, http.StatusOK)
}*/

/*
// getMyDevices swagger:route GET /me/devices device getMyDevices
//
// retrieves your devices
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
//			data: []Device
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"

	func (c *PodiumController) GetMyDevicesHandler(w http.ResponseWriter, r *http.Request) {
		ok := false
		var user *core.User
		if ok, user = c.Controller.GetUser(w, r); !ok {
			_ = user
			return
		}

		devices := Devices{}

		c.ormDB.Set("gorm:auto_preload", true).Where("user_id = ?", user.ID).Find(&devices)

		c.SendJSON(w, &devices, http.StatusOK)
	}
*/
func (c *PodiumController) GetDeviceSystemVersionLastPublishHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/

	systemVersion := &DeviceSystemVersion{}

	c.ormDB.Set("gorm:auto_preload", true).Debug().Where("is_publish = true").Order("LENGTH(system_version) DESC, system_version DESC").First(&systemVersion)

	c.SendJSON(w, &systemVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SaveDeviceSystemVersionHandler(w http.ResponseWriter, r *http.Request) {
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

	systemVersion := DeviceSystemVersion{}
	if err := c.GetContent(&systemVersion, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	systemVersion.SystemVersion = strings.TrimSpace(systemVersion.SystemVersion)

	if systemVersion.SystemVersion == "" {
		err := errors.New("System version must not be empty")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	if c.ormDB.NewRecord(&systemVersion) {
		checkSystemVersion := DeviceSystemVersion{}
		c.ormDB.Set("gorm:auto_preload", true).Where("system_version = ?", systemVersion.SystemVersion).Last(&checkSystemVersion)

		if checkSystemVersion.ID > 0 {
			err := errors.New("System version already exists.")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}

		if systemVersion.DeviceSystemVersionTypes == nil || (systemVersion.DeviceSystemVersionTypes != nil && len(systemVersion.DeviceSystemVersionTypes) == 0) {
			err := errors.New("No update added.")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}

		deviceTypes := []uint{DeviceType_APK_PRO, DeviceType_Podium, DeviceType_PORTAL}
		for _, item := range systemVersion.DeviceSystemVersionTypes {
			index := c.Find(deviceTypes, item.DeviceTypeVersion.DeviceType.ID)
			if index > -1 && len(deviceTypes) > 0 {
				deviceTypes = append(deviceTypes[:index], deviceTypes[index+1:]...)
			}
		}

		lastPublishedSystemVersion := DeviceSystemVersion{}
		c.ormDB.Set("gorm:auto_preload", true).Where("is_publish = true").Last(&lastPublishedSystemVersion)

		if lastPublishedSystemVersion.ID > 0 {
			for _, deviceType := range deviceTypes {
				deviceSystemVersionType := DeviceSystemVersionType{}
				deviceTypeVersion := DeviceTypeVersion{}

				switch deviceType {
				case DeviceType_APK_PRO:
					deviceTypeVersion = lastPublishedSystemVersion.GetLastDeviceTypeVersion(c.ormDB, DeviceType_APK_PRO)
					break
				case DeviceType_Podium:
					deviceTypeVersion = lastPublishedSystemVersion.GetLastDeviceTypeVersion(c.ormDB, DeviceType_Podium)
					break
				case DeviceType_PORTAL:
					deviceTypeVersion = lastPublishedSystemVersion.GetLastDeviceTypeVersion(c.ormDB, DeviceType_PORTAL)
					break
				}

				if deviceTypeVersion.ID > 0 {
					deviceSystemVersionType.DeviceTypeVersion = deviceTypeVersion
					systemVersion.DeviceSystemVersionTypes = append(systemVersion.DeviceSystemVersionTypes, deviceSystemVersionType)
				}
			}
		} else {
			if len(deviceTypes) != 0 {
				err := errors.New("For device types, an update must be uploaded!")
				c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
				return
			}
		}

		for _, item := range systemVersion.DeviceSystemVersionTypes {
			if item.DeviceTypeVersion.ID == 0 {
				if item.DeviceTypeVersion.Version == "" {
					err := errors.New("Version must not be empty")
					c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
					return
				}

				if item.DeviceTypeVersion.DeviceType.ID == 0 {
					err := errors.New("No DeviceType with that name")
					c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
					return
				}
			}
		}

		//Zu erst wurde überprüft, angegben sind
		c.ormDB.Set("gorm:save_associations", false).Create(&systemVersion)
		for key, item := range systemVersion.DeviceSystemVersionTypes {

			if item.DeviceTypeVersion.ID == 0 {
				item.DeviceTypeVersion.OnlyForNewUpdateSystem = true
				c.ormDB.Debug().Set("gorm:save_associations", false).Create(&item.DeviceTypeVersion)
			}

			if item.ID == 0 {
				item.DeviceSystemVersionId = systemVersion.ID
				item.DeviceTypeVersionId = item.DeviceTypeVersion.ID
				c.ormDB.Set("gorm:save_associations", false).Create(&item)
			}
			systemVersion.DeviceSystemVersionTypes[key] = item
		}
	} else {
		systemVersionDB := DeviceSystemVersion{}
		c.ormDB.Set("gorm:auto_preload", true).First(&systemVersionDB, systemVersion.ID)
		if systemVersionDB.IsPublish {
			err := errors.New("System version cannot be edited because it has already been published.")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}

		//HIER DARF ES NICHT AUF gepublished gesetzt werden, da für gibt es eine extra Route
		systemVersion.IsPublish = systemVersionDB.IsPublish // false

		if systemVersionDB.SystemVersion != systemVersion.SystemVersion {
			checkSystemVersion := DeviceSystemVersion{}
			c.ormDB.Set("gorm:auto_preload", true).Where("system_version = ?", systemVersion.SystemVersion).Last(&checkSystemVersion)
			if checkSystemVersion.ID > 0 {
				err := errors.New("System version already exists.")
				c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
				return
			}
		}

		for _, item := range systemVersion.DeviceSystemVersionTypes {
			if item.DeviceTypeVersion.ID == 0 {
				if item.DeviceTypeVersion.Version == "" {
					err := errors.New("Version must not be empty")
					c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
					return
				}

				if item.DeviceTypeVersion.DeviceType.ID == 0 {
					err := errors.New("No DeviceType with that name")
					c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
					return
				}

				for key, versionTypeDB := range systemVersionDB.DeviceSystemVersionTypes {
					//if versionTypeDB.DeviceTypeVersion.DeviceType.ID == item.DeviceTypeVersion.DeviceType.ID{
					if versionTypeDB.DeviceTypeVersion.DeviceType.ID == item.DeviceTypeVersion.DeviceType.ID {
						if versionTypeDB.DeviceTypeVersion.Version != item.DeviceTypeVersion.Version {

							//c.ormDB.Debug().Set("gorm:save_associations", false).Delete(&versionTypeDB.DeviceTypeVersion)
							c.ormDB.Debug().Set("gorm:save_associations", false).Delete(&versionTypeDB)

							item.ID = 0
							item.DeviceTypeVersion.ID = 0
							c.ormDB.Debug().Set("gorm:save_associations", false).Save(&item.DeviceTypeVersion)
							item.DeviceSystemVersionId = systemVersion.ID
							item.DeviceTypeVersionId = item.DeviceTypeVersion.ID
							c.ormDB.Set("gorm:save_associations", false).Save(&item)
							systemVersion.DeviceSystemVersionTypes[key] = item
						}
						//}
					}
				}

			}
		}

		c.ormDB.Set("gorm:save_associations", false).Save(&systemVersion)

		c.ormDB.Set("gorm:auto_preload", true).First(&systemVersion, systemVersion.ID)
	}

	//c.ormDB.Set("gorm:auto_preload", true).Last(&systemVersion, systemId)
	c.SendJSON(w, &systemVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) UploadDeviceSystemVersionHandler(w http.ResponseWriter, r *http.Request) {
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
	deviceTypeVersionId, _ := strconv.ParseInt(vars["deviceTypeVersionId"], 10, 64)

	deviceType := r.FormValue("device_type")
	version := r.FormValue("version")

	typeVersion := DeviceTypeVersion{}
	c.ormDB.Set("gorm:auto_preload", true).Where("id = ? AND version = ? AND device_type_id IN (SELECT id FROM device_types WHERE type_name = ?)", deviceTypeVersionId, version, deviceType).Last(&typeVersion)

	if typeVersion.ID == 0 {
		err := errors.New("Version not found!")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	if typeVersion.DeviceType.TypeName == "" || typeVersion.Version == "" {
		err := errors.New("Version or device type not found!")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	systemVersion := DeviceSystemVersion{}
	c.ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT device_system_version_id FROM device_system_version_types WHERE device_type_version_id = ?) AND is_publish = true", typeVersion.ID).Order("publish_date DESC").First(&systemVersion)

	if systemVersion.ID > 0 {
		err := errors.New("Version can not be changed, because the already published.")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	err := os.MkdirAll(fmt.Sprintf(core.GetUploadFilepath()+"devices/%s/version/%s/", typeVersion.DeviceType.TypeName, typeVersion.Version), 0777)
	r.ParseMultipartForm(32 << 20)
	//var err error
	for fileKey, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); nil != err {
				//status = http.StatusInternalServerError
				_ = infile
				log.Println(err)
				return
			}
			// open destination
			var outfile *os.File
			log.Println(hdr.Filename)
			pos := strings.LastIndex(hdr.Filename, "/")
			filename := hdr.Filename[pos+1:]
			filePath := fmt.Sprintf(core.GetUploadFilepath()+"devices/%s/version/%s/%s", typeVersion.DeviceType.TypeName, typeVersion.Version, filename)
			if outfile, err = os.Create(filePath); nil != err {
				//status = http.StatusInternalServerError
				log.Println(err)
			}
			// 32K buffer copy
			var written int64
			if written, err = io.Copy(outfile, infile); nil != err {
				//status = http.StatusInternalServerError
				log.Println(err)
				return
			}
			log.Println("uploaded file:" + hdr.Filename + ";length:" + strconv.Itoa(int(written)))

			fileType := "APK"
			if strings.Contains(strings.ToUpper(fileKey), "DEB") {
				fileType = "DEB"
			} else if strings.Contains(strings.ToUpper(fileKey), "SIG") {
				fileType = "SIG"
			}
			switch fileType {
			case "APK":
				typeVersion.APKFileName = filename
				break
			case "DEB":
				typeVersion.DebFileName = filename
				break
			case "SIG":
				typeVersion.SigFileName = filename
				break
			}
		}
	}
	c.ormDB.Set("gorm:save_associations", false).Save(&typeVersion)

	c.SendJSON(w, &typeVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) PublishDeviceSystemVersionHandler(w http.ResponseWriter, r *http.Request) {
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
	systemVersionId, _ := strconv.ParseInt(vars["systemVersionId"], 10, 64)

	systemVersion := DeviceSystemVersion{}
	c.ormDB.Debug().Set("gorm:auto_preload", true).Last(&systemVersion, systemVersionId)

	if systemVersion.ID == 0 {
		err := errors.New("System Version not found!")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	if systemVersion.SystemVersion == "" {
		err := errors.New("Invalid version of the system version!")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	if systemVersion.IsPublish {
		err := errors.New("Version already published!")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	//Prüfen, ob alle notwendigen DeviceTypen hinzugefügt wurde
	deviceTypes := []uint{DeviceType_APK_PRO, DeviceType_Podium, DeviceType_PORTAL}
	for _, item := range systemVersion.DeviceSystemVersionTypes {
		index := c.Find(deviceTypes, item.DeviceTypeVersion.DeviceType.ID)
		if index > -1 && len(deviceTypes) > 0 {
			deviceTypes = append(deviceTypes[:index], deviceTypes[index+1:]...)
		}
	}

	missingVersions := ""
	for _, item := range deviceTypes {
		if missingVersions != "" {
			missingVersions += ", "
		}
		switch item {
		case DeviceType_PORTAL:
			missingVersions += "Portal"
			break
		case DeviceType_APK_PRO:
			missingVersions += "Apk Pro"
			break
		case DeviceType_Podium:
			missingVersions += "Podium"
			break
		}
	}
	if missingVersions != "" {
		errorMessage := "It is missing the following version to publish: "
		errorMessage += missingVersions
		err := errors.New(errorMessage)
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	//Pürfune, ob alle Dateien hochgeladen wurden.
	deviceTypesFiles := []uint{DeviceType_APK_PRO, DeviceType_Podium}
	for _, item := range systemVersion.DeviceSystemVersionTypes {
		index := c.Find(deviceTypesFiles, item.DeviceTypeVersion.DeviceType.ID)

		if index > -1 && len(deviceTypesFiles) > 0 {
			switch item.DeviceTypeVersion.DeviceTypeId {
			case DeviceType_APK_PRO:
				if item.DeviceTypeVersion.APKFileName != "" {
					deviceTypesFiles = append(deviceTypesFiles[:index], deviceTypesFiles[index+1:]...)
				}
				break
			case DeviceType_Podium:
				if item.DeviceTypeVersion.DebFileName != "" && item.DeviceTypeVersion.SigFileName != "" {
					deviceTypesFiles = append(deviceTypesFiles[:index], deviceTypesFiles[index+1:]...)
				}
				break
			}
		}
	}

	missingFiles := ""
	for _, item := range deviceTypesFiles {
		if missingFiles != "" {
			missingFiles += ", "
		}
		switch item {
		case DeviceType_APK_PRO:
			missingFiles += "Apk Pro"
			break
		case DeviceType_Podium:
			missingFiles += "Podium"
			break
		}
	}
	if missingFiles != "" {
		errorMessage := "It is missing the following files to publish: "
		errorMessage += missingFiles
		err := errors.New(errorMessage)
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	c.ormDB.Model(&systemVersion).Where("id = ?", systemVersion.ID).Update("is_publish", true)
	c.ormDB.Model(&systemVersion).Where("id = ?", systemVersion.ID).Update("publish_by_id", user.ID)
	c.ormDB.Model(&systemVersion).Where("id = ?", systemVersion.ID).Update("publish_date", tools.NullTime{Time: time.Now(), Valid: true})
	systemVersion.PublishBy = *user
	c.SendJSON(w, &systemVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDeviceSystemVersionHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/
	vars := mux.Vars(r)
	systemId, _ := strconv.ParseInt(vars["systemId"], 10, 64)

	systemVersion := &DeviceSystemVersion{}

	c.ormDB.Set("gorm:auto_preload", true).Last(&systemVersion, systemId)

	c.SendJSON(w, &systemVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetLastDeviceSystemVersionHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/
	vars := mux.Vars(r)
	systemId, _ := strconv.ParseInt(vars["systemId"], 10, 64)

	systemVersion := &DeviceSystemVersion{}

	c.ormDB.Set("gorm:auto_preload", true).Last(&systemVersion, systemId)

	c.SendJSON(w, &systemVersion, http.StatusOK)
}

func (c *PodiumController) GetCurrentPortalSystemVersionHandler(w http.ResponseWriter, r *http.Request) {

	lastSystemVersion := &DeviceSystemVersion{}

	c.ormDB.Set("gorm:auto_preload", false).Debug().Where("is_publish = true").Order("LENGTH(system_version) DESC, system_version DESC").First(&lastSystemVersion)

	systemVersion := &DeviceSystemVersion{}
	systemVersion.ID = lastSystemVersion.ID
	systemVersion.SystemVersion = lastSystemVersion.SystemVersion

	c.SendJSON(w, &systemVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDeviceTypeVersionHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/
	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["typeId"], 10, 64)

	typeVersion := &DeviceTypeVersion{}

	if typeId > 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("device_type_id = ?", typeId).Last(&typeVersion)
	} else {
		c.ormDB.Debug().Set("gorm:auto_preload", true).Joins("LEFT JOIN device_system_version_types ON device_type_version_id = device_type_versions.id").Joins("LEFT JOIN device_system_versions ON device_system_version_id = device_system_versions.id").Where("device_type_id IN (SELECT id FROM device_types WHERE type_name = ?) AND is_publish = true", vars["typeName"]).Order("publish_date DESC").First(&typeVersion)
	}

	c.SendJSON(w, &typeVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDeviceTypeVersionFileHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}*/

	vars := mux.Vars(r)
	typeId, _ := strconv.ParseInt(vars["typeId"], 10, 64)
	versionId, _ := strconv.ParseInt(vars["versionId"], 10, 64)
	fileType := strings.ToUpper(vars["fileType"])

	typeVersion := &DeviceTypeVersion{}
	if typeId > 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("device_type_id = ?", typeId).Find(&typeVersion, versionId)
	} else {
		c.ormDB.Set("gorm:auto_preload", true).Where("device_type_id IN (SELECT id FROM device_types WHERE type_name = ?)", vars["typeName"]).Find(&typeVersion, versionId)
	}

	if typeVersion.ID > 0 {
		filepath := core.GetUploadFilepath() + "devices/" + typeVersion.DeviceType.TypeName + "/version/" + typeVersion.Version + "/"
		filename := ""
		switch fileType {
		case "DEB":
			filename = typeVersion.DebFileName
		case "SIG":
			filename = typeVersion.SigFileName
		case "APK":
			filename = typeVersion.APKFileName
		}
		log.Println(filepath + filename)
		w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		w.Header().Add("Access-Control-Allow-Origin", "*")
		http.ServeFile(w, r, filepath+filename)
		return
	}

	c.HandleErrorWithStatus(errors.New("File not found"), w, http.StatusNotFound)
}

// Retrieve Files
func (c *PodiumController) CreateDeviceTypeVersionHandler(w http.ResponseWriter, r *http.Request) {
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
	deviceType := r.FormValue("device_type")
	version := r.FormValue("version")
	systemVersion := r.FormValue("system_version")

	newVersion := DeviceTypeVersion{}
	newVersion.Version = version
	newVersion.SystemVersion = systemVersion

	if version == "" {
		err := errors.New("Version must not be empty")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	if systemVersion == "" {
		err := errors.New("System version must not be empty")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	c.ormDB.Set("gorm:auto_preload", false).Where("type_name = ?", deviceType).Find(&newVersion.DeviceType)
	if newVersion.DeviceType.ID == 0 {
		err := errors.New("No DeviceType with that name")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	newVersion.DeviceTypeId = newVersion.DeviceType.ID
	err := os.MkdirAll(fmt.Sprintf(core.GetUploadFilepath()+"devices/%s/version/%s/", newVersion.DeviceType.TypeName, newVersion.Version), 0777)
	if err != nil {
		log.Println(err)
	}
	r.ParseMultipartForm(32 << 20)
	//var err error
	for fileKey, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			// open uploaded
			var infile multipart.File
			if infile, err = hdr.Open(); nil != err {
				//status = http.StatusInternalServerError
				_ = infile
				log.Println(err)
				return
			}
			// open destination
			var outfile *os.File
			log.Println(hdr.Filename)
			pos := strings.LastIndex(hdr.Filename, "/")
			filename := hdr.Filename[pos+1:]
			filePath := fmt.Sprintf(core.GetUploadFilepath()+"devices/%s/version/%s/%s", newVersion.DeviceType.TypeName, newVersion.Version, filename)
			if outfile, err = os.Create(filePath); nil != err {
				//status = http.StatusInternalServerError
				log.Println(err)
			}
			// 32K buffer copy
			var written int64
			if written, err = io.Copy(outfile, infile); nil != err {
				//status = http.StatusInternalServerError
				log.Println(err)
				return
			}
			log.Println("uploaded file:" + hdr.Filename + ";length:" + strconv.Itoa(int(written)))

			fileType := "APK"
			if strings.Contains(strings.ToUpper(fileKey), "DEB") {
				fileType = "DEB"
			} else if strings.Contains(strings.ToUpper(fileKey), "SIG") {
				fileType = "SIG"
			}
			switch fileType {
			case "APK":
				newVersion.APKFileName = filename
				break
			case "DEB":
				newVersion.DebFileName = filename
				break
			case "SIG":
				newVersion.SigFileName = filename
				break
			}

		}
	}

	c.ormDB.Set("gorm:save_associations", false).Create(&newVersion)

	c.SendJSON(w, newVersion, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDeviceTypesHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/
	types := &DeviceTypes{}

	c.ormDB.Where("show_in_portal = 1").Set("gorm:auto_preload", true).Find(&types)

	c.SendJSON(w, &types, http.StatusOK)
}

// getDoctor swagger:route GET /doctors/{doctorId} doctor getDoctor
//
// retrieves informations about a doctor
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: doctorId
//     in: path
//     description: the ID for the doctor
//     required: true
//     type: integer
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: Doctor
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) SavePracticeDeviceHandler(w http.ResponseWriter, r *http.Request) {

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	if !c.isPractice(user) {
		err := errors.New("only practices can add devices")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	pracDevice := PracticeDevice{}
	if err := c.GetContent(&pracDevice, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	practice := c.getPractice(user)
	pracDevice.PracticeId = practice.ID
	c.ormDB.Set("gorm:save_associations", false).Save(&pracDevice.Device)
	pracDevice.DeviceId = pracDevice.Device.ID
	c.ormDB.Set("gorm:save_associations", false).Save(&pracDevice)

	c.SendJSON(w, &pracDevice, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetDeviceHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/

	vars := mux.Vars(r)
	deviceId, _ := strconv.ParseInt(vars["deviceId"], 10, 64)

	device := &Device{}

	c.ormDB.Set("gorm:auto_preload", true).Find(&device, deviceId)

	c.ormDB.Set("gorm:auto_preload", true).Where("device_id = ?", device.ID).Order("measurement_date DESC").First(&device.LastMeasurement)

	c.SendJSON(w, &device, http.StatusOK)
}

func (c *PodiumController) TestMailHandler(w http.ResponseWriter, r *http.Request) {
	err := core.SendMail("info@podium.care", []string{"jonathan.cremer@symblcrowd.de", "dr.russell.payne@outlook.com", "russell.payne@thermetrix.com"}, []string{}, []string{}, "TEST", "TEST", []string{})

	c.SendJSON(w, &err, http.StatusOK)
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//   - name: Authorization
//     in: header
//     description: "Bearer " + token
//     required: true
//     type: string
//   - name: userId
//     in: path
//     description: the ID of the other conversation partner
//     required: true
//     type: string
//
// Responses:
//
//	   default: HandleErrorData
//			  200:
//				data: []Message
//	       401: HandleErrorData "unauthorized"
//	       403: HandleErrorData "no Permission"
func (c *PodiumController) GetRiskDefinitionsHandler(w http.ResponseWriter, r *http.Request) {
	/*ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}
	*/
	riskDefinitions := RiskDefinitions{}

	c.ormDB.Set("gorm:auto_preload", true).Find(&riskDefinitions)

	c.SendJSON(w, &riskDefinitions, http.StatusOK)
}

func (c *PodiumController) insertRiskDefinitions() {
	c.ormDB.Exec("DELETE FROM risk_definitions")
	tmps := RiskDefinitions{
		{Model: core.Model{ID: 1}, Title: "low risk", Shortcut: "LR", Description: "visit clinician within next 12 months", SortValue: 2, SortValueAsc: 4},
		{Model: core.Model{ID: 2}, Title: "medium risk", Shortcut: "MR", Description: "visit clinician within next 6 weeks or sooner if situation worsens (nice says 3-6 months)", SortValue: 3, SortValueAsc: 3},
		{Model: core.Model{ID: 3}, Title: "high risk", Shortcut: "HR", Description: "visit clinician within next 2 weeks or sooner if situation worsens  (nice says 1-2 months)", SortValue: 4, SortValueAsc: 2},
		{Model: core.Model{ID: 4}, Title: "active problem", Shortcut: "AP", Description: "visit clinician within a week or sooner if situation worsens (nice sas 1-2 weeks)", SortValue: 1, SortValueAsc: 1},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) insertAppointmentStatusDefs() {
	c.ormDB.Exec("DELETE FROM appointment_status_defs")
	tmps := AppointmentStatusDefs{
		{Model: core.Model{ID: 1}, StatusName: "Anfrage durch Patient", StatusDescription: ""},
		{Model: core.Model{ID: 2}, StatusName: "Terminvorschlag durch Doktor (fragt der Doktor an, hat es direkt den Status, da Doktor initial immer Termin mitgibt)", StatusDescription: ""},
		{Model: core.Model{ID: 3}, StatusName: "Akzeptiert durch Patient", StatusDescription: ""},
		{Model: core.Model{ID: 4}, StatusName: "Abgelehnt durch Patient (danach keine weitere Aktion mehr möglich, Patient/Doktor muss neues Appointment stellen)", StatusDescription: ""},
		{Model: core.Model{ID: 5}, StatusName: "Abgelehnt durch Doktor (danach keine weitere Aktion mehr möglich, Patient/Doktor muss neues Appointment stellen)", StatusDescription: ""},
		{Model: core.Model{ID: 6}, StatusName: "Anfrage Reschedule durch Patient", StatusDescription: ""},
		{Model: core.Model{ID: 7}, StatusName: "Anfrage Reschedule durch Doktor", StatusDescription: ""},
		{Model: core.Model{ID: 8}, StatusName: "Reschedule abgelehnt (not used at moment)", StatusDescription: ""},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

// TODO TEST
func (c *PodiumController) insertDeviceTypes() {
	c.ormDB.Exec("DELETE FROM device_types")
	tmps := DeviceTypes{
		{Model: core.Model{ID: uint(DeviceType_Podium)}, TypeName: "podium", ShowInPortal: false},
		{Model: core.Model{ID: uint(DeviceType_APK_HOME)}, TypeName: "apk_home"},
		{Model: core.Model{ID: uint(DeviceType_APK_PRO)}, TypeName: "apk_pro"},
		{Model: core.Model{ID: uint(DeviceType_APK_REMOTE)}, TypeName: "apk_remote"},
		{Model: core.Model{ID: uint(DeviceType_PORTAL)}, TypeName: "portal"},
	}
	for _, tmp := range tmps {
		c.ormDB.Debug().Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) insertRewardMonetaryDiscounts() {
	c.ormDB.Exec("DELETE FROM reward_monetary_discounts")
	tmps := RewardMonetaryDiscounts{
		{Model: core.Model{ID: 1}, DiscountValue: 0.5, ConsecutiveScansThreshold: 10},
		{Model: core.Model{ID: 2}, DiscountValue: 1.5, ConsecutiveScansThreshold: 20},
		{Model: core.Model{ID: 3}, DiscountValue: 2.5, ConsecutiveScansThreshold: 30},
		{Model: core.Model{ID: 4}, DiscountValue: 6, ConsecutiveScansThreshold: 60},
		{Model: core.Model{ID: 5}, DiscountValue: 15, ConsecutiveScansThreshold: 120},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) insertRewardStarRatings() {
	c.ormDB.Exec("DELETE FROM reward_star_ratings")
	tmps := RewardStarRatings{
		{Model: core.Model{ID: 1}, StarLevel: 1, ConsecutiveScansThreshold: 10},
		{Model: core.Model{ID: 2}, StarLevel: 2, ConsecutiveScansThreshold: 20},
		{Model: core.Model{ID: 3}, StarLevel: 3, ConsecutiveScansThreshold: 30},
		{Model: core.Model{ID: 4}, StarLevel: 4, ConsecutiveScansThreshold: 60},
		{Model: core.Model{ID: 5}, StarLevel: 5, ConsecutiveScansThreshold: 120},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) insertStandardQuestionTemplates() {

	ids := []uint{
		1,
		2,
		3,
		4,
		5,
		6,
		7,
		8,
		9,
		10,
		11,
		12,
		13,
		14,
		15,
		16,
		17,
		18,
		159,
		160,
	}

	c.ormDB.Exec("DELETE FROM question_templates WHERE id IN (?)", ids)
	tmps := QuestionTemplates{
		{Model: core.Model{ID: 1}, DoctorId: 0, QuestionType: 1, RecurringRule: "SETUP", QuestionText: "Have you every had any ulceration?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Ulceration"},
		{Model: core.Model{ID: 2}, DoctorId: 0, QuestionType: 1, RecurringRule: "SETUP", QuestionText: "Have you every had any amputation?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Amputation"},
		{Model: core.Model{ID: 3}, DoctorId: 0, QuestionType: 1, RecurringRule: "SETUP", QuestionText: "Have you every had a kidney transplant?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Kidney transplant"},
		{Model: core.Model{ID: 4}, DoctorId: 0, QuestionType: 1, RecurringRule: "SETUP", QuestionText: "Have you every been diagnosed with peripheral artery disease or had a related intervention?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "peripheral artery disease"},
		{Model: core.Model{ID: 5}, DoctorId: 0, QuestionType: 1, RecurringRule: "SETUP", QuestionText: "Do you have a deformity on either foot?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Deformity"},
		{Model: core.Model{ID: 6}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Describe your present foot health", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Foot health"},
		{Model: core.Model{ID: 7}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Redness that stays when foot is elevated?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Redness - Elevated"},
		{Model: core.Model{ID: 8}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do either of your feet feel hot to touch? (hotter than each other or surroundings)", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Hotness"},
		{Model: core.Model{ID: 9}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Are any of your toe nails damaged or infected?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Damaged nails"},
		{Model: core.Model{ID: 10}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Are they ever numb?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Numb"},
		{Model: core.Model{ID: 11}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do they ever tingle?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Tingle"},
		{Model: core.Model{ID: 12}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do they every feel like insects are crawling on them?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Insects"},
		{Model: core.Model{ID: 13}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Are your feet swollen?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Swollen"},
		{Model: core.Model{ID: 14}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Redness when feet are on ground, pale when elevated?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Redness - Ground"},
		{Model: core.Model{ID: 15}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do either of your feet feel cold to touch? (colder than each other or surroundings)", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Coldness"},
		{Model: core.Model{ID: 16}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do they ever burn or hurt when elevated (i.e. in bed), but feel better when sitting or standing?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Burn - Elevated"},
		{Model: core.Model{ID: 17}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do you change your socks daily?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Socks"},
		{Model: core.Model{ID: 18}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Do you wear any tight fitting hosiery? (socks/tights)", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Hosiery"},
		{Model: core.Model{ID: 159}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Is there a loss of protective sensation as determined by a 10g monofilament test or similar?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "Monofilament"},
		{Model: core.Model{ID: 160}, DoctorId: 0, QuestionType: 2, RecurringRule: "DAILY-SCAN", QuestionText: "Is there evidence of peripheral arterial disease?", QuestionFormat: 0, IsRequired: true, PatientId: 0, QuestionKeyword: "arterial disease"},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) insertStandardQuestionTemplateAnswers() {

	ids := []uint{
		1,
		2,
		3,
		4,
		5,
		6,
		7,
		8,
		9,
		10,
		11,
		12,
		13,
		14,
		15,
		16,
		17,
		18,
		19,
		20,
		21,
		22,
		23,
		24,
		25,
		26,
		27,
		28,
		29,
		30,
		31,
		32,
		33,
		34,
		35,
		36,
		37,
		38,
		39,
		40,
		324,
		325,
		326,
		327,
		328,
		329,
	}

	c.ormDB.Exec("DELETE FROM question_template_answers WHERE id IN (?)", ids)
	tmps := QuestionTemplateAnswers{
		{Model: core.Model{ID: 1}, QuestionTemplateId: 1, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 2}, QuestionTemplateId: 1, AnswerValue: 1, RiskRating: 3, AnswerText: "yes"},
		{Model: core.Model{ID: 3}, QuestionTemplateId: 2, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 4}, QuestionTemplateId: 2, AnswerValue: 1, RiskRating: 3, AnswerText: "yes"},
		{Model: core.Model{ID: 5}, QuestionTemplateId: 3, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 6}, QuestionTemplateId: 3, AnswerValue: 1, RiskRating: 3, AnswerText: "yes"},
		{Model: core.Model{ID: 7}, QuestionTemplateId: 4, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 8}, QuestionTemplateId: 4, AnswerValue: 1, RiskRating: 3, AnswerText: "yes"},
		{Model: core.Model{ID: 9}, QuestionTemplateId: 5, AnswerValue: 0, RiskRating: 1, AnswerText: "no"},
		{Model: core.Model{ID: 10}, QuestionTemplateId: 5, AnswerValue: 1, RiskRating: 1, AnswerText: "minor"},
		{Model: core.Model{ID: 11}, QuestionTemplateId: 5, AnswerValue: 2, RiskRating: 2, AnswerText: "major"},
		{Model: core.Model{ID: 12}, QuestionTemplateId: 5, AnswerValue: 3, RiskRating: 2, AnswerText: "yes, but not sure"},
		{Model: core.Model{ID: 13}, QuestionTemplateId: 6, AnswerValue: 0, RiskRating: 1, AnswerText: "healthy"},
		{Model: core.Model{ID: 14}, QuestionTemplateId: 6, AnswerValue: 1, RiskRating: 1, AnswerText: "dry"},
		{Model: core.Model{ID: 15}, QuestionTemplateId: 6, AnswerValue: 2, RiskRating: 2, AnswerText: "heavy callus"},
		{Model: core.Model{ID: 16}, QuestionTemplateId: 6, AnswerValue: 3, RiskRating: 4, AnswerText: "open ulcer or wound"},
		{Model: core.Model{ID: 17}, QuestionTemplateId: 7, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 18}, QuestionTemplateId: 7, AnswerValue: 1, RiskRating: 4, AnswerText: "yes"},
		{Model: core.Model{ID: 19}, QuestionTemplateId: 8, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 20}, QuestionTemplateId: 8, AnswerValue: 1, RiskRating: 3, AnswerText: "yes"},
		{Model: core.Model{ID: 21}, QuestionTemplateId: 9, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 22}, QuestionTemplateId: 9, AnswerValue: 1, RiskRating: 4, AnswerText: "yes"},
		{Model: core.Model{ID: 23}, QuestionTemplateId: 10, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 24}, QuestionTemplateId: 10, AnswerValue: 1, RiskRating: 2, AnswerText: "yes"},
		{Model: core.Model{ID: 25}, QuestionTemplateId: 11, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 26}, QuestionTemplateId: 11, AnswerValue: 1, RiskRating: 2, AnswerText: "yes"},
		{Model: core.Model{ID: 27}, QuestionTemplateId: 12, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 28}, QuestionTemplateId: 12, AnswerValue: 1, RiskRating: 2, AnswerText: "yes"},
		{Model: core.Model{ID: 29}, QuestionTemplateId: 13, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 30}, QuestionTemplateId: 13, AnswerValue: 1, RiskRating: 4, AnswerText: "yes"},
		{Model: core.Model{ID: 31}, QuestionTemplateId: 14, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 32}, QuestionTemplateId: 14, AnswerValue: 1, RiskRating: 4, AnswerText: "yes"},
		{Model: core.Model{ID: 33}, QuestionTemplateId: 15, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 34}, QuestionTemplateId: 15, AnswerValue: 1, RiskRating: 3, AnswerText: "yes"},
		{Model: core.Model{ID: 35}, QuestionTemplateId: 16, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 36}, QuestionTemplateId: 16, AnswerValue: 1, RiskRating: 4, AnswerText: "yes"},
		{Model: core.Model{ID: 37}, QuestionTemplateId: 17, AnswerValue: 0, RiskRating: 2, AnswerText: "no"},
		{Model: core.Model{ID: 38}, QuestionTemplateId: 17, AnswerValue: 1, RiskRating: 0, AnswerText: "yes"},
		{Model: core.Model{ID: 39}, QuestionTemplateId: 18, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 40}, QuestionTemplateId: 18, AnswerValue: 1, RiskRating: 2, AnswerText: "yes"},
		{Model: core.Model{ID: 324}, QuestionTemplateId: 159, AnswerValue: 1, RiskRating: 2, AnswerText: "yes"},
		{Model: core.Model{ID: 325}, QuestionTemplateId: 159, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 326}, QuestionTemplateId: 159, AnswerValue: 0, RiskRating: 0, AnswerText: "N/A"},
		{Model: core.Model{ID: 327}, QuestionTemplateId: 160, AnswerValue: 1, RiskRating: 2, AnswerText: "yes"},
		{Model: core.Model{ID: 328}, QuestionTemplateId: 160, AnswerValue: 0, RiskRating: 0, AnswerText: "no"},
		{Model: core.Model{ID: 329}, QuestionTemplateId: 160, AnswerValue: 0, RiskRating: 0, AnswerText: "N/A"},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) insertRewardUserLevels() {

	c.ormDB.Exec("DELETE FROM reward_user_levels")
	tmps := RewardUserLevels{
		{Model: core.Model{ID: 1}, Title: "Experienced", ScansThreshold: 360},
		{Model: core.Model{ID: 2}, Title: "Super User", ScansThreshold: 720},
		{Model: core.Model{ID: 3}, Title: "Veteran", ScansThreshold: 1080},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func (c *PodiumController) InitSetIsTenMinsApart() {
	practices := Practices{}

	c.ormDB.Set("gorm:auto_preload", false).Find(&practices)
	for _, practice := range practices {

		userId := practice.UserId

		doctors := Doctors{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Find(&doctors)

		for _, doctor := range doctors { //USER_ID = 545,  Doctor = 385

			patients := Patients{}
			//Gib mir alle Patienten von der Praxis, die einen Scan haben
			c.ormDB.Debug().Where("id IN (SELECT patient_id FROM measurements WHERE user_id = ?)", userId).Find(&patients)

			for _, patient := range patients {
				measurementsShared := MeasurementsShared{}
				lastGoodMeasurementShared := MeasurementShared{}

				db := c.ormDB.Debug().Where("measurement_shareds.doctor_id = ? AND measurements.user_id = ? AND measurements.patient_id = ?", doctor.ID, userId, patient.ID)
				db = db.Joins("LEFT JOIN measurements ON measurements.id = measurement_shareds.measurement_id")
				db.Set("gorm:auto_preload", true).Order("measurement_date ASC").Find(&measurementsShared)

				for _, measurementShared := range measurementsShared {
					tenMinApart := false

					if measurementShared.ID == 0 || measurementShared.Measurement.MeasurementDate.Time.Unix()-lastGoodMeasurementShared.Measurement.MeasurementDate.Time.Unix() > 600 {
						lastGoodMeasurementShared = measurementShared
						tenMinApart = true
					}

					//Soll nur den Datensatz anpassen, wenn es eine Änderung gibt.
					if measurementShared.TenMinsApart != tenMinApart {
						c.ormDB.Model(&MeasurementShared{}).Where("id = ?", measurementShared.ID).Update("ten_mins_apart", tenMinApart)
					}
				}
			}
		}
	}
}

/*

UPDATE question_templates
SET custom_answers=true
Where question_type = 3 AND isNull(question_templates.deleted_at) and question_templates.id IN
(SELECT qta.question_template_id FROM question_template_answers qta WHERE qta.answer_text != 'yes' AND qta.answer_text != 'no' OR
(SELECT COUNT(*) FROM question_template_answers tmp WHERE tmp.question_template_id = qta.question_template_id GROUP BY tmp.question_template_id) != 2 GROUP BY qta.question_template_id);
*/
