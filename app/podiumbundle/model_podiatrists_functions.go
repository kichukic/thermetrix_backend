package podiumbundle

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"thermetrix_backend/app/core"
	"thermetrix_backend/app/websocket"
)

func (doctor *Doctor) Validate(ormDb *gorm.DB) bool {
	doctor.Errors = make(map[string]string)

	if doctor.Practice.ID == 0 {
		doctor.Errors["practice"] = "practice not found"
	}

	if doctor.Practice.AccountsAbbreviation == "" {
		doctor.Errors["accounts_abbreviation"] = "practice accounts abbreviation empty"
	}

	for _, doctorUser := range doctor.Users {
		user := doctorUser.User
		// As requested by TMX, usernames should only be unique within each practice
		if doctorUser.User.ID == 0 {
			user.Username = fmt.Sprintf("%s_%s", doctor.Practice.AccountsAbbreviation, doctorUser.User.Username)
		}
		if !user.Validate(ormDb) {
			for key, element := range user.Errors {
				doctor.Errors[key] = element
			}
		}

	}

	if len(doctor.Errors) > 0 {
		return false
	}

	return true
}

func (doctor *Doctor) Save(ormDB *gorm.DB) (bool, error) {

	isNewDoctor := true
	if doctor.ID == 0 {
		ormDB.Set("gorm:save_associations", false).Create(&doctor)
	} else {
		isNewDoctor = false
		doctorDb := Doctor{}
		ormDB.First(&doctorDb, doctor.ID)
		if doctorDb.ID == 0 {
			return false, errors.New("user not found")
		}
		ormDB.Set("gorm:save_associations", false).Save(&doctor)
	}

	// --- User account
	for key, doctorUser := range doctor.Users {
		if doctorUser.Status == 0 {
			doctorUser.Status = DoctorUserStatus_Doctor
		}
		if doctorUser.Status == DoctorUserStatus_Doctor {
			doctorUser.User.UserType = 2

			if doctor.Practice.ID == 0 {
				return false, errors.New("no practice")
			} else {
				if doctor.Practice.User.ID == 0 {
					ormDB.Preload("User").First(&doctor.Practice, doctor.Practice.ID)
				}
			}
			doctorUser.User.CreatedBy = doctor.Practice.User.ID

			if _, err := doctorUser.User.Save(ormDB); err != nil {
				return false, err
			} else {
				doctorUser.UserId = doctorUser.User.ID

				doctorUserDb := DoctorUser{}
				ormDB.Where("doctor_id=? AND user_id=?", doctorUser.DoctorId, doctorUser.UserId).First(&doctorUserDb)

				if doctorUserDb.DoctorId == 0 {
					doctorUser.DoctorId = doctor.ID
					ormDB.Set("gorm:save_associations", false).Create(&doctorUser)
				} else {
					ormDB.Set("gorm:save_associations", false).Save(&doctorUser)
				}

				doctor.Users[key] = doctorUser
			}

			if doctor.Practice.ID == 0 {
				return false, errors.New("no practice")
			}
		}
	}

	// --- Devices (many2many)
	for key, doctorDevice := range doctor.Devices {
		if _, err := doctorDevice.Device.Save(ormDB); err != nil {
			continue
		} else {
			doctorDevice.DeviceId = doctorDevice.Device.ID
		}

		doctorDeviceDb := DoctorDevice{}
		ormDB.Where("doctor_id=? AND device_id=?", doctorDevice.DoctorId, doctorDevice.DeviceId).First(&doctorDeviceDb)
		if doctorDeviceDb.DoctorId == 0 {
			doctorDevice.DoctorId = doctor.ID
			ormDB.Set("gorm:save_associations", false).Create(&doctorDevice)
		} else {
			ormDB.Set("gorm:save_associations", false).Save(&doctorDevice)
		}
		doctor.Devices[key] = doctorDevice
	}

	// --- Practice (many2many)
	if doctor.Practice.ID == 0 {
		return false, errors.New("no practice")
	} else {
		practiceDoctorDb := PracticeDoctor{}
		practiceDoctorDb.DoctorId = doctor.ID
		practiceDoctorDb.PracticeId = doctor.Practice.ID
		ormDB.Where("practice_id=? AND doctor_id=?", doctor.Practice.ID, doctor.ID).First(&practiceDoctorDb)
		if practiceDoctorDb.DoctorId == 0 {
			ormDB.Set("gorm:save_associations", false).Create(&practiceDoctorDb)
		} else {
			ormDB.Set("gorm:save_associations", false).Save(&practiceDoctorDb)
		}
	}

	// ToDo SpecialistFields

	if isNewDoctor {
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Add doctor", web3socket.Websocket_Add, web3socket.Websocket_Doctor, doctor.ID, nil)
	} else {
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Updated doctor", web3socket.Websocket_Update, web3socket.Websocket_Doctor, doctor.ID, nil)
	}

	return true, nil
}

func (patient *Patient) GetPairedDoctors(ormDB *gorm.DB) Doctors {

	doctors := Doctors{}
	ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM doctor_patient_relations WHERE consent_status = 2 AND patient_id =?)", patient.ID).Find(&doctors)

	return doctors
}

func (doctor *Doctor) GetUsers(ormDB *gorm.DB) core.Users {
	users := core.Users{}
	ormDB.Where("system_accounts.id IN (SELECT doctor_users.user_id FROM doctor_users WHERE doctor_users.doctor_id IN (?)", doctor.ID).Find(&users)
	return users
}

func (doctor *Doctor) GetPracticeUserId(ormDB *gorm.DB) uint {

	practice := doctor.GetPractice(ormDB)
	return practice.UserId
}

func (doctor *Doctor) GetPractice(ormDB *gorm.DB) Practice {
	practice := Practice{}
	ormDB.Where("practices.id IN (SELECT practice_doctors.practice_id FROM practice_doctors WHERE practice_doctors.doctor_id =?)", doctor.ID).Find(&practice)
	return practice
}

func (deviceSystemVersion *DeviceSystemVersion) GetLastDeviceTypeVersion(ormDB *gorm.DB, deviceTypeId uint) DeviceTypeVersion {
	version := DeviceTypeVersion{}
	ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT device_type_version_id FROM device_system_version_types WHERE device_system_version_id = ?) AND device_type_id = ?", deviceSystemVersion.ID, deviceTypeId).First(&version)
	return version

}

func (device *Device) Save(ormDB *gorm.DB) (bool, error) {
	device.DeviceTypeId = device.DeviceType.ID

	if device.ID == 0 {
		ormDB.Set("gorm:save_associations", false).Create(&device)
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Created device", web3socket.Websocket_Add, web3socket.Websocket_Device, device.ID, nil)
	} else {
		deviceDb := Device{}
		ormDB.First(&deviceDb, device.ID)
		if deviceDb.ID == 0 {
			return false, errors.New("user not found")
		}
		ormDB.Set("gorm:save_associations", false).Save(&device)
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Updated device", web3socket.Websocket_Update, web3socket.Websocket_Device, device.ID, nil)
	}

	return true, nil
}

func (patient *Patient) GetPractice(ormDB *gorm.DB) Practice {
	practice := Practice{}
	query := ""
	if patient.User.CreatedBy > 0 {
		query += fmt.Sprintf("user_id = %v OR ", patient.User.CreatedBy)
	}
	query += fmt.Sprintf("id IN (SELECT practice_id FROM practice_doctors WHERE doctor_id IN (SELECT doctor_id FROM doctor_patient_relations WHERE patient_id = %v AND consent_status IN (2)))", patient.ID)

	ormDB.Debug().Where(query).First(&practice)

	return practice
}

func (patient *Patient) Validate(ormDb *gorm.DB) bool {
	patient.Errors = make(map[string]string)

	if patient.Practice.ID == 0 {
		patient.Errors["practice"] = "practice not found"
	}

	if patient.Practice.AccountsAbbreviation == "" {
		patient.Errors["accounts_abbreviation"] = "practice accounts abbreviation empty"
	}

	user := patient.User
	// As requested by TMX, usernames should only be unique within each practice
	if ormDb.NewRecord(&patient.User) {
		user.Username = fmt.Sprintf("%s_%s", patient.Practice.AccountsAbbreviation, patient.User.Username)
	}

	if !user.Validate(ormDb) {
		for key, element := range user.Errors {
			patient.Errors[key] = element
		}
	}

	if len(patient.Errors) > 0 {
		return false
	}

	return true
}

func (file *MeasurementFile) GetAbsolutePath() string {
	return core.GetUploadFilepath() + file.Filepath
}
