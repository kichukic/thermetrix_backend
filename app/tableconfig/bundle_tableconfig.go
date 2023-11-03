package tableconfig

import (
	"github.com/jinzhu/gorm"
	"net/http"
	"thermetrix_backend/app/core"
)

type TableConfigBundle struct {
	routes []core.Route
}

func NewTableConfigBundle(ormDB *gorm.DB, users *map[string]core.User) core.Bundle {
	hc := NewTableConfigController(ormDB, users)

	r := []core.Route{
		core.Route{Method: http.MethodGet, Path: "/tableconfig/configs", Handler: hc.GetDefaultTableConfigsHandler},
		core.Route{Method: http.MethodGet, Path: "/tableconfig/configs/{configTypeName}", Handler: hc.GetTableConfigHandler},
		core.Route{Method: http.MethodGet, Path: "/tableconfig/configs/{configTypeName}", Handler: hc.SaveTableConfig4UserHandler},

		core.Route{Method: http.MethodOptions, Path: "/tableconfig/{rest:.*}", Handler: hc.OptionsHandler},
	}

	return &TableConfigBundle{
		routes: r,
	}
}

// GetRoutes implement interface core.Bundle
func (b *TableConfigBundle) GetRoutes() []core.Route {
	return b.routes
}

func (b *TableConfigBundle) GetModuleId() uint {
	return 9
}
