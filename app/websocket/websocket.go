package web3socket

import (
	"fmt"
	"github.com/gorilla/websocket"
//	"gotools/tools"
	tools "github.com/kirillDanshin/nulltime"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	Websocket_Measurements       = "MEASUREMENTS"
	Websocket_SharedMeasurements = "SHAREDMEASUREMENTS"
	Websocket_Account_Locked     = "ACCOUNTLOCKED"
	Websocket_Account_Unlocked   = "ACCOUNTUNLOCKED"
	Websocket_Patients           = "PATIENTS"
	Websocket_Podiatrist         = "PODIATRIST"
	Websocket_Appointments       = "APPOINTMENTS"
	Websocket_Invites            = "INVITES"
	Websocket_Chats              = "CHATS"
	Websocket_Messages           = "MESSAGES"
	Websocket_Monitoring         = "MONITORING"
	Websocket_All                = "ALL"
)

const (
	Websocket_Update = "UPDATE"
	Websocket_Add    = "ADD"
	Websocket_Delete = "DELETE"
)

type RegisteredMessageType struct {
	MessageType string `json:"message_type"`
	SpecifiedId uint   `json:"specified_id"`
}

type RegisteredMessageTypes []RegisteredMessageType

var WebsocketUsers = make(map[uint]map[*websocket.Conn]RegisteredMessageTypes)

//var websocketClients = make(map[*websocket.Conn]bool) // connected clients

var Broadcast = make(chan WSHeaderMessage)   // broadcast channel
var UserChannel = make(chan WSHeaderMessage) // user channel

// Configure the upgrader
var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == http.MethodPost {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}

func HandleBroadcastMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-Broadcast
		// Send it out to every client that is currently connected

		for userId, usersWebsockets := range WebsocketUsers {
			if msg.UserId == 0 || msg.UserId == userId {
				// wenn keine User-ID angegeben, dann Berechtigung prüfen
				if msg.UserId == 0 {
					// TODO check Permission
					hasPermission := true
					if !hasPermission {
						continue
					}
				}
				for client, areas := range usersWebsockets {
					needsToSend := false
					for _, area := range areas {
						if (area.MessageType == msg.Message.MessageType || area.MessageType == Websocket_All) && (area.SpecifiedId == 0 || area.SpecifiedId == msg.Message.ForeignId) {
							needsToSend = true
							break
						}
					}
					if needsToSend {
						err := client.WriteJSON(&msg.Message)
						// log.Println(msg)
						if err != nil {
							log.Printf("error: %v", err)
							client.Close()
							delete(usersWebsockets, client)
						}
					}
				}

			}
		}
	}
}

func HandleUserMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-UserChannel
		// Send it out to every client that is currently connected

		for client := range WebsocketUsers[msg.UserId] {
			err := client.WriteJSON(msg.Message)
			//log.Println(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(WebsocketUsers[msg.UserId], client)
			}
		}
	}
}

func SendBroadCastWebsocketDataInfoMessage(message string, action string, foreignType string, foreignId uint, data interface{}) {
	var wsMsg WebsocketMessage = WebsocketMessage{
		MessageType: "DATA",
		Timestamp:   tools.NullTime{Time: time.Now(), Valid: true},
		Message:     message,
		ForeignType: foreignType,
		ForeignId:   foreignId,
		Action:      action,
		Data:        data,
	}
	headerMsg := WSHeaderMessage{UserId: 0, Message: wsMsg}
	Broadcast <- headerMsg
}

// userI = 0 -> alle User, die entsprechend berechtigt sind
func SendWebsocketDataInfoMessage(message string, action string, foreignType string, foreignId uint, userIds []uint, data interface{}) {
	if userIds == nil || len(userIds) == 0 {
		return
	}
	for _, userId := range userIds {
		if userId > 0 {
			var wsMsg WebsocketMessage = WebsocketMessage{
				MessageType: "DATA",
				Timestamp:   tools.NullTime{Time: time.Now(), Valid: true},
				Message:     message,
				ForeignType: foreignType,
				ForeignId:   foreignId,
				Action:      action,
				Data:        data,
			}
			headerMsg := WSHeaderMessage{UserId: userId, Message: wsMsg}
			Broadcast <- headerMsg
		}
	}
}

func SendWebsocketNotification(action string, foreignId uint) {

	var msg Notification = Notification{Title: "Report fertig", Message: "Der Report xyz bla blubb ist fertig und kann angezeigt werden", Priority: 2, Timestamp: tools.NullTime{Time: time.Now(), Valid: true}}
	var wsMsg WebsocketMessage = WebsocketMessage{MessageType: "NOTIFICATION", Data: msg}
	headerMsg := WSHeaderMessage{UserId: 1, Message: wsMsg}

	Broadcast <- headerMsg

}
