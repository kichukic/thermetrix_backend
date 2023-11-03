package systembundle

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/tealeg/xlsx"
	"log"
	"strconv"
	"strings"
	"thermetrix_backend/app/core"
	"thermetrix_backend/app/podiumbundle"
	"time"
)

var genderNumbers = map[string]string{}

const (
	ExcelSheet_Practice = "Practice"
	ExcelSheet_Doctors  = "Podiatrists"
	ExcelSheet_Patients = "Patients"
	ExcelSheet_Settings = "Settings"

	ExcelHeader_Abbreviation_r = "Abbreviation*"
	ExcelHeader_Name_r         = "Name*"
	ExcelHeader_Postcode_r     = "Postcode*"
	ExcelHeader_Username_r     = "Username*"
	ExcelHeader_Password_r     = "Password*"
	ExcelHeader_Email_r        = "Email*"
	ExcelHeader_UseForPod_r    = "Use for Pod1? (Y/N)*"

	ExcelHeader_FirstName              = "FirstName"
	ExcelHeader_LastName               = "LastName"
	ExcelHeader_Gender                 = "Gender"
	ExcelHeader_Email                  = "Email"
	ExcelHeader_Phone                  = "Phone"
	ExcelHeader_Postcode               = "Postcode"
	ExcelHeader_Town                   = "Town"
	ExcelHeader_Country                = "Country"
	ExcelHeader_AddressLine1           = "AddressLine1"
	ExcelHeader_AddressLine2           = "AddressLine2"
	ExcelHeader_StandardWelcomeMessage = "StandardWelcomeMessage"

	ExcelHeader_BirthDate = "BirthDate"
	ExcelHeader_County    = "County"

	LogoType_MainLogo             = "main_logo"
	LogoType_MainLogoWhite        = "main_logo_white"
	LogoType_PartnerLogo          = "partner_logo"
	LogoType_LoginBackground      = "login_background"
	LogoType_SmallLoginBackground = "small_login_background"
)

func (c *SystemController) addGenders() {
	genderNumbers["male"] = "0"
	genderNumbers["female"] = "1"
	genderNumbers["intersex"] = "2"
	genderNumbers["prefer not to say"] = "3"
}

func (c *SystemController) importWholePracticeFromExcel(fileName string) (podiumbundle.Practice, []string, error) {
	c.addGenders()

	importErrors := []string{}

	log.Println(fileName)
	f, err := xlsx.OpenFile(fileName)
	if err != nil {
		return podiumbundle.Practice{}, nil, err
	}

	practiceSheet := f.Sheet[ExcelSheet_Practice]
	patientsSheet := f.Sheet[ExcelSheet_Patients]
	settingsSheet := f.Sheet[ExcelSheet_Settings]
	if practiceSheet == nil || patientsSheet == nil || settingsSheet == nil {
		err := errors.New("datasheet not found")
		log.Println(err)
		return podiumbundle.Practice{}, nil, err
	}

	// Get settings from settings tab
	// genders, err = getSetting(f.Sheet[DataSheet_Settings], "Gender")

	newPractice, newUser, newDoctor, newUserForDoctor, importErrorsPractice := getPractice(c.ormDB, practiceSheet)
	newDoctors, newDoctorUsers, importErrorsDoctor := getDoctors(c.ormDB, practiceSheet, 6, &newPractice)
	newPatients, newPatientUsers, importErrorsPatients := getPatients(c.ormDB, patientsSheet, 1, &newPractice)

	if importErrorsPractice != nil && len(importErrorsPractice) > 0 {
		importErrors = append(importErrors, importErrorsPractice...)
	}

	if importErrorsDoctor != nil && len(importErrorsDoctor) > 0 {
		importErrors = append(importErrors, importErrorsDoctor...)
	}

	if importErrorsPatients != nil && len(importErrorsPatients) > 0 {
		importErrors = append(importErrors, importErrorsPatients...)
	}

	if len(importErrors) > 0 {
		return podiumbundle.Practice{}, importErrors, nil
	} else {
		// Create new Practice and corresponding doctors
		c.ormDB.Set("gorm:save_associations", false).Save(&newUser)
		newPractice.UserId = newUser.ID
		c.ormDB.Set("gorm:save_associations", false).Save(&newPractice)

		//CAR-0077 Chagne no more Practice and Podiatrist, with the same username
		newPractice.HasSameDoctor = false
		if newPractice.HasSameDoctor {
			newUserForDoctor.CreatedBy = newPractice.UserId
			c.ormDB.Set("gorm:save_associations", false).Save(&newUserForDoctor)
			c.ormDB.Set("gorm:save_associations", false).Save(&newDoctor)
			newDoctorUser := podiumbundle.DoctorUser{}
			newDoctorUser.UserId = newUserForDoctor.ID
			newDoctorUser.DoctorId = newDoctor.ID
			newDoctorUser.Status = 1
			c.ormDB.Set("gorm:save_associations", false).Save(&newDoctorUser)
			newPracticeDoctor := podiumbundle.PracticeDoctor{}
			newPracticeDoctor.PracticeId = newPractice.ID
			newPracticeDoctor.DoctorId = newDoctor.ID
			//ToDo Allow_Practice_Login always == 0 ??? RS
			c.ormDB.Set("gorm:save_associations", false).Save(&newPracticeDoctor)
		}
		c.savePatients(newUser.ID, newPatients, newPatientUsers)

		c.saveDoctors(newUser.ID, newPractice.ID, newDoctors, newDoctorUsers)

		return newPractice, nil, nil
	}
}

func (c *SystemController) importDoctorsAndPatientsFromExcel(fileName string, practiceUser *core.User) ([]string, error) {

	c.addGenders()

	importErrors := []string{}

	log.Println(fileName)
	f, err := xlsx.OpenFile(fileName)
	if err != nil {
		return nil, err
	}

	doctorsSheet := f.Sheet[ExcelSheet_Doctors]
	patientsSheet := f.Sheet[ExcelSheet_Patients]
	settingsSheet := f.Sheet[ExcelSheet_Settings]
	if doctorsSheet == nil || patientsSheet == nil || settingsSheet == nil {
		err := errors.New("datasheet not found")
		log.Println(err)
		return nil, err
	}

	practice := c.getPractice(practiceUser)

	newDoctors, newDoctorUsers, importErrorsDoctor := getDoctors(c.ormDB, doctorsSheet, 1, practice)

	//Verknüpft die errors, um alle Fehler den User zu zeigen
	if importErrorsDoctor != nil && len(importErrorsDoctor) > 0 {
		importErrors = append(importErrors, importErrorsDoctor...)
	}

	newPatients, newPatientUsers, importErrorsPatients := getPatients(c.ormDB, patientsSheet, 1, practice)
	if importErrorsPatients != nil && len(importErrorsPatients) > 0 {
		importErrors = append(importErrors, importErrorsPatients...)
	}

	if len(importErrors) > 0 {
		return importErrors, nil
	} else {
		practice := c.getPractice(practiceUser)
		c.saveDoctors(practiceUser.ID, practice.ID, newDoctors, newDoctorUsers)
		c.savePatients(practiceUser.ID, newPatients, newPatientUsers)

		return nil, nil
	}

}

func (c *SystemController) importDoctorsFromExcel(fileName string, practiceUser *core.User) ([]string, error) {

	c.addGenders()

	importErrors := []string{}

	log.Println(fileName)
	f, err := xlsx.OpenFile(fileName)
	if err != nil {
		return nil, err
	}

	doctorsSheet := f.Sheet[ExcelSheet_Doctors]
	settingsSheet := f.Sheet[ExcelSheet_Settings]
	if doctorsSheet == nil || settingsSheet == nil {
		err := errors.New("datasheet not found")
		log.Println(err)
		return nil, err
	}

	practice := c.getPractice(practiceUser)

	if practice.AccountsAbbreviation == "" {
		err := errors.New("practice abbreviation empty")
		log.Println(err)
		return nil, err
	}

	newDoctors, newDoctorUsers, importErrors := getDoctors(c.ormDB, doctorsSheet, 1, practice)

	if len(importErrors) > 0 {
		return importErrors, nil
	} else {
		c.saveDoctors(practiceUser.ID, practice.ID, newDoctors, newDoctorUsers)

		return nil, nil
	}
}

func (c *SystemController) importPatientsFromExcel(fileName string, practiceUser *core.User) ([]string, error) {

	importErrors := []string{}

	c.addGenders()

	log.Println(fileName)
	f, err := xlsx.OpenFile(fileName)
	if err != nil {
		return nil, err
	}

	patientsSheet := f.Sheet[ExcelSheet_Patients]
	settingsSheet := f.Sheet[ExcelSheet_Settings]
	if patientsSheet == nil || settingsSheet == nil {
		err := errors.New("datasheet not found")
		log.Println(err)
		return nil, err
	}

	practice := c.getPractice(practiceUser)

	newPatients, newPatientUsers, importErrors := getPatients(c.ormDB, patientsSheet, 1, practice)

	if len(importErrors) > 0 {
		return importErrors, nil
	} else {
		// Create new Patients for Practice
		c.savePatients(practiceUser.ID, newPatients, newPatientUsers)

		return nil, nil
	}
}

//CreatedById = user.id
func (c *SystemController) saveDoctors(createdById uint, practiceId uint, newDoctors podiumbundle.Doctors, newDoctorUsers core.Users) {

	// Create new Doctors for Practice
	for i, newDoctor := range newDoctors {
		newUserForDoctor := newDoctorUsers[i]
		newUserForDoctor.CreatedBy = createdById
		c.ormDB.Set("gorm:save_associations", false).Save(&newUserForDoctor)
		c.ormDB.Set("gorm:save_associations", false).Save(&newDoctor)
		newDoctorUser := podiumbundle.DoctorUser{}
		newDoctorUser.UserId = newUserForDoctor.ID
		newDoctorUser.DoctorId = newDoctor.ID
		newDoctorUser.Status = 1
		c.ormDB.Set("gorm:save_associations", false).Save(&newDoctorUser)
		newPracticeDoctor := podiumbundle.PracticeDoctor{}
		newPracticeDoctor.PracticeId = practiceId
		newPracticeDoctor.DoctorId = newDoctor.ID
		//ToDo Allow_Practice_Login always == 0 ??? RS
		c.ormDB.Set("gorm:save_associations", false).Save(&newPracticeDoctor)
	}
}

//CreatedById = user.id
func (c *SystemController) savePatients(createdById uint, newPatients podiumbundle.Patients, newPatientUsers core.Users) {
	// Create new Patients for Practice
	for i, newPatient := range newPatients {
		newUserForPatient := newPatientUsers[i]
		newUserForPatient.CreatedBy = createdById
		c.ormDB.Set("gorm:save_associations", false).Save(&newUserForPatient)
		newPatient.UserId = newUserForPatient.ID
		c.ormDB.Set("gorm:save_associations", false).Save(&newPatient)
	}
}

func GetHeaderIndexes(row *xlsx.Row) map[string]int {
	headers := make(map[string]int)
	for i, cell := range row.Cells {
		switch cell.String() {
		default:
			headers[cell.String()] = i
			break
		}
	}
	return headers
}

func getSetting(sheet *xlsx.Sheet, settingName string) ([]string, error) {
	settings := []string{}
	headers := map[string]int{}

	if len(sheet.Rows) == 0 {
		err := errors.New("sheet is empty")
		log.Println(err)
		return nil, err
	}
	for rowNr, row := range sheet.Rows {
		if rowNr == 0 {
			headers = GetHeaderIndexes(row)
		}
		if rowNr > 0 {
			if val, ok := headers[settingName]; ok {
				settings = append(settings, row.Cells[val].Value)
			} else {
				err := errors.New(fmt.Sprintf("setting column %s not found", settingName))
				log.Println(err)
				return nil, err
			}
		}
	}

	return settings, nil
}

func createUser(row *xlsx.Row, rowNr int, headers map[string]int, userType core.UserType, emailRequired bool, abbreviation string) (core.User, []string) {
	newUser := core.User{}
	importErrors := []string{}

	newUser.RegisteredAt = core.Now()
	newUser.UserType = userType
	newUser.IsActive = true

	//Set Username if not empty
	if val := getString(row, headers, ExcelHeader_Username_r); val != "" {
		newUser.Username = abbreviation + "_" + val
	} else {
		err := errors.New(fmt.Sprintf("row %d: column '%s' is required", rowNr+1, ExcelHeader_Username_r))
		log.Println(err)
		importErrors = append(importErrors, err.Error())
	}

	// Validate email (if required or if not required but set) and set if ok
	if emailRequired {
		if val := getString(row, headers, ExcelHeader_Email_r); val != "" {
			err := core.ValidateFormat(val)
			if err != nil {
				newImportErrorMessage := fmt.Sprintf("row %d: %s", rowNr, err.Error())
				importErrors = append(importErrors, newImportErrorMessage)
			} else {
				newUser.Email = val
			}
		} else {
			err := errors.New(fmt.Sprintf("row %d: column '%s' is required", rowNr+1, ExcelHeader_Email))
			importErrors = append(importErrors, err.Error())
		}
	} else if val := getString(row, headers, ExcelHeader_Email); val != "" {
		err := core.ValidateFormat(val)
		if err != nil {
			newImportErrorMessage := fmt.Sprintf("row %d: %s", rowNr, err.Error())
			importErrors = append(importErrors, newImportErrorMessage)
		} else {
			newUser.Email = val
		}
	}

	//Validate password (if required or if not required but set) and set if ok
	if userType != 1 {
		if val := getString(row, headers, ExcelHeader_Password_r); val != "" {
			err := core.ValidatePassword(val)
			if err != nil {
				newImportErrorMessage := fmt.Sprintf("row %d: %s", rowNr, err.Error())
				importErrors = append(importErrors, newImportErrorMessage)
			} else {
				newUser.PasswordX = core.GetMD5Hash(val)
				newUser.Password = newUser.PasswordX
			}
		} else {
			err := errors.New(fmt.Sprintf("row %d: column '%s' is required", rowNr+1, ExcelHeader_Password_r))
			importErrors = append(importErrors, err.Error())
		}
	} else {
		if val := getString(row, headers, ExcelHeader_Password_r); val != "" {
			err := core.ValidatePassword(val)
			if err != nil {
				newImportErrorMessage := fmt.Sprintf("row %d: %s", rowNr, err.Error())
				importErrors = append(importErrors, newImportErrorMessage)
			} else {
				newUser.PasswordX = core.GetMD5Hash(val)
				newUser.Password = newUser.PasswordX
			}
		}
	}

	return newUser, importErrors
}

func validateUsers(ormDB *gorm.DB, users core.Users, userType core.UserType) []string {
	importErrors := []string{}
	usernames := []string{}
	userEmails := []string{}

	for _, user := range users {
		usernames = append(usernames, user.Username)
		userEmails = append(userEmails, user.Email)
	}

	existingUsersByUsername := core.Users{}
	existingUsersByEmail := core.Users{}

	ormDB.Where("user_type = ? AND username IN (?)", userType, usernames).Find(&existingUsersByUsername)
	ormDB.Where("user_type = ? AND email IN (?)", userType, userEmails).Find(&existingUsersByEmail)

	for _, user := range users {
		for _, existingUser := range existingUsersByUsername {
			if strings.ToLower(existingUser.Username) == strings.ToLower(user.Username) {
				err := errors.New(fmt.Sprintf("username '%s' already exists", user.Username))
				log.Println(err)
				importErrors = append(importErrors, err.Error())
				break
			}
		}
		// Since 1.4.1, usernames are now unique, not emails
		/*
			if user.Email != "" {
				for _, existingUser := range existingUsersByEmail {
					if strings.ToLower(existingUser.Email) == strings.ToLower(user.Email) {
						err := errors.New(fmt.Sprintf("email '%s' already exists", user.Email))
						log.Println(err)
						importErrors = append(importErrors, err.Error())
						break
					}
				}
			}
		*/
	}

	return importErrors
}

func getPractice(ormDB *gorm.DB, sheet *xlsx.Sheet) (podiumbundle.Practice, core.User, podiumbundle.Doctor, core.User, []string) {
	newPractice := podiumbundle.Practice{}
	newUserForPractice := core.User{}
	newDoctor := podiumbundle.Doctor{}
	newUserForDoctor := core.User{}
	headers := map[string]int{}
	importErrors := []string{}
	newImportErrors := []string{}

	for rowNr, row := range sheet.Rows {
		if rowNr == 1 {
			headers, newImportErrors = checkHeadersExcelRow(row, ExcelSheet_Practice)
			if len(newImportErrors) > 0 {
				importErrors = append(importErrors, newImportErrors...)
				return newPractice, newUserForPractice, newDoctor, newUserForDoctor, importErrors
			}
		}

		if rowNr == 2 {
			// Create Practice
			//Set Name if not empty
			if val := getString(row, headers, ExcelHeader_Name_r); val != "" {
				newPractice.Name = val
			} else {
				err := errors.New(fmt.Sprintf("Error for practice in row %d: column '%s' is required", rowNr+1, ExcelHeader_Name_r))
				log.Println(err)
				importErrors = append(importErrors, err.Error())
			}

			//Set abbreviation if not empty
			if val := getString(row, headers, ExcelHeader_Abbreviation_r); val != "" {
				newPractice.AccountsAbbreviation = val
			} else {
				err := errors.New(fmt.Sprintf("Error for practice in row %d: column '%s' is required", rowNr+1, ExcelHeader_Abbreviation_r))
				log.Println(err)
				importErrors = append(importErrors, err.Error())
			}

			// Create User for Practice
			newUserForPractice, newImportErrors = createUser(row, rowNr+1, headers, core.UserTypePractice, true, newPractice.AccountsAbbreviation)
			if len(newImportErrors) > 0 {
				for _, importError := range newImportErrors {
					newImportErrorMessage := fmt.Sprintf("Error for practice in %s", importError)
					importErrors = append(importErrors, newImportErrorMessage)
				}
			}

			//Set Postcode if not empty
			if val := getString(row, headers, ExcelHeader_Postcode_r); val != "" {
				newPractice.Postcode = val
			} else {
				err := errors.New(fmt.Sprintf("Error for practice in row %d: column '%s' is required", rowNr+1, ExcelHeader_Postcode_r))
				log.Println(err)
				importErrors = append(importErrors, err.Error())
			}

			// Create Doctor and Doctor User, if necessary
			if val := getString(row, headers, ExcelHeader_UseForPod_r); val == "y" {
				newPractice.HasSameDoctor = true

				newUserForDoctor, newImportErrors = createUser(row, rowNr+1, headers, core.UserTypeDoctor, true, newPractice.AccountsAbbreviation)
				if len(newImportErrors) > 0 {
					for _, importError := range newImportErrors {
						newImportErrorMessage := fmt.Sprintf("Error for practice in %s", importError)
						importErrors = append(importErrors, newImportErrorMessage)
					}
				}

				newDoctor.Name = newPractice.Name
				newDoctor.Email = newUserForDoctor.Email
				newDoctor.Postcode = newPractice.Postcode

				newDoctor.IsRegistered = true
			}
			break
		}
	}
	usersToTest := core.Users{}
	usersToTest = append(usersToTest, newUserForPractice)
	newImportErrors = validateUsers(ormDB, usersToTest, core.UserTypePractice)
	if len(newImportErrors) > 0 {
		for _, importError := range newImportErrors {
			newImportErrorMessage := fmt.Sprintf("Error for practice: %s", importError)
			importErrors = append(importErrors, newImportErrorMessage)
		}
	}
	usersToTest = core.Users{}
	usersToTest = append(usersToTest, newUserForDoctor)
	newImportErrors = validateUsers(ormDB, usersToTest, core.UserTypeDoctor)
	if len(newImportErrors) > 0 {
		for _, importError := range newImportErrors {
			newImportErrorMessage := fmt.Sprintf("Error for practice: %s", importError)
			importErrors = append(importErrors, newImportErrorMessage)
		}
	}

	if len(importErrors) > 0 {
		return newPractice, newUserForPractice, newDoctor, newUserForDoctor, importErrors
	} else {
		return newPractice, newUserForPractice, newDoctor, newUserForDoctor, nil
	}
}

func getDoctors(ormDB *gorm.DB, sheet *xlsx.Sheet, headerIndex int, practice *podiumbundle.Practice) (podiumbundle.Doctors, core.Users, []string) {
	startRow := headerIndex + 1

	newDoctors := podiumbundle.Doctors{}
	newUsers := core.Users{}
	headers := map[string]int{}
	importErrors := []string{}
	newImportErrors := []string{}

	for rowNr, row := range sheet.Rows {
		if rowNr == headerIndex {
			headers, newImportErrors = checkHeadersExcelRow(row, ExcelSheet_Doctors)
			if len(newImportErrors) > 0 {
				importErrors = append(importErrors, newImportErrors...)
				return newDoctors, newUsers, importErrors
			}
		}
		if rowNr >= startRow {
			if isEmptyExcelRow(row) {
				continue
			}

			//Debug
			log.Println(len(row.Cells))

			// Create Doctor and User for each row
			newUser, newImportErrors := createUser(row, rowNr+1, headers, core.UserTypeDoctor, false, practice.AccountsAbbreviation)
			if len(newImportErrors) > 0 {
				for _, importError := range newImportErrors {
					newImportErrorMessage := fmt.Sprintf("Error for doctors in %s", importError)
					importErrors = append(importErrors, newImportErrorMessage)
				}
			}
			newUsers = append(newUsers, newUser)

			newDoctor := podiumbundle.Doctor{}

			if val := getString(row, headers, ExcelHeader_FirstName); val != "" {
				newDoctor.FirstName = val
			}
			if val := getString(row, headers, ExcelHeader_LastName); val != "" {
				newDoctor.LastName = val
			}
			// Check if Gender matches a defined gender
			if val := getString(row, headers, ExcelHeader_Gender); val != "" {
				if valGender, ok := genderNumbers[strings.ToLower(val)]; ok {
					newDoctor.Gender = valGender
				}
			}
			// Validate Email and set if ok
			if val := getString(row, headers, ExcelHeader_Email); val != "" {
				err := core.ValidateFormat(val)
				if err != nil {
					newImportErrorMessage := fmt.Sprintf("row %d: %s", rowNr, err.Error())
					importErrors = append(importErrors, newImportErrorMessage)
				} else {
					newUser.Email = val
					// a doctor may have multiple users
					// so the email needs to be set on the doctor
					// too so the frontend shows this value
					newDoctor.Email = val
				}
			}
			if val := getString(row, headers, ExcelHeader_Phone); val != "" {
				newDoctor.Phone = val
			}
			if val := getString(row, headers, ExcelHeader_Postcode); val != "" {
				newDoctor.Postcode = val
			}
			if val := getString(row, headers, ExcelHeader_Town); val != "" {
				newDoctor.Town = val
			}
			if val := getString(row, headers, ExcelHeader_Country); val != "" {
				newDoctor.Country = val
			}
			if val := getString(row, headers, ExcelHeader_AddressLine1); val != "" {
				newDoctor.AddressLine1 = val
			}
			if val := getString(row, headers, ExcelHeader_AddressLine2); val != "" {
				newDoctor.AddressLine2 = val
			}
			if val := getString(row, headers, ExcelHeader_StandardWelcomeMessage); val != "" {
				newDoctor.StandardWelcomeMessage = val
			}

			newDoctor.IsRegistered = true

			newDoctors = append(newDoctors, newDoctor)
		}
	}

	newImportErrors = validateUsers(ormDB, newUsers, core.UserTypeDoctor)
	if len(newImportErrors) > 0 {
		for _, importError := range newImportErrors {
			newImportErrorMessage := fmt.Sprintf("Error for doctor: %s", importError)
			importErrors = append(importErrors, newImportErrorMessage)
		}
	}

	if len(importErrors) > 0 {
		return newDoctors, newUsers, importErrors
	} else {
		return newDoctors, newUsers, nil
	}
}

func getPatients(ormDB *gorm.DB, sheet *xlsx.Sheet, headerIndex int, practice *podiumbundle.Practice) (podiumbundle.Patients, core.Users, []string) {
	startRow := headerIndex + 1

	newPatients := podiumbundle.Patients{}
	newUsers := core.Users{}
	headers := map[string]int{}
	importErrors := []string{}
	newImportErrors := []string{}

	for rowNr, row := range sheet.Rows {
		if rowNr == headerIndex {
			headers, newImportErrors = checkHeadersExcelRow(row, ExcelSheet_Patients)
			if len(newImportErrors) > 0 {
				importErrors = append(importErrors, newImportErrors...)
				return newPatients, newUsers, importErrors
			}
		}
		if rowNr >= startRow {
			if isEmptyExcelRow(row) {
				continue
			}

			//Debug
			log.Println(len(row.Cells))

			// Create Patient and User for each row
			newUser, newImportErrors := createUser(row, rowNr, headers, core.UserTypePatient, false, practice.AccountsAbbreviation)
			if len(newImportErrors) > 0 {
				for _, importError := range newImportErrors {
					newImportErrorMessage := fmt.Sprintf("Error for patient in %s", importError)
					importErrors = append(importErrors, newImportErrorMessage)
				}
			}
			newUsers = append(newUsers, newUser)

			newPatient := podiumbundle.Patient{}

			if val := getString(row, headers, ExcelHeader_FirstName); val != "" {
				newPatient.FirstName = val
			}
			if val := getString(row, headers, ExcelHeader_LastName); val != "" {
				newPatient.LastName = val
			}
			// Check if Gender matches a defined gender
			if val := getString(row, headers, ExcelHeader_Gender); val != "" {
				if valGender, ok := genderNumbers[strings.ToLower(val)]; ok {
					newPatient.Gender = valGender
				}
			}
			if val := getString(row, headers, ExcelHeader_BirthDate); val != "" {
				newPatient.BirthDate = core.Now()
				newPatient.BirthDate = getDateFromExcelString(val)
			}
			if val := getString(row, headers, ExcelHeader_Phone); val != "" {
				newPatient.Phone = val
			}
			if val := getString(row, headers, ExcelHeader_Postcode); val != "" {
				newPatient.Postcode = val
			}
			if val := getString(row, headers, ExcelHeader_Town); val != "" {
				newPatient.Town = val
			}
			if val := getString(row, headers, ExcelHeader_County); val != "" {
				newPatient.County = val
			}
			if val := getString(row, headers, ExcelHeader_Country); val != "" {
				newPatient.Country = val
			}
			if val := getString(row, headers, ExcelHeader_AddressLine1); val != "" {
				newPatient.AddressLine1 = val
			}
			if val := getString(row, headers, ExcelHeader_AddressLine2); val != "" {
				newPatient.AddressLine2 = val
			}

			newPatients = append(newPatients, newPatient)
		}
	}

	newImportErrors = validateUsers(ormDB, newUsers, core.UserTypePatient)
	if len(newImportErrors) > 0 {
		for _, importError := range newImportErrors {
			newImportErrorMessage := fmt.Sprintf("Error for patient: %s", importError)
			importErrors = append(importErrors, newImportErrorMessage)
		}
	}

	if len(importErrors) > 0 {
		return newPatients, newUsers, importErrors
	} else {
		return newPatients, newUsers, nil
	}
}

// Return int value of cell at header index. The string is empty when header or cell is empty
func getBool(r *xlsx.Row, headers map[string]int, s string) bool {
	if val, ok := headers[s]; ok {
		if val < len(r.Cells) {
			return r.Cells[val].Bool()
		}
	}
	return false
}

// Return bool value of cell at header index. The string is empty when header or cell is empty
func getFloat(r *xlsx.Row, headers map[string]int, s string) float64 {
	if val, ok := headers[s]; ok {
		if val < len(r.Cells) {
			if cellValue, err := r.Cells[val].Float(); err == nil {
				return cellValue
			}
		}
	}
	return 0
}

// Return float value of cell at header index. The string is empty when header or cell is empty
func getInt(r *xlsx.Row, headers map[string]int, s string) int {
	if val, ok := headers[s]; ok {
		if val < len(r.Cells) {
			if cellValue, err := r.Cells[val].Int(); err == nil {
				return cellValue
			}
		}
	}
	return 0
}

// Return string value of cell at header index. The string is empty when header or cell is empty
func getString(r *xlsx.Row, headers map[string]int, s string) string {
	if val, ok := headers[s]; ok {
		if val < len(r.Cells) {
			return r.Cells[val].String()
		}
	}
	return ""
}

func isEmptyExcelRow(r *xlsx.Row) bool {
	isEmpty := true
	for _, cell := range r.Cells {
		if cell.Value != "" {
			isEmpty = false
			break
		}
	}
	return isEmpty
}

// Check integrity of excel file headers in dependence of string s (Practice, Doctors, Patient)
func checkHeadersExcelRow(r *xlsx.Row, s string) (map[string]int, []string) {
	headers := map[string]int{}
	errors := []string{}

	if isEmptyExcelRow(r) {
		errors = append(errors, "header row missing")
		return headers, errors
	}

	headers = GetHeaderIndexes(r)
	switch s {
	case ExcelSheet_Practice:
		if _, ok := headers[ExcelHeader_Name_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Practice, ExcelHeader_Name_r))
		}
		if _, ok := headers[ExcelHeader_Postcode_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Practice, ExcelHeader_Postcode_r))
		}
		if _, ok := headers[ExcelHeader_Username_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Practice, ExcelHeader_Username_r))
		}
		if _, ok := headers[ExcelHeader_Password_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Practice, ExcelHeader_Password_r))
		}
		if _, ok := headers[ExcelHeader_Email_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Practice, ExcelHeader_Email_r))
		}
		if _, ok := headers[ExcelHeader_UseForPod_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Practice, ExcelHeader_UseForPod_r))
		}
		break
	case ExcelSheet_Doctors:
		if _, ok := headers[ExcelHeader_Username_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Username_r))
		}
		if _, ok := headers[ExcelHeader_Password_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Password_r))
		}
		if _, ok := headers[ExcelHeader_FirstName]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_FirstName))
		}
		if _, ok := headers[ExcelHeader_LastName]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_LastName))
		}
		if _, ok := headers[ExcelHeader_Gender]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Gender))
		}
		if _, ok := headers[ExcelHeader_Email]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Email))
		}
		if _, ok := headers[ExcelHeader_Phone]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Phone))
		}
		if _, ok := headers[ExcelHeader_Postcode]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Postcode))
		}
		if _, ok := headers[ExcelHeader_Town]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Town))
		}
		if _, ok := headers[ExcelHeader_Country]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_Country))
		}
		if _, ok := headers[ExcelHeader_AddressLine1]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_AddressLine1))
		}
		if _, ok := headers[ExcelHeader_AddressLine2]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_AddressLine2))
		}
		if _, ok := headers[ExcelHeader_StandardWelcomeMessage]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Doctors, ExcelHeader_StandardWelcomeMessage))
		}
		break
	case ExcelSheet_Patients:
		if _, ok := headers[ExcelHeader_Username_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Username_r))
		}
		if _, ok := headers[ExcelHeader_Password_r]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Password_r))
		}
		if _, ok := headers[ExcelHeader_FirstName]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_FirstName))
		}
		if _, ok := headers[ExcelHeader_LastName]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_LastName))
		}
		if _, ok := headers[ExcelHeader_BirthDate]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_BirthDate))
		}
		if _, ok := headers[ExcelHeader_Gender]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Gender))
		}
		if _, ok := headers[ExcelHeader_Email]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Email))
		}
		if _, ok := headers[ExcelHeader_Phone]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Phone))
		}
		if _, ok := headers[ExcelHeader_Postcode]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Postcode))
		}
		if _, ok := headers[ExcelHeader_County]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_County))
		}
		if _, ok := headers[ExcelHeader_Town]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Town))
		}
		if _, ok := headers[ExcelHeader_Country]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_Country))
		}
		if _, ok := headers[ExcelHeader_AddressLine1]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_AddressLine1))
		}
		if _, ok := headers[ExcelHeader_AddressLine2]; !ok {
			errors = append(errors, fmt.Sprintf("%s: header %s missing", ExcelSheet_Patients, ExcelHeader_AddressLine2))
		}
		break
	}

	return headers, errors
}

func getDateFromExcelString(documentDateString string) core.NullTime {
	//log.Println(documentDateString)
	tmp := strings.Split(documentDateString, "-")
	dotSplitted := false
	if len(tmp) != 3 {
		dotSplitted = true
		tmp = strings.Split(documentDateString, ".")
	}
	documentDate := core.NullTime{}
	if len(tmp) == 3 {
		day, _ := strconv.Atoi(tmp[1])
		month, _ := strconv.Atoi(tmp[0])
		year, _ := strconv.Atoi(tmp[2])
		if dotSplitted {
			day, _ = strconv.Atoi(tmp[0])
			month, _ = strconv.Atoi(tmp[1])
			year, _ = strconv.Atoi(tmp[2])
		}
		if year < 1900 {
			year = 2000 + year
		}
		documentDate = core.NullTime{Time: time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), Valid: true}
	} else {

		documentDateFloat, err := strconv.ParseFloat(documentDateString, 64)
		if err == nil {
			documentDate.Time = xlsx.TimeFromExcelTime(documentDateFloat, false)
			documentDate.Valid = true
		}

	}
	return documentDate
}
