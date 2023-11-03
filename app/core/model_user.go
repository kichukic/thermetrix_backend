package core

import "github.com/jinzhu/gorm"

const PasswordRegex = `^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]{8,}$`
const PasswordMessage = "password needs to be at least 8 characters long and needs at least one lowercase, uppercase and special character as well as one digit"
const PasswordMinLength = 8

type UserType uint

const (
	UserTypeSystem   UserType = 0
	UserTypePatient  UserType = 1
	UserTypeDoctor   UserType = 2
	UserTypePractice UserType = 3
)

// 0 - system; 1 - patient; 2 - doctor; 3 -practice
// swagger:model
type User struct {
	Model
	Username          string   `json:"username,omitempty"`
	UserType          UserType `json:"user_type,omitempty"` // 0 - system; 1 - patient; 2 - doctor; 3 -practice
	Email             string   `json:"email,omitempty"`
	Token             string   `json:"token,omitempty" gorm:"-"`
	Password          string   `json:"-"`
	PasswordX         string   `json:"password,omitempty" gorm:"-"`
	PasswordRepeat    string   `json:"password_repeat,omitempty" gorm:"-"`
	IsActive          bool     `json:"is_active"`
	IsSysadmin        bool     `json:"is_sysadmin,omitempty"`
	IsPasswordExpired bool     `json:"is_password_expired,omitempty"`
	RegisteredAt      NullTime `json:"registered_at,omitempty"`

	IsTutorialSeen bool `json:"is_tutorial_seen"`
	TosAccepted    bool `json:"tos_accepted"`
	//TestCommend
	//Patient					*Patient				`json:"patient,omitempty" gorm:"-"`
	//Doctor					*Doctor					`json:"doctor,omitempty" gorm:"-"`

	CreatedBy  uint `json:"created_by"`
	HomeUser   bool `json:"home_user"`   // True can you Home-App
	RemoteUser bool `json:"remote_user"` // True can you Remote-App

	SettingId uint                 `json:"-"`
	Setting   SystemAccountSetting `json:"setting"`
	Errors    map[string]string    `json:"-" gorm:"-"`
}

type Users []User

func (User) TableName() string {
	return "system_accounts"
}

type SystemAccountSetting struct {
	Model
	AutomaticLogoutTimer       uint `json:"automatic_logout_timer"`
	ActiveAutomaticLogoutTimer bool `json:"active_automatic_logout_timer"`
}

type SystemAccountSettings []SystemAccountSetting

func (user *User) AfterFind(tx *gorm.DB) (err error) {

	if !user.RegisteredAt.Valid {
		user.RegisteredAt.Time = user.CreatedAt
		user.RegisteredAt.Valid = true
	}

	return
}
