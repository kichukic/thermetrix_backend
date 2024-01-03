package podiumbundle

import "strings"
import "time"

// PatientImages struct to hold patient ID and images
type PatientImages struct {
    Measurement_id string `gorm:"column:measurement_id"`
    PatientID      string `gorm:"column:patient_id"`
    ImagePath      string `gorm:"column:image_path"`
    Images         string `gorm:"type:LONGTEXT;column:images"`
    IsDFA_Complete bool   `gorm:"column:is_dfa_complete;default:false"`
    ImageKey        string `gorm:"column:image_key"`
    CreatedAt      time.Time `gorm:"column:created_at"`
}



type PatientImage struct {
    Measurement_id string  `gorm:"column:measurement_id"`
    PatientID      string  `gorm:"column:patient_id"`
    ImagePath      *string `json:"url" gorm:"column:image_path;default:null"` // Pointer to allow NULL
    Images         string `gorm:"type:LONGTEXT;column:images;default:null"`  // Pointer to allow NULL
    ImageKey        string `gorm:"column:image_key"`
    IsDFA_Complete bool   `gorm:"column:is_dfa_complete;default:false"`       // Pointer to allow NULL
}

// type PatientImage struct {
//     Images []string `json:"images"`
// }

// Function to create a new instance of PatientImages
func NewPatientImages(Measurement_id string,patientID string, images []string) *PatientImages {
	// Serialize the images slice to a comma-separated string before storing in the database
	imageStr := strings.Join(images, ",")
    now := time.Now()
	return &PatientImages{
        Measurement_id: Measurement_id,
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
