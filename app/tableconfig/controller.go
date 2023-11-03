package tableconfig

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"thermetrix_backend/app/core"
)

type TableConfigController struct {
	core.Controller
	ormDB *gorm.DB
}

func NewTableConfigController(ormDB *gorm.DB, users *map[string]core.User) *TableConfigController {

	if core.Config.Database.Debug {
		ormDB = ormDB.Debug()
	}

	c := &TableConfigController{
		Controller: core.Controller{Users: users},
		ormDB:      ormDB,
	}

	if core.Config.Database.DoAutoMigrate {
		c.ormDB.AutoMigrate(&TableConfigUserSetting{})
	}

	return c
}

func (c *TableConfigController) GetDefaultTableConfigsHandler(w http.ResponseWriter, r *http.Request) {
	configs := TableConfigs{}
	for key, tableConfig := range loadedTableConfigs {
		config := TableConfig{
			ConfigType: key,
			Config:     tableConfig,
		}
		configs = append(configs, config)
	}
	c.SendJSON(w, &configs, http.StatusOK)
}
func (c *TableConfigController) GetTableConfigHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configTypeName := vars["configTypeName"]
	if data, err := GetTableConfig(c.ormDB, r, configTypeName); err != nil || data == nil {
		c.HandleError(err, w)
	} else {
		c.SendJSON(w, &data, http.StatusOK)
	}
}

func GetTableConfig(ormDB *gorm.DB, r *http.Request, configTypeName string) (interface{}, error) {
	configTypeName = checkConfigTypeName(configTypeName)
	onlyDefault, mergeWithDefault := false, false
	_ = mergeWithDefault // TODO implement
	defaultPath := getTableConfigPath(configTypeName)
	defaultPath += "default.json"
	if configStringDefault, err := ioutil.ReadFile(defaultPath); err != nil {
		return nil, errors.New("default config does not exists")
	} else {
		// always reload from file system
		//if asString {
		//	if onlyDefault {
		//		return string(configStringDefault), nil
		//	} else {
		//		// get from Database for User
		//		tableConfigUserSetting := TableConfigUserSetting{}
		//		ormDB.Where("user_id=? AND table_config_type_name=?", user.GetId(), configTypeName).First(&tableConfigUserSetting)
		//		if tableConfigUserSetting.ID > 0 {
		//			return tableConfigUserSetting.TableHeaderConfigData, nil
		//		} else {
		//			// nur default senden
		//			return string(configStringDefault), nil
		//		}
		//	}
		//} else {
		configDataDefault := SCTableConfig{}
		json.Unmarshal(configStringDefault, &configDataDefault)
		if onlyDefault {
			return &configDataDefault, nil
		} else {
			// get from Database for User
			tableConfigUserSetting := TableConfigUserSetting{}
			user := core.User{}
			ormDB.Where("user_id=? AND table_config_type_name=?", user.ID, configTypeName).First(&tableConfigUserSetting)
			if tableConfigUserSetting.ID > 0 {
				tableHeadersDisplayUser := []string{}
				json.Unmarshal([]byte(tableConfigUserSetting.TableHeaderDisplayConfigData), &tableHeadersDisplayUser)
				configDataDefault.TableHeadersDisplay = tableHeadersDisplayUser
				return &configDataDefault, nil
			} else {
				// nur default senden
				return &configDataDefault, nil
			}
		}
		//}
	}

	return nil, errors.New("Not found")
}

func (c *TableConfigController) SaveTableConfig4UserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configTypeName := vars["configTypeName"]
	scTableConfig := SCTableConfig{}
	if err := c.GetContent(&scTableConfig, r); err != nil {
		log.Println(err)
		if c.HandleError(err, w) {
			return
		}
	}
	data, _ := json.Marshal(&scTableConfig)
	configTypeName = checkConfigTypeName(configTypeName)
	// check if already table config exists for User
	tableConfigUserSetting := TableConfigUserSetting{}
	user := core.User{}
	c.ormDB.Where("user_id=? AND table_config_type_name=?", user.ID, configTypeName).First(&tableConfigUserSetting)
	if tableConfigUserSetting.ID == 0 {
		// new
		tableConfigUserSetting.UserId = user.ID
		tableConfigUserSetting.TableConfigTypeName = configTypeName
		tableConfigUserSetting.TableHeaderDisplayConfigData = string(data)
		c.ormDB.Create(&tableConfigUserSetting)
	} else {
		tableConfigUserSetting.TableHeaderDisplayConfigData = string(data)
		c.ormDB.Save(&tableConfigUserSetting)
	}

	c.SendJSON(w, &scTableConfig, http.StatusOK)
}
