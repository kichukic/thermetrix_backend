package podiumbundle

import (
	"net/http"
	"thermetrix_backend/app/core"

	"github.com/jinzhu/gorm"
)

// PodiumBundle handle fleet resources
type PodiumBundle struct {
	routes []core.Route
}

// NewPodiumBundle instance
func NewPodiumBundle(ormDB *gorm.DB, users *map[string]core.User) core.Bundle {
	//Test Tag 1.4 Commit
	//km := NewSystemSQLMapper(db)
	hc := NewPodiumController(ormDB, users)
	//Comment
	r := []core.Route{
		core.Route{Method: http.MethodGet, Path: "/doctors", Handler: hc.GetDoctors4AppHandler},
		core.Route{Method: http.MethodGet, Path: "/admin/doctors", Handler: hc.GetDoctorsHandler},
		core.Route{Method: http.MethodGet, Path: "/doctors/{doctorId:[0-9]+}", Handler: hc.GetDoctorHandler},
		core.Route{Method: http.MethodDelete, Path: "/doctors/{doctorId:[0-9]+}", Handler: hc.DeleteDoctorHandler},
		core.Route{Method: http.MethodPost, Path: "/doctors", Handler: hc.SaveDoctorHandler},
		core.Route{Method: http.MethodPost, Path: "/doctors/{doctorId:[0-9]+}/accounts", Handler: hc.SaveAccountForDoctorHandler},
		core.Route{Method: http.MethodGet, Path: "/me/doctors", Handler: hc.GetMyDoctorsHandler},

		core.Route{Method: http.MethodGet, Path: "/me/doctors/specific", Handler: hc.GetDoctorsSpecificHandler},
		core.Route{Method: http.MethodGet, Path: "/me/doctors/{doctorId:[0-9]+}/specific", Handler: hc.GetDoctorsSpecificHandler},

		core.Route{Method: http.MethodGet, Path: "/me/doctors/{doctorId:[0-9]+}/patients", Handler: hc.GetMyDoctorPatientsHandler},

		//Above DONE
		core.Route{Method: http.MethodGet, Path: "/patients", Handler: hc.GetPatientsHandler},
		core.Route{Method: http.MethodGet, Path: "/patients/{patientId:[0-9]+}", Handler: hc.GetPatientHandler},
		core.Route{Method: http.MethodPost, Path: "/patients", Handler: hc.SavePatientHandler},
		core.Route{Method: http.MethodGet, Path: "/me/patients", Handler: hc.GetMyPatientsHandler},

		//Hier anfangen
		core.Route{Method: http.MethodPost, Path: "/doctors/transfer", Handler: hc.TransferDoctorDataHandler},

		//only if doctor

		core.Route{Method: http.MethodGet, Path: "/me/statistics", Handler: hc.GetMyStatisticsHandler}, //only if doctor

		core.Route{Method: http.MethodPut, Path: "/me/tutorial/seen", Handler: hc.SetTutorialSeenHandler},

		//		core.Route{Method: http.MethodPost, Path: "/patients/register", Handler: hc.RegisterPatientHandler},

		core.Route{Method: http.MethodGet, Path: "/me/appointments", Handler: hc.GetMyAppointmentsHandler},
		core.Route{Method: http.MethodGet, Path: "/me/measurements", Handler: hc.GetMyMeasurementsHandler}, //only if patient

		core.Route{Method: http.MethodGet, Path: "/me/conversations", Handler: hc.GetMyChatsHandler},
		core.Route{Method: http.MethodGet, Path: "/me/conversations/{userId:[0-9]+}", Handler: hc.GetChatMessagesHandler},
		core.Route{Method: http.MethodGet, Path: "/me/conversations/{userId:[0-9]+}/{messageId:[0-9]+}", Handler: hc.GetNewChatMessagesHandler},
		core.Route{Method: http.MethodDelete, Path: "/me/conversations/{userId:[0-9]+}/{messageId:[0-9]+}", Handler: hc.DeleteChatMessagesHandler},
		core.Route{Method: http.MethodDelete, Path: "/me/conversations/{userId:[0-9]+}/messages/{messageId:[0-9]+}", Handler: hc.DeleteChatMessagesHandler},
		core.Route{Method: http.MethodPost, Path: "/me/conversations", Handler: hc.SaveChatMessageHandler},

		core.Route{Method: http.MethodGet, Path: "/me/consents", Handler: hc.GetMyConsentsHandler},
		core.Route{Method: http.MethodGet, Path: "/me/consents/pending", Handler: hc.GetMyPendingConsentsHandler},
		//TODO SM ASK JC was istconsent

		core.Route{Method: http.MethodPost, Path: "/me/consents", Handler: hc.SaveConsentHandler},
		core.Route{Method: http.MethodPatch, Path: "/me/consents/{consentId:[0-9]+}/accept", Handler: hc.AcceptConsentHandler},
		core.Route{Method: http.MethodPatch, Path: "/me/consents/{consentId:[0-9]+}/decline", Handler: hc.DeclineConsentHandler},

		core.Route{Method: http.MethodPost, Path: "/doctors/approach", Handler: hc.SaveDoctorApproachHandler},

		core.Route{Method: http.MethodDelete, Path: "/me/consents/{consentId:[0-9]+}", Handler: hc.DeleteConsentHandler},

		core.Route{Method: http.MethodGet, Path: "/me/user", Handler: hc.GetMyProfileHandler},
		core.Route{Method: http.MethodPost, Path: "/me/user", Handler: hc.SaveMyProfileHandler},
		core.Route{Method: http.MethodPost, Path: "/user/{userId:[0-9]+}/setting", Handler: hc.SaveAccountSettingHandler},

		core.Route{Method: http.MethodGet, Path: "/me/notifications", Handler: hc.GetMyNotificationsHandler},
		core.Route{Method: http.MethodPut, Path: "/me/notifications", Handler: hc.SetMyNotificationsReadHandler},
		core.Route{Method: http.MethodPut, Path: "/me/notifications/{notificationId:[0-9]+}", Handler: hc.SetMyNotificationsReadHandler},

		/*
			core.Route{Method:  http.MethodGet, 	Path:    "/me/devices", Handler: hc.GetMyDevicesHandler, }, //only if patient
		*/

		core.Route{Method: http.MethodGet, Path: "/appointments", Handler: hc.GetAppointmentsHandler},
		core.Route{Method: http.MethodGet, Path: "/appointments/requests", Handler: hc.GetOpenAppointmentsHandler},
		core.Route{Method: http.MethodGet, Path: "/appointments/{appointmentId:[0-9]+}", Handler: hc.GetAppointmentHandler},
		core.Route{Method: http.MethodPost, Path: "/appointments", Handler: hc.SaveAppointmentHandler},

		core.Route{Method: http.MethodPatch, Path: "/appointments/{appointmentId:[0-9]+}/accept", Handler: hc.AcceptAppointmentHandler},
		core.Route{Method: http.MethodPatch, Path: "/appointments/{appointmentId:[0-9]+}/decline", Handler: hc.DeclineAppointmentHandler},
		core.Route{Method: http.MethodPatch, Path: "/appointments/{appointmentId:[0-9]+}/reschedule", Handler: hc.RescheduleAppointmentHandler},
		core.Route{Method: http.MethodPost, Path: "/appointments/{appointmentId:[0-9]+}/reschedule", Handler: hc.RescheduleAppointmentHandler},

		core.Route{Method: http.MethodGet, Path: "/questions/templates", Handler: hc.GetQuestionTemplatesHandler},
		core.Route{Method: http.MethodGet, Path: "/questions/templates/{questionTemplateId:[0-9]+}", Handler: hc.GetQuestionTemplateHandler},
		core.Route{Method: http.MethodPost, Path: "/questions/templates", Handler: hc.SaveQuestionTemplateHandler},
		core.Route{Method: http.MethodGet, Path: "/questions/templates/patient/{patientId:[0-9]+}", Handler: hc.GetPatientQuestionTemplatesHandler},
		core.Route{Method: http.MethodGet, Path: "/questionnaires", Handler: hc.GetPatientQuestionnairesHandler},
		core.Route{Method: http.MethodGet, Path: "/questionnaires/unanswered", Handler: hc.GetUnansweredPatientQuestionnairesHandler},
		core.Route{Method: http.MethodGet, Path: "/questionnaires/podiatrists", Handler: hc.GetPodiatristQuestionsFromPatientQuestionnairesHandler},
		core.Route{Method: http.MethodGet, Path: "/questionnaires/{questionnaireId:[0-9]+}", Handler: hc.GetPatientQuestionnaireHandler},
		core.Route{Method: http.MethodPost, Path: "/questionnaires", Handler: hc.SavePatientQuestionnaireHandler},
		core.Route{Method: http.MethodPost, Path: "/questionnaires/{questionnaireId:[0-9]+}/question", Handler: hc.SavePatientQuestionnaireQuestionHandler},

		core.Route{Method: http.MethodGet, Path: "/questionnaires/questions/latest", Handler: hc.GetLatestPatientQuestionnaireQuestionsHandler},

		// for statistic / chart
		core.Route{Method: http.MethodGet, Path: "/questions/{questionTemplateId:[0-9]+}/patient/{patientId:[0-9]+}", Handler: hc.GetPatientQuestionnaireQuestionsHandler},

		// Charts für App
		core.Route{Method: http.MethodGet, Path: "/me/questions/templates", Handler: hc.GetMyQuestionTemplatesHandler},
		core.Route{Method: http.MethodGet, Path: "/me/questions/{questionTemplateId:[0-9]+}", Handler: hc.GetMyQuestionnaireQuestionsHandler},

		core.Route{Method: http.MethodPost, Path: "/questionnaires/patient/{patientId}", Handler: hc.CreatePatientQuestionnaireHandler},
		core.Route{Method: http.MethodGet, Path: "/questionnaires/daily", Handler: hc.CreateDailyPatientQuestionnairesHandler},
		//core.Route{Method: http.MethodGet, Path: "/questionnaires/daily/test", Handler: hc.CreateDailyPatientQuestionnaireTestHandler},

		core.Route{Method: http.MethodGet, Path: "/me/questionnaires", Handler: hc.GetSetupPatientQuestionnaireHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/next", Handler: hc.GetMeasurementNextHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/previous", Handler: hc.GetMeasurementPreviousHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/images/next", Handler: hc.GetMeasurementImagesNextHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/images/previous", Handler: hc.GetMeasurementImagesPreviousHandler},

		//TODO SM
		core.Route{Method: http.MethodGet, Path: "/measurements", Handler: hc.GetMeasurementsHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/download-report", Handler: hc.GetMeasurementsCsvFileHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/red", Handler: hc.GetMeasurementsWithStatusRedHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/blue", Handler: hc.GetMeasurementsWithStatusBlueHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/redblue", Handler: hc.GetMeasurementsWithStatusRedBlueHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/patient/{patientId:[0-9]+}", Handler: hc.GetMeasurementsForPatientHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}", Handler: hc.GetMeasurementHandler},
		// Deprecated for admin frontend
		core.Route{Method: http.MethodGet, Path: "/measurements/table-config", Handler: hc.GetMeasurementsTableConfigHandler},
		//DONE
		core.Route{Method: http.MethodPost, Path: "/measurements", Handler: hc.RetrieveMeasurementHandler},

		core.Route{Method: http.MethodDelete, Path: "/measurements/{measurementId:[0-9]+}", Handler: hc.DeleteMeasurementHandler},

		//END DONE
		core.Route{Method: http.MethodPost, Path: "/measurements/share", Handler: hc.ShareMeasurementHandler},
		core.Route{Method: http.MethodDelete, Path: "/measurements/shared/{measurementSharedId:[0-9]+}", Handler: hc.DeleteSharedMeasurementHandler},

		core.Route{Method: http.MethodPost, Path: "/measurement/{measurementId:[0-9]+}/riskRating", Handler: hc.SaveMeasurementRiskRatingHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/questionnaire", Handler: hc.GetMeasurementQuestionnaireHandler},

		core.Route{Method: http.MethodPost, Path: "/measurements/{measurementId:[0-9]+}/questionnaire", Handler: hc.SaveMeasurementQuestionnaireHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/setup-questionnaire", Handler: hc.GetMeasurementSetupQuestionnaireHandler},

		core.Route{Method: http.MethodPost, Path: "/measurements/{measurementId:[0-9]+}/favorite", Handler: hc.ToggleMeasurementFavoriteHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/files", Handler: hc.GetMeasurementsFilesHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/files", Handler: hc.GetMeasurementFilesHandler},
		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/files/{fileType}", Handler: hc.GetMeasurementFileHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/portal/images", Handler: hc.GetMeasurementImagesForPortalHandler},
		/*
			core.Route{Method:  http.MethodGet, 	Path:    "/patients/{patientId:[0-9]+}/measurements", Handler: hc.GetMeasurementsForPatientHandler, },
			core.Route{Method:  http.MethodGet, 	Path:    "/patients/{patientId:[0-9]+}/measurements/red", Handler: hc.GetMeasurementsForPatientWithStatusRedHandler, },
			core.Route{Method:  http.MethodGet, 	Path:    "/patients/{patientId:[0-9]+}/measurements/blue", Handler: hc.GetMeasurementsForPatientWithStatusBlueHandler, },
			core.Route{Method:  http.MethodGet, 	Path:    "/patients/{patientId:[0-9]+}/measurements/redblue", Handler: hc.GetMeasurementsForPatientWithStatusRedBlueHandler, },
			core.Route{Method:  http.MethodGet, 	Path:    "/patients/{patientId:[0-9]+}/measurements/{measurementId:[0-9]+}", Handler: hc.GetMeasurementHandler, },
		*/

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/annotations", Handler: hc.GetAnnotationsForMeasurementHandler},
		core.Route{Method: http.MethodPost, Path: "/measurements/{measurementId:[0-9]+}/annotations", Handler: hc.SaveAnnotationForMeasurementHandler},
		core.Route{Method: http.MethodDelete, Path: "/measurements/{measurementId:[0-9]+}/annotations/{annotationId:[0-9]+}", Handler: hc.DeleteAnnotationForMeasurementHandler},

		core.Route{Method: http.MethodGet, Path: "/measurements/{measurementId:[0-9]+}/pdf", Handler: hc.ExportMeasurementHandler},
		core.Route{Method: http.MethodPost, Path: "/measurements/{measurementId:[0-9]+}/send", Handler: hc.SendMailMeasurementHandler},

		core.Route{Method: http.MethodGet, Path: "/podiatrists/crawl", Handler: hc.CrawlPodiatristsHandler},

		/*
			core.Route{Method:  http.MethodGet, 	Path:    "/devices/", Handler: hc.GetDevicesHandler, }, // not used at the moment
			core.Route{Method:  http.MethodGet, 	Path:    "/devices/{deviceId:[0-9]+}", Handler: hc.GetDeviceHandler, }, // not used at the moment
			core.Route{Method:  http.MethodGet, 	Path:    "/devicetypes", Handler: hc.GetDeviceTypesHandler, }, // not used at the moment
			core.Route{Method:  http.MethodGet, 	Path:    "/devicetypes/{deviceTypeId:[0-9]+}", Handler: hc.GetDeviceTypeHandler, }, // not used at the moment

		*/

		core.Route{Method: http.MethodGet, Path: "/devices/system/version/portal-last", Handler: hc.GetCurrentPortalSystemVersionHandler},
		core.Route{Method: http.MethodGet, Path: "/devices/system/version/publish-last", Handler: hc.GetDeviceSystemVersionLastPublishHandler},
		core.Route{Method: http.MethodPost, Path: "/devices/system/version", Handler: hc.SaveDeviceSystemVersionHandler},
		core.Route{Method: http.MethodPost, Path: "/devices/type/{deviceTypeVersionId:[0-9]+}/upload/file", Handler: hc.UploadDeviceSystemVersionHandler},

		//core.Route{Method: http.MethodPost, Path: "/devices/system/version/update", Handler: hc.UpdateDeviceSystemVersionHandler},
		core.Route{Method: http.MethodPost, Path: "/devices/system/version/{systemVersionId:[0-9]+}/publish", Handler: hc.PublishDeviceSystemVersionHandler},

		core.Route{Method: http.MethodGet, Path: "/devices/system/{systemId:[0-9]+}/version", Handler: hc.GetDeviceSystemVersionHandler},

		core.Route{Method: http.MethodGet, Path: "/devices/type/{typeId:[0-9]+}/version", Handler: hc.GetDeviceTypeVersionHandler},
		core.Route{Method: http.MethodGet, Path: "/devices/type/{typeName}/version", Handler: hc.GetDeviceTypeVersionHandler},
		core.Route{Method: http.MethodGet, Path: "/devices/type/{typeId:[0-9]+}/version/{versionId:[0-9]+}/file/{fileType}", Handler: hc.GetDeviceTypeVersionFileHandler},
		core.Route{Method: http.MethodGet, Path: "/devices/type/{typeName}/version/{versionId:[0-9]+}/file/{fileType}", Handler: hc.GetDeviceTypeVersionFileHandler},
		core.Route{Method: http.MethodPost, Path: "/devices/version", Handler: hc.CreateDeviceTypeVersionHandler},
		core.Route{Method: http.MethodPost, Path: "/practice/device", Handler: hc.SavePracticeDeviceHandler},
		core.Route{Method: http.MethodGet, Path: "/devices/types", Handler: hc.GetDeviceTypesHandler},
		core.Route{Method: http.MethodGet, Path: "/devices/{deviceId:[0-9]+}", Handler: hc.GetDeviceHandler},
		core.Route{Method: http.MethodGet, Path: "/risks/definitions", Handler: hc.GetRiskDefinitionsHandler},

		core.Route{Method: http.MethodGet, Path: "/practices", Handler: hc.GetPracticesHandler},
		core.Route{Method: http.MethodGet, Path: "/practices/{practiceId:[0-9]+}", Handler: hc.GetPracticeHandler},
		core.Route{Method: http.MethodGet, Path: "/practices/{practiceId:[0-9]+}/doctors", Handler: hc.GetPracticeDoctorsHandler},
		core.Route{Method: http.MethodGet, Path: "/practices/{practiceId:[0-9]+}/patients", Handler: hc.GetPracticePatientsHandler},
		core.Route{Method: http.MethodGet, Path: "/practices/{practiceId:[0-9]+}/devices", Handler: hc.GetPracticeDevicesHandler},
		core.Route{Method: http.MethodPost, Path: "/practices", Handler: hc.SavePracticeHandler},

		core.Route{Method: http.MethodPost, Path: "/sendmail", Handler: hc.SendMail1},
		core.Route{Method: http.MethodPost, Path: "/patientImages", Handler: hc.SaveImagesForPatient},
		core.Route{Method: http.MethodGet, Path: "/ScanHistory/{MeasurementID}", Handler: hc.getPatientImagesFromDB},
		core.Route{Method: http.MethodGet, Path: "/ScanHistory", Handler: hc.ScanHistory},
		core.Route{Method: http.MethodGet, Path: "/serve-image/{filepath}", Handler: hc.serveImageHandler},

		//core.Route{Method: http.MethodGet, Path: "/mail/test", Handler: hc.TestMailHandler},
		//core.Route{Method: http.MethodPost, Path: "/practice/accounts", Handler: hc.GetPractice},

		core.Route{Method: http.MethodOptions, Path: "/{rest:.*}", Handler: hc.OptionsHandler},
	}
	
	return &PodiumBundle{
		routes: r,
	}
}

// GetRoutes implement interface core.Bundle
func (b *PodiumBundle) GetRoutes() []core.Route {
	return b.routes
}
