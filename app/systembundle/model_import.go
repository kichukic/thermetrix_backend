package systembundle

import "thermetrix_backend/app/core"

const (
	DataSheet_Practice = "Practice"
	DataSheet_Patients = "Patients"
	DataSheet_Settings = "Settings"
)

type ImportError struct {
	core.Model
	DataSheet    string `json:"data_sheet"`
	RowNumber    uint   `json:"row_number"`
	ErrorMessage string `json:error_message`
}

type ImportErrors []ImportError
