package systembundle

import (
	"github.com/jinzhu/gorm"
	"net/http"
	"thermetrix_backend/app/core"
)

// SystemBundle handle kitties resources
type SystemBundle struct {
	routes []core.Route
}

// NewSystemBundle instance
func NewSystemBundle(ormDB *gorm.DB, Users *map[string]core.User) core.Bundle {
	hc := NewSystemController(ormDB, Users)

	r := []core.Route{

		core.Route{Method: http.MethodPost, Path: "/import/whole-practice", Handler: hc.ImportWholePracticeHandler},

		core.Route{Method: http.MethodPost, Path: "/system/practice/{practiceId:[0-9]+}/import/doctors", Handler: hc.ImportDoctorsForAdminHandler},
		core.Route{Method: http.MethodPost, Path: "/system/me/import/doctors", Handler: hc.ImportDoctorsForPracticeHandler},

		core.Route{Method: http.MethodPost, Path: "/system/practice/{practiceId:[0-9]+}/import/patients", Handler: hc.ImportPatientsForAdminHandler},
		core.Route{Method: http.MethodPost, Path: "/system/me/import/patients", Handler: hc.ImportPatientsForPracticeHandler},

		core.Route{Method: http.MethodPost, Path: "/system/practice/{practiceId:[0-9]+}/import/doctors-patients", Handler: hc.ImportDoctorsPatientsForAdminHandler},
		core.Route{Method: http.MethodPost, Path: "/system/me/import/doctors-patients", Handler: hc.ImportDoctorsPatientsForPracticeHandler},

		core.Route{Method: http.MethodPost, Path: "/system/login", Handler: hc.Login},
		core.Route{Method: http.MethodPost, Path: "/system/logout", Handler: hc.Logout},
		core.Route{Method: http.MethodGet, Path: "/system/logo/{logo_type}", Handler: hc.LogoHandler},

		core.Route{Method: http.MethodPost, Path: "/patients/register", Handler: hc.RegisterPatientHandler},
		core.Route{Method: http.MethodPost, Path: "/register/patient", Handler: hc.RegisterPatientHandler},

		core.Route{Method: http.MethodPost, Path: "/register/practice", Handler: hc.RegisterPracticeHandler},
		core.Route{Method: http.MethodPost, Path: "/system/user/lock", Handler: hc.LockUserHandler},
		core.Route{Method: http.MethodPost, Path: "/system/user/unlock", Handler: hc.UnlockUserHandler},
		core.Route{Method: http.MethodPost, Path: "/system/user/password/request", Handler: hc.RequestPasswordResetHandler},
		core.Route{Method: http.MethodPost, Path: "/system/user/password/reset", Handler: hc.ResetPasswordHandler},
		core.Route{Method: http.MethodPost, Path: "/system/user/tos/accept", Handler: hc.TosAcceptedHandler},
		core.Route{Method: http.MethodGet, Path: "/system/tos/pdf", Handler: hc.GetTosPdfHandler},
		core.Route{Method: http.MethodGet, Path: "/system/tutorial/pdf", Handler: hc.GetTutroialPdfHandler},

		core.Route{Method: http.MethodGet, Path: "/system/frontend-translations", Handler: hc.GetFrontendTranslationsHandler},
		//Done
		core.Route{Method: http.MethodPost, Path: "/register/doctor", Handler: hc.RegisterDoctorHandler},

		core.Route{Method: http.MethodGet, Path: "/system/users", Handler: hc.GetUsersHandler},

		core.Route{Method: http.MethodGet, Path: "/system/server-config", Handler: hc.GetServerConfigHandler},

		/*
			core.Route{Method:  http.MethodGet, Path:    "/system/users/{id:[0-9]+}", Handler: hc.GetUserHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/users/contact/{cid:[0-9]+}", Handler: hc.GetUsersForContactHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/users/contacts", Handler: hc.GetContactsForUsersHandler, },

			core.Route{Method:  http.MethodPost, Path:    "/system/users/generate", Handler: hc.GeneratePasswordHandler, },
			core.Route{Method:  http.MethodPost, Path:    "/system/users", Handler: hc.SaveUserHandler, },
			core.Route{Method:  http.MethodPost, Path:    "/system/users/profile", Handler: hc.SaveUserProfileHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/users", Handler: hc.GetUsersHandler, },
			core.Route{Method:  http.MethodPost, Path:    "/system/users/lock", Handler: hc.LockUserHandler, },
			core.Route{Method:  http.MethodPost, Path:    "/system/users/unlock", Handler: hc.UnlockUserHandler, },


			core.Route{Method:  http.MethodGet, Path:    "/system/permissions", Handler: hc.GetAllPermissionsOverviewHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/permissions/roles/{rid:[0-9]+}", Handler: hc.GetAllPermissionsOverviewHandler, },

			core.Route{Method:  http.MethodGet, Path:    "/system/permissions/roles", Handler: hc.GetRolesHandler, },
			core.Route{Method:  http.MethodPost, Path:    "/system/permissions/roles", Handler: hc.SavePermissionsForRoleHandler, },



			core.Route{Method:  http.MethodPost, Path:    "/system/log/frontend", Handler: hc.LogFrontendEventHandler, },


			core.Route{Method:  http.MethodGet, Path:    "/system/log", Handler: hc.GetSystemLogsHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/users/activities", Handler: hc.GetUserActivitiesHandler, },



			core.Route{Method:  http.MethodPost, Path:    "/system/tooltips", Handler: hc.SaveSystemTooltipHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/tooltips", Handler: hc.GetSystemTooltipsHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/tooltips/{tooltipId:[0-9]+}", Handler: hc.GetSystemTooltipHandler, },
			core.Route{Method:  http.MethodGet, Path:    "/system/tooltips/key/{tooltipKey}", Handler: hc.GetSystemTooltipByKeyHandler, },
		*/

		core.Route{Method: http.MethodGet, Path: "/ws/ticket", Handler: hc.GetWSTicketHandler},
		core.Route{Method: http.MethodGet, Path: "/ws/test", Handler: hc.SendWSTestMessageHandler},
		core.Route{Method: http.MethodGet, Path: "/ws/{ticket}", Handler: hc.HandleConnections},

		core.Route{Method: http.MethodOptions, Path: "/system/{rest:.*}", Handler: hc.OptionsHandler},
		core.Route{Method: http.MethodOptions, Path: "/ws/{rest:.*}", Handler: hc.OptionsHandler},
	}

	if core.Config.Database.DeleteData {
		hc.DeleteDataForUsersNotIn(core.Config.Database.UsersToKeep)
	}
	return &SystemBundle{
		routes: r,
	}
}

// GetRoutes implement interface core.Bundle
func (b *SystemBundle) GetRoutes() []core.Route {
	return b.routes
}
