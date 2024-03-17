
package podiumbundle

import "time"

type CoverLetter struct {
    MeasurementID string    `gorm:"column:measurement_id"`
    PatientID     string    `gorm:"column:patient_id"`
    CoverLetter   string    `gorm:"column:cover_letter"`
    IsReferred    bool      `gorm:"column:is_referred"`
    CreatedAt     time.Time `gorm:"column:created_at"`
}
