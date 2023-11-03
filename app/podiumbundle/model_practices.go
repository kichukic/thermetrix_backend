package podiumbundle

import (
	"github.com/jinzhu/gorm"
	"thermetrix_backend/app/core"
)

type Practice struct {
	core.Model
	UserId               uint             `json:"-"`
	User                 core.User        `json:"user"`
	Name                 string           `json:"name"`
	Postcode             string           `json:"postcode"`
	Doctors              []PracticeDoctor `json:"doctors"`
	Devices              []PracticeDevice `json:"devices"`
	HasSameDoctor        bool             `json:"has_same_doctor"`
	PracticeContract     PracticeContract `json:"practice_contract"`
	AccountsAbbreviation string           `json:"accounts_abbreviation"`

	Patients Patients `json:"patients" gorm:"-"`

	Errors map[string]string `json:"-" gorm:"-"`
}

type Practices []Practice

type PracticeDoctor struct {
	core.Model
	PracticeId         uint   `json:"-"`
	DoctorId           uint   `json:"-"`
	Doctor             Doctor `json:"doctor"`
	AllowPracticeLogin bool   `json:"allow_practice_login"`
}

type PracticeContract struct {
	core.Model
	PracticeId        uint          `json:"-"`
	ContractStartDate core.NullTime `json:"contract_start_date"`
	ContractEndDate   core.NullTime `json:"contract_end_date"`
	ContractTerm      string        `json:"contract_term"`
	CountDevices      uint          `json:"count_devices"`
	PaymentAmount     int64         `json:"payment_amount"`
	PaymentPeriod     string        `json:"payment_period"`
}

type PracticeDevice struct {
	core.Model
	PracticeId uint   `json:"-"`
	DeviceId   uint   `json:"-"`
	Device     Device `json:"device"`
}

type PracticeStatistic struct {
	CountCustomers       int64           `json:"count_customers"`
	CountDoctors         int64           `json:"count_doctors"`
	CountScansThisYear   int64           `json:"count_scans_this_year"`
	CountScansThisMonth  int64           `json:"count_scans_this_month"`
	CountScansThisWeek   int64           `json:"count_scans_this_week"`
	CountScansToday      int64           `json:"count_scans_today"`
	CountScansTotal      int64           `json:"count_scans_total"`
	CountSuccessfulScans int64           `json:"count_successful_scans"`
	CountScansPerMonth   []ScansPerMonth `json:"scans_per_month"`
	CountScansPerWeek    []ScansPerWeek  `json:"scans_per_week"`
}

type ScansPerMonth struct {
	Year       string `json:"year"`
	Month      string `json:"month"`
	CountScans int64  `json:"count_scans"`
}

type ScansPerWeek struct {
	Year       string `json:"year"`
	Month      string `json:"month"`
	Week       string `json:"week"`
	CountScans int64  `json:"count_scans"`
}

type DoctorTransferHelper struct {
	NewDoctor Doctor    `json:"new_doctor"`
	OldDoctor Doctor    `json:"old_doctor"`
	Patients  []Patient `json:"patients"`
}

func (practice *Practice) getFullDoctors(ormDB *gorm.DB) Doctors {

	doctors := Doctors{}
	ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(&doctors)

	return doctors
}

func (practice *Practice) getDoctors(ormDB *gorm.DB) Doctors {
	doctors := Doctors{}
	ormDB.Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(&doctors)

	return doctors
}

func (practice *Practice) getSmallDoctors(ormDB *gorm.DB) SmallDoctors {
	doctors := SmallDoctors{}
	ormDB.Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(&doctors)

	return doctors
}

func (practice *Practice) getDoctorsIds(ormDB *gorm.DB) []uint {

	tmp := []uint{}

	//doctors := Doctors{}
	//ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(tmp)
	return tmp
}
