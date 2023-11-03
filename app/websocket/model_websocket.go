package web3socket

import (
//	"gotools/tools"
	tools "github.com/kirillDanshin/nulltime"
)

const (
	Websocket_Practice    = "PRACTICE"
	Websocket_Doctor      = "DOCTOR"
	Websocket_UserAccount = "USER_ACCOUNT"
	Websocket_Device      = "DEVICE"
	Websocket_Patient     = "PATIENT"
	Websocket_Measurement = "MEASUREMENT"
)

// Define our message object
type WSHeaderMessage struct {
	UserId  uint             `json:"user_id"`
	Message WebsocketMessage `json:"message"`
}

/*
type WebsocketMessage struct {
	MessageType string      `json:"message_type"`
	Data        interface{} `json:"data"`
}*/
type WebsocketMessage struct {
	MessageType string         `json:"message_type"`
	Timestamp   tools.NullTime `json:"timestamp"`
	Status      int            `json:"status,omitempty"`
	Message     string         `json:"message,omitempty"`
	ForeignType string         `json:"foreign_type,omitempty"`
	ForeignId   uint           `json:"foreign_id,omitempty"`
	Action      string         `json:"action,omitempty"`
	Data        interface{}    `json:"data,omitempty"`
}

type WebsocketMessageHeartbeat struct {
	Site       string         `json:"site"`
	LastAction tools.NullTime `json:"last_action"`
}

type Notification struct {
	Title     string         `json:"title"`
	Type      string         `json:"type"`
	Message   string         `json:"message"`
	Priority  int64          `json:"priority"`
	Timestamp tools.NullTime `json:"timestamp"`
}
