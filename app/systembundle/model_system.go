package systembundle

import (
	"thermetrix_backend/app/core"
)

type LogData struct {
	LogLevel int64  `json:"log_level"`
	LogFile  string `json:"log_file"`
	LogLine  int64  `json:"log_line"`
	LogText  string `json:"log_text"`
	LogRoute string `json:"log_route"`
}

type SystemSetting struct {
	core.Model
	SettingsKey   string `json:"settings_key" gorm:"type:varchar(100);unique_index"`
	SettingsValue string `json:"settings_value" gorm:"type:TEXT"`

	Errors map[string]string `json:"-" gorm:"-"`
}

//
//
// swagger:model SystemLog
type SystemLog struct {
	ID              uint          `json:"id" gorm:"primary_key"`
	UserId          uint          `json:"-"`
	User            core.User     `json:"user"`
	LogType         uint          `json:"log_type"`
	LogDate         core.NullTime `json:"log_date"`
	LogTitle        string        `json:"log_title"`
	LogText         string        `json:"log_text"`
	CurrentPage     string        `json:"current_page" gorm:"-"`
	CurrentPageDate core.NullTime `json:"current_page_date" gorm:"-"`

	Errors map[string]string `json:"-" gorm:"-"`
}

type SystemLogs []SystemLog

func (SystemLog) TableName() string {
	return "system_log"
}

type SystemAccountsSession struct {
	core.Model
	AccountId    uint          `json:"-"`
	Account      core.User     `json:"account"`
	SessionToken string        `json:"session_token" gorm:"type:VARCHAR(36);unique_index"`
	LoginTime    core.NullTime `json:"login_time"`
}
type SystemAccountsSessions []SystemAccountsSession

//
//
// swagger:model SystemLog
type SystemFrontendLog struct {
	ID       uint          `json:"id" gorm:"primary_key"`
	UserId   uint          `json:"-"`
	User     core.User     `json:"user"`
	LogType  uint          `json:"log_type"`
	LogDate  core.NullTime `json:"log_date"`
	LogTitle string        `json:"log_title"`
	LogText  string        `json:"log_text"`

	Errors map[string]string `json:"-" gorm:"-"`
}

type SystemFrontendLogs []SystemFrontendLog

type SystemAccountPasswordReset struct {
	core.Model
	UserId  uint      `json:"-"`
	User    core.User `json:"user"`
	Token   string    `json:"token"`
	IsValid bool      `json:"is_valid"`
}

type ResetPasswordHelper struct {
	Username       string `json:"username"`
	Token          string `json:"token"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"password_repeat"`
}

type SystemFrontendTranslationKeyTranslation struct {
	core.Model
	SystemFrontendTranslationKeyId      uint                              `json:"-"`
	SystemFrontendTranslationKey        SystemFrontendTranslationKey      `json:"translation_key"`
	SystemFrontendTranslationLanguageId uint                              `json:"-"`
	SystemFrontendTranslationLanguage   SystemFrontendTranslationLanguage `json:"translation_language"`

	Translation string `json:"translation"`
}

type SystemFrontendTranslationKeyTranslations []SystemFrontendTranslationKeyTranslation

type SystemServerConfig struct {
	CustomerServerKey string `json:"customer_server_key"`
}

type SystemFrontendTranslationKey struct {
	core.Model
	TranslationKey string `json:"key"`
}

type SystemFrontendTranslationKeys []SystemFrontendTranslationKey

type SystemFrontendTranslationLanguage struct {
	core.Model
	Language     string `json:"language"`
	LanguageCode string `json:"language_code"`
}

type SystemFrontendTranslationLanguages []SystemFrontendTranslationLanguage
