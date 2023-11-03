package podiumbundle

import (
	"encoding/json"
//	"gotools/tools"
	tools "github.com/kirillDanshin/nulltime"
	"thermetrix_backend/app/core"
)

const (
	DoctorUserStatus_Doctor = 1
)

/*
noch offen:
	- Unit
	- UnitGroup
	- Import


*/

// (Verbindung Podiatrist - Patient; Patient - Patient; Patient - Angehörige
// --> wird so nicht von Richards API übernommen.
// wir haben eine Verbindung zwischen Patient und Doctor, die gespeichert wird
// für die Verknüpfung zwischen Patient und anderen Usern gibt es dann eine extra Tabelle
/*type Consent struct {
	PatientId					uint					`json:"-"`
	Patient						Patient					`json:"patient"`
	DoctorId					uint					`json:"-"`
	Doctor						Doctor					`json:"doctor"`
	SpecialistFieldId			uint					`json:"-"`
	ConsentStatus				uint					`json:"consent_type"`
	ConsentType					uint					`json:"-"` // 1 - Patient - Doctor; 2 - Patient - User
}*/

type DoctorPatientRelation struct {
	core.Model
	PatientId         uint          `json:"-"`
	Patient           Patient       `json:"patient"`
	DoctorId          uint          `json:"-"`
	Doctor            Doctor        `json:"doctor"`
	SpecialistFieldId uint          `json:"-"`
	ConsentStatus     uint          `json:"consent_status"` // 1 - Patient requested doctor; 2 - doctor accepted; --> If doctior created User, the user (if is patient) is autmatically added, with status 2
	ConsentType       uint          `json:"consent_type"`   // 1 - Patient - Doctor; 2 - Doctor - Patient
	ConsentDate       core.NullTime `json:"consent_date"`
}

type DoctorPatientRelations []DoctorPatientRelation

type Doctor struct {
	core.Model
	Name                   string            `json:"name"`
	FirstName              string            `json:"first_name"`
	LastName               string            `json:"last_name"`
	AddressLine1           string            `json:"address_line1"`
	AddressLine2           string            `json:"address_line2"`
	Postcode               string            `json:"postcode"`
	County                 string            `json:"county"`
	Town                   string            `json:"town"`
	Country                string            `json:"country"`
	Phone                  string            `json:"phone"`
	Email                  string            `json:"email"`
	Latitude               float64           `json:"latitude"`
	Longitude              float64           `json:"longitude"`
	StandardWelcomeMessage string            `json:"standard_welcome_message" gorm:"default:'Hello'"`
	SpecialistFields       []SpecialistField `json:"specialist_fields" gorm:"many2many:podium_doctor_specialist_fields"`
	IsRegistered           bool              `json:"is_registered"`
	Users                  []DoctorUser      `json:"users"`
	Gender                 string            `json:"gender"`

	HasHomeVisit bool   `json:"has_home_visit"`
	Description  string `json:"description"`
	CrawlId      uint   `json:"-"`

	HasHomeUsers bool `json:"has_home_users"`

	Devices []DoctorDevice `json:"devices"`

	CountScans           int64    `json:"count_scans" gorm:"-"`
	CountSuccessfulScans int64    `json:"count_successful_scans" gorm:"-"`
	Distance             float64  `json:"distance" gorm:"-"`
	Practice             Practice `json:"practice,omitempty" gorm:"-"`

	Errors map[string]string `json:"-" gorm:"-"`
}
type Doctors []Doctor

type SmallDoctor struct {
	core.Model
	Name         string       `json:"name"`
	FirstName    string       `json:"first_name"`
	LastName     string       `json:"last_name"`
	Email        string       `json:"email"`
	IsRegistered bool         `json:"is_registered"`
	Users        []DoctorUser `json:"users"`
}
type SmallDoctors []SmallDoctor

func (SmallDoctor) TableName() string {
	return "doctors"
}

type DoctorUser struct {
	core.Model
	DoctorId uint      `json:"-"`
	UserId   uint      `json:"-"`
	User     core.User `json:"user"`
	Status   uint      `json:"-"` // 1 - Doctor; 2 - Assistant
}

type DoctorApproach struct {
	core.Model
	DoctorId     uint    `json:"-"`
	Doctor       Doctor  `json:"doctor"`
	PatientId    uint    `json:"-"`
	Patient      Patient `json:"patient"`
	ApproachType string  `json:"approach_type"`
}

type RewardStarRating struct {
	core.Model
	StarLevel                 int64 `json:"star_level"`
	ConsecutiveScansThreshold int64 `json:"consecutive_scans_threshold"`
}
type RewardStarRatings []RewardStarRating

type RewardMonetaryDiscount struct {
	core.Model
	DiscountValue             float64 `json:"discount_value"`
	ConsecutiveScansThreshold int64   `json:"consecutive_scans_threshold"`
}
type RewardMonetaryDiscounts []RewardMonetaryDiscount

type RewardUserLevel struct {
	core.Model
	Title          string `json:"title"`
	ScansThreshold int64  `json:"scans_threshold"`
}
type RewardUserLevels []RewardUserLevel

type Patient struct {
	core.Model
	UserId        uint            `json:"-"`
	User          core.User       `json:"user"`
	FirstName     string          `json:"first_name"`
	LastName      string          `json:"last_name"`
	Postcode      string          `json:"postcode"`
	County        string          `json:"county"`
	Town          string          `json:"town"`
	Country       string          `json:"country"`
	AddressLine1  string          `json:"address_line1"`
	AddressLine2  string          `json:"address_line2"`
	Phone         string          `json:"phone"`
	Gender        string          `json:"gender"`
	BirthDate     core.NullTime   `json:"birth_date"`
	Rewards       []PatientReward `json:"rewards,omitempty"`
	CurrentReward PatientReward   `json:"current_reward,omitempty" gorm:"-"`

	ConsecutiveScans int64 `json:"consecutive_scans" gorm:"-"`
	TotalScans       int64 `json:"total_scans" gorm:"-"`

	RiskRating int64 `json:"risk_rating"`

	LastAppointmentDate           core.NullTime `json:"last_appointment_date" gorm:"-"`
	LastMeasurementDate           core.NullTime `json:"last_measurement_date" gorm:"-"`
	RedMeasurementsLastWeekCount  int64         `json:"red_measurements_last_week_count" gorm:"-"`
	BlueMeasurementsLastWeekCount int64         `json:"blue_measurements_last_week_count" gorm:"-"`

	LastScanQuestionnaire *PatientQuestionnaire `json:"last_scan_questionnaire" gorm:"-"`
	LastQuestionnaire     *PatientQuestionnaire `json:"last_questionnaire" gorm:"-"`

	CountScans int64 `json:"count_scans" gorm:"-"`

	Devices []PatientDevice `json:"devices"`

	SetupComplete       bool `json:"setup_complete"`
	HasPairedPodiatrist bool `json:"has_paired_podiatrist"`
	PracticeCanEdit     bool `json:"practice_can_edit" gorm:"-"`
	AddedOwnQuestions   bool `json:"added_own_questions" gorm:"-"`

	Practice Practice `json:"practice,omitempty" gorm:"-"`

	Errors map[string]string `json:"-" gorm:"-"`
}
type Patients []Patient

type PatientReward struct {
	core.Model
	RewardStarRatingId       uint                   `json:"-"`
	RewardStarRating         RewardStarRating       `json:"star_rating"`
	RewardMonetaryDiscountId uint                   `json:"-"`
	RewardMonetaryDiscount   RewardMonetaryDiscount `json:"monetary_discount"`
	RewardUserLevelId        uint                   `json:"-"`
	RewardUserLevel          RewardUserLevel        `json:"user_level"`
	PatientId                uint                   `json:"-"`
	RewardDate               core.NullTime          `json:"reward_date"`
}

type SpecialistField struct {
	core.Model
	Title string `json:"title"`
}
type SpecialistFields []SpecialistField

type Appointment struct {
	core.Model

	DoctorId  uint    `json:"-"`
	Doctor    Doctor  `json:"doctor"`
	PatientId uint    `json:"-"`
	Patient   Patient `json:"patient"`

	DateTimeFrom core.NullTime `json:"date_time_from"`

	DateTimeTo core.NullTime `json:"date_time_to"`
	Content    string        `json:"content" gorm:"type:TEXT"`

	RequestDateTime core.NullTime `json:"request_date_time"`
	//RequestStatus		int							`json:"request_status"`

	AppointmentStatusId uint              `json:"-"`
	AppointmentStatus   AppointmentStatus `json:"request_status"`

	OriginAppointmentId uint `json:"-"`
	//OriginAppointment		*Appointment					`json:"origin_appointment" gorm:"PRELOAD:false"`
	OriginAppointment *Appointment `json:"-" gorm:"PRELOAD:false"` //BUG WITH JSON

}
type Appointments []Appointment

//
//
// swagger:model AppointmentStatus
type AppointmentStatus struct {
	core.Model
	StatusDefId         uint                 `json:"-"`
	StatusDef           AppointmentStatusDef `json:"-"`
	StatusDate          core.NullTime        `json:"status_date" gorm:"type:datetime"`
	StatusName          string               `json:"status_name" gorm:"-"`
	SourceAppointmentId uint                 `json:"-"`
	CreatedById         uint                 `json:"-"`
	CreatedBy           core.User            `json:"created_by,omitempty" gorm:"-"`
	Errors              map[string]string    `json:"-" gorm:"-"`
}

func (appointmentStatus *AppointmentStatus) MarshalJSON() ([]byte, error) {
	type Alias AppointmentStatus
	return json.Marshal(&struct {
		ID         uint   `json:"id"`
		StatusName string `json:"status_name"`
		*Alias
	}{
		ID:         appointmentStatus.StatusDef.ID,
		StatusName: appointmentStatus.StatusDef.StatusName,
		Alias:      (*Alias)(appointmentStatus),
	})
}

//
//
// swagger:model AppointmentStatusDef
type AppointmentStatusDef struct {
	core.Model
	StatusName        string            `json:"status_name"`
	StatusDescription string            `json:"status_description" gorm:"type:TEXT"`
	Errors            map[string]string `json:"-" gorm:"-"`
}
type AppointmentStatusDefs []AppointmentStatusDef

/*
1: Anfrage durch Patient
2: Terminvorschlag durch Doktor (fragt der Doktor an, hat es direkt den Status, da Doktor initial immer Termin mitgibt)
3: Akzeptiert durch Patient
4: Abgelehnt durch Patient (danach keine weitere Aktion mehr möglich, Patient/Doktor muss neues Appointment stellen)
5: Abgelehnt durch Doktor (danach keine weitere Aktion mehr möglich, Patient/Doktor muss neues Appointment stellen)
6: Anfrage Reschedule durch Patient
7: Anfrage Reschedule durch Doktor
8: Reschedule abgelehnt
*/

type Annotation struct {
	core.Model
	UserId        uint        `json:"-"`
	User          core.User   `json:"-"`
	HelperUser    HelperUser  `json:"user"`
	MeasurementId uint        `json:"-"`
	Measurement   Measurement `json:"measurement"`
	//AnnotationType		int						`json:"annotation_type"`
	AnnotationTime core.NullTime `json:"annotation_time"`

	VisibilityStatus int `json:"visibility_status"` // 1 - alle; 2 - nur Ersteller

	Subject string `json:"subject"`
	Content string `json:"content"`

	IsDeleted bool `json:"is_deleted"` //Flag used because notes need to be displayed crossed out when they are "deleted"

	DoctorId uint `json:"doctor_id" gorm:"-"`
	//TODO locator?
}
type Annotations []Annotation

type MeasurementLightbox struct {
	core.Model
	Title  string                     `json:"title"`
	Images []MeasurementLightboxImage `json:"images"`
}

type MeasurementLightboxImage struct {
	core.Model
	DownloadUrl string `json:"download_url"`
}

type Measurement struct {
	core.Model

	PatientId uint    `json:"-"`
	Patient   Patient `json:"patient,omitempty" gorm:"PRELOAD,false"` //Patient

	MeasurementDate  core.NullTime     `json:"measurement_date"`
	MeasurementFiles []MeasurementFile `json:"measurement_files"`

	HotspotDetected     string  `json:"hotspot_detected" gorm:"type:ENUM('NONE', 'LEFT', 'RIGHT', 'BOTH')"`
	ColdspotDetected    string  `json:"coldspot_detected" gorm:"type:ENUM('NONE', 'LEFT', 'RIGHT', 'BOTH')"`
	DeviceTemperature   float64 `json:"device_temperature"`
	DeviceHumidity      float64 `json:"device_humidity"`
	DeviceBattery       float64 `json:"device_battery"`
	DeviceVersion       string  `json:"device_version"`
	DeviceSystemVersion string  `json:"device_system_version"`

	AppVersion       string     `json:"app_version"`
	AppSystemVersion string     `json:"app_system_version"`
	AppBuild         int64      `json:"app_build"`
	AppDeviceTypeId  uint       `json:"-"`
	AppDeviceType    DeviceType `json:"app_device_type"`

	TmpDate string `json:"-"`

	Questionnaire      *PatientQuestionnaire `json:"questionnaire" gorm:"-"`
	DailyQuestionnaire *PatientQuestionnaire `json:"daily_questionnaire" gorm:"-"`
	SetupQuestionnaire *PatientQuestionnaire `json:"setup_questionnaire" gorm:"-"`
	//PreviousSetupQuestionnaire *PatientQuestionnaire `json:"setup_questionnaire" gorm:"-"`

	DeviceId   uint                  `json:"-"`
	Device     *Device               `json:"device,omitempty" gorm:"PRELOAD,false"`
	UserId     uint                  `json:"-"`
	DoctorId   uint                  `json:"-"`
	Doctor     *Doctor               `json:"doctor,omitempty" gorm:"PRELOAD,false"`
	IsFavorite bool                  `json:"is_favorite" gorm:"-"`
	Favorites  []MeasurementFavorite `json:"-"`

	MeasurementRisk *MeasurementDoctorRisk `json:"measurement_risk,omitempty" gorm:"-"`
	Practice        *Practice              `json:"practice,omitempty" gorm:"-"`
}
type Measurements []Measurement

type MeasurementFavorite struct {
	core.Model
	MeasurementId uint `json:"-"`
	UserId        uint `json:"user_id"`
}

type MeasurementDoctorRisk struct {
	core.Model
	MeasurementId    uint           `json:"-"`
	DoctorId         uint           `json:"-"`
	RiskDefinitionId uint           `json:"-"`
	RiskDefinition   RiskDefinition `json:"risk_definition"`
	MeasurementDate  core.NullTime  `json:"measurement_date,omitempty" gorm:"-"`
}
type MeasurementDoctorRisks []MeasurementDoctorRisk

type MeasurementFile struct {
	core.Model
	MeasurementId   uint   `json:"-"`
	MeasurementType string `json:"measurement_type" gorm:"type:ENUM('NORMAL', 'THERMAL', 'REPORT', 'STATISTIC', 'DYNAMIC', 'T0', 'INPUT', 'FOOT_POSITIONING', 'DYNAMIC_STATISTIC')"`
	Label           string `json:"-" gorm:"-"`
	Filepath        string `json:"-"`
}

type MeasurementShared struct {
	core.Model
	MeasurementId uint        `json:"measurement_id"`
	Measurement   Measurement `json:"measurement"`
	DoctorId      uint        `json:"doctor_id"`
	Doctor        Doctor      `json:"doctor"`
	TenMinsApart  bool        `json:"ten_min_apart"`
}

type MeasurementsShared []MeasurementShared

// Device vom User
type Device struct {
	core.Model
	DeviceIdentifier string       `json:"device_identifier"`
	DeviceMac        string       `json:"device_mac"`
	DeviceSerial     string       `json:"device_serial"`
	DeviceTypeId     uint         `json:"-"`
	DeviceType       DeviceType   `json:"device_type"`
	LastMeasurement  Measurement  `json:"last_measurement" gorm:"-"`
	LastDeviceStatus DeviceStatus `json:"last_device_status" gorm:"-"`
}
type Devices []Device

type DeviceStatus struct {
	core.Model
	DeviceId             uint          `json:"device_id"`
	MeasurementId        uint          `json:"measurement_id"`
	StatusDate           core.NullTime `json:"status_date"`
	Valid                bool          `json:"valid"`
	Temperature          float64       `json:"temperature"`
	Humidity             float64       `json:"humidity"`
	Battery              float64       `json:"battery"`
	LeftMinTemperature   float64       `json:"left_min_temperature"`
	LeftMeanTemperature  float64       `json:"left_mean_temperature"`
	LeftMaxTemperature   float64       `json:"left_max_temperature"`
	RightMinTemperature  float64       `json:"right_min_temperature"`
	RightMeanTemperature float64       `json:"right_mean_temperature"`
	RightMaxTemperature  float64       `json:"right_max_temperature"`
	DiffMinTemperature   float64       `json:"diff_min_temperature"`
	DiffMeanTemperature  float64       `json:"diff_mean_temperature"`
	DiffMaxTemperature   float64       `json:"diff_max_temperature"`
}

type DoctorDevice struct {
	core.Model
	DoctorId uint   `json:"-"`
	DeviceId uint   `json:"-"`
	Device   Device `json:"device"`
}

type PatientDevice struct {
	core.Model
	PatientId uint   `json:"-"`
	DeviceId  uint   `json:"-"`
	Device    Device `json:"device"`
}

type Sensor struct {
	core.Model
	SensorIdentifier string `json:"sensor_identifier"`
}
type Sensors []Sensor

type DeviceType struct {
	core.Model
	TypeName     string       `json:"type_name"`
	SensorTypes  []SensorType `json:"sensor_types"`
	ShowInPortal bool         `json:"-"`
}
type DeviceTypes []DeviceType

type DeviceSystemVersion struct {
	core.Model
	IsPublish                bool                     `json:"is_publish"`
	SystemVersion            string                   `json:"system_version"`
	DeviceSystemVersionTypes DeviceSystemVersionTypes `json:"device_versions"`
	PublishDate              tools.NullTime           `json:"publish_date"`
	PublishById              uint                     `json:"-"`
	PublishBy                core.User                `json:"publish_by"`
}

type DeviceSystemVersionType struct {
	core.Model
	DeviceSystemVersionId uint `json:"-"`

	DeviceTypeVersionId uint              `json:"-"`
	DeviceTypeVersion   DeviceTypeVersion `json:"device_version"`
}
type DeviceSystemVersionTypes []DeviceSystemVersionType

type DeviceTypeVersion struct {
	core.Model
	DeviceTypeId           uint       `json:"-"`
	DeviceType             DeviceType `json:"device_type"`
	Version                string     `json:"version"`
	SystemVersion          string     `json:"system_version"`
	APKFileName            string     `json:"apk_filename"`
	DebFileName            string     `json:"deb_filename"`
	SigFileName            string     `json:"sig_filename"`
	OnlyForNewUpdateSystem bool       `json:"-"`
}

const (
	DeviceType_Podium     uint = 1
	DeviceType_APK_HOME   uint = 2
	DeviceType_APK_PRO    uint = 3
	DeviceType_APK_REMOTE uint = 4
	DeviceType_PORTAL     uint = 5
)

type SensorType struct {
	core.Model
}
type SensorTypes []SensorType

// CreatorId = 0: system generated, not deletable, for all podiatrists, ...
// QuestionnaireType: 1 - App Start (only defineid by system); 2 - after measurement (only defined by system); 3 - after measurement, individuell, defined by podiatrist
/*type QuestionnaireTemplate struct {
	core.Model
	QuestionnaireTitle		string							`json:"questionnaire_title"`
	QuestionnaireType		int								`json:"questionnaire_type"`
	Questions				[]QuestionnaireTemplateQuestion	`json:"questions"`
	PatientId				uint							`json:"-"`
	Patient					Patient							`json:"patient"`
}
type QuestionnaireTemplates []QuestionnaireTemplate
*/

// CreatorId = 0: system generated, not deletable, for all podiatrists, ...
type QuestionTemplate struct {
	core.Model
	DoctorId        uint                     `json:"-"`
	QuestionType    int                      `json:"question_type"` //1- Setup, 2- System, 3- Doctor
	DateFrom        core.NullTime            `json:"date_from"`
	DateTo          core.NullTime            `json:"date_to"`
	RecurringRule   string                   `json:"recurring_rule"`
	QuestionText    string                   `json:"question_text"`
	QuestionKeyword string                   `json:"question_keyword"`
	QuestionFormat  int                      `json:"question_format"`
	IsRequired      bool                     `json:"is_required"`
	Answers         []QuestionTemplateAnswer `json:"answers"`
	PatientId       uint                     `json:"-"`
	Patient         *Patient                 `json:"patient,omitempty" gorm:"PRELOAD:false"`
	CustomAnswers   bool                     `json:"custom_answers"`
}
type QuestionTemplates []QuestionTemplate

type QuestionTemplateAnswer struct {
	core.Model
	QuestionTemplateId uint   `json:"-"`
	AnswerValue        uint   `json:"value"`
	AnswerText         string `json:"answer"`
	RiskRating         int64  `json:"risk"` // 1 - low; 2 - mid; 3 - high; 4 - akut
}
type QuestionTemplateAnswers []QuestionTemplateAnswer

type PatientQuestionnaire struct {
	core.Model
	//QuestionnaireTitle		string							`json:"questionnaire_title"`
	//QuestionnaireType		int								`json:"questionnaire_type"`
	Questions         []PatientQuestionnaireQuestion `json:"questions"`
	QuestionnaireDate core.NullTime                  `json:"questionnaire_date"`
	PatientId         uint                           `json:"-"`
	Patient           Patient                        `json:"patient,omitempty"`
	MeasurementId     uint                           `json:"-"`
	//Patient				Patient					`json:"patient"`
}
type PatientQuestionnaires []PatientQuestionnaire

// CreatorId = 0: system generated, not deletable, for all podiatrists, ...
type PatientQuestionnaireQuestion struct {
	core.Model
	DeletedAtPublic        core.NullTime          `json:"deleted_at"`
	DoctorId               uint                   `json:"-"`
	PatientQuestionnaireId uint                   `json:"-"`
	TemplateQuestionId     uint                   `json:"-"`
	TemplateQuestion       QuestionTemplate       `json:"template_question"`
	QuestionDate           core.NullTime          `json:"question_date"`
	AnswerId               uint                   `json:"-"`
	Answer                 QuestionTemplateAnswer `json:"answer"`
}
type PatientQuestionnaireQuestions []PatientQuestionnaireQuestion

type RiskCalculation struct {
	core.Model
	PatientId                uint           `json:"-"`
	PatientQuestionnaireId   uint           `json:"-"`
	CalculationDate          core.NullTime  `json:"calculation_date" gorm:"type:DATETIME"`
	IsInitialRiskCalculation bool           `json:"is_initial_risk_calculation"`
	RiskRatingId             int64          `json:"-"`
	RiskRating               RiskDefinition `json:"risk"`
}
type RiskCalculations []RiskCalculation

type RiskDefinition struct {
	core.Model
	Shortcut     string `json:"shortcut"`
	Title        string `json:"title"`
	Description  string `json:"description" gorm:"type:TEXT"`
	SortValue    uint   `json:"-"`
	SortValueAsc uint   `json:"-"`
}
type RiskDefinitions []RiskDefinition

type HelperConversation struct {
	InterlocutorId     uint       `json:"-"`
	Interlocutor       core.User  `json:"-" gorm:"-"`
	InterlocutorHelper HelperUser `json:"interlocutor"`
	UnreadMessages     uint       `json:"unread_messages"`
	LastMessageId      uint       `json:"-"`
	LastMessage        Message    `json:"last_message"`
	Messages           []Message  `json:"messages,omitempty"`
	DoctorId           uint       `json:"-"`
}
type HelperConversations []HelperConversation

func (d HelperConversations) Len() int      { return len(d) }
func (d HelperConversations) Swap(i, j int) { d[i], d[j] = d[j], d[i] }
func (d HelperConversations) Less(i, j int) bool {
	return d[i].LastMessage.MessageTime.Time.Unix() > d[j].LastMessage.MessageTime.Time.Unix()
}

type Message struct {
	core.Model
	MessageTime core.NullTime `json:"message_time"`
	DoctorId    uint          `json:"-"` // optional
	SenderId    uint          `json:"-"`
	Sender      core.User     `json:"sender"`
	RecipientId uint          `json:"-"`
	Recipient   core.User     `json:"recipient"`
	IsUnread    bool          `json:"is_unread"`
	MessageText string        `json:"message_text" gorm:"type:TEXT"`
	Attachments []Measurement `json:"attachments,omitempty"  gorm:"many2many:messages_measurements"`
}
type Messages []Message

type HelperUser struct {
	core.User
	Patient  *Patient  `json:"patient,omitempty"`
	Doctor   *Doctor   `json:"doctor,omitempty"`
	Practice *Practice `json:"practice,omitempty"`
}

type Notification struct {
	core.Model
	UserId           uint                 `json:"-"`
	NotificationType int64                `json:"notification_type"` // 1 - Chat, 2 - Appointment, 3 - Neue Frage zu beantwortem, 4 App-Welcome Message, TODO SM neue Notification NEuie Frage 5 Neue Frage vorhanden beim Nächsten Scan
	Visible          bool                 `json:"visible"`
	ForeignId        uint                 `json:"foreign_id"`
	SourceUrl        string               `json:"source_url"`
	Title            string               `json:"title"`
	Message          string               `json:"message"`
	NotificationDate core.NullTime        `json:"notification_date"`
	Actions          []NotificationAction `json:"actions,omitempty"`

	Appointment *Appointment `json:"appointment,omitempty" gorm:"-"`
	//User				*core.User				`json:"doctor,omitempty" gorm:"-"`
	Interlocutor       *core.User  `json:"-" gorm:"-"`
	InterlocutorHelper *HelperUser `json:"interlocutor" gorm:"-"`
}
type Notifications []Notification

type NotificationAction struct {
	core.Model
	NotificationId uint   `json:"-"`
	Title          string `json:"string"`
	ActionUrl      string `json:"action_url"`
}

type NotificationActions []NotificationAction

type HelperStatistic struct {
	core.Model
	NumberOfPatients      int64         `json:"number_of_patients"`
	NewPatientsThisMonth  int64         `json:"new_patients_this_month"`
	AppointmentsThisMonth int64         `json:"appointments_this_month"`
	SubscriptionUntil     core.NullTime `json:"subscription_until"`
}

type HelperStatistics []HelperStatistic
