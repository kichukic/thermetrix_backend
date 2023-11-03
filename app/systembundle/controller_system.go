package systembundle

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"thermetrix_backend/app/core"
	"thermetrix_backend/app/podiumbundle"
	"thermetrix_backend/app/websocket"
	"time"
)

var WSTickets map[string]string
var LogFile *os.File

const MODULE_ID = 1

// SystemController struct
type SystemController struct {
	core.Controller
	//hm SystemMapper
	db    *sql.DB
	ormDB *gorm.DB
}

// NewSystemController instance
func NewSystemController(ormDB *gorm.DB, Users *map[string]core.User) *SystemController {
	WSTickets = make(map[string]string)

	c := &SystemController{
		Controller: core.Controller{Users: Users},
		ormDB:      ormDB,
	}

	c.insertRiskDefinitions()
	if core.Config.Database.DoAutoMigrate {
		ormDB.AutoMigrate(&SystemFrontendLog{}, &SystemSetting{})
		ormDB.AutoMigrate(&core.User{}, &SystemAccountsSession{}, &SystemLog{})
		ormDB.AutoMigrate(&core.SystemAccountSetting{})
		ormDB.AutoMigrate(&SystemAccountPasswordReset{})
		ormDB.AutoMigrate(&SystemFrontendTranslationKeyTranslation{}, &SystemFrontendTranslationLanguage{}, &SystemFrontendTranslationKey{})

		if core.Config.Database.DoInsert {
			c.insertRiskDefinitions()
		}
	}

	go web3socket.HandleUserMessages()
	go web3socket.HandleBroadcastMessages()

	return c
}

//SystemFrontendTranslationLanguage

func (c *SystemController) insertRiskDefinitions() {
	c.ormDB.Exec("DELETE FROM system_frontend_translation_language")
	tmps := SystemFrontendTranslationLanguages{
		{Model: core.Model{ID: 1}, LanguageCode: "en-US", Language: "en-US"},
	}
	for _, tmp := range tmps {
		c.ormDB.Set("gorm:save_associations", false).Create(&tmp)
	}
}

func LogEvent(logData LogData) {
	return
}

func (c *SystemController) LogFrontendEventHandler(w http.ResponseWriter, r *http.Request) {

	var user *core.User
	var ok bool

	if ok, user = c.TryGetUser(w, r); !ok {
		_ = user
		//return
	}

	log.Println(user)

	userText := ""

	if user != nil {
		userText = fmt.Sprintf("(Logged User: %d - %s)", user.ID, user.Username)
	}

	var logData LogData

	if err := c.GetContent(&logData, r); err != nil {
		return
	}

	logFileName := fmt.Sprintf("logs/frontend_%s.log", time.Now().Format("20060102"))

	if LogFile == nil || LogFile.Name() != logFileName {
		var err error
		LogFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Println(err)
		}
	}
	logLevel := ""

	switch logData.LogLevel {
	case 0:
		logLevel = "NORMAL"
	case 1:
		logLevel = "WARNING"
	case 2:
		logLevel = "DEBUG"
	case 3:
		logLevel = "ERROR"
	case 4:
		logLevel = "FATAL"
	default:
		logLevel = "UNKNOWN"
	}
	text := fmt.Sprintf("%s%s|%s:%d|%s|%s|%s\n", time.Now().Format("2006-01-02 15:04-05"), userText, logData.LogFile, logData.LogLine, logLevel, logData.LogRoute, logData.LogText)

	if _, err := LogFile.WriteString(text); err != nil {
		log.Println(err)
	}

	c.SendJSON(w, &logData, http.StatusOK)

}

func arrayToString(a []int, delim string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
	//return strings.Trim(strings.Join(strings.Split(fmt.Sprint(a), " "), delim), "[]")
	//return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(a)), delim), "[]")
}

// getSystemLogs swagger:route GET /system/log system getSystemLogs
//
// retrieves all SystemLogs
//
// produces:
// - application/json
// parameters:
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//        200:
//	       data: []SystemLog
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *SystemController) GetSystemLogsHandler(w http.ResponseWriter, r *http.Request) {

	logs := &SystemLogs{}
	c.ormDB.Set("gorm:auto_preload", true).Find(&logs)

	c.SendJSON(w, &logs, http.StatusOK)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // number of letter indices fitting in 63 bits
)

func GenerateRandomString(n int) string {
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

func (c *SystemController) DeleteDataForUsersNotIn(usernames []string) {
	users := core.Users{}
	userIds := make([]uint, 0)
	patients := podiumbundle.Patients{}
	patientIds := make([]uint, 0)
	doctors := podiumbundle.Doctors{}
	doctorIds := make([]uint, 0)
	practices := podiumbundle.Practices{}
	practiceIds := make([]uint, 0)
	c.ormDB.Debug().Unscoped().Where(`username IN (?)`, usernames).Find(&users)

	for _, user := range users {
		userIds = append(userIds, user.ID)
	}

	c.ormDB.Debug().Unscoped().Where(`id IN (SELECT doctor_id FROM doctor_users du WHERE user_id IN (?))`, userIds).Find(&doctors)

	for _, doctor := range doctors {
		doctorIds = append(doctorIds, doctor.ID)
	}

	c.ormDB.Debug().Unscoped().Where(`user_id IN (?)`, userIds).Find(&patients)

	for _, patient := range patients {
		patientIds = append(patientIds, patient.ID)
	}
	c.ormDB.Debug().Unscoped().Where(`user_id IN (?)`, userIds).Find(&practices)
	for _, practice := range practices {
		practiceIds = append(practiceIds, practice.ID)
	}
	log.Println("test")

	c.ormDB.Debug().Where("id NOT IN (?)", patientIds).Delete(&podiumbundle.Patient{})
	c.ormDB.Debug().Where("patient_id NOT IN (?)", patientIds).Delete(&podiumbundle.PatientDevice{})
	c.ormDB.Debug().Where("patient_id NOT IN (?)", patientIds).Delete(&podiumbundle.PatientReward{})
	c.ormDB.Debug().Where("patient_id NOT IN (?)", patientIds).Delete(&podiumbundle.PatientQuestionnaire{})
	c.ormDB.Debug().Where("patient_questionnaire_id IN (SELECT id FROM patient_questionnaires WHERE deleted_at IS NOT NULL)").Delete(&podiumbundle.PatientQuestionnaireQuestion{})
	c.ormDB.Debug().Where("patient_questionnaire_id IN (SELECT id FROM patient_questionnaires WHERE deleted_at IS NOT NULL) OR patient_id NOT IN(?)", patientIds).Delete(&podiumbundle.RiskCalculation{})
	c.ormDB.Debug().Where("id NOT IN (?)", practiceIds).Delete(&podiumbundle.Practice{})
	c.ormDB.Debug().Where("practice_id NOT IN (?)", practiceIds).Delete(&podiumbundle.PracticeDevice{})
	c.ormDB.Debug().Where("practice_id NOT IN (?) OR doctor_id NOT IN (?)", practiceIds, doctorIds).Delete(&podiumbundle.PracticeDoctor{})
	c.ormDB.Debug().Where("doctor_id NOT IN (?) OR patient_id NOT IN (?)", doctorIds, patientIds).Delete(&podiumbundle.Appointment{})
	c.ormDB.Debug().Where("source_appointment_id IN (SELECT id FROM appointments WHERE deleted_at IS NOT NULL)").Delete(&podiumbundle.AppointmentStatus{})
	c.ormDB.Debug().Where("doctor_id NOT IN (?) OR patient_id NOT IN (?)", doctorIds, patientIds).Delete(&podiumbundle.DoctorApproach{})
	c.ormDB.Debug().Where("doctor_id NOT IN (?) OR patient_id NOT IN (?)", doctorIds, patientIds).Delete(&podiumbundle.DoctorPatientRelation{})
	c.ormDB.Debug().Where("doctor_id NOT IN (?) OR patient_id NOT IN (?)", doctorIds, patientIds).Where("doctor_id > 0").Delete(&podiumbundle.QuestionTemplate{})
	c.ormDB.Debug().Where("question_template_id IN (SELECT id FROM question_templates WHERE deleted_at IS NOT NULL)").Delete(&podiumbundle.QuestionTemplateAnswer{})
	c.ormDB.Debug().Where("id NOT IN (?)", doctorIds).Delete(&podiumbundle.Doctor{})
	c.ormDB.Debug().Where("doctor_id NOT IN (?)", doctorIds).Delete(&podiumbundle.DoctorDevice{})
	c.ormDB.Debug().Where("doctor_id NOT IN (?) OR user_id NOT IN (?)", doctorIds, userIds).Delete(&podiumbundle.DoctorUser{})
	c.ormDB.Debug().Where("(doctor_id NOT IN (?) AND doctor_id > 0) OR user_id NOT IN (?) OR patient_id NOT IN (?)", doctorIds, userIds, patientIds).Delete(&podiumbundle.Measurement{})
	c.ormDB.Debug().Where("measurement_id IN (SELECT id FROM measurements WHERE deleted_at IS NOT NULL) OR doctor_id NOT IN (?)", doctorIds).Delete(&podiumbundle.MeasurementShared{})
	c.ormDB.Debug().Where("measurement_id IN (SELECT id FROM measurements WHERE deleted_at IS NOT NULL)").Delete(&podiumbundle.MeasurementFile{})
	c.ormDB.Debug().Where("(doctor_id NOT IN (?) AND doctor_id > 0) OR sender_id NOT IN (?) OR recipient_id NOT IN (?)", doctorIds, userIds, userIds).Delete(&podiumbundle.Message{})
	//c.ormDB.Where("message_id IN (SELECT id FROM messages WHERE deleted_at IS NOT NULL)", doctorIds).Delete(&podiumbundle.Message{}.Attachments)

	c.ormDB.Debug().Where("id NOT IN (?)", userIds).Delete(&core.User{})
	c.ormDB.Debug().Where("account_id NOT IN (?)", userIds).Delete(&SystemAccountsSession{})
	c.ormDB.Debug().Where("user_id NOT IN (?)", userIds).Delete(&podiumbundle.Notification{})
	c.ormDB.Debug().Where("user_id NOT IN (?)", userIds).Delete(SystemAccountPasswordReset{})
	c.ormDB.Debug().Where("user_id NOT IN (?)", userIds).Delete(podiumbundle.Annotation{})
}
