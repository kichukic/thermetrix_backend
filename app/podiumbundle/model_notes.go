package podiumbundle

import "time"

type notes struct {
	PatientID     string    `json:"patient_id" gorm:"column:patient_id"`
	MeasurementID string    `json:"measurement_id" gorm:"column:measurement_id"`
	Notes         string    `json:"notes" gorm:"column:notes"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}


type note struct {
    MeasurementID string `gorm:"column:measurement_id"`
    PatientID     string `gorm:"column:patient_id"`
    Notes         string `gorm:"column:notes"`
}