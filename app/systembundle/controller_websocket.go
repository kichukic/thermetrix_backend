package systembundle

import (
	"errors"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
//	"gotools/tools"
	tools "github.com/kirillDanshin/nulltime"
	"log"
	"net/http"
	"strings"
	"thermetrix_backend/app/core"
	websocket2 "thermetrix_backend/app/websocket"
	"time"
)

func (c *SystemController) GetWSTicketHandler(w http.ResponseWriter, r *http.Request) {

	sessionToken := ""

	auth := r.Header.Get("Authorization")
	//	log.Println("auth: ", auth)
	//	log.Printf("USERS: %v", c.Users)
	if len(auth) != len("Bearer 9871b73e-df71-4780-5ed6-b2cbee85f3b5") {
		c.HandleUnauthorizedError(errors.New("Not authorized"), w)
		return
	} else {
		tmp := strings.Split(auth, " ")
		//log.Println(tmp[1])
		//log.Println(Users)
		if _, ok := (*c.Users)[tmp[1]]; !ok {
			c.HandleUnauthorizedError(errors.New("Session invalid"), w)
			return
		} else {
			sessionToken = tmp[1]
		}
	}

	ticket := c.RandomString(32)

	WSTickets[ticket] = sessionToken

	c.SendJSON(w, &ticket, http.StatusOK)
}

func (c *SystemController) SendWSTestMessageHandler(w http.ResponseWriter, r *http.Request) {

	websocket2.SendWebsocketDataInfoMessage("Test", "Test2", "Test3", 12, nil, nil)

	/*
		var msg Notification = Notification{Title: "Report fertig", Message: "Der Report xyz bla blubb ist fertig und kann angezeigt werden", Priority: 2, Timestamp: tools.NullTime{Time: time.Now(), Valid: true}}
		var wsMsg WebsocketMessage = WebsocketMessage{MessageType: "NOTIFICATION", Data: msg}
		headerMsg := WSHeaderMessage{UserId: 1, Message: wsMsg}

		broadcast <- headerMsg
	*/
	c.SendJSON(w, "OK", http.StatusOK)
}

func (c *SystemController) SendWSActiveHandler(w http.ResponseWriter, r *http.Request) {

	var msg websocket2.Notification = websocket2.Notification{Title: "Report fertig", Message: "Der Report xyz bla blubb ist fertig und kann angezeigt werden", Priority: 2, Timestamp: tools.NullTime{Time: time.Now(), Valid: true}}
	var wsMsg websocket2.WebsocketMessage = websocket2.WebsocketMessage{MessageType: "REQUEST", Data: msg}
	headerMsg := websocket2.WSHeaderMessage{UserId: 1, Message: wsMsg}

	websocket2.Broadcast <- headerMsg

	c.SendJSON(w, "OK", http.StatusOK)
}

func (c *SystemController) SendBreakMessageHandler(w http.ResponseWriter, r *http.Request) {

	var msg websocket2.Notification = websocket2.Notification{Type: "SNDBRKMSG", Title: "Achtung", Message: "Bitte alle nicht gespeicherten Änderungen innerhalb von 5 Minuten löschen", Priority: 3, Timestamp: tools.NullTime{Time: time.Now(), Valid: true}}
	var wsMsg websocket2.WebsocketMessage = websocket2.WebsocketMessage{MessageType: "SNDBRKMSG", Data: msg}
	headerMsg := websocket2.WSHeaderMessage{UserId: 1, Message: wsMsg}

	websocket2.Broadcast <- headerMsg

	c.SendJSON(w, "OK", http.StatusOK)
}

func (c *SystemController) SendHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	c.ormDB.Exec("DELETE FROM system_frontend_logs WHERE log_type=10;")
	var msg websocket2.Notification = websocket2.Notification{Type: "HEARTBEAT", Title: "", Message: "", Priority: -1, Timestamp: tools.NullTime{Time: time.Now(), Valid: true}}
	var wsMsg websocket2.WebsocketMessage = websocket2.WebsocketMessage{MessageType: "HEARTBEAT", Data: msg}
	headerMsg := websocket2.WSHeaderMessage{UserId: 1, Message: wsMsg}

	websocket2.Broadcast <- headerMsg

	c.SendJSON(w, "OK", http.StatusOK)

}

func (c *SystemController) HandleConnections(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	ticket := vars["ticket"]
	auth := WSTickets[ticket]

	//	log.Println(formatRequest(r))

	if user, ok := (*c.Users)[auth]; !ok {
		c.HandleError(errors.New("Ticket invalid"), w)
		return
	} else {

		// Upgrade initial GET request to a websocket
		ws, err := websocket2.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		// Make sure we close the connection when the function returns
		defer ws.Close()

		// Register our new client
		if _, ok := websocket2.WebsocketUsers[user.ID]; !ok {
			websocket2.WebsocketUsers[user.ID] = make(map[*websocket.Conn]websocket2.RegisteredMessageTypes)
		}

		websocket2.WebsocketUsers[user.ID][ws] = websocket2.RegisteredMessageTypes{{MessageType: websocket2.Websocket_All, SpecifiedId: 0}}

		//		log.Println(ws)

		for {
			var msg websocket2.WebsocketMessage
			// Read in a new message as JSON and map it to a Message object
			err := ws.ReadJSON(&msg)
			log.Println(msg)
			if err != nil {
				log.Printf("error: %v", err)
				//delete(WebsocketUsers[user.ID], ws)
				break
			}

			if msg.MessageType == "HEARTBEAT" || msg.MessageType == "SNDBRKMSG" {
				frontendLog := SystemFrontendLog{LogDate: core.NullTime{Time: time.Now(), Valid: true}, UserId: uint(user.ID), LogType: 10, LogTitle: "current screen", LogText: msg.Data.(string)}
				c.ormDB.Set("gorm:save_associations", false).Save(&frontendLog)
			}

			// Send the newly received message to the broadcast channel
			//broadcast <- msg
		}
	}
}
