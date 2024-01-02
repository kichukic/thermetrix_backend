package podiumbundle

import "strings"
import "time"

// PatientImages struct to hold patient ID and images
type PatientImages struct {
    PatientID      string `gorm:"column:patient_id"`
    ImagePath      string `gorm:"column:image_path"`
    Images         string `gorm:"type:LONGTEXT;column:images"`
    IsDFA_Complete bool   `gorm:"column:is_dfa_complete;default:false"`
    CreatedAt      time.Time `gorm:"column:created_at"`
}

type PatientImage struct {
    PatientID      string `gorm:"column:patient_id"`
    ImagePath string `json:"url"`
    Images    string `gorm:"type:LONGTEXT;column:images"`
    IsDFA_Complete bool   `gorm:"column:is_dfa_complete;default:false"`
}

// type PatientImage struct {
//     Images []string `json:"images"`
// }

// Function to create a new instance of PatientImages
func NewPatientImages(patientID string, images []string) *PatientImages {
	// Serialize the images slice to a comma-separated string before storing in the database
	imageStr := strings.Join(images, ",")
    now := time.Now()
	return &PatientImages{
		PatientID: patientID,
		Images:    imageStr,
        IsDFA_Complete: false,
        CreatedAt:  now,
	}
}

// Function to get the deserialized images
func (p *PatientImages) GetImages() []string {
	// Deserialize the comma-separated string to a slice of strings
	return strings.Split(p.Images, ",")
}
