package tableconfig

import "thermetrix_backend/app/core"

type TableConfigUserSetting struct {
	core.Model
	UserId                       uint   `json:"-"`
	TableConfigTypeName          string `json:"table_config_type_name"`
	TableHeaderDisplayConfigData string `json:"table_header_display_config_data" gorm:"type:LONGTEXT"`
}
type TableConfigUserSettings []TableConfigUserSetting


// type PatientImages struct {
// 	ID         uint   // Primary key ID field
// 	PatientID  string // Patient ID field
// 	ImagePath  string // File path of the image

// }