package podiumbundle

import (
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"
	"thermetrix_backend/app/core"
	"time"
)

// getNotifications swagger:route GET /me/notifications me getNotifications
//
// retrieves measurements
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) GetMyNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	notifications := Notifications{}

	db := c.ormDB.Set("gorm:auto_preload", true).Where("user_id = ? ", user.ID) //and visible = 1

	db = db.Order("id")

	if len(r.URL.Query()) > 0 {
		values := r.URL.Query()
		if val, ok := values["last_refresh"]; ok && len(val) > 0 {
			if val[0] != "" {
				db = db.Where("updated_at > ?", val[0])
			} else {
				db = db.Where("visible = 1")
			}
		} else {
			db = db.Where("visible = 1")
		}
	} else {
		db = db.Where("visible = 1")
	}

	db.Find(&notifications)

	for key, notification := range notifications {

		if notification.NotificationType == 1 {
			interlocutor := core.User{}
			tmpId := strings.Replace(notification.SourceUrl, "/me/conversations/", "", -1)
			fid, _ := strconv.Atoi(tmpId)
			c.ormDB.Find(&interlocutor, fid)
			notification.InterlocutorHelper = &HelperUser{}
			notification.InterlocutorHelper.User = interlocutor
			notification.InterlocutorHelper.PasswordX = ""
			if c.isPatient(&interlocutor) {
				notification.InterlocutorHelper.Patient = c.getPatient(&interlocutor)
				//c.ormDB.Set("gorm:auto_preload", true).Where("sender_id = ? or recipient_id =?", notification.InterlocutorHelper.ID, chat.InterlocutorHelper.ID).Where("doctor_id = ?", c.getDoctor(user).ID).Last(&chat.LastMessage)
			} else if c.isDoctor(&interlocutor) {
				notification.InterlocutorHelper.Doctor = c.getDoctor(&interlocutor)
				//c.ormDB.Set("gorm:auto_preload", true).Where("sender_id = ? or recipient_id =?", user.ID, user.ID).Where("doctor_id = ?", notification.InterlocutorHelper.Doctor).Last(&chat.LastMessage)
			}

		} else if notification.NotificationType == 2 {
			appointment := Appointment{}
			c.ormDB.Set("gorm:auto_preload", true).First(&appointment, notification.ForeignId)
			notification.Appointment = &appointment
		}

		notifications[key] = notification
	}

	c.SendJSON(w, &notifications, http.StatusOK)
}

// setMyNotificationsRead swagger:route PUT /me/notifications me setMyNotificationsRead
//
// retrieves measurements
//
// produces:
// - application/json
//	+ name: Authorization
//    in: header
//    description: "Bearer " + token
//    required: true
//    type: string
// Responses:
//    default: HandleErrorData
//		  200:
//			data: []Measurement
//        401: HandleErrorData "unauthorized"
//        403: HandleErrorData "no Permission"
func (c *PodiumController) SetMyNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
	ok := false
	var user *core.User
	if ok, user = c.GetUser(w, r); !ok {
		_ = user
		return
	}

	db := c.ormDB.Set("gorm:save_associations", false).Model(&Notification{}).Where("user_id = ?", user.ID)
	vars := mux.Vars(r)
	notificationId, _ := strconv.ParseInt(vars["notificationId"], 10, 64)
	if notificationId > 0 {
		db = db.Where("id = ?", notificationId)
	}
	db.Update("visible", 0)

	c.SendJSON(w, &Notification{}, http.StatusOK)
}

func (c *PodiumController) CreateNotification(userId uint, notificationType int64, title string, message string, foreignId uint, sourceUrl string, actions []NotificationAction) {

	notification := Notification{
		Title:            title,
		Message:          message,
		UserId:           userId,
		NotificationDate: core.NullTime{Time: time.Now(), Valid: true},
		ForeignId:        foreignId,
		SourceUrl:        sourceUrl,
		NotificationType: notificationType,
		Actions:          actions,
		Visible:          true,
	}

	c.ormDB.Set("gorm:save_associations", true).Create(&notification)

	// Idea: sent now a push notification

}
