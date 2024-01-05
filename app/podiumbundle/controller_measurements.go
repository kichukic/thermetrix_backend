package podiumbundle

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/jung-kurt/gofpdf"
//	"gotools/tools"
	tools "github.com/kirillDanshin/nulltime"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"thermetrix_backend/app/core"
	"thermetrix_backend/app/tableconfig"
	"thermetrix_backend/app/websocket"
	"time"
)

func (c *PodiumController) GetMeasurementsTableConfigHandler(w http.ResponseWriter, r *http.Request) {
	if tableConfig, err := tableconfig.GetTableConfig(c.ormDB, r, "measurements"); err != nil || tableConfig == nil {
		c.HandleError(err, w)
	} else {
		c.SendJSON(w, &tableConfig, http.StatusOK)
	}
}

// getMeasurements swagger:route GET /measurements measurements getMeasurements
//
// retrieves measurements
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	paging := c.GetPaging(r.URL.Query())
	db, dbTotalCount := c.CreateWhereConditionsMeasurements(r.URL.Query(), r, user)

	measurements := Measurements{}

	offset := paging.Offset
	//JumpTo
	urlQuery := r.URL.Query()
	if len(urlQuery) > 0 {
		dbJumpTo := db
		values := urlQuery
		measurementId := 0
		if val, ok := values["jump_to"]; ok && len(val) > 0 {
			measurementId, _ = strconv.Atoi(val[0])
		}

		if measurementId > 0 {
			currentMeasurement := Measurement{}
			c.ormDB.
				Set("gorm:auto_preload", false).
				Preload("Patient").
				Preload("Practice").
				Preload("Practice.User").
				Preload("Doctor").
				Preload("Doctor.Users").
				Preload("Doctor.Users.User").
				First(&currentMeasurement, measurementId)

			doctor := Doctor{}
			c.ormDB.
				Set("gorm:auto_preload", true).
				First(&doctor, currentMeasurement.DoctorId)
				//.Preload("Doctor.Users")
				//.Preload("Doctor.Users.User")

			currentMeasurement.Doctor = &doctor
			values := r.URL.Query()
			if currentMeasurement.Doctor == nil {
				currentMeasurement.Doctor = &Doctor{}
				currentMeasurement.Doctor.Users = []DoctorUser{{User: core.User{}}}
			}

			if c.isDoctor(user) {
				doctor := &Doctor{}
				doctor = c.getDoctor(user)

				if doctor.ID > 0 {
					tmp := MeasurementDoctorRisk{}
					c.ormDB.
						Set("gorm:auto_preload", true).
						Where("measurement_id = ?", currentMeasurement.ID).
						Where("doctor_id = ?", doctor.ID).First(&tmp)

					currentMeasurement.MeasurementRisk = &tmp
				}

				measurementFavorite := MeasurementFavorite{}
				c.ormDB.Set("gorm:auto_preload", false).
					Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).
					First(&measurementFavorite)

				if measurementFavorite.ID > 0 {
					currentMeasurement.IsFavorite = true
				}
			}

			if c.isPractice(user) {
				measurementFavorite := MeasurementFavorite{}
				c.ormDB.Set("gorm:auto_preload", false).
					Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).
					First(&measurementFavorite)
				if measurementFavorite.ID > 0 {
					currentMeasurement.IsFavorite = true
				}
			}

			if val, ok := values["order"]; ok && len(val) > 0 {
				if val[0] != "" {
					if strings.Contains(val[0], ",") {
						sortSplit := strings.Split(val[0], ",")
						sortKey := sortSplit[0]
						sortDirection := sortSplit[1]
						switch sortKey {
						case "practice":

							break
						case "doctor":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("username > (SELECT username FROM system_accounts WHERE (SELECT user_id FROM doctor_users WHERE system_accounts.id = user_id  AND doctor_id = (SELECT doctor_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) OR username = (SELECT username FROM system_accounts WHERE (SELECT user_id FROM doctor_users WHERE system_accounts.id = user_id  AND doctor_id = (SELECT doctor_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("(username < ?) OR (isNUll(username) AND measurements.id <= ? OR username = ? AND measurements.id <= ?)", currentMeasurement.Doctor.Users[0].User.Username, currentMeasurement.ID, currentMeasurement.Doctor.Users[0].User.Username, currentMeasurement.ID)
							}
							break
						case "patient":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("username > (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) OR username = (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("username < (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) OR username = (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							}
							break
						case "town":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("town > (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) OR town = (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("town < (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) OR town = (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							}
							break
						case "date":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date > (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date < (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							}

							break
						case "time":
							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE TIME(m.measurement_date) > (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) OR TIME(m.measurement_date) = (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE TIME(m.measurement_date) < (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) OR TIME(m.measurement_date) = (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							}
							break
						case "status":

						case "measurement_date":
							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date > (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date < (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							}
							break
						case "measurement_risk":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("risk_definitions.sort_value > ? OR (risk_definitions.sort_value = ? AND measurements.id <= ? OR ISNULL(risk_definitions.sort_value) AND measurements.id <= ? )", currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("risk_definitions.sort_value_asc > ? OR (risk_definitions.sort_value_asc = ? AND measurements.id <= ? OR ? = 0 AND ISNULL(risk_definitions.sort_value_asc) AND measurements.id <= ?)", currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, measurementId, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, measurementId)
							}
							break
						case "is_favorite":
							numberBool := 0
							if currentMeasurement.IsFavorite {
								numberBool = 1
							}

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where(" ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) > ?) OR ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) = ? AND measurements.id <= ?)", user.ID, numberBool, user.ID, numberBool, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where(" ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) < ?) OR ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) = ? AND measurements.id <= ?)", user.ID, numberBool, user.ID, numberBool, measurementId)
							}
							break
						default:
							break
						}
					}
				}
			}

			positionInList := 0
			//			dbJumpTo.Debug().Model(&Measurements{}).Count(&positionInList)
			tmpMeasurements := Measurements{}
			dbJumpTo.Debug().
				Set("gorm:auto_preload", false).
				Model(&Measurements{}).
				Order("measurements.id").
				Where("ISNULL(measurements.deleted_at)").
				Find(&tmpMeasurements)

			positionInList = len(tmpMeasurements) - 1
			if positionInList < 0 {
				positionInList = 0
			}
			if paging.Limit > 0 {
				pageOfElement := positionInList / paging.Limit
				offset = pageOfElement * paging.Limit
				paging.Page = pageOfElement
			}
		}
	}

	if user.IsSysadmin {
		db = db.
			Preload("Device").
			Preload("Doctor").
			Preload("Doctor.Users").
			Preload("Doctor.Users.User")

		db.Debug().
			Set("gorm:auto_preload", false).
			Preload("Patient").
			Preload("Patient.User").
			Preload("MeasurementFiles").
			Limit(paging.Limit).
			Offset(offset).
			Order("measurements.id").
			Where("ISNULL(measurements.deleted_at)").
			Find(&measurements)

		dbTotalCount.
			Model(&Measurements{}).
			Where("ISNULL(measurements.deleted_at)").
			Count(&paging.TotalCount)
	} else {
		db.Debug().
			Set("gorm:auto_preload", true).
			Limit(paging.Limit).
			Offset(offset).
			Preload("Device").
			Preload("MeasurementFiles").
			Order("measurements.id").
			Where("isNUll(measurements.deleted_at)").
			Find(&measurements)

		dbTotalCount.
			Model(&Measurements{}).
			Where("ISNULL(measurements.deleted_at)").
			Count(&paging.TotalCount)
	}

	if paging.PerPage > 0 {
		paging.TotalPage = paging.TotalCount / paging.PerPage
		if paging.TotalPage <= 0 {
			paging.TotalPage = 1
		}
	}
	doctorId := uint(0)
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		doctorId = doctor.ID
	}

	if doctorId > 0 {
		for key, measurement := range measurements {
			tmp := MeasurementDoctorRisk{}
			c.ormDB.Debug().
				Set("gorm:auto_preload", true).
				Where("measurement_id = ?", measurement.ID).
				Where("doctor_id = ?", doctorId).
				First(&tmp)

			measurement.MeasurementRisk = &tmp
			measurements[key] = measurement
		}
	}

	if user.IsSysadmin {
		for key, measurement := range measurements {
			//c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)
			if measurement.DoctorId > 0 {
				practice := c.getPracticeOfDoctor(measurement.DoctorId)
				measurement.Practice = &practice
				measurements[key] = measurement
			}
		}
	}

	c.SendJSONPaging(w, r, paging, &measurements, http.StatusOK)
}

func (c *PodiumController) GetMeasurementsCsvFileHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	paging := c.GetPaging(r.URL.Query())
	db, dbTotalCount := c.CreateWhereConditionsMeasurements(r.URL.Query(), r, user)

	measurements := Measurements{}

	//offset := paging.Offset
	//JumpTo
	urlQuery := r.URL.Query()
	if len(urlQuery) > 0 {
		dbJumpTo := db
		values := urlQuery
		measurementId := 0
		if val, ok := values["jump_to"]; ok && len(val) > 0 {
			measurementId, _ = strconv.Atoi(val[0])
		}

		if measurementId > 0 {
			currentMeasurement := Measurement{}
			c.ormDB.Set("gorm:auto_preload", false).Preload("Patient").Preload("Practice").Preload("Practice.User").Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User").First(&currentMeasurement, measurementId)

			doctor := Doctor{}
			c.ormDB.Set("gorm:auto_preload", true).First(&doctor, currentMeasurement.DoctorId) //.Preload("Doctor.Users").Preload("Doctor.Users.User")
			currentMeasurement.Doctor = &doctor
			values := r.URL.Query()
			if currentMeasurement.Doctor == nil {
				currentMeasurement.Doctor = &Doctor{}
				currentMeasurement.Doctor.Users = []DoctorUser{{User: core.User{}}}
			}

			if c.isDoctor(user) {
				doctor := &Doctor{}
				doctor = c.getDoctor(user)

				if doctor.ID > 0 {
					tmp := MeasurementDoctorRisk{}
					c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = ?", currentMeasurement.ID).Where("doctor_id = ?", doctor.ID).First(&tmp)
					currentMeasurement.MeasurementRisk = &tmp
				}

				measurementFavorite := MeasurementFavorite{}
				c.ormDB.Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).First(&measurementFavorite)
				if measurementFavorite.ID > 0 {
					currentMeasurement.IsFavorite = true
				}
			}

			if c.isPractice(user) {
				measurementFavorite := MeasurementFavorite{}
				c.ormDB.Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).First(&measurementFavorite)
				if measurementFavorite.ID > 0 {
					currentMeasurement.IsFavorite = true
				}
			}

			if val, ok := values["order"]; ok && len(val) > 0 {
				if val[0] != "" {
					if strings.Contains(val[0], ",") {
						sortSplit := strings.Split(val[0], ",")
						sortKey := sortSplit[0]
						sortDirection := sortSplit[1]
						switch sortKey {
						case "practice":

							break
						case "doctor", "doctors":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("username > (SELECT username FROM system_accounts WHERE (SELECT user_id FROM doctor_users WHERE system_accounts.id = user_id  AND doctor_id = (SELECT doctor_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) OR username = (SELECT username FROM system_accounts WHERE (SELECT user_id FROM doctor_users WHERE system_accounts.id = user_id  AND doctor_id = (SELECT doctor_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("(username < ?) OR (isNUll(username) AND measurements.id <= ? OR username = ? AND measurements.id <= ?)", currentMeasurement.Doctor.Users[0].User.Username, currentMeasurement.ID, currentMeasurement.Doctor.Users[0].User.Username, currentMeasurement.ID)
							}
							break
						case "patient":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("username > (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) OR username = (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("username < (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) OR username = (SELECT username FROM system_accounts WHERE (SELECT user_id FROM patients WHERE system_accounts.id = user_id  AND id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							}
							break
						case "town":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("town > (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) OR town = (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("town < (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) OR town = (SELECT town FROM patients WHERE id = (SELECT patient_id FROM measurements WHERE id = ? Limit 1) Limit 1) AND measurements.id <= ?", measurementId, measurementId, measurementId)
							}
							break
						case "date":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date > (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date < (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							}

							break
						case "time":
							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE TIME(m.measurement_date) > (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) OR TIME(m.measurement_date) = (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE TIME(m.measurement_date) < (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) OR TIME(m.measurement_date) = (SELECT TIME(measurement_date) FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							}
							break
						case "status":

						case "measurement_date":
							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date > (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("id IN (SELECT m.id FROM measurements as m WHERE m.measurement_date < (SELECT measurement_date FROM measurements WHERE id = ?) OR m.measurement_date = (SELECT measurement_date FROM measurements WHERE id = ?) AND m.id <= ?)", measurementId, measurementId, measurementId)
							}
							break
						case "measurement_risk":

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where("risk_definitions.sort_value > ? OR (risk_definitions.sort_value = ? AND measurements.id <= ? OR ISNULL(risk_definitions.sort_value) AND measurements.id <= ? )", currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, measurementId, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where("risk_definitions.sort_value_asc > ? OR (risk_definitions.sort_value_asc = ? AND measurements.id <= ? OR ? = 0 AND ISNULL(risk_definitions.sort_value_asc) AND measurements.id <= ?)", currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, measurementId, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, measurementId)
							}
							break
						case "is_favorite":
							numberBool := 0
							if currentMeasurement.IsFavorite {
								numberBool = 1
							}

							if "desc" == sortDirection {
								dbJumpTo = dbJumpTo.Where(" ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) > ?) OR ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) = ? AND measurements.id <= ?)", user.ID, numberBool, user.ID, numberBool, measurementId)
							} else {
								dbJumpTo = dbJumpTo.Where(" ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) < ?) OR ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) = ? AND measurements.id <= ?)", user.ID, numberBool, user.ID, numberBool, measurementId)
							}
							break
						default:
							break
						}
					}
				}
			}

			positionInList := 0
			//			dbJumpTo.Debug().Model(&Measurements{}).Count(&positionInList)
			tmpMeasurements := Measurements{}
			dbJumpTo.Debug().Set("gorm:auto_preload", false).Model(&Measurements{}).Order("measurements.id").Where("ISNULL(measurements.deleted_at)").Find(&tmpMeasurements)

			positionInList = len(tmpMeasurements) - 1
			if positionInList < 0 {
				positionInList = 0
			}
			if paging.Limit > 0 {
				pageOfElement := positionInList / paging.Limit
				//offset = pageOfElement * paging.Limit
				paging.Page = pageOfElement
			}
		}
	}

	if user.IsSysadmin {
		db = db.Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User")
		db.Set("gorm:auto_preload", false).Preload("Patient").Preload("Device").Preload("Patient.User").Order("measurements.id").Where("ISNULL(measurements.deleted_at)").Find(&measurements)
		dbTotalCount.Model(&Measurements{}).Where("ISNULL(measurements.deleted_at)").Count(&paging.TotalCount)
	} else {
		db.Set("gorm:auto_preload", true).Preload("Device").Order("measurements.id").Where("isNUll(measurements.deleted_at)").Find(&measurements)
		dbTotalCount.Model(&Measurements{}).Where("ISNULL(measurements.deleted_at)").Count(&paging.TotalCount)
	}

	if paging.PerPage > 0 {
		paging.TotalPage = paging.TotalCount / paging.PerPage
		if paging.TotalPage <= 0 {
			paging.TotalPage = 1
		}
	}
	doctorId := uint(0)
	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		doctorId = doctor.ID
	}

	if doctorId > 0 {
		for key, measurement := range measurements {
			tmp := MeasurementDoctorRisk{}
			c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = ?", measurement.ID).Where("doctor_id = ?", doctorId).First(&tmp)
			measurement.MeasurementRisk = &tmp
			measurements[key] = measurement
		}
	}

	if user.IsSysadmin {
		for key, measurement := range measurements {
			//c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)
			if measurement.DoctorId > 0 {
				practice := c.getPracticeOfDoctor(measurement.DoctorId)
				measurement.Practice = &practice
				measurements[key] = measurement
			}
		}
	}

	/*
	   workingFolder := core.RandomString(32)
	   	X tmpPath := "tmp/" + workingFolder

	   	os.MkdirAll(tmpPath, 0777)
	*/
	tmpPath := c.GetTmpUploadPathWithRandomStringCount(32)

	tmpFileName := fmt.Sprintf("%s.csv", "measurements")
	tmpFullFilePath := filepath.Join(tmpPath, tmpFileName)
	f, err := os.Create(tmpFullFilePath)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)

	serverName := core.Config.Server.CustomName

	var headerRow [][]string
	headerRow = append(headerRow, []string{
		"Server name",
		"Scan id",
		"Practice username",
		"Device serial number",
		"Clinician username",
		"Patient username",
		"Measurement date",
		"Measurement time",
		"Device firmware version",
		"App version",
		"App system version",
	})

	writer.WriteAll(headerRow)

	timeOffset := int(0)
	hTimeOffset := r.Header["X-Timezone-Offset"]
	timeZone := r.Header["X-Timezone"]
	_ = timeZone
	if len(hTimeOffset) > 0 {
		tmp, err := strconv.Atoi(hTimeOffset[0])
		if err == nil {
			timeOffset = tmp
		}
	}
	_ = timeOffset
	var loc *time.Location
	if timeZone != nil {
		loc, _ = time.LoadLocation(timeZone[0])
	}

	//if err == nil {

	var dataToWrite [][]string
	for _, measurement := range measurements {

		patientName := ""
		patientName = measurement.Patient.FirstName

		if len(patientName) > 0 {
			patientName += ", "
		}

		patientName += measurement.Patient.LastName

		patientName = measurement.Patient.User.Username

		device := Device{}
		doctor := Doctor{}
		practice := Practice{}

		if measurement.Device != nil {
			device = *measurement.Device
		}
		if measurement.Doctor != nil {
			doctor = *measurement.Doctor
		}
		if measurement.Practice != nil {
			practice = *measurement.Practice
		}

		doctorName := ""
		if doctor.Name == "" {
			doctorName = doctor.FirstName

			if len(doctorName) > 0 {
				doctorName += ", "
			}

			doctorName += doctor.LastName
		} else {
			doctorName = doctor.Name
		}

		if len(doctor.Users) > 0 {
			doctorName = doctor.Users[0].User.Username
		}

		practiceName := ""
		practiceName = practice.Name
		practiceName = ""

		if len(practiceName) == 0 {
			practiceName = practice.User.Username
		}

		measurementDate := measurement.MeasurementDate.Time
		if loc != nil {
			measurementDate = measurement.MeasurementDate.Time.In(loc)
		}

		row := []string{
			serverName,
			strconv.Itoa(int(measurement.ID)),
			practiceName,
			device.DeviceSerial,
			doctorName,
			patientName,
			measurementDate.Format("02.01.2006"),
			measurementDate.Format(time.Kitchen),
			measurement.DeviceVersion,
			measurement.AppVersion,
			measurement.AppSystemVersion,
		}
		dataToWrite = append(dataToWrite, row)
	}

	writer.WriteAll(dataToWrite)
	writer.Flush()

	c.SendFile(w, r, filepath.Join(tmpPath, tmpFileName))
}

func (c *PodiumController) GetMeasurementPreviousHandler(w http.ResponseWriter, r *http.Request) {
	c.GetMeasurementNavigation(w, r, "previous")
}

func (c *PodiumController) GetMeasurementNextHandler(w http.ResponseWriter, r *http.Request) {
	c.GetMeasurementNavigation(w, r, "next")
}

func (c *PodiumController) GetMeasurementImagesPreviousHandler(w http.ResponseWriter, r *http.Request) {
	c.GetMeasurementImagesNavigation(w, r, "previous")
}

func (c *PodiumController) GetMeasurementImagesNextHandler(w http.ResponseWriter, r *http.Request) {
	c.GetMeasurementImagesNavigation(w, r, "next")
}

// getMeasurements swagger:route GET /measurements measurements getMeasurements
//
// retrieves measurements
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementImagesNavigation(w http.ResponseWriter, r *http.Request, navigationType string) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	currentMeasurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	//	navigationType string

	currentMeasurement := Measurement{}
	//db, _ := c.CreateWhereConditionsMeasurements(r.URL.Query(), r, user)
	//dbTotalCount.Model(&Measurements{}).Count(&paging.TotalCount)
	navigateToMeasurement := Measurement{}
	c.ormDB.Debug().Set("gorm:auto_preload", false).Preload("Patient").Preload("Patient.User").Preload("Practice").Preload("Practice.User").Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User").First(&currentMeasurement, currentMeasurementId)

	doctor := &Doctor{}
	if c.isDoctor(user) {
		doctor = c.getDoctor(user)
		if doctor.ID > 0 {
			tmp := MeasurementDoctorRisk{}
			c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = ?", currentMeasurementId).Where("doctor_id = ?", doctor.ID).First(&tmp)
			currentMeasurement.MeasurementRisk = &tmp
		}

		measurementFavorite := MeasurementFavorite{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).First(&measurementFavorite)
		if measurementFavorite.ID > 0 {
			currentMeasurement.IsFavorite = true
		}
	}

	if c.isPractice(user) {
		measurementFavorite := MeasurementFavorite{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).First(&measurementFavorite)
		if measurementFavorite.ID > 0 {
			currentMeasurement.IsFavorite = true
		}
	}

	values := r.URL.Query()
	if currentMeasurement.Doctor == nil {
		currentMeasurement.Doctor = &Doctor{}
		currentMeasurement.Doctor.Users = []DoctorUser{{User: core.User{}}}
	}
	db := c.CreateWhereMeasurementsForNavigation(values, r, currentMeasurement, navigationType, user)
	db.Set("gorm:auto_preload", true).Debug().Where("measurements.deleted_at IS NULL").First(&navigateToMeasurement) //.Preload("Practice").Preload("Practice.User").Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User")

	if navigateToMeasurement.ID == 0 {
		err := errors.New("No more Measurements")
		c.HandleError(err, w)
		return
	}

	measurementLightBox := MeasurementLightbox{}
	measurementLightBox.ID = navigateToMeasurement.ID

	timeOffset := int(0)
	hTimeOffset := r.Header["X-Timezone-Offset"]
	timeZone := r.Header["X-Timezone"]
	_ = timeZone
	if len(hTimeOffset) > 0 {
		tmp, err := strconv.Atoi(hTimeOffset[0])
		if err == nil {
			timeOffset = tmp
		}
	}
	_ = timeOffset
	loc, _ := time.LoadLocation(timeZone[0])
	//if err == nil {
	tmpTime := navigateToMeasurement.MeasurementDate.Time.In(loc)
	//}
	//return t, err

	//loc := time.FixedZone(timeZone[0], timeOffset)
	//nativeDate := time.Date(navigateToMeasurement.MeasurementDate.Time.Year(), navigateToMeasurement.MeasurementDate.Time.Month(), navigateToMeasurement.MeasurementDate.Time.Day(), navigateToMeasurement.MeasurementDate.Time.Hour(), navigateToMeasurement.MeasurementDate.Time.Minute(),0, 0, loc)
	//tmpTime := navigateToMeasurement.MeasurementDate.Time.Add(-1 * time.Duration(timeOffset) * time.Minute)
	measurementLightBox.Title = tmpTime.Format("02/01/2006") + " @ " + tmpTime.Format("15:04") //fmt.Sprintf("%v/%v/%v @ %v:%v", tmpTime.Day(), int(tmpTime.Month()), tmpTime.Year(), tmpTime.Hour(),tmpTime.Minute()) //

	image := MeasurementLightboxImage{}
	measurementLightBox.Images = append(measurementLightBox.Images, image)
	measurementLightBox.Images = append(measurementLightBox.Images, image)
	measurementLightBox.Images = append(measurementLightBox.Images, image)
	measurementLightBox.Images = append(measurementLightBox.Images, image)

	for _, file := range navigateToMeasurement.MeasurementFiles {
		url := fmt.Sprintf("/measurements/%v/files/%s", navigateToMeasurement.ID, file.MeasurementType)
		if file.MeasurementType == "THERMAL" {
			measurementLightBox.Images[0].DownloadUrl = url

		} else if file.MeasurementType == "DYNAMIC" {
			measurementLightBox.Images[1].DownloadUrl = url

		} else if file.MeasurementType == "STATISTIC" {
			measurementLightBox.Images[2].DownloadUrl = url

		} else if file.MeasurementType == "DYNAMIC_STATISTIC" {
			measurementLightBox.Images[3].DownloadUrl = url

		} else if file.MeasurementType == "NORMAL" {
			tmpImage := MeasurementLightboxImage{}
			tmpImage.DownloadUrl = url
			measurementLightBox.Images = append(measurementLightBox.Images, tmpImage)
		} else {

		}
	}
	for i := len(measurementLightBox.Images) - 1; i >= 0; i-- {
		if measurementLightBox.Images[i].DownloadUrl == "" {
			measurementLightBox.Images = append(measurementLightBox.Images[:i], measurementLightBox.Images[i+1:]...)
		}
	}

	c.SendJSON(w, &measurementLightBox, http.StatusOK)

}

// getMeasurements swagger:route GET /measurements measurements getMeasurements
//
// retrieves measurements
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementNavigation(w http.ResponseWriter, r *http.Request, navigationType string) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	currentMeasurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	//	navigationType string

	currentMeasurement := Measurement{}
	//db, _ := c.CreateWhereConditionsMeasurements(r.URL.Query(), r, user)
	//dbTotalCount.Model(&Measurements{}).Count(&paging.TotalCount)
	navigateToMeasurement := Measurement{}
	c.ormDB.Debug().Set("gorm:auto_preload", false).Preload("Patient").Preload("Patient.User").Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User").First(&currentMeasurement, currentMeasurementId)

	doctor := &Doctor{}
	if c.isDoctor(user) {
		doctor = c.getDoctor(user)
		if doctor.ID > 0 {
			tmp := MeasurementDoctorRisk{}
			c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = ?", currentMeasurementId).Where("doctor_id = ?", doctor.ID).First(&tmp)
			currentMeasurement.MeasurementRisk = &tmp
		}

		measurementFavorite := MeasurementFavorite{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).First(&measurementFavorite)
		if measurementFavorite.ID > 0 {
			currentMeasurement.IsFavorite = true
		}
	}

	if c.isPractice(user) {
		measurementFavorite := MeasurementFavorite{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", currentMeasurement.ID, user.ID).First(&measurementFavorite)
		if measurementFavorite.ID > 0 {
			currentMeasurement.IsFavorite = true
		}
	}

	values := r.URL.Query()
	if currentMeasurement.Doctor == nil {
		currentMeasurement.Doctor = &Doctor{}
		currentMeasurement.Doctor.Users = []DoctorUser{{User: core.User{}}}
	}
	db := c.CreateWhereMeasurementsForNavigation(values, r, currentMeasurement, navigationType, user)
	db.Set("gorm:auto_preload", true).Debug().Where("measurements.deleted_at IS NULL").First(&navigateToMeasurement) //.Preload("Practice").Preload("Practice.User").Preload("Doctor").Preload("Doctor.Users").Preload("Doctor.Users.User")

	if navigateToMeasurement.ID == 0 {
		err := errors.New("No more Measurements")
		c.HandleError(err, w)
		return
	}

	if doctor.ID > 0 {

		tmp := MeasurementDoctorRisk{}
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = ?", navigateToMeasurement.ID).Where("doctor_id = ?", doctor.ID).First(&tmp)
		navigateToMeasurement.MeasurementRisk = &tmp

		measurementFavorite := MeasurementFavorite{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", navigateToMeasurement.ID, user.ID).First(&measurementFavorite)
		if measurementFavorite.ID > 0 {
			navigateToMeasurement.IsFavorite = true
		}
	}
	if c.isPractice(user) {
		measurementFavorite := MeasurementFavorite{}
		c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND user_id = ?", navigateToMeasurement.ID, user.ID).First(&measurementFavorite)
		if measurementFavorite.ID > 0 {
			navigateToMeasurement.IsFavorite = true
		}
	}

	c.SendJSON(w, &navigateToMeasurement, http.StatusOK)

}

func (c *PodiumController) CreateWhereMeasurementsForNavigation(urlQuery url.Values, r *http.Request, currentMeasurement Measurement, navigation string, user *core.User) *gorm.DB {

	db := c.ormDB.Debug()
	//dbTotalCount := c.ormDB.Debug()
	//next, previous

	if c.isPatient(user) {
		patient := c.getPatient(user)
		db = db.Where("measurements.patient_id = ?", patient.ID)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		docPractice := doctor.GetPractice(c.ormDB)
		//dbTotalCount = dbTotalCount.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2)", doctor.ID).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID)

		db = db.Unscoped().Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", docPractice.ID)
		db = db.Select("*, (measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?)) as is_favorite", user.ID)
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		db = db.Unscoped().Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", practice.ID)

		db = db.Select("*, (measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?)) as is_favorite", user.ID)
	}

	if len(urlQuery) > 0 {
		values := urlQuery

		//navigation := "next"

		//db = db.Debug().Where("contacts_contacts.last_name + contacts_contacts.first_name + contacts_contacts.org_name > (Select sub.last_name+sub.first_name+sub.org_name FROM contacts_contacts AS sub WHERE sub.id == ?)", currentContactId)
		/*if val, ok := values["navigation"]; ok && len(val) > 0 {
			if val[0] != "" {
				navigation = val[0]
			}
		}*/

		/*if val, ok := values["types"]; ok && len(val) > 0 {
			if val[0] != "" {
				//types := strings.Split(strings.ToUpper(val[0]), ",")
				types := strings.Split(strings.ToUpper(val[0]), ",")
				db = db.Where("contacts_contacts.id IN (SELECT contact_id FROM contacts_contact_types WHERE is_active = 1 AND contact_type_def_id IN (SELECT id FROM contacts_contact_type_defs WHERE deleted_at IS NULL AND (id IN (?) OR contact_type_name IN (?))))", types, types)
				//	dbTotalCount = dbTotalCount.Where("id IN (SELECT contact_id FROM contacts_contact_types WHERE is_active = 1 AND contact_type_def_id IN (SELECT id FROM contacts_contact_type_defs WHERE deleted_at IS NULL AND (id IN (?) OR contact_type_name IN (?))))", types, types)
			}
		}
		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"
				db = db.Where("org_name LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR birthname LIKE ? OR name_affix LIKE ? OR suffix LIKE ? OR debitor_number LIKE ? OR id IN (SELECT contact_id FROM contacts_contact_types WHERE is_active = 1 AND contact_type_number LIKE ?)", search, search, search, search, search, search, search, search)
				//	dbTotalCount = dbTotalCount.Where("org_name LIKE ? OR first_name LIKE ? OR last_name LIKE ? OR birthname LIKE ? OR name_affix LIKE ? OR suffix LIKE ? OR debitor_number LIKE ? OR id IN (SELECT contact_id FROM contacts_contact_types WHERE is_active = 1 AND contact_type_number LIKE ?)", search, search, search, search, search, search, search, search)
			}
		}*/

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"

				hTime := r.Header["X-Timezone"]
				h, err := time.Parse("15:04", val[0])
				h = h.AddDate(2020, 1, 1)

				queryHasTimeOrDate := false
				if err == nil { //&&!fullDate
					queryHasTimeOrDate = true
					db = db.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR hour(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) = hour(?) AND minute(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) = minute(?))", hTime, search, search, search, search, hTime, h, hTime, h)
				} else {
					var date core.NullTime
					tmp := val[0]
					date.FromString(tmp)

					if len(tmp) >= 4 {
						if date.Valid {
							queryHasTimeOrDate = true
							db = db.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR date(measurements.measurement_date) = date(?) OR time(measurements.measurement_date) = time(?))", hTime, search, search, search, search, date, date)

						}
					}
				}
				if !queryHasTimeOrDate {
					db = db.Where("YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR	HOUR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MINUTE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?))", hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, search, search, search)
				}
			}
		}

		if val, ok := values["last_refresh"]; ok && len(val) > 0 {
			if val[0] != "" {
				last_refresh := val[0]
				db = db.Where("updated_at > ?", last_refresh)
			}
		}

		if val, ok := values["measurement_date_from"]; ok && len(val) > 0 {
			if val[0] != "" {
				measurement_date_from := val[0]
				db = db.Where("measurement_date >= ?", measurement_date_from)
			}
		}

		if val, ok := values["order"]; ok && len(val) > 0 {
			//db = db.Debug().Where("id > ?", currentContactId)
			if val[0] != "" {
				if strings.Contains(val[0], ",") {
					sortSplit := strings.Split(val[0], ",")
					sortKey := sortSplit[0]
					sortDirection := sortSplit[1]

					largerSmall := ""
					secondOrderById := ""
					if strings.ToLower(navigation) == "previous" {
						//Previous
						if strings.ToLower(sortDirection) == "desc" {
							largerSmall = ">"
							secondOrderById = "<"
						} else {
							//ASC
							sortDirection = "desc"
							largerSmall = "<"
							secondOrderById = "<"
						}
					} else {
						//Next
						if strings.ToLower(sortDirection) == "desc" {
							largerSmall = "<"
							secondOrderById = ">"
						} else {
							largerSmall = ">"
							secondOrderById = ">"
						}
					}

					switch sortKey {

					case "practice":
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
							}
						}
						db = db.Joins("LEFT JOIN doctors ON measurements.doctor_id = doctors.id").Joins("LEFT JOIN practice_doctors ON practice_doctors.doctor_id = doctors.id").Joins("LEFT JOIN practices ON practices.id = practice_doctors.practice_id").Joins("Left JOIN system_accounts ON practices.user_id = system_accounts.id")
						where := fmt.Sprintf("(username = ? AND measurements.id %s ?) OR (username %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.Practice.User.Username, currentMeasurement.ID, currentMeasurement.Practice.User.Username)
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						break
					case "doctor":
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
							}
						}
						db = db.Joins("LEFT JOIN doctor_users ON measurements.doctor_id = doctor_users.doctor_id").Joins("Left JOIN system_accounts ON doctor_users.user_id = system_accounts.id")
						where := fmt.Sprintf("(username = ? AND measurements.id %s ?) OR (username %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.Doctor.Users[0].User.Username, currentMeasurement.ID, currentMeasurement.Doctor.Users[0].User.Username)
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						break
					case "patient":
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
							}
						}

						db = db.Joins("LEFT JOIN patients ON measurements.patient_id = patients.id").Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						where := fmt.Sprintf("(username = ? AND measurements.id %s ?) OR (username %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.Patient.User.Username, currentMeasurement.ID, currentMeasurement.Patient.User.Username)
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						break
					case "town":
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
							}
						}

						db = db.Joins("LEFT JOIN patients ON measurements.patient_id = patients.id")
						where := fmt.Sprintf("(town = ? AND measurements.id %s ?) OR (town %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.Patient.Town, currentMeasurement.ID, currentMeasurement.Patient.Town)
						db = db.Order(fmt.Sprintf("town %s", sortDirection))
						break
					case "date":
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
								largerSmall = ">"
								secondOrderById = ">"
							}
						} else {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "desc"
								largerSmall = "<"
								secondOrderById = "<"
							}
						}
						where := fmt.Sprintf("(measurement_date = ? AND measurements.id %s ?) OR (measurement_date %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.MeasurementDate, currentMeasurement.ID, currentMeasurement.MeasurementDate)
						db = db.Order(fmt.Sprintf("measurement_date %s", sortDirection))
						break
					case "time":
						if strings.ToLower(navigation) == "next" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								largerSmall = "<"
								secondOrderById = "<"
							}
						}
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
							}
						}
						where := fmt.Sprintf("(Time(measurement_date) = TIME(?) AND measurements.id %s ?) OR (Time(measurement_date) %s Time(?))", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.MeasurementDate, currentMeasurement.ID, currentMeasurement.MeasurementDate)
						db = db.Order(fmt.Sprintf("TIME(measurement_date) %s", sortDirection))
						break
					case "measurement_date":
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
								largerSmall = ">"
								secondOrderById = ">"
							}
						} else {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "desc"
								largerSmall = "<"
								secondOrderById = "<"
							}
						}
						where := fmt.Sprintf("(measurement_date = ? AND measurements.id %s ?) OR (measurement_date %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, currentMeasurement.MeasurementDate, currentMeasurement.ID, currentMeasurement.MeasurementDate)
						db = db.Order(fmt.Sprintf("measurement_date %s", sortDirection))
						break
					case "measurement_risk":
						//db := db = db.Where("measurement_doctor_risks.doctor_id = ?", doctor.ID)
						db = db.Joins("LEFT JOIN measurement_doctor_risks ON measurements.id = measurement_doctor_risks.measurement_id").Joins("LEFT JOIN risk_definitions ON measurement_doctor_risks.risk_definition_id = risk_definitions.id")
						/*

							sortDirection = "desc"
									largerSmall = "<"
									secondOrderById = ">"
						*/

						/*if strings.ToLower(navigation) == "previous" {
							if sortSplit[1] == "asc" {

							}else{

							}
						}else{
							if sortSplit[1] == "asc" {

							} else {

							}
						}*/
						if sortSplit[1] == "asc" {

							if strings.ToLower(navigation) == "next" {

								if currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc == 0 {
									largerSmall = "<"
									sortDirection = "DESC"
									where := fmt.Sprintf("(ISNULL(risk_definitions.sort_value_asc) AND measurements.id %s ?)", secondOrderById) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.ID)
									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value_asc %s", sortDirection))
								} else {
									largerSmall = "<"
									sortDirection = "DESC"
									where := fmt.Sprintf("(risk_definitions.sort_value_asc = ? AND measurements.id %s ?) OR (risk_definitions.sort_value_asc %s ?) OR (? = 1 AND ISNULL(risk_definitions.sort_value_asc))", secondOrderById, largerSmall) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc)
									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value_asc %s", sortDirection))
								}
							} else {
								if currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc == 0 {
									sortDirection = "ASC"
									largerSmall = ">"
									secondOrderById = "<"
									where := fmt.Sprintf("(ISNULL(risk_definitions.sort_value_asc) AND measurements.id %s ?) OR (risk_definitions.sort_value_asc = 1)", secondOrderById) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.ID)
									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value_asc %s", sortDirection))
								} else {
									sortDirection = "ASC"
									largerSmall = ">"
									secondOrderById = "<"
									where := fmt.Sprintf("(risk_definitions.sort_value_asc = ? AND measurements.id %s ?) OR (risk_definitions.sort_value_asc %s ?) ", secondOrderById, largerSmall) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValueAsc)

									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value_asc %s", sortDirection))
								}
							}

							/*where := fmt.Sprintf("(risk_definitions.sort_value > 1 AND risk_definitions.sort_value %s ? AND measurements.id %s ?) OR (risk_definitions.sort_value = ? AND measurements.id %s ?) OR (risk_definitions.sort_value %s ?)", largerSmall, secondOrderById, secondOrderById, largerSmall)
							db = db.Debug().Where(where, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue) //, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue,  currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.ID)

							db = db.Debug().Order(fmt.Sprintf("case when risk_definitions.sort_value = 1 Then 99 when isNUll(risk_definitions.sort_value) Then 100 ELSE risk_definitions.sort_value END ASC"))*/

						} else {

							//NEXT
							if strings.ToLower(navigation) == "next" {
								if currentMeasurement.MeasurementRisk.RiskDefinition.SortValue == 0 {
									where := fmt.Sprintf("(ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)", secondOrderById) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.ID)
									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value %s", sortDirection))
								} else {
									where := fmt.Sprintf("(risk_definitions.sort_value = ? AND measurements.id %s ?) OR (risk_definitions.sort_value %s ?) OR (? = 1 AND ISNULL(risk_definitions.sort_value))", secondOrderById, largerSmall) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue)

									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value %s", sortDirection))
								}
							} else {
								//prev
								if currentMeasurement.MeasurementRisk.RiskDefinition.SortValue == 0 {
									sortDirection = "asc"
									where := fmt.Sprintf("(ISNULL(risk_definitions.sort_value) AND measurements.id %s ?) OR (risk_definitions.sort_value = 1)", secondOrderById) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.ID)
									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value %s", sortDirection))
								} else {
									sortDirection = "asc"
									where := fmt.Sprintf("(risk_definitions.sort_value = ? AND measurements.id %s ?) OR (risk_definitions.sort_value %s ?) ", secondOrderById, largerSmall) //OR (ISNULL(risk_definitions.sort_value) AND measurements.id %s ?)
									db = db.Debug().Where(where, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue, currentMeasurement.ID, currentMeasurement.MeasurementRisk.RiskDefinition.SortValue)

									db = db.Debug().Order(fmt.Sprintf("risk_definitions.sort_value %s", sortDirection))
								}
							}

						}

						break
					case "is_favorite":
						numberBool := 0
						if currentMeasurement.IsFavorite {
							numberBool = 1
						}
						if strings.ToLower(navigation) == "previous" {
							if strings.ToLower(sortSplit[1]) == "desc" {
								sortDirection = "asc"
							}
						}

						where := fmt.Sprintf("((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) = ? AND measurements.id %s ?) OR ((measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?) != 0) %s ?)", secondOrderById, largerSmall)
						db = db.Debug().Where(where, user.ID, numberBool, currentMeasurement.ID, user.ID, numberBool)
						db = db.Order(fmt.Sprintf("is_favorite %s", sortDirection))
						break
					default:
						db = db.Order(fmt.Sprintf("%s %s", sortKey, sortDirection))
						break
					}

					//Der Nächste soll immer an der ersten stller der Liste sein, deswegen machen wir bei der ZWEITE SORTIERUNG (ID) DESC
					if strings.ToLower(navigation) == "previous" {
						if (sortKey == "date" || sortKey == "measurement_date") && strings.ToLower(sortSplit[1]) == "desc" {
							db = db.Order(fmt.Sprintf("measurements.id %s", "asc"))
						} else {
							db = db.Order(fmt.Sprintf("measurements.id %s", "desc"))
						}
					} else {
						if (sortKey == "time") && strings.ToLower(sortSplit[1]) == "desc" {
							db = db.Order(fmt.Sprintf("measurements.id %s", "desc"))
						}
					}

				}

				//if strings.ToLower(navigation) == "previous" {
				//	db = db.Order(fmt.Sprintf("measurements.id %s", "desc"))
				//}

			}
		}
	}

	return db
}

func (c *PodiumController) CreateWhereConditionsMeasurements(urlQuery url.Values, r *http.Request, user *core.User) (*gorm.DB, *gorm.DB) {

	db := c.ormDB
	dbTotalCount := c.ormDB.Debug()
	if c.isPatient(user) {
		patient := c.getPatient(user)
		db = db.Where("measurements.patient_id = ?", patient.ID)
		dbTotalCount = dbTotalCount.Where("measurements.patient_id = ?", patient.ID)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		docPractice := doctor.GetPractice(c.ormDB)
		//db = db.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2)", doctor.ID).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID)
		//dbTotalCount = dbTotalCount.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2)", doctor.ID).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID)

		db = db.Unscoped().Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", docPractice.ID)
		dbTotalCount = dbTotalCount.Unscoped().Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", docPractice.ID)

		db = db.Select("*, (measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?)) as is_favorite", user.ID)
		dbTotalCount = dbTotalCount.Select("*, (measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?)) as is_favorite", user.ID)
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		db = db.Unscoped().Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", practice.ID)
		dbTotalCount = dbTotalCount.Unscoped().Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", practice.ID)

		db = db.Select("*, (measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?)) as is_favorite", user.ID)
		dbTotalCount = dbTotalCount.Select("*, (measurements.id IN (SELECT measurement_id FROM measurement_favorites WHERE deleted_at IS NULL AND user_id = ?)) as is_favorite", user.ID)
	}

	if len(urlQuery) > 0 {
		values := urlQuery

		if val, ok := values["search"]; ok && len(val) > 0 {
			if val[0] != "" {
				search := "%" + val[0] + "%"

				hTime := r.Header["X-Timezone"]
				h, err := time.Parse("15:04", val[0])
				h = h.AddDate(2020, 1, 1)

				queryHasTimeOrDate := false
				if err == nil { //&&!fullDate
					queryHasTimeOrDate = true
					db = db.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR hour(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) = hour(?) AND minute(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) = minute(?))", hTime, search, search, search, search, hTime, h, hTime, h)
					dbTotalCount = dbTotalCount.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR hour(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) = hour(?) AND minute(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) = minute(?))", hTime, search, search, search, search, hTime, h, hTime, h)
				} else {
					var date core.NullTime
					tmp := val[0]
					date.FromString(tmp)

					if len(tmp) >= 4 {
						if date.Valid {
							queryHasTimeOrDate = true
							db = db.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR date(measurements.measurement_date) = date(?) OR time(measurements.measurement_date) = time(?))", hTime, search, search, search, search, date, date)
							dbTotalCount = dbTotalCount.Where("(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?)) OR date(measurements.measurement_date) = date(?) OR time(measurements.measurement_date) = time(?))", hTime, search, search, search, search, date, date)
						}
					}
				}
				if !queryHasTimeOrDate {
					db = db.Where("measurements.doctor_id IN (SELECT doctor_id FROM doctor_users WHERE ISNULL(deleted_at) AND user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?)) OR YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR	HOUR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MINUTE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?))", search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, search, search, search)
					dbTotalCount = dbTotalCount.Where("measurements.doctor_id IN (SELECT doctor_id FROM doctor_users WHERE ISNULL(deleted_at) AND user_id IN (SELECT id FROM system_accounts WHERE username LIKE ?)) OR YEAR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR DAYOFMONTH(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR HOUR(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR MINUTE(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', ?), measurements.measurement_date)) LIKE ? OR (Select username From system_accounts WHERE system_accounts.id = (Select user_id From patients Where patients.id = measurements.patient_id)) Like ? OR (Select patients.town From patients Where patients.id = measurements.patient_id) Like ? OR measurements.id IN (SELECT measurement_doctor_risks.measurement_id FROM measurement_doctor_risks WHERE measurement_doctor_risks.risk_definition_id IN (SELECT risk_definitions.id FROM risk_definitions WHERE risk_definitions.shortcut LIKE ?))", search, hTime, search, hTime, search, hTime, search, hTime, search, hTime, search, search, search, search)
				}
			}
		}

		if val, ok := values["last_refresh"]; ok && len(val) > 0 {
			if val[0] != "" {
				last_refresh := val[0]
				db = db.Where("updated_at > ?", last_refresh)
			}
		}

		if val, ok := values["measurement_date_from"]; ok && len(val) > 0 {
			if val[0] != "" {
				measurement_date_from := val[0]
				db = db.Where("measurement_date >= ?", measurement_date_from)
			}
		}

		///measurementFrom := false
		//measurementTo := false
		if val, ok := values["filter"]; ok && len(val) > 0 {
			tmpFilters := make(map[string][]string)
			for _, filter := range val {
				data := strings.Split(filter, ",")
				if len(data) > 1 {
					switch strings.ToLower(data[0]) {
					/*
					case "measurement_from":
						var measurementFrom tools.NullTime
						measurementFrom.FromString(data[1])

						/*	if measurementFrom.Valid {
							db = db.Where("YEAR(measurement_date) >= YEAR(?) AND MONTH(measurement_date) >= MONTH(?) AND DAY(measurement_date) >= DAY(?) AND HOUR(measurement_date) >= 0 AND MINUTE(measurement_date) >= 0", data[1], data[1], data[1])
							dbTotalCount = dbTotalCount.Where("YEAR(measurement_date) >= YEAR(?) AND MONTH(measurement_date) >= MONTH(?) AND DAY(measurement_date) >= DAY(?) AND HOUR(measurement_date) >= 0 AND MINUTE(measurement_date) >= 0", data[1], data[1], data[1])
						}
						if measurementFrom.Valid {
							db = db.Where("measurement_date >= ?", measurementFrom)
							dbTotalCount = dbTotalCount.Where("measurement_date >= ?", measurementFrom)
						}
					case "measurement_to":
						var measurementTo tools.NullTime
						measurementTo.FromString(data[1])
						/*if measurementTo.Valid {
							db = db.Where("YEAR(measurement_date) <= YEAR(?) AND MONTH(measurement_date) <= MONTH(?) AND DAY(measurement_date) <= DAY(?) AND HOUR(measurement_date) < 24 AND MINUTE(measurement_date) < 60", data[1], data[1], data[1])
							dbTotalCount = dbTotalCount.Where("YEAR(measurement_date) <= YEAR(?) AND MONTH(measurement_date) <= MONTH(?) AND DAY(measurement_date) <= DAY(?) AND HOUR(measurement_date) < 24 AND MINUTE(measurement_date) < 60", data[1], data[1], data[1])
						}
						if measurementTo.Valid {
							db = db.Where("DATE(measurement_date) <= DATE(?)", measurementTo)
							dbTotalCount = dbTotalCount.Where("DATE(measurement_date) <= DATE(?)", measurementTo)
						}
					*/
					case "user_practices", "user_doctors", "user_patients", "practices", "doctors", "patients", "measurement":
						tmpFilters[data[0]] = append(tmpFilters[data[0]], data[1])
					}
				}
				log.Println(filter)
			}

			for key, filterData := range tmpFilters {
				switch key {
				case "user_practices":
					db = db.Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (SELECT id FROM practices WHERE user_id IN (?))))", filterData)
					dbTotalCount = dbTotalCount.Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (SELECT id FROM practices WHERE user_id IN (?))))", filterData)
				case "user_doctors":
					db = db.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM doctor_users WHERE user_id IN (?) AND ISNULL(deleted_at)) AND consent_status=2)", filterData).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM doctor_users WHERE user_id IN (?) AND ISNULL(deleted_at)))", filterData)
					dbTotalCount = dbTotalCount.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM doctor_users WHERE user_id IN (?) AND ISNULL(deleted_at)) AND consent_status=2)", filterData).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM doctor_users WHERE user_id IN (?) AND ISNULL(deleted_at)))", filterData)
				case "user_patients":
					db = db.Where("measurements.patient_id IN (SELECT id FROM patients WHERE user_id IN (?))", filterData)
					dbTotalCount = dbTotalCount.Where("measurements.patient_id IN (SELECT id FROM patients WHERE user_id IN (?))", filterData)
				case "practices":
					db = db.Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (?)))", filterData)
					dbTotalCount = dbTotalCount.Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id IN (?)))", filterData)
				case "doctors":
					db = db.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id IN (?) AND consent_status=2)", filterData).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (?))", filterData)
					dbTotalCount = dbTotalCount.Unscoped().Where("measurements.patient_id IN (SELECT doctor_patient_relations.patient_id FROM doctor_patient_relations WHERE doctor_id IN (?) AND consent_status=2)", filterData).Where("measurements.id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (?))", filterData)
				case "patients":
					db = db.Where("measurements.patient_id IN (?)", filterData)
					dbTotalCount = dbTotalCount.Where("measurements.patient_id IN (?)", filterData)
				case "measurement":
					db = db.Where("measurements.id IN (?)", filterData)
					dbTotalCount = dbTotalCount.Where("measurements.id IN (?)", filterData)
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
					case "practice":
						db = db.Joins("LEFT JOIN doctors ON measurements.doctor_id = doctors.id").Joins("LEFT JOIN practice_doctors ON practice_doctors.doctor_id = doctors.id").Joins("LEFT JOIN practices ON practices.id = practice_doctors.practice_id").Joins("Left JOIN system_accounts ON practices.user_id = system_accounts.id")
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						dbTotalCount = dbTotalCount.Joins("LEFT JOIN doctors ON measurements.doctor_id = doctors.id").Joins("LEFT JOIN practice_doctors ON practice_doctors.doctor_id = doctors.id").Joins("LEFT JOIN practices ON practices.id = practice_doctors.practice_id").Joins("Left JOIN system_accounts ON practices.user_id = system_accounts.id")
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("username %s", sortDirection))
						break
					case "doctor":
						db = db.Joins("LEFT JOIN doctors ON measurements.doctor_id = doctors.id").Joins("Left JOIN doctor_users ON doctors.id = doctor_users.doctor_id").Joins("Left JOIN system_accounts ON doctor_users.user_id = system_accounts.id")
						dbTotalCount = dbTotalCount.Joins("LEFT JOIN doctors ON measurements.doctor_id = doctors.id").Joins("Left JOIN doctor_users ON doctors.id = doctor_users.doctor_id").Joins("Left JOIN system_accounts ON doctor_users.user_id = system_accounts.id")
						//dbTotalCount = dbTotalCount.Joins("LEFT JOIN doctors ON measurements.doctor_id = doctors.id").Joins("Left JOIN system_accounts ON doctors.user_id = system_accounts.id")

						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("username %s", sortDirection))
						break
					case "patient":
						//	db = db.Order(fmt.Sprintf("%s+0 %s, %s %s", sortKey, sortDirection, sortKey, sortDirection))
						db = db.Joins("LEFT JOIN patients ON measurements.patient_id = patients.id").Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						db = db.Order(fmt.Sprintf("username %s", sortDirection))
						dbTotalCount = dbTotalCount.Joins("LEFT JOIN patients ON measurements.patient_id = patients.id").Joins("Left JOIN system_accounts ON patients.user_id = system_accounts.id")
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("username %s", sortDirection))
						break
					case "town":
						//	db = db.Order(fmt.Sprintf("%s+0 %s, %s %s", sortKey, sortDirection, sortKey, sortDirection))
						db = db.Joins("LEFT JOIN patients ON measurements.patient_id = patients.id")
						db = db.Order(fmt.Sprintf("town %s", sortDirection))
						dbTotalCount = dbTotalCount.Joins("LEFT JOIN patients ON measurements.patient_id = patients.id")
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("town %s", sortDirection))
						break
					case "date":
						//	db = db.Order(fmt.Sprintf("%s+0 %s, %s %s", sortKey, sortDirection, sortKey, sortDirection))
						db = db.Order(fmt.Sprintf("measurement_date %s", sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("measurement_date %s", sortDirection))
						break
					case "time":

						//hTimezone := r.Header["X-Timezone"]

						//db = db.Order(fmt.Sprintf("TIME(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', '%s'), measurements.measurement_date)) %s", hTimezone[0], sortDirection))
						//dbTotalCount = dbTotalCount.Order(fmt.Sprintf("TIME(COALESCE(CONVERT_TZ(measurements.measurement_date, 'UTC', '%s'), measurements.measurement_date)) %s",hTimezone[0], sortDirection))
						db = db.Order(fmt.Sprintf("Time(measurement_date) %s", sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("Time(measurement_date) %s", sortDirection))
						break
					case "status":
						db = db.Order(fmt.Sprintf("hotspot_detected %s, coldspot_detected %s", sortDirection, sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("hotspot_detected %s, coldspot_detected %s", sortDirection, sortDirection))

					case "measurement_date":
						//	db = db.Order(fmt.Sprintf("%s+0 %s, %s %s", sortKey, sortDirection, sortKey, sortDirection))
						db = db.Order(fmt.Sprintf("measurement_date %s", sortDirection))
						dbTotalCount = dbTotalCount.Order(fmt.Sprintf("measurement_date %s", sortDirection))
						break
					case "measurement_risk":
						//db := db.Where("measurement_doctor_risks.doctor_id = ?", doctor.ID)

						if c.isDoctor(user) {
							doctor := c.getDoctor(user)
							db = db.Joins("LEFT JOIN measurement_doctor_risks ON measurements.id = measurement_doctor_risks.measurement_id AND measurement_doctor_risks.doctor_id = ?", doctor.ID).Joins("LEFT JOIN risk_definitions ON measurement_doctor_risks.risk_definition_id = risk_definitions.id")
							dbTotalCount = dbTotalCount.Joins("LEFT JOIN measurement_doctor_risks ON measurements.id = measurement_doctor_risks.measurement_id AND measurement_doctor_risks.doctor_id = ?", doctor.ID).Joins("LEFT JOIN risk_definitions ON measurement_doctor_risks.risk_definition_id = risk_definitions.id")
						} else {
							db = db.Joins("LEFT JOIN measurement_doctor_risks ON measurements.id = measurement_doctor_risks.measurement_id").Joins("LEFT JOIN risk_definitions ON measurement_doctor_risks.risk_definition_id = risk_definitions.id")
							dbTotalCount = dbTotalCount.Joins("LEFT JOIN measurement_doctor_risks ON measurements.id = measurement_doctor_risks.measurement_id").Joins("LEFT JOIN risk_definitions ON measurement_doctor_risks.risk_definition_id = risk_definitions.id")
						}

						if "asc" == sortDirection {
							//db = db.Debug().Order("case when risk_definitions.sort_value = 1 Then 5 when isNUll(risk_definitions.sort_value) Then 6 ELSE risk_definitions.sort_value END ASC")
							db = db.Order("risk_definitions.sort_value_asc DESC")
						} else {
							db = db.Order(fmt.Sprintf("risk_definitions.sort_value %s", sortDirection))

							//dbTotalCount = dbTotalCount.Joins("LEFT JOIN measurement_doctor_risks ON measurements.id = measurement_doctor_risks.measurement_id")
							//dbTotalCount = dbTotalCount.Order(fmt.Sprintf("measurement_doctor_risks.id %s", sortDirection))
						}
						break
					case "device_serial":
						db = db.Joins("LEFT JOIN devices ON measurements.device_id = devices.id")
						dbTotalCount = dbTotalCount.Joins("LEFT JOIN devices ON measurements.device_id = devices.id")

						db = db.Order(fmt.Sprintf("devices.device_serial %s", sortDirection))
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

// getRedMeasurements swagger:route GET /measurements/red measurements getRedMeasurements
//
// retrieves measurements with hotspot detected
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementsWithStatusRedHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	measurements := Measurements{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Where("hotspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2)", doctor.ID).Where("id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID).Where("hotspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("hotspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	}

	for key, measurement := range measurements {
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)
		measurements[key] = measurement
	}

	c.SendJSON(w, &measurements, http.StatusOK)
}

// getBlueMeasurements swagger:route GET /measurements/blue measurements getBlueMeasurements
//
// retrieves measurements with coldspot detected
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementsWithStatusBlueHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	measurements := Measurements{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Where("coldspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2)", doctor.ID).Where("coldspot_detected != 'NONE'").Where("id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID).Order("measurement_date DESC").Find(&measurements)
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("coldspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	}

	for key, measurement := range measurements {
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)
		measurements[key] = measurement
	}

	c.SendJSON(w, &measurements, http.StatusOK)
}

// getRedBlueMeasurements swagger:route GET /measurements/redblue measurements getRedBlueMeasurements
//
// retrieves measurements with hotspot and coldspot detected
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementsWithStatusRedBlueHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	measurements := Measurements{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Where("hotspot_detected != 'NONE' OR coldspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2)", doctor.ID).Where("hotspot_detected != 'NONE' OR coldspot_detected != 'NONE'").Where("id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID).Order("measurement_date DESC").Find(&measurements)
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("hotspot_detected != 'NONE' OR coldspot_detected != 'NONE'").Order("measurement_date DESC").Find(&measurements)
	}

	for key, measurement := range measurements {
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)
		measurements[key] = measurement
	}

	c.SendJSON(w, &measurements, http.StatusOK)
}

// getRedBlueMeasurements swagger:route GET /measurements/redblue measurements getRedBlueMeasurements
//
// retrieves measurements with hotspot and coldspot detected
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
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementsForPatientHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	measurements := Measurements{}

	vars := mux.Vars(r)
	patientId, _ := strconv.ParseInt(vars["patientId"], 10, 64)

	if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).Where("patient_id = ?", patientId).Where("id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID).Find(&measurements)
	}

	for key, measurement := range measurements {
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)
		measurements[key] = measurement
	}

	c.SendJSON(w, &measurements, http.StatusOK)
}

// getMeasurement swagger:route GET /measurement/{measurementId} measurements getMeasurement
//
// retrieves appointments for user, expected if user is system user, he get all appointments
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
//			data: Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	measurement := Measurement{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).First(&measurement, measurementId)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ? AND consent_status>=2)", doctor.ID).Where("id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID).First(&measurement, measurementId)
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).Where("user_id = ? OR id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?))", user.ID, practice.ID).Find(&measurement, measurementId)
	} else if user.UserType == 0 {
		c.ormDB.Unscoped().Set("gorm:auto_preload", true).First(&measurement, measurementId)
	}

	c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)

	if measurement.MeasurementFiles == nil {
		measurement.MeasurementFiles = make([]MeasurementFile, 0)
	}
	measurement.SetupQuestionnaire = &PatientQuestionnaire{}
	c.ormDB.Set("gorm:auto_preload", true).Where("DATE(questionnaire_date) <= DATE(?)", measurement.MeasurementDate).Where("patient_id = ?", measurement.PatientId).Where("id IN (SELECT patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE qt.question_type = 1)").Order("questionnaire_date DESC").First(&measurement.SetupQuestionnaire)

	c.SendJSON(w, &measurement, http.StatusOK)
}

// deleteMeasurement swagger:route DELETE /measurement/{measurementId} measurements deleteMeasurement
//
// delete a measurement
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
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) DeleteMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	//TODO SM TEST
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	measurement := Measurement{}

	wsIds := []uint{}

	if measurementId > 0 {
		if c.isPatient(user) {
			patient := c.getPatient(user)
			c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).Delete(&measurement, measurementId)

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

			go web3socket.SendWebsocketDataInfoMessage("Delete measurment", web3socket.Websocket_Delete, web3socket.Websocket_Measurements, uint(measurementId), wsIds, nil)
		} else if c.isDoctor(user) {
			measurementShared := &MeasurementShared{}
			doctor := c.getDoctor(user)
			c.ormDB.Where("doctor_id =?", doctor.ID).Where("measurement_id = ?", measurementId).Delete(&measurementShared)

			wsIds = append(wsIds, user.ID)
			practiceId := doctor.GetPracticeUserId(c.ormDB)
			if practiceId > 0 {
				wsIds = append(wsIds, practiceId)
			}
			go web3socket.SendWebsocketDataInfoMessage("Delete measurement", web3socket.Websocket_Delete, web3socket.Websocket_Measurements, uint(measurementId), wsIds, nil)
		} else if c.isPractice(user) {
			measurementShared := &MeasurementShared{}
			practice := c.getPractice(user)

			wsMeasurementShareds := []MeasurementShared{}
			c.ormDB.Debug().Where("doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?)", practice.ID).Where("measurement_id = ?", measurementId).Find(&wsMeasurementShareds)
			c.ormDB.Debug().Where("doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?)", practice.ID).Where("measurement_id = ?", measurementId).Delete(&measurementShared)

			wsIds = append(wsIds, user.ID)
			for _, item := range wsMeasurementShareds {
				if item.DoctorId > 0 {
					userId := c.getMainUserIdFromDoctorId(item.DoctorId)
					if userId > 0 {
						wsIds = append(wsIds, userId)
					}
				}
			}
			if len(wsIds) > 0 {
				go web3socket.SendWebsocketDataInfoMessage("Delete measurment", web3socket.Websocket_Delete, web3socket.Websocket_Measurements, uint(measurementId), wsIds, nil)
			}
		} else if user.UserType == 0 {
			c.ormDB.Set("gorm:auto_preload", true).Delete(&measurement, measurementId)
		}
	}

	c.SendJSON(w, measurement, http.StatusOK)
}

// getMeasurementFile swagger:route GET /measurement/{measurementId}/files/{fileType} measurements getMeasurementFile
//
// retrieves appointments for user, expected if user is system user, he get all appointments
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
//			data: Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMeasurementFileHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)
	fileType := strings.ToUpper(vars["fileType"])
	if fileType != "THERMAL" && fileType != "REPORT" && fileType != "STATISTIC" && fileType != "DYNAMIC" && fileType != "DYNAMIC_STATISTIC" {
		fileType = "NORMAL"
	}

	measurement := Measurement{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).First(&measurement, measurementId)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)

		practice := doctor.GetPractice(c.ormDB)
		c.ormDB.Set("gorm:auto_preload", true).Unscoped().Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ? ) AND consent_status>=2)", practice.ID).First(&measurement, measurementId)

		//c.ormDB.Set("gorm:auto_preload", true).Unscoped().Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ? AND consent_status>=2)", doctor.ID).First(&measurement, measurementId)
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		c.ormDB.Set("gorm:auto_preload", true).Unscoped().Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ? ) AND consent_status>=2)", practice.ID).First(&measurement, measurementId)
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).Unscoped().First(&measurement, measurementId)
	}
	if measurement.ID > 0 {
		// measurement wurde gefunden, User ist berechtigt es zu sehen
		for _, file := range measurement.MeasurementFiles {
			if file.MeasurementType == fileType {
				w.Header().Set("Content-Disposition", `inline; filename="measurement`+strconv.Itoa(int(measurement.ID))+`_`+file.MeasurementType+`.jpg"`)
				w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
				w.Header().Add("Access-Control-Allow-Origin", "*")
				log.Println(file.GetAbsolutePath())
				http.ServeFile(w, r, file.GetAbsolutePath())
				return
			}
		}
	}
	c.HandleErrorWithStatus(errors.New("File not found"), w, http.StatusNotFound)
}

func (c *PodiumController) GetMeasurementsFilesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	db, _ := c.CreateWhereConditionsMeasurements(r.URL.Query(), r, user)

	measurements := Measurements{}

	if user.IsSysadmin {
		db = db.Preload("MeasurementFiles").Preload("Patient").Preload("Patient.User")
	}

	db.Debug().Set("gorm:auto_preload", false).Find(&measurements)

	if len(measurements) == 0 {
		err := errors.New("No scan found.")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	now := tools.NullTime { Time: time.Now(), Valid: true }
	filePrefix := fmt.Sprintf(`measurement_images_%v.%v.%v`, now.Time.Day(), int(now.Time.Month()), now.Time.Year())
	fileNames := []string{}

	//X tmpPath := "tmp/" + core.RandomString(10)
	//tmpFolder := tmpPath + "/" + filePrefix
	tmpPath := c.GetTmpUploadPath()
	tmpFolder := tmpPath + filePrefix

	tmpFileName, _ := CreateFolder(tmpFolder, nil)

	for _, measurement := range measurements {
		// measurement wurde gefunden, User ist berechtigt es zu sehen
		fileNames = []string{}
		for _, file := range measurement.MeasurementFiles {
			fileNames = append(fileNames, core.GetUploadFilepath()+file.Filepath)
		}

		folder := fmt.Sprintf(`%s_images_from_%v.%v.%v_%v.%v`, measurement.Patient.User.Username, measurement.MeasurementDate.Time.Day(), int(measurement.MeasurementDate.Time.Month()), measurement.MeasurementDate.Time.Year(), measurement.MeasurementDate.Time.Hour(), measurement.MeasurementDate.Time.Minute())
		path := tmpFileName + "/" + folder
		CreateFolder(path, fileNames)
	}

	if len(fileNames) == 0 {
		err := errors.New("No images found")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	fileNameZip, err := c.ZipWriter(filePrefix+".zip", tmpFolder)

	if err != nil {
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	c.SendFile(w, r, fileNameZip)
}

func CreateFolder(folderFileName string, fileNames []string) (string, error) {

	os.MkdirAll(folderFileName, 0777)

	if fileNames != nil && len(fileNames) > 0 {
		for _, originalFileName := range fileNames {

			dat, err := ioutil.ReadFile(originalFileName)
			if err != nil {
				fmt.Println(err)
			}

			splitOriginalFileName := strings.Split(originalFileName, "/")

			newFileName := originalFileName
			if len(splitOriginalFileName) > 0 {
				newFileName = splitOriginalFileName[len(splitOriginalFileName)-1]
			}

			file, err := os.Create(folderFileName + "/" + newFileName)

			if err != nil {
				fmt.Println(err)
			}
			_, err = file.Write(dat)
			if err != nil {
				fmt.Println(err)
			}

			defer file.Close()
		}
	}

	return folderFileName, nil
}

func (c *PodiumController) ZipWriter(zipFile string, originalFolder string) (string, error) {

	//X tmpPath := "tmp/" + core.RandomString(10)
	//tmpFilename := tmpPath + "/" + zipFile
	tmpPath := c.GetTmpUploadPath()
	tmpFilename := tmpPath + zipFile

	os.MkdirAll(tmpPath, 0777)

	// Get a Buffer to Write To
	outFile, err := os.Create(tmpFilename)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	err = addFiles(w, originalFolder+"/", "")

	if err != nil {
		return "", err
	}

	err = w.Close()
	if err != nil {
		return "", err
	}

	return tmpFilename, nil
}

func addFiles(w *zip.Writer, basePath, baseInZip string) error {
	// Open the Directory
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		fmt.Println(basePath + file.Name())
		if !file.IsDir() {
			dat, err := ioutil.ReadFile(basePath + file.Name())
			if err != nil {
				fmt.Println(err)
			}

			// Add some files to the archive.
			f, err := w.Create(baseInZip + file.Name())
			if err != nil {
				fmt.Println(err)
			}
			_, err = f.Write(dat)
			if err != nil {
				fmt.Println(err)
			}
		} else if file.IsDir() {

			// Recurse
			newBase := basePath + file.Name() + "/"
			fmt.Println("Recursing and Adding SubDir: " + file.Name())
			fmt.Println("Recursing and Adding SubDir: " + newBase)

			addFiles(w, newBase, baseInZip+file.Name()+"/")
		}
	}

	return nil
}

func (c *PodiumController) GetMeasurementFilesHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	measurement := Measurement{}

	c.ormDB.Debug().Set("gorm:auto_preload", true).First(&measurement, measurementId)

	userPatient := core.User{}
	c.ormDB.Debug().Set("gorm:auto_preload", false).Where("id IN (SELECT user_id FROM patients WHERE id = ?)", measurement.PatientId).First(&userPatient)

	fileName := fmt.Sprintf(`measurement_images_%s`, userPatient.Username)
	dateFrom := ""
	if measurement.MeasurementDate.Valid {
		dateFrom = "%s_%v.%v.%v_%v.%v"
		dateFrom = fmt.Sprintf(dateFrom, measurement.Patient.User.Username, measurement.MeasurementDate.Time.Day(), int(measurement.MeasurementDate.Time.Month()), measurement.MeasurementDate.Time.Year(), measurement.MeasurementDate.Time.Hour(), measurement.MeasurementDate.Time.Minute())
	}
	fullFileName := fileName + dateFrom + ".zip"

	fileNames := []string{}

	// measurement wurde gefunden, User ist berechtigt es zu sehen

	for _, file := range measurement.MeasurementFiles {
		fileNames = append(fileNames, core.GetUploadFilepath()+file.Filepath)
	}

	if len(fileNames) == 0 {
		err := errors.New("No images found")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	fileNameZip, _ := core.ZipFiles(fullFileName, fileNames)

	c.SendFile(w, r, fileNameZip)
}

func (c *PodiumController) GetMeasurementImagesForPortalHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	measurement := Measurement{}

	c.ormDB.Debug().Set("gorm:auto_preload", true).First(&measurement, measurementId)

	//userPatient := core.User{}
	//c.ormDB.Debug().Set("gorm:auto_preload", false).Where("id IN (SELECT user_id FROM patients WHERE id = ?)", measurement.PatientId).First(&userPatient)

	// measurement wurde gefunden, User ist berechtigt es zu sehen

	images := []MeasurementFile{}
	for _, file := range measurement.MeasurementFiles {
		if file.MeasurementType == "THERMAL" || file.MeasurementType == "STATISTIC" || file.MeasurementType == "NORMAL" {
			images = append(images, file)
		}
	}

	c.SendJSON(w, &images, http.StatusOK)
}

// Retrieve Files
func (c *PodiumController) RetrieveMeasurementHandler(w http.ResponseWriter, r *http.Request) {

	deviceDataRequired := false
	deviceStatusValid := true

	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	clientId := r.Header.Get("Client")

	if !c.isPatient(user) && !c.isPractice(user) {
		err := errors.New("only patient or practice can save measurements")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	patient := &Patient{}
	doctor := &Doctor{}
	if c.isPatient(user) {
		patient = c.getPatient(user)
	} else if c.isPractice(user) {
		c.ormDB.Set("gorm:auto_preload", false).Find(&patient, r.FormValue("patient_id"))
		c.ormDB.Set("gorm:auto_preload", false).Find(&doctor, r.FormValue("doctor_id"))
	}

	measurementDateStr := r.FormValue("measurement_date")
	hotspotDetected := r.FormValue("hotspot_detected")
	coldspotDetected := r.FormValue("coldspot_detected")
	deviceTemperatureStr := r.FormValue("device_temperature")
	deviceHumidityStr := r.FormValue("device_humidity")
	deviceBatteryStr := r.FormValue("device_battery")
	deviceIdStr := r.FormValue("device_id")
	deviceMacStr := r.FormValue("device_mac")
	deviceSerialStr := r.FormValue("device_serial")
	deviceTypeStr := r.FormValue("device_type")
	deviceVersionStr := r.FormValue("device_version")
	deviceSystemVersionStr := r.FormValue("device_system_version")

	leftMinTemperatureStr := r.FormValue("left_min_temperature")
	leftMeanTemperatureStr := r.FormValue("left_mean_temperature")
	leftMaxTemperatureStr := r.FormValue("left_max_temperature")
	rightMinTemperatureStr := r.FormValue("right_min_temperature")
	rightMeanTemperatureStr := r.FormValue("right_mean_temperature")
	rightMaxTemperatureStr := r.FormValue("right_max_temperature")
	diffMinTemperatureStr := r.FormValue("diff_min_temperature")
	diffMeanTemperatureStr := r.FormValue("diff_mean_temperature")
	diffMaxTemperatureStr := r.FormValue("diff_max_temperature")

	//phoneDeviceIdentifier := r.FormValue("phone_device_id")
	//phoneDeviceName := r.FormValue("phone_device_name")
	appVersionStr := r.FormValue("app_version")
	appBuildStr := r.FormValue("app_build")
	appSystemVersionStr := r.FormValue("app_system_version")
	appVersionDeviceTypeNameStr := r.FormValue("app_device_type_name")

	appDeviceType := DeviceType{}
	if len(appVersionDeviceTypeNameStr) > 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("type_name = ?", appVersionDeviceTypeNameStr).First(&appDeviceType)
	}

	measurementDate := core.NullTime{}
	measurementDate.FromString(measurementDateStr)

	//Prüfe, ob der Scan schon mal geschickt wurde
	measurementDB := Measurement{}
	c.ormDB.Set("gorm:auto_preload", true).Where("measurement_date = ? AND patient_id = ?", measurementDate, patient.ID).First(&measurementDB)
	if measurementDB.ID > 0 {
		questionnaire := PatientQuestionnaire{}
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = ? AND patient_id = ?", measurementDB.ID, patient.ID).First(&questionnaire)
		measurementDB.Questionnaire = &questionnaire
			
		if c.isPractice(user) {
			dailyQuestionnaire := PatientQuestionnaire{}
			c.ormDB.Debug().Set("gorm:auto_preload", true).Where("measurement_id = 0").Where("patient_id = ?", measurementDB.PatientId).Where("DATE(questionnaire_date) = DATE(?)", measurementDB.MeasurementDate).Where("id IN (SELECT pqq.patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE pqq.answer_id = 0 AND qt.recurring_rule IN ('DAILY', 'WEEKLY', 'MONTHLY'))").Order("id desc").First(&dailyQuestionnaire)
			measurementDB.DailyQuestionnaire = &dailyQuestionnaire
			if measurementDB.DailyQuestionnaire == nil || measurementDB.DailyQuestionnaire.ID == 0 {
				measurementDB.DailyQuestionnaire, _ = CreatePatientQuestionnaire(*c.ormDB, measurementDB.PatientId, []int64{2, 3}, []string{"DAILY", "WEEKLY", "MONTHLY"}, 0)
			}
			c.ormDB.Set("gorm:auto_preload", false).First(&measurementDB.Patient, measurementDB.PatientId)
			//if !measurementDB.Patient.SetupComplete {
			measurementDB.SetupQuestionnaire, _ = CreatePatientQuestionnaire(*c.ormDB, measurementDB.PatientId, []int64{1}, []string{"SETUP"}, 0)
			//}
		}
		c.SendJSON(w, measurementDB, http.StatusOK)
		return
	}

	device := Device{}
	deviceType := DeviceType{}

	c.ormDB.Where("type_name = ?", deviceTypeStr).Find(&deviceType)
	if deviceType.ID == 0 {
		err := errors.New("invalid Data: no valid DeviceType")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	c.ormDB.Where("device_mac = ?", deviceMacStr).Find(&device)

	//c.ormDB.Where("device_mac = ? AND device_serial = ?", deviceMacStr, deviceSerialStr).Find(&device)
	if device.ID == 0 {
		device.DeviceIdentifier = deviceIdStr
		device.DeviceMac = deviceMacStr
		device.DeviceSerial = deviceSerialStr
		device.DeviceType = deviceType
		device.DeviceTypeId = deviceType.ID
		c.ormDB.Set("gorm:save_associations", false).Create(&device)
	}
	if device.ID > 0 {
		if device.DeviceIdentifier != deviceIdStr && deviceIdStr != "" {
			c.ormDB.Model(&Device{}).Where("id = ?", device.ID).Update("device_identifier", deviceIdStr)
		}
		if c.isPatient(user) {
			patDevice := PatientDevice{}
			c.ormDB.Where("patient_id = ? AND device_id = ?", patient.ID, device.ID).Find(&patDevice)
			if patDevice.ID == 0 {
				patDevice.Device = device
				patDevice.DeviceId = device.ID
				patDevice.PatientId = patient.ID
				c.ormDB.Create(&patDevice)
			}
		}

		if c.isPractice(user) {

			if doctor.ID > 0 {
				podDevice := DoctorDevice{}
				c.ormDB.Where("doctor_id = ? AND device_id = ?", doctor.ID, device.ID).Find(&podDevice)
				if podDevice.ID == 0 {
					podDevice.Device = device
					podDevice.DeviceId = device.ID
					podDevice.DoctorId = doctor.ID
					c.ormDB.Create(&podDevice)
				}
			}
			practice := c.getPractice(user)
			pracDevice := PracticeDevice{}
			c.ormDB.Where("practice_id = ? AND device_id = ?", practice.ID, device.ID).Find(&pracDevice)
			if pracDevice.ID == 0 {
				pracDevice.Device = device
				pracDevice.DeviceId = device.ID
				pracDevice.PracticeId = practice.ID
				c.ormDB.Create(&pracDevice)
			}
		}
	}

	if !measurementDate.Valid {
		err := errors.New("invalid Data: no measurement date")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	if hotspotDetected != "NONE" && hotspotDetected != "LEFT" && hotspotDetected != "RIGHT" && hotspotDetected != "BOTH" {
		err := errors.New("invalid Data: hotspot detected")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	if coldspotDetected != "NONE" && coldspotDetected != "LEFT" && coldspotDetected != "RIGHT" && coldspotDetected != "BOTH" {
		err := errors.New("invalid Data: coldspot detected")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	deviceTemperature, err := strconv.ParseFloat(deviceTemperatureStr, 64)
	if err != nil {
		err := errors.New("invalid Data: no valid deviceTemperature")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	deviceHumidity, err := strconv.ParseFloat(deviceHumidityStr, 64)
	if err != nil {
		err := errors.New("invalid Data: no valid deviceHumidity")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}
	deviceBattery, err := strconv.ParseFloat(deviceBatteryStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid deviceBattery")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}

	appBuild, err := strconv.ParseInt(appBuildStr, 10, 64)
	if err != nil {
		appBuild = 0
	}

	leftMinTemperature, err := strconv.ParseFloat(leftMinTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid leftMinTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	leftMeanTemperature, err := strconv.ParseFloat(leftMeanTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid leftMeanTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	leftMaxTemperature, err := strconv.ParseFloat(leftMaxTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid leftMaxTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	rightMinTemperature, err := strconv.ParseFloat(rightMinTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid rightMinTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	rightMeanTemperature, err := strconv.ParseFloat(rightMeanTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid rightMeanTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	rightMaxTemperature, err := strconv.ParseFloat(rightMaxTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid rightMaxTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	diffMinTemperature, err := strconv.ParseFloat(diffMinTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid diffMinTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	diffMeanTemperature, err := strconv.ParseFloat(diffMeanTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid diffMeanTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}
	diffMaxTemperature, err := strconv.ParseFloat(diffMaxTemperatureStr, 64)
	if err != nil {
		deviceStatusValid = false
		if deviceDataRequired {
			err := errors.New("invalid Data: no valid diffMaxTemperature")
			c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
			return
		}
	}

	measurement := Measurement{
		PatientId:           patient.ID,
		MeasurementDate:     measurementDate,
		HotspotDetected:     hotspotDetected,
		ColdspotDetected:    coldspotDetected,
		DeviceTemperature:   deviceTemperature,
		DeviceHumidity:      deviceHumidity,
		DeviceBattery:       deviceBattery,
		DeviceVersion:       deviceVersionStr,
		TmpDate:             measurementDateStr,
		MeasurementFiles:    make([]MeasurementFile, 0),
		UserId:              user.ID,
		DoctorId:            doctor.ID,
		DeviceId:            device.ID,
		AppBuild:            appBuild,
		AppVersion:          appVersionStr,
		AppSystemVersion:    appSystemVersionStr,
		DeviceSystemVersion: deviceSystemVersionStr,
		AppDeviceTypeId:     appDeviceType.ID,
		AppDeviceType:       appDeviceType,
	}

	measurementSharedDoctorIds := []uint{}
	c.ormDB.Create(&measurement)
	if doctor.ID > 0 {
		newShare := &MeasurementShared{}
		newShare.DoctorId = doctor.ID
		newShare.MeasurementId = measurement.ID
		c.ormDB.Create(&newShare)

		pairing := &DoctorPatientRelation{}
		c.ormDB.Where("consent_status = 2 AND doctor_id = ? AND patient_id = ?", doctor.ID, patient.ID).Find(&pairing)
		if pairing.ID == 0 {
			pairing.DoctorId = doctor.ID
			pairing.PatientId = patient.ID
			pairing.ConsentStatus = 2
			pairing.ConsentType = 2
			pairing.ConsentDate = core.NullTime{time.Now(), true}
			c.ormDB.Create(&pairing)
			message := Message{
				DoctorId:    doctor.ID,
				IsUnread:    true,
				MessageText: doctor.StandardWelcomeMessage,
				SenderId:    user.ID,
				RecipientId: patient.UserId,
				MessageTime: core.NullTime{Time: time.Now(), Valid: true},
			}
			c.ormDB.Create(&message)
			c.CreateNotification(message.Recipient.ID, 1, fmt.Sprintf("New Message from %s", message.Sender.Username), message.MessageText, message.ID, fmt.Sprintf("/me/conversations/%d", message.Sender.ID), nil)
		}
	} else if clientId == core.CLIENT_APK_Remote && core.CLIENT_APK_Remote != core.Client_APK_Home {
		pairings := DoctorPatientRelations{}
		c.ormDB.Where("consent_status = 2 AND patient_id = ?", patient.ID).Find(&pairings)
		for _, pairing := range pairings {
			newShare := &MeasurementShared{}
			newShare.DoctorId = pairing.DoctorId
			newShare.MeasurementId = measurement.ID
			c.ormDB.Debug().Create(&newShare)
			measurementSharedDoctorIds = append(measurementSharedDoctorIds, pairing.DoctorId)
		}

	}
	CalculateRewardForPatient(c.ormDB, patient.ID, true)

	err = os.MkdirAll(fmt.Sprintf(core.GetUploadFilepath()+"uploads/patients/%d/recordings/%d/", patient.ID, measurement.ID), 0777)
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
			relativeFileName := fmt.Sprintf("uploads/patients/%d/recordings/%d/%s", patient.ID, measurement.ID, filename)
			filePath := fmt.Sprintf(core.GetUploadFilepath() + relativeFileName)
			//if outfile, err = os.Create("./uploaded/" + hdr.Filename); nil != err {
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

			measurementType := "NORMAL"

			if strings.Contains(strings.ToUpper(fileKey), "THERMAL") {
				measurementType = "THERMAL"
			} else if strings.Contains(strings.ToUpper(fileKey), "REPORT") {
				measurementType = "REPORT"
			} else if strings.EqualFold(strings.ToUpper(fileKey), "DYNAMIC_STATISTIC") {
				measurementType = "DYNAMIC_STATISTIC"
			} else if strings.Contains(strings.ToUpper(fileKey), "STATISTIC") {
				measurementType = "STATISTIC"
			} else if strings.Contains(strings.ToUpper(fileKey), "DYNAMIC") {
				measurementType = "DYNAMIC"
			} else if strings.Contains(strings.ToUpper(fileKey), "T0") {
				measurementType = "T0"
			} else if strings.Contains(strings.ToUpper(fileKey), "INPUT") {
				measurementType = "INPUT"
			} else if strings.Contains(strings.ToUpper(fileKey), "FOOT_POSITIONING") {
				measurementType = "FOOT_POSITIONING"
			}

			measurementFile := MeasurementFile{
				MeasurementId:   measurement.ID,
				Filepath:        relativeFileName,
				MeasurementType: measurementType,
			}
			c.ormDB.Create(&measurementFile)
			measurement.MeasurementFiles = append(measurement.MeasurementFiles, measurementFile)

			//res.Write([]byte("uploaded file:" + hdr.Filename + ";length:" + strconv.Itoa(int(written))))
		}
	}

	measurement.Questionnaire, _ = CreatePatientQuestionnaire(*c.ormDB, measurement.PatientId, []int64{2, 3}, []string{"DAILY-SCAN", "WEEKLY-SCAN", "MONTHLY-SCAN"}, int64(measurement.ID))
	if c.isPractice(user) {
		dailyQuestionnaire := PatientQuestionnaire{}
		db := c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id = 0")
		db = db.Where("DATE(questionnaire_date) = DATE(NOW())").Where("patient_id = ?", measurement.PatientId)
		db.Where("id IN (SELECT pqq.patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE pqq.answer_id = 0 AND qt.recurring_rule IN ('DAILY', 'WEEKLY', 'MONTHLY'))").Order("id desc").First(&dailyQuestionnaire)
		measurement.DailyQuestionnaire = &dailyQuestionnaire
		if measurement.DailyQuestionnaire == nil || measurement.DailyQuestionnaire.ID == 0 {
			measurement.DailyQuestionnaire, _ = CreatePatientQuestionnaire(*c.ormDB, measurement.PatientId, []int64{2, 3}, []string{"DAILY", "WEEKLY", "MONTHLY"}, 0)
		}
		c.ormDB.Set("gorm:auto_preload", false).Find(&measurement.Patient, measurement.PatientId)
		//if !measurement.Patient.SetupComplete {
		measurement.SetupQuestionnaire, _ = CreatePatientQuestionnaire(*c.ormDB, measurement.PatientId, []int64{1}, []string{"SETUP"}, 0)
		//}
	}

	if device.ID > 0 {
		newDeviceStatus := DeviceStatus{}
		newDeviceStatus.MeasurementId = measurement.ID
		newDeviceStatus.StatusDate = measurement.MeasurementDate
		newDeviceStatus.DeviceId = device.ID

		newDeviceStatus.Valid = deviceStatusValid
		newDeviceStatus.Temperature = deviceTemperature
		newDeviceStatus.Humidity = deviceHumidity
		newDeviceStatus.Battery = deviceBattery
		newDeviceStatus.LeftMinTemperature = leftMinTemperature
		newDeviceStatus.LeftMeanTemperature = leftMeanTemperature
		newDeviceStatus.LeftMaxTemperature = leftMaxTemperature
		newDeviceStatus.RightMinTemperature = rightMinTemperature
		newDeviceStatus.RightMeanTemperature = rightMeanTemperature
		newDeviceStatus.RightMaxTemperature = rightMaxTemperature
		newDeviceStatus.DiffMinTemperature = diffMinTemperature
		newDeviceStatus.DiffMeanTemperature = diffMeanTemperature
		newDeviceStatus.DiffMaxTemperature = diffMaxTemperature
		c.ormDB.Set("gorm:save_associations", false).Create(&newDeviceStatus)
	}

	/*
		data := struct {
			Status		int			`json:"status"`
			Message		string		`json:"message"`
			Measurement	Measurement		`json:"measurement"`
		}{
			58,
			"Files added",
			measurement,
		}
	*/

	if !c.isPatient(user) && !c.isPractice(user) {
		err := errors.New("only patient or practice can save measurements")
		c.HandleErrorWithStatus(err, w, http.StatusNotAcceptable)
		return
	}

	c.SendJSON(w, measurement, http.StatusOK)

	ids := []uint{user.ID}

	if c.isPractice(user) {
		//Only Practies and Doctor
		if patient.UserId > 0 {
			ids = append(ids, patient.UserId)
		}

		if doctor.ID > 0 {
			doctorIds := c.getMainUserIdFromDoctorId(doctor.ID)
			ids = append(ids, doctorIds)
		}
	} else if c.isPatient(user) {

		userIdsOfDoctors := c.getMainUserIdsFromDoctorIds(measurementSharedDoctorIds)
		userIdOfPractices := c.getPracticeUserIdsFromDoctorIds(measurementSharedDoctorIds)

		ids = append(ids, userIdsOfDoctors...)
		ids = append(ids, userIdOfPractices...)
	}
	if len(ids) > 0 {
		go web3socket.SendWebsocketDataInfoMessage("Add Measurement", web3socket.Websocket_Add, web3socket.Websocket_Measurements, measurement.ID, ids, nil)
		//Nur Practice
		go web3socket.SendWebsocketDataInfoMessage("Update Doctors", web3socket.Websocket_Update, web3socket.Websocket_Podiatrist, measurement.DoctorId, ids, nil)
		//Podiatrist and Practice
		go web3socket.SendWebsocketDataInfoMessage("Update Patients", web3socket.Websocket_Update, web3socket.Websocket_Patients, measurement.PatientId, ids, nil)
	}
	c.SetIsTenMinsApart(measurement, user)
}

func (c *PodiumController) SetIsTenMinsApart(measurement Measurement, user *core.User) {

	if c.isPractice(user) {

		if measurement.DoctorId > 0 {

			prevTenMinsApart := Measurement{}

			c.ormDB.Debug().Where("measurement_date <= ? AND id != ? AND patient_id = ? AND user_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND ten_mins_apart AND deleted_at IS NULL))", measurement.MeasurementDate, measurement.ID, measurement.PatientId, user.ID, measurement.DoctorId).Order("measurement_date DESC").First(&prevTenMinsApart)

			lastGoodMeasurement := prevTenMinsApart

			if prevTenMinsApart.ID == 0 || measurement.MeasurementDate.Time.Unix()-prevTenMinsApart.MeasurementDate.Time.Unix() > 600 {
				log.Println(measurement.ID)
				log.Println(measurement.DoctorId)
				err := c.ormDB.Model(&MeasurementShared{}).Where("measurement_id = ? AND doctor_id = ?", measurement.ID, measurement.DoctorId).Update("ten_mins_apart", true).Error
				log.Println(err)
				lastGoodMeasurement = measurement
			}

			lateMeasurementsShared := MeasurementsShared{}
			//c.ormDB.Where("measurement_date >= ? AND id != ? AND patient_id = ? AND user_id = ? AND (doctor_id > 0 AND id IN (SELECT measurement_id FROM measurement_shareds WHERE doctor_id = ? AND ten_min_apart AND deleted_at IS NULL))", measurement.MeasurementDate, measurement.ID, measurement.Patient.ID, user.ID, measurement.Doctor.ID).Order("measurement_date ASC").Find(&lateMeasurements)
			//c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ? and measurement_id = (SELECT id FROM measurements WHERE measurement_date >= ? AND id != ? AND patient_id = ? AND user_id = ?)", measurement.Doctor.ID, measurement.MeasurementDate, measurement.ID, measurement.Patient.ID, user.ID).Order("measurement_date ASC").Find(&lateMeasurementsShared)

			db := c.ormDB.Debug().Where("measurement_shareds.doctor_id = ? AND measurements.user_id = ? AND measurements.patient_id = ? AND measurement_date >= ? AND measurements.id != ?", measurement.DoctorId, user.ID, measurement.PatientId, measurement.MeasurementDate, measurement.ID)
			db = db.Joins("LEFT JOIN measurements ON measurements.id = measurement_shareds.measurement_id")
			db.Set("gorm:auto_preload", true).Order("measurement_date ASC").Find(&lateMeasurementsShared)

			for _, lateMeasurementShared := range lateMeasurementsShared {
				tenMinApart := false

				if lateMeasurementShared.Measurement.MeasurementDate.Time.Unix()-lastGoodMeasurement.MeasurementDate.Time.Unix() > 600 {
					lastGoodMeasurement = lateMeasurementShared.Measurement
					tenMinApart = true
				}

				//Soll nur den Datensatz anpassen, wenn es eine Änderung gibt.
				if lateMeasurementShared.TenMinsApart != tenMinApart {
					c.ormDB.Model(&MeasurementShared{}).Where("id = ?", lateMeasurementShared.ID).Update("ten_mins_apart", tenMinApart)
				}
			}
		}

	} else {

	}
}

// getAnnotationsForMeasurement swagger:route GET /measurements/{measurementId}/annotations measurements getAnnotationsForMeasurement
//
// retrieves annotations for measurement
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
//			data: []Annotation
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetAnnotationsForMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	annotations := Annotations{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id = ?)", patient.ID).Where("measurement_id=?", measurementId).Find(&annotations)
	} else if c.isDoctor(user) {
		//Alle Doctoren von der gleichen Praxis sollen alle Notizen sehen
		doctor := c.getDoctor(user)
		practice := c.getPracticeOfDoctor(doctor.ID)
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status=2))", practice.ID).Where("measurement_id=?", measurementId).Find(&annotations)
		//c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2))", doctor.ID).Where("measurement_id=?", measurementId).Find(&annotations)
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status=2))", practice.ID).Where("measurement_id=?", measurementId).Find(&annotations)
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id=?", measurementId).Find(&annotations)
	}

	for key, annotation := range annotations {
		helperUser := HelperUser{}
		helperUser.User = annotation.User
		log.Println(helperUser.UserType)
		if c.isPatient(&helperUser.User) {
			log.Println(annotation.ID)
			helperUser.Patient = c.getPatient(&helperUser.User)
			log.Println(helperUser.Patient)
			//&helperUser.Patient.User = nil
		} else if c.isDoctor(&helperUser.User) {
			helperUser.Doctor = c.getDoctor(&helperUser.User)
			log.Println(helperUser.Doctor)
		} else if c.isPractice(&helperUser.User) {
			helperUser.Practice = c.getPractice(&helperUser.User)
			log.Println(helperUser.Practice)
		}
		annotation.HelperUser = helperUser
		annotations[key] = annotation
	}

	c.SendJSON(w, &annotations, http.StatusOK)
}

// deleteAnnotation swagger:route DELETE /measurement/{measurementId}/annotations/{annotationId} measurements deleteAnnotation
//
// delete a measurement
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
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) DeleteAnnotationForMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)
	annotationId, _ := strconv.ParseInt(vars["annotationId"], 10, 64)

	annotation := Annotation{}

	if annotationId > 0 {
		if c.isPatient(user) {
			patient := c.getPatient(user)
			// c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id = ?)", patient.ID).Where("measurement_id=?", measurementId).Delete(&annotation, annotationId)

			c.ormDB.Set("gorm:auto_preload", true).Model(&annotation).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id = ?)", patient.ID).Where("measurement_id=? AND id=?", measurementId, annotationId).Update("is_deleted", true)

		} else if c.isDoctor(user) {
			doctor := c.getDoctor(user)
			//c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2))", doctor.ID).Where("measurement_id=?", measurementId).Delete(&annotation, annotationId)

			c.ormDB.Set("gorm:auto_preload", true).Model(&annotation).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2))", doctor.ID).Where("measurement_id=? AND id=?", measurementId, annotationId).Update("is_deleted", true)

		} else if c.isPractice(user) {
			practice := c.getPractice(user)
			//c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id=? AND consent_status=2))", doctor.ID).Where("measurement_id=?", measurementId).Delete(&annotation, annotationId)

			c.ormDB.Set("gorm:auto_preload", true).Model(&annotation).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status=2))", practice.ID).Where("measurement_id=? AND id=?", measurementId, annotationId).Update("is_deleted", true)

		} else if user.UserType == 0 {
			//c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id=?", measurementId).Delete(&annotation, annotationId)

			c.ormDB.Set("gorm:auto_preload", true).Model(&annotation).Where("measurement_id=? AND id=?", measurementId, annotationId).Update("is_deleted", true)

		}
	}

	c.SendJSON(w, "", http.StatusOK)
}

// saveAnnotationForMeasurement swagger:route POST /measurements/{measurementId}/annotations annotations saveAnnotationForMeasurement
//
// saves an annotation for a measurement
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
//			data: Message
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) SaveAnnotationForMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User

	if ok, user = c.Controller.GetUser(w, r); !ok || user.UserType == 0 {
		_ = user
		return
	}

	annotation := &Annotation{}
	if err := c.GetContent(&annotation, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if strings.TrimSpace(annotation.Content) == "" {
		err := errors.New("Please enter an annotation")
		c.HandleError(err, w)
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	annotationDB := &Annotation{}
	c.ormDB.Debug().Set("gorm:auto_preload", false).Where("measurement_id = ? AND annotation_time = ? AND user_id = ?", measurementId, annotation.AnnotationTime, annotation.HelperUser.ID).First(&annotationDB)
	if annotationDB.ID > 0 {
		c.SendJSON(w, &annotation, http.StatusOK)
		return
	}
	measurement := Measurement{}
	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).First(&measurement, measurementId)
	} else if c.isDoctor(user) {
		//doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement, measurementId)
		//Where("doctor_id = ?", doctor.ID) funktioniert nicht, da muss auf consent geguckt werden
	} else if c.isPractice(user) {
		//doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement, measurementId)
		//Where("doctor_id = ?", doctor.ID) funktioniert nicht, da muss auf consent geguckt werden
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement, measurementId)
	}

	// check if measurement exists and user can add annotation (has permission)

	if measurement.ID > 0 {
		annotation.MeasurementId = measurement.ID
		if c.isPatient(user) || c.isDoctor(user) {
			annotation.UserId = user.ID
			annotation.User = *user
		} else if c.isPractice(user) {

			if annotation.DoctorId > 0 {
				c.ormDB.Set("gorm:auto_preload", false).Where("id IN (SELECT user_id FROM doctor_users WHERE doctor_id = ?)", annotation.DoctorId).Find(&annotation.User)
			} else {
				annotation.User = *user
			}

			annotation.UserId = annotation.User.ID
		}
		c.ormDB.Set("gorm:save_associations", false).Save(&annotation)
		log.Println(user)

		helperUser := HelperUser{}
		helperUser.User = annotation.User
		if c.isPatient(&annotation.User) {
			helperUser.Patient = c.getPatient(user)
		} else if c.isDoctor(&annotation.User) {
			helperUser.Doctor = c.getDoctor(user)
		} else if c.isPractice(&annotation.User) {
			helperUser.Practice = c.getPractice(user)
		}
		annotation.HelperUser = helperUser

		c.SendJSON(w, &annotation, http.StatusOK)
		return
	} else {
		err := errors.New("This is not your scan")
		c.HandleError(err, w)
		return
	}

}

// shareMeasurement swagger:route POST /measurement/share measurements shareMeasurement
//
// share a measurement
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
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) ShareMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}
	if !c.isPatient(user) && !c.isSysadmin(user) && !c.isDoctor(user) {
		err := errors.New("You are not allowed to share a scan")
		c.HandleError(err, w)
		return
	}

	sharedMeasurement := &MeasurementShared{}

	if err := c.GetContent(&sharedMeasurement, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	c.ormDB.Set("gorm:save_associations", false).Create(&sharedMeasurement)

	c.SendJSON(w, sharedMeasurement, http.StatusOK)
}

// deleteMeasurement swagger:route DELETE /measurement/{measurementId} measurements deleteMeasurement
//
// delete a measurement
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
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) DeleteSharedMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementSharedId, _ := strconv.ParseInt(vars["measurementSharedId"], 10, 64)

	measurementShared := MeasurementShared{}

	if measurementSharedId > 0 {
		if c.isPatient(user) {
			patient := c.getPatient(user)
			c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id IN (SELECT id FROM measurements WHERE patient_id = ?)", patient.ID).Delete(&measurementShared, measurementSharedId)
		} else if c.isDoctor(user) {
			doctor := c.getDoctor(user)
			c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id = ?", doctor.ID).Delete(&measurementShared, measurementSharedId)

		} else if c.isPractice(user) {
			practice := c.getPractice(user)
			c.ormDB.Set("gorm:auto_preload", true).Where("doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?)", practice.ID).Delete(&measurementShared, measurementSharedId)

		} else if user.UserType == 0 {
			c.ormDB.Set("gorm:auto_preload", true).Delete(&measurementShared, measurementSharedId)
		}
	}

	c.SendJSON(w, nil, http.StatusOK)
}

// shareMeasurement swagger:route POST /measurement/share measurements shareMeasurement
//
// share a measurement
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
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) ToggleMeasurementFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}
	/*
		if !c.isDoctor(user) {
			err := errors.New("You are not allowed to favorite a scan")
			c.HandleError(err, w)
			return
		}
	*/

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	favorite := &MeasurementFavorite{}

	c.ormDB.Where("measurement_id = ? AND user_id = ?", measurementId, user.ID).Find(&favorite)

	if favorite.ID > 0 {
		c.ormDB.Delete(&favorite)
	} else {
		favorite.UserId = user.ID
		favorite.MeasurementId = uint(measurementId)
		c.ormDB.Set("gorm:save_associations", false).Create(&favorite)

	}

	c.SendJSON(w, favorite, http.StatusOK)
}

func CalculateRewardForPatient(db *gorm.DB, patientId uint, save bool) (*PatientReward, int, int) {
	reward := &PatientReward{}
	reward.PatientId = patientId
	reward.RewardDate = core.NullTime{time.Now(), true}
	oldReward := &PatientReward{}
	db.Set("gorm:auto_preload", true).Where("patient_id = ?", patientId).Order("reward_date DESC").First(&oldReward)

	countConsecutiveScans := 0
	countScans := 0

	scans := &Measurements{}
	date := core.NullTime{time.Now(), true}

	db.Set("gorm:auto_preload", false).Where("patient_id = ?", patientId).Order("measurement_date DESC").Find(&scans)
	countScans = len(*scans)
	log.Println(countScans)

	db.Set("gorm:auto_preload", false).Where("patient_id = ?", patientId).Group("DATE(measurement_date)").Order("measurement_date DESC").Find(&scans)
	for _, scan := range *scans {
		log.Println(scan.ID)
		if DateEqual(date.Time, scan.MeasurementDate.Time) {
			countConsecutiveScans++
			date.Time = date.Time.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	log.Println(countConsecutiveScans)
	log.Println(countScans)

	db.Where("consecutive_scans_threshold <= ?", countConsecutiveScans).Order("consecutive_scans_threshold ASC").First(&reward.RewardStarRating)
	db.Where("consecutive_scans_threshold <= ?", countConsecutiveScans).Order("consecutive_scans_threshold ASC").First(&reward.RewardMonetaryDiscount)
	db.Where("scans_threshold <= ?", countScans).Order("scans_threshold ASC").First(&reward.RewardUserLevel)

	if oldReward.RewardStarRating.ConsecutiveScansThreshold > reward.RewardStarRating.ConsecutiveScansThreshold {
		reward.RewardStarRating = oldReward.RewardStarRating
	}
	if save {
		db.Set("gorm:save_associations", false).Create(&reward)
	}
	return reward, countScans, countConsecutiveScans
}

func GetDate(toRound time.Time) time.Time {
	return time.Date(toRound.Year(), toRound.Month(), toRound.Day(), 0, 0, 0, 0, toRound.Location())
}

func DateEqual(date1, date2 time.Time) bool {
	log.Println(date1)
	log.Println(date2)
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func (c *PodiumController) 	ExportMeasurementHandler(w http.ResponseWriter, r *http.Request) {
    println("ExportMeasurementHandler", r.URL.Path)
    ok := false
    var user *core.User
    if ok, user = c.GetUser(w, r); !ok {
        _ = user
        return
    }

    vars := mux.Vars(r)
    measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

    // Print statement to check measurement ID
    fmt.Println("Measurement ID:", measurementId)

    timeOffset := int64(0)
    hTimeOffset := r.Header["X-Timezone-Offset"]
    log.Println(hTimeOffset)
    if len(hTimeOffset) > 0 {
        tmp, err := strconv.Atoi(hTimeOffset[0])
        if err == nil {
            timeOffset = int64(tmp) * 60
        }
    }

    measurement := Measurement{}

    if c.isPatient(user) {
        patient := c.getPatient(user)
        c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).First(&measurement, measurementId)
    } else if c.isDoctor(user) {
        doctor := c.getDoctor(user)

        practice := doctor.GetPractice(c.ormDB)
        c.ormDB.Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status>=2)", practice.ID).First(&measurement, measurementId)
    } else if c.isPractice(user) {
        practice := c.getPractice(user)
        c.ormDB.Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status>=2)", practice.ID).First(&measurement, measurementId)
    } else if user.UserType == 0 {
        c.ormDB.Set("gorm:auto_preload", true).First(&measurement, measurementId)
    }

    // Print statement to check measurement data after retrieval
    fmt.Println("Measurement:", measurement)

    c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)

    if measurement.MeasurementFiles == nil {
        measurement.MeasurementFiles = make([]MeasurementFile, 0)
    }

    // Print statement to check measurement data before PDF creation
    fmt.Println("Measurement before PDF creation:", measurement)

    timestamp := measurement.MeasurementDate.Time.Unix()
    timestamp = timestamp - timeOffset

    measurement.MeasurementDate.Time = time.Unix(timestamp, 0)
    fileName := c.createMeasurementPDF(measurement, timeOffset)

    log.Println(fileName)

    // Print statement to check the file name created for the PDF
    fmt.Println("PDF File Name:", fileName)

    w.Header().Set("Content-Disposition", `inline; filename="Scan `+strconv.Itoa(int(measurement.ID))+`.pdf"`)
    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Add("Access-Control-Allow-Origin", "*")

    http.ServeFile(w, r, fileName)
}

type HelperSendMail struct {
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

func (c *PodiumController) SendMailMeasurementHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	helperSendMail := HelperSendMail{}
	if err := c.GetContent(&helperSendMail, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	measurement := Measurement{}

	if c.isPatient(user) {
		patient := c.getPatient(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id = ?", patient.ID).First(&measurement, measurementId)
	} else if c.isDoctor(user) {
		doctor := c.getDoctor(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id = ? AND consent_status>=2)", doctor.ID).Where("id IN (SELECT measurement_id FROM measurement_shareds WHERE deleted_at IS NULL AND doctor_id = ?)", doctor.ID).First(&measurement, measurementId)
	} else if c.isPractice(user) {
		practice := c.getPractice(user)
		c.ormDB.Set("gorm:auto_preload", true).Where("patient_id IN (SELECT patient_id FROM doctor_patient_relations WHERE doctor_id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id = ?) AND consent_status>=2)", practice.ID).First(&measurement, measurementId)
	} else if user.UserType == 0 {
		c.ormDB.Set("gorm:auto_preload", true).First(&measurement, measurementId)
	}

	// TEMP
	//c.ormDB.Set("gorm:auto_preload", true).First(&measurement, measurementId)

	c.ormDB.Set("gorm:auto_preload", true).First(&measurement.Patient, measurement.PatientId)

	if measurement.MeasurementFiles == nil {
		measurement.MeasurementFiles = make([]MeasurementFile, 0)
	}

	timeOffset := int64(0)
	hTimeOffset := r.Header["X-Timezone-Offset"]
	log.Println(hTimeOffset)
	if len(hTimeOffset) > 0 {
		tmp, err := strconv.Atoi(hTimeOffset[0])
		if err == nil {
			timeOffset = int64(tmp) * 60
		}
	}
	measurement.MeasurementDate.Time = time.Unix(measurement.MeasurementDate.Time.Unix()-timeOffset, 0)
	fileName := c.createMeasurementPDF(measurement, timeOffset)

	//http.ServeFile(w, r, fileName)

	podiumMail := PodiumMail{
		Subject: helperSendMail.Subject,
		Body:    helperSendMail.Body,
		From:    "info@podium.care", //user.Email,
		ReplyTo: user.Email,
		To:      helperSendMail.To,
		Cc:      helperSendMail.Cc,
		Bcc:     helperSendMail.Bcc,
		/*SmtpHost:     "mail.podium.care",
		SmtpPort:     "465",
		SmtpUsername: "info@podium.care",
		SmtpPassword: "w-fDhrC4e", //"bNrpyQqVAxgJgZ6z",*/
	}

	if len(helperSendMail.To) > 0 {

		if !SendMailAttach(podiumMail, []string{fileName}) {
			err := errors.New("No recipient")
			if c.HandleError(err, w) {
				return
			}
		}

		c.SendJSON(w, "ok", http.StatusOK)
	} else {
		err := errors.New("No recipient")
		if c.HandleError(err, w) {
			return
		}
	}
}

type PodiumMail struct {
	core.Model
	Subject string   `json:"subject"`
	Body    string   `json:"body" gorm:"TYPE:LONGTEXT"`
	From    string   `json:"from"`
	ReplyTo string   `json:"reply_to"`
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	/*
		SmtpHost     string   `json:"smtp_host"`
		SmtpPort     string   `json:"smtp_port"`
		SmtpUsername string   `json:"smtp_username"`
		SmtpPassword string   `json:"smtp_password"`*/

	Errors map[string]string `json:"-" gorm:"-"`
}
type PodiumMails []PodiumMail

func SendMailAttach(podiumMail PodiumMail, attachments []string) bool {
	if err := core.SendMail(podiumMail.From, parseAddresses(podiumMail.To), parseAddresses(podiumMail.Cc), parseAddresses(podiumMail.Bcc), podiumMail.Subject, podiumMail.Body, attachments); err != nil {
		log.Println(err)
		return false
	} else {
		log.Println("Done Mail")
		return true
	}
}

func parseAddresses(addresses []string) []string {
	newAddresses := make([]string, 0)
	for _, address := range addresses {
		if address != "" {
			newAddresses = append(newAddresses, strings.Split(address, ";")...)
		}
	}
	return newAddresses
}

func (c *PodiumController) createMeasurementPDF(scan Measurement, timeOffset int64) string {

	//X tmpPath := "tmp/" + core.RandomString(10) + "/"
	tmpPath := c.GetTmpUploadPath()
	os.MkdirAll(tmpPath, 0777)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)

	pdf.SetFooterFuncLpi(func(lastPage bool) {
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 10, scan.Patient.FirstName+" "+scan.Patient.LastName+"", "", 0, "L", false, 0, "")
		pdf.SetY(-12)
		//pdf.CellFormat(0, 10, "Scan of "+scan.MeasurementDate.Time.Format("02/01/2006"), "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 10, "Scan on "+scan.MeasurementDate.Time.Format("02/01/2006")+" @ "+scan.MeasurementDate.Time.Format("15:04"), "", 0, "L", false, 0, "")

		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d of %d                            ", pdf.PageNo(), 2), "", 0, "C", false, 0, "")
	})

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 18)

	pdf.Text(10, 15, "Scan on "+scan.MeasurementDate.Time.Format("02/01/2006")+" @ "+scan.MeasurementDate.Time.Format("15:04"))

	pdf.Image("logo_podium.png", 140, 3, 50, 0, false, "", 0, "")

	x := 0.0
	pdf.SetFont("Arial", "", 12)
	y := 22.0
	text := ""
	if scan.Patient.LastName != "" {
		if scan.Patient.FirstName != "" {
			text += fmt.Sprintf("Patient: %s %s", scan.Patient.LastName, scan.Patient.FirstName)
		} else {
			text += fmt.Sprintf("Patient: %s", scan.Patient.LastName)
		}
	} else {
		patientUser := core.User{}
		c.ormDB.Set("gorm:auto_preload", true).First(&patientUser, scan.Patient.UserId)
		text = fmt.Sprintf("Patient: %s", patientUser.Username)
	}
	if text != "" {
		pdf.Text(10, y, text)
		y += 6
	}

	text = ""
	c.ormDB.Set("gorm:auto_preload", true).First(&scan.Patient, scan.PatientId)
	if scan.DoctorId > 0 {
		doc := Doctor{}
		c.ormDB.Set("gorm:auto_preload", true).First(&doc, scan.DoctorId)
		docName := ""
		if doc.LastName != "" {
			docName = fmt.Sprintf("%s %s", doc.FirstName, doc.LastName)
		} else {
			docUser := core.User{}
			c.ormDB.Set("gorm:auto_preload", true).Where("id=(SELECT user_id FROM doctor_users WHERE doctor_id=?)", doc.ID).First(&docUser)
			docName = docUser.Username
		}
		if text != "" {
			text += fmt.Sprintf("scanned by Clinician: %s", docName)
		} else {
			text += fmt.Sprintf("scanned by Clinician: %s", docName)
		}
	}
	if text != "" {
		pdf.Text(10, y, text)
		y += 6
	}
	text = ""
	practice := Practice{}
	c.ormDB.Set("gorm:auto_preload", true).Where("user_id=?", scan.UserId).First(&practice)
	if practice.ID > 0 {
		if practice.Name != "" {
			text += fmt.Sprintf("at Practice: %s", practice.Name)
		} else {
			practiceUser := core.User{}
			c.ormDB.Set("gorm:auto_preload", true).First(&practiceUser, practice.UserId)
			text += fmt.Sprintf("at Practice: %s", practiceUser.Username)
		}
	}
	if text != "" {
		pdf.Text(10, y, text)
		y += 6
	}

	y += 3

	//Patient: [name], scanned by Podiatrist: [name] at Practice: [name]
	/*text := ""
	if scan.Patient.LastName != "" {
		if scan.Patient.FirstName != "" {
			text += fmt.Sprintf("Patient: %s %s", scan.Patient.LastName, scan.Patient.FirstName)
		} else {
			text += fmt.Sprintf("Patient: %s", scan.Patient.LastName)
		}
	} else {
		patientUser := core.User{}
		c.ormDB.Set("gorm:auto_preload", true).First(&patientUser, scan.Patient.UserId)
		text = fmt.Sprintf("Patient: %s", patientUser.Username)
	}

	c.ormDB.Set("gorm:auto_preload", true).First(&scan.Patient, scan.PatientId)
	if scan.DoctorId > 0 {
		doc := Doctor{}
		c.ormDB.Set("gorm:auto_preload", true).First(&doc, scan.DoctorId)
		docName := ""
		if doc.LastName != "" {
			docName = fmt.Sprintf("%s %s", doc.FirstName, doc.LastName)
		} else {
			docUser := core.User{}
			c.ormDB.Set("gorm:auto_preload", true).Where("id=(SELECT user_id FROM doctor_users WHERE doctor_id=?)", doc.ID).First(&docUser)
			docName = docUser.Username
		}
		if text != "" {
			text += fmt.Sprintf(", scanned by Podiatrist: %s", docName)
		} else {
			text += fmt.Sprintf("scanned by Podiatrist: %s", docName)
		}
	}

	practice := Practice{}
	c.ormDB.Set("gorm:auto_preload", true).Where("user_id=?", scan.UserId).First(&practice)
	if practice.ID > 0 {
		if practice.Name != "" {
			text += fmt.Sprintf(" at Practice: %s", practice.Name)
		} else {
			practiceUser := core.User{}
			c.ormDB.Set("gorm:auto_preload", true).First(&practiceUser, practice.UserId)
			text += fmt.Sprintf(" at Practice: %s", practiceUser.Username)
		}
	}

	pdf.Text(10, 22, text)*/

	pdf.SetFont("Arial", "B", 14)

	xShift := float64(0)
	files := make([]MeasurementFile, 5)

	for _, file := range scan.MeasurementFiles {

		if file.MeasurementType == "THERMAL" {
			file.Label = "THERMAL (Absolute scale)"
			files[1] = file

		} else if file.MeasurementType == "DYNAMIC" {
			file.Label = "THERMAL (Dynamic scale)"
			files[2] = file

		} else if file.MeasurementType == "STATISTIC" {
			file.Label = "Histogram (Absolute scale)"
			files[3] = file

		} else if file.MeasurementType == "DYNAMIC_STATISTIC" {
			file.Label = "Histogram (Dynamic scale)"
			files[4] = file

		} else if file.MeasurementType == "NORMAL" {
			file.Label = file.MeasurementType
			files[0] = file
		} else {

		}
	}
	for i := len(files) - 1; i >= 0; i-- {
		_, err := os.Stat(files[i].GetAbsolutePath())
		if files[i].ID == 0 || err != nil {
			files = append(files[:i], files[i+1:]...)
		}
	}

	/*	for _, file := range scan.MeasurementFiles {
		// TODO Remove once RP wants it added again
		//'NORMAL', 'THERMAL', 'REPORT', 'STATISTIC', 'DYNAMIC', 'T0', 'INPUT', 'FOOT_POSITIONING', 'DYNAMIC_STATISTIC'
		if file.MeasurementType == "NORMAL" || file.MeasurementType == "THERMAL" || file.MeasurementType == "DYNAMIC" || file.MeasurementType == "STATISTIC" || file.MeasurementType == "DYNAMIC_STATISTIC" { // != "REPORT"
			files = append(files, file)
		}
	}*/
	pdf.SetFont("Arial", "B", 12)
	for i, scanFile := range files {
		if len(files) > 3 && i == 2 {
			y += 65
			xShift = -120
		}
		x = float64(10+60*i) + xShift
		pdf.Text(x, y, strings.ReplaceAll(scanFile.Label, "_", " "))
		pdf.Image(scanFile.GetAbsolutePath(), x, y+3, 50, 0, false, "", 0, "")
	}
	pdf.SetFont("Arial", "B", 14)
	y += 65

	if false {
		pdf.Text(15, y, "Scan rating: ")
		pdf.Image("icon_smiley_happy.png", 45, y-6, 8, 0, false, "", 0, "")
		y += 10
	}
	questionnaire := &PatientQuestionnaire{}

	c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id =?", scan.ID).Find(&questionnaire)

	setupQuestionnaire := &PatientQuestionnaire{}

	c.ormDB.Debug().Set("gorm:auto_preload", true).Select("patient_questionnaires.*, ABS(TIMESTAMPDIFF(SECOND, ?, questionnaire_date)) as SecondsBetweenDates", scan.MeasurementDate).Where("questionnaire_date <= ?", scan.MeasurementDate).Where("patient_id = ?", scan.PatientId).Where("id IN (SELECT patient_questionnaire_id FROM patient_questionnaire_questions pqq LEFT JOIN question_templates qt ON pqq.template_question_id = qt.id WHERE qt.question_type = 1 AND pqq.answer_id > 0)").Order("SecondsBetweenDates ASC").First(&setupQuestionnaire)

	_, maxY := pdf.GetPageSize()
	if len(setupQuestionnaire.Questions) > 0 {
		pdf.Text(15, y, "Setup-Questions")
		y += 7
		for i, q := range setupQuestionnaire.Questions {
			pdf.SetFont("Arial", "", 11)
			pdf.Text(15, y, fmt.Sprintf("SQ%d: %s", (i+1), q.TemplateQuestion.QuestionText))
			pdf.SetFont("Arial", "B", 11)
			pdf.Text(15, y+5, fmt.Sprintf("A%d: %s", (i+1), q.Answer.AnswerText))
			y += 12

			if y > maxY-20 {
				pdf.AddPage()
				y = 15
			}
		}
	}

	pdf.SetFont("Arial", "B", 14)
	pdf.Text(15, y, "Questions")
	y += 7
	_, maxY = pdf.GetPageSize()
	pageAdded := false
	for i, q := range questionnaire.Questions {

		pdf.SetFont("Arial", "", 11)
		pdf.Text(15, y, fmt.Sprintf("Q%d: %s", (i+1), q.TemplateQuestion.QuestionText))
		pdf.SetFont("Arial", "B", 11)
		pdf.Text(15, y+5, fmt.Sprintf("A%d: %s", (i+1), q.Answer.AnswerText))
		y += 12

		if y > maxY-20 {
			pdf.AddPage()
			pageAdded = true
			y = 15
		}
	}

	annotations := Annotations{}

	c.ormDB.Set("gorm:auto_preload", true).Where("measurement_id=?", scan.ID).Group("annotation_time").Find(&annotations)

	for key, annotation := range annotations {
		helperUser := HelperUser{}
		helperUser.User = annotation.User
		if c.isPatient(&annotation.User) {
			helperUser.Patient = c.getPatient(&annotation.User)
			//&helperUser.Patient.User = nil
		} else if c.isDoctor(&annotation.User) {
			helperUser.Doctor = c.getDoctor(&annotation.User)
		} else if c.isPractice(&annotation.User) {
			helperUser.Practice = c.getPractice(&annotation.User)
		}

		annotation.HelperUser = helperUser
		annotations[key] = annotation
	}

	if !pageAdded {
		pdf.AddPage()
		y = 15
	} else {
		y += 15
	}

	pdf.SetFont("Arial", "B", 14)
	pdf.Text(15, y, "Notes")
	y += 7
	for _, annotation := range annotations {
		pdf.SetFont("Arial", "", 11)
		annotationUser := ""
		if annotation.HelperUser.Doctor != nil {
			annotationUser = annotation.HelperUser.Doctor.FirstName + " " + annotation.HelperUser.Doctor.LastName
		} else if annotation.HelperUser.Patient != nil {
			annotationUser = annotation.HelperUser.Patient.FirstName + " " + annotation.HelperUser.Patient.LastName
		} else if annotation.HelperUser.Practice != nil {
			annotationUser = annotation.HelperUser.Practice.Name
		}

		annotationTimestamp := annotation.AnnotationTime.Time.Unix()
		annotationTimestamp = annotationTimestamp - timeOffset
		annotation.AnnotationTime.Time = time.Unix(annotationTimestamp, 0)

		pdf.Text(15, y, fmt.Sprintf("%s - %s", annotation.AnnotationTime.Time.Format("02/01/2006")+" @ "+annotation.AnnotationTime.Time.Format("15:04"), annotationUser))
		pdf.SetFont("Arial", "B", 11)
		pdf.Text(15, y+5, fmt.Sprintf("%s", annotation.Content))
		y += 12
	}

	fileName := fmt.Sprintf(tmpPath + fmt.Sprintf("exportScan%d.pdf", scan.ID))
	err := pdf.OutputFileAndClose(fileName)
	if err != nil {
		log.Println(err)
	}

	return fileName
}

// DeleteChatMessage swagger:route DELETE /me/conversations/{userId} chats deleteChatMessage
//
// retrieves all Messages of a Chat
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
//	+ name: userId
//    in: path
//    description: the ID of the other conversation partner
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: []Message
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) SaveMeasurementRiskRatingHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.Controller.GetUser(w, r); !ok {
		_ = user
		return
	}

	vars := mux.Vars(r)
	measurementId, _ := strconv.ParseInt(vars["measurementId"], 10, 64)

	if !c.isDoctor(user) {
		err := errors.New("You are not allowed to change the risks!")
		c.HandleError(err, w)
		return
	}

	doctor := c.getDoctor(user)

	measurementDoctorRisk := MeasurementDoctorRisk{}
	if err := c.GetContent(&measurementDoctorRisk, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}

	if c.ormDB.NewRecord(&measurementDoctorRisk) {
		measurementDoctorRisk.DoctorId = doctor.ID
		measurementDoctorRisk.MeasurementId = uint(measurementId)
		c.ormDB.Set("gorm:save_associations", false).Create(&measurementDoctorRisk)
	} else {
		measurementDoctorRiskDB := MeasurementDoctorRisk{}
		c.ormDB.Set("gorm:auto_preload", true).First(&measurementDoctorRiskDB, measurementDoctorRisk.ID)
		measurementDoctorRiskDB.RiskDefinition = measurementDoctorRisk.RiskDefinition
		measurementDoctorRiskDB.RiskDefinitionId = measurementDoctorRisk.RiskDefinition.ID
		c.ormDB.Set("gorm:save_associations", false).Save(&measurementDoctorRiskDB)
	}

	c.SendJSON(w, &measurementDoctorRisk, http.StatusOK)

	go web3socket.SendWebsocketDataInfoMessage("Update risk of measurement", web3socket.Websocket_Add, web3socket.Websocket_Measurements, uint(measurementId), []uint{user.ID}, nil)
}
