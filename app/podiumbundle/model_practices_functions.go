package podiumbundle

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"thermetrix_backend/app/websocket"
)

func (practice *Practice) Save(ormDB *gorm.DB) (bool, error) {
	if practice.ID == 0 {
		ormDB.Set("gorm:save_associations", false).Create(&practice)
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Created practice", web3socket.Websocket_Add, web3socket.Websocket_Practice, practice.ID, nil)
	} else {
		practiceDb := Practice{}
		ormDB.First(&practiceDb, practice.ID)
		if practiceDb.ID == 0 {
			return false, errors.New("practice not found")
		}
		practice.HasSameDoctor = practiceDb.HasSameDoctor
		ormDB.Set("gorm:save_associations", false).Save(&practice)
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Updated practice", web3socket.Websocket_Update, web3socket.Websocket_Practice, practice.ID, nil)
	}

	// -- User account
	practice.User.UserType = 3
	if _, err := practice.User.Save(ormDB); err != nil {
		return false, err
	} else {
		practice.UserId = practice.User.ID
		ormDB.Set("gorm:save_associations", false).Save(&practice)
	}

	// --- Clinicians (many2many)
	for key, practiceDoctor := range practice.Doctors {
		if _, err := practiceDoctor.Doctor.Save(ormDB); err != nil {
			continue
		} else {
			practiceDoctor.DoctorId = practiceDoctor.Doctor.ID
		}

		practiceDoctorDb := PracticeDoctor{}
		ormDB.Where("practice_id=? AND doctor_id=?", practiceDoctor.PracticeId, practiceDoctor.DoctorId).First(&practiceDoctorDb)
		if practiceDoctorDb.PracticeId == 0 {
			practiceDoctor.PracticeId = practice.ID
			ormDB.Set("gorm:save_associations", false).Create(&practiceDoctor)
		} else {
			ormDB.Set("gorm:save_associations", false).Save(&practiceDoctor)
		}

		practice.Doctors[key] = practiceDoctor
	}

	// --- Devices
	for key, practiceDevice := range practice.Devices {
		if _, err := practiceDevice.Device.Save(ormDB); err != nil {
			continue
		} else {
			practiceDevice.DeviceId = practiceDevice.Device.ID
		}

		practiceDeviceDb := PracticeDevice{}
		ormDB.Where("practice_id=? AND device_id=?", practiceDevice.PracticeId, practiceDevice.DeviceId).First(&practiceDeviceDb)
		if practiceDeviceDb.PracticeId == 0 {
			practiceDevice.PracticeId = practice.ID
			ormDB.Set("gorm:save_associations", false).Create(&practiceDevice)
		} else {
			ormDB.Set("gorm:save_associations", false).Save(&practiceDevice)
		}
		practice.Devices[key] = practiceDevice
	}

	// ToDo Contracts

	return true, nil
}

func (p *Practice) Validate(ormDb *gorm.DB) bool {
	p.Errors = make(map[string]string)

	if p.AccountsAbbreviation == "" {
		p.Errors["accounts_abbreviation"] = "accounts abbreviation empty"
	}

	if p.ID == 0 {
		practiceDb := Practice{}
		ormDb.Where("name = ?", p.Name).First(&practiceDb)
		if practiceDb.ID > 0 {
			p.Errors["practice_name"] = "practice name already exists"
		}

		practiceDb = Practice{}
		ormDb.Where("accounts_abbreviation = ?", p.AccountsAbbreviation).First(&practiceDb)
		// Practice can be saved with the same abbreviation
		if practiceDb.ID > 0 {
			p.Errors["accounts_abbreviation"] = "accounts abbreviation already exists"
		}
	} else {
		practiceDb := Practice{}
		ormDb.Where("name = ? AND id != ?", p.Name, p.ID).First(&practiceDb)
		if practiceDb.ID > 0 {
			p.Errors["practice_name"] = "practice name already exists"
		}

		practiceDb = Practice{}
		ormDb.Where("accounts_abbreviation = ? AND id != ?", p.AccountsAbbreviation, p.ID).First(&practiceDb)
		// Practice can be saved with the same abbreviation
		if practiceDb.ID > 0 {
			p.Errors["accounts_abbreviation"] = "accounts abbreviation already exists"
		}
	}

	user := p.User
	// As requested by TMX, usernames should only be unique within each practice
	if user.ID == 0 {
		user.Username = fmt.Sprintf("%s_%s", p.AccountsAbbreviation, p.User.Username)
	}

	if !user.Validate(ormDb) {
		for key, element := range user.Errors {
			p.Errors[key] = element
		}
	}

	if len(p.Errors) > 0 {
		return false
	}

	return true
}

func (practice *Practice) GetDoctors(ormDB *gorm.DB) Doctors {
	doctors := Doctors{}
	ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(&doctors)

	return doctors
}

func (practice *Practice) GetDoctorIds(ormDB *gorm.DB) []uint {

	tmp := []uint{}

	//doctors := Doctors{}
	//ormDB.Set("gorm:auto_preload", true).Where("id IN (SELECT doctor_id FROM practice_doctors WHERE practice_id =?)", practice.ID).Find(tmp)
	return tmp
}
