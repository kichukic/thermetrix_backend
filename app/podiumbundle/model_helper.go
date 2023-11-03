package podiumbundle

import "thermetrix_backend/app/core"

type HelperSystemAccountsSession struct {
	core.Model
	AccountId    uint          `json:"-"`
	Account      core.User     `json:"account"`
	SessionToken string        `json:"session_token" gorm:"type:VARCHAR(36);unique_index"`
	LoginTime    core.NullTime `json:"login_time"`
}

func (HelperSystemAccountsSession) TableName() string {
	return "system_accounts_sessions"
}
