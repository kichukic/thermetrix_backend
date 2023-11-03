package core

import (
	"errors"
	"github.com/jinzhu/gorm"
	"log"
	"thermetrix_backend/app/websocket"
	"time"
)

//Save save user account
func (user *User) Save(ormDB *gorm.DB) (bool, error) {
	if user.PasswordX != "" {
		// check new password
		if user.PasswordX != user.PasswordRepeat {
			return false, errors.New("password and repetition of password are not the same")
		}
		err := ValidatePassword(user.PasswordX)
		if err != nil {
			return false, errors.New(PasswordMessage)
		}

		/*
			regex, err := regexp.Compile(PasswordRegex)
			if err != nil {

			}
			if !regex.MatchString(user.PasswordX) {
				return false, errors.New(PasswordMessage)
			}
		*/
		user.Password = GetMD5Hash(user.PasswordX)

	}
	if user.ID == 0 {
		user.IsActive = true
		user.RegisteredAt.Time = time.Now()
		user.RegisteredAt.Valid = true
		ormDB.Set("gorm:save_associations", false).Create(&user)
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Created user account", web3socket.Websocket_Add, web3socket.Websocket_UserAccount, user.ID, nil)
	} else {
		userDB := User{}
		ormDB.First(&userDB, user.ID)
		if user.Password == "" {
			user.Password = userDB.Password
		}
		ormDB.Set("gorm:save_associations", false).Save(&user)
		go web3socket.SendBroadCastWebsocketDataInfoMessage("Updated user account", web3socket.Websocket_Add, web3socket.Websocket_UserAccount, user.ID, nil)
	}

	return true, nil
}

// Validate validate user account
func (user *User) Validate(ormDb *gorm.DB) bool {
	user.Errors = make(map[string]string)

	//if user.Email == "" {
	//	user.Errors["email"] = "email empty"
	//}

	if user.Username == "" {
		user.Errors["username"] = "username empty"
	}

	if user.PasswordX != user.PasswordRepeat {
		user.Errors["password_repetition"] = "password and repetition of password are not the same"
	}

	if user.PasswordX != "" {
		err := ValidatePassword(user.PasswordX)
		if err != nil {
			log.Println(err)
			user.Errors["password"] = err.Error()
		}
		/*
			regex, err := regexp.Compile(PasswordRegex)
			if err != nil {
				log.Println(err)
			}
			if !regex.MatchString(user.PasswordX) {
				user.Errors["password"] = PasswordMessage
			}*/
	}

	if user.ID == 0 {
		/*
			countAccounts := 0
			sqlQuery := `SELECT COUNT(*) FROM system_accounts WHERE username=?`
			err := db.QueryRow(sqlQuery, user.Username).Scan(&countAccounts)
			log.Println(err)
			if countAccounts > 0 {

				user.Errors = make(map[string]string)
				user.Errors["username"] = "username already exists"
			}
		*/

		userDb := User{}
		ormDb.Where("username = ?", user.Username).First(&userDb)

		if userDb.ID > 0 {
			user.Errors["username"] = "username already exists"
		}
	}

	if len(user.Errors) > 0 {
		return false
	}

	return true
}
