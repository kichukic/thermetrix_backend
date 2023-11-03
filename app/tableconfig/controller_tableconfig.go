package tableconfig

import (
	"encoding/json"
	"errors"
//	"gotools/tools"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"strconv"
	"thermetrix_backend/app/core"
)

var loadedTableConfigs map[string]SCTableConfig

func Register4TableConfig(tableStruct interface{}) error {
	if loadedTableConfigs == nil {
		loadedTableConfigs = make(map[string]SCTableConfig)
	}
	tableConfigTypeName := getType(tableStruct)
	tableConfigTypeName = checkConfigTypeName(tableConfigTypeName)
	if _, ok := loadedTableConfigs[tableConfigTypeName]; ok {
		return errors.New("already exists")
	}
	// check if already exists in file system
	if tableConfig, err := getDefaultTableConfig(tableConfigTypeName); err != nil || tableConfig == nil {
		// not in file system, create default
		if tableConfig, err := createDefaultTableConfig(tableStruct, tableConfigTypeName); err == nil && tableConfig != nil {
			loadedTableConfigs[tableConfigTypeName] = *tableConfig
		}
	} else {
		loadedTableConfigs[tableConfigTypeName] = *tableConfig
	}
	return nil
}

func checkConfigTypeName(tableConfigType string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Println(err)
	}
	return strings.ToLower(reg.ReplaceAllString(tableConfigType, ""))
}
func getType(myvar interface{}) string {
	if t := reflect.TypeOf(myvar); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}
func getDefaultTableConfig(tableConfigType string) (*SCTableConfig, error) {
	path := getTableConfigPath(tableConfigType)
	filePath := path + "default.json"
	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}

	if data, err := ioutil.ReadFile(filePath); err != nil {
		return nil, err
	} else {
		tableConfig := SCTableConfig{}
		if err = json.Unmarshal(data, &tableConfig); err != nil {
			return nil, err
		}
		return &tableConfig, nil
	}

}
func createDefaultTableConfig(tableStruct interface{}, tableConfigType string) (*SCTableConfig, error) {
	if tableStruct == nil {
		return nil, errors.New("not valid")
	}
	path := getTableConfigPath(tableConfigType)
	filePath := path + "default.json"
	if _, err := os.Stat(filePath); err == nil {
		return nil, errors.New("already exists")
	}
	tableConfig := GetTableHeader(tableStruct)
	// saveToFile
	data, _ := json.Marshal(&tableConfig)
	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return nil, err
	}

	return &tableConfig, nil
}

func getTableConfigPath(tableConfigType string) string {
	tableConfigPath := core.Config.Server.TableConfigPath
	if tableConfigPath == "" {
		tableConfigPath = "./config/tableconfig"
	}
	if !strings.HasSuffix(tableConfigPath, "/") {
		tableConfigPath += "/"
	}
	tableConfigPath += tableConfigType + "/"
	os.MkdirAll(tableConfigPath, 0777)
	return tableConfigPath
}

func GetTableHeader(test interface{}) SCTableConfig {
	return GetTableHeaderWithTag(test, "sctable")
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake  = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func GetTableHeaderWithTag(test interface{}, tagName string) SCTableConfig {
	tableConfig := SCTableConfig{}
	tableHeaders := SCTableHeaders{}
	tableHeadersDisplay := []string{}
	tableActions := SCTableActions{}

	t := reflect.TypeOf(test)
	log.Println("Type:", t.Name())
	log.Println("Kind:", t.Kind())
	for i := 0; i < t.NumField(); i++ {
		tableHeader := SCTableHeader{}
		// Get the field, returns https://golang.org/pkg/reflect/#StructField
		field := t.Field(i)

		isDefaultDisplay := false
		isAction := false
		// Get the field tag value
		tag := field.Tag.Get(tagName)
		if tag == "-" || tag == "" {
			continue
		}
		tag = strings.Replace(tag, "#;", "###***+++***###", -1)
		tags := strings.Split(tag, ";")
		for _, tagData := range tags {
			tagData = strings.Replace(tagData, "###***+++***###", "#;", -1)
			tmp := strings.Split(tagData, ":")
			key := tmp[0]
			value := ""
			if len(tmp) > 1 {
				value = tmp[1]
			}

			switch key {
			case "title":
				tableHeader.Title = value // TODO Translate
			case "subtitle":
				tableHeader.SubtitleTitle = value
			case "sticky":
				tableHeader.Sticky, _ = strconv.ParseBool(value)
			case "subtitleDataKey":
				tableHeader.SubtitleDisplayBy = value
			case "dataKey":
				tableHeader.Index = value
			case "displayBy":
				tableHeader.DisplayBy = value
			case "align":
				tableHeader.Align = value
			case "concatWith":
				tableHeader.ConcatWith = value
			case "isDefaultDisplay":
				if value != "" {
					isDefaultDisplay, _ = strconv.ParseBool(value)
				} else {
					isDefaultDisplay = true
				}
			case "actions":
				isAction = true
				if len(tmp) > 3 {
					action := SCTableAction{
						Index: tmp[1],
						Label: tmp[2],
						Icon:  tmp[3],
					}
					tableActions = append(tableActions, action)
				}
			}
		}

		if isAction {
			continue
		}

		log.Printf("%d. %v (%v), tag: '%v'\n", i+1, field.Name, field.Type.Name(), tag)

		if tableHeader.Title == "" {
			tableHeader.Title = field.Name
		}
		if tableHeader.Type == "" {
			switch field.Type.Name() {
			case "int", "float64", "uint", "int64", "float32", "int32", "uint64", "uint32":
				tableHeader.Type = "number"
			case "core.CurrencyNumber", "CurrencyNumber":
				tableHeader.Type = "currency"
			case "time.Time", "*time.Time", "tools.NullTime", "*tools.NullTime":
				tableHeader.Type = "date"
			default:
				tableHeader.Type = "string"
			}
		}

		if tableHeader.Index == "" {
			tableHeader.Index = ToSnakeCase(field.Name)
		}
		if tableHeader.DisplayBy == "" {
			tableHeader.DisplayBy = tableHeader.Index
		}

		tableHeaders = append(tableHeaders, tableHeader)
		if isDefaultDisplay {
			tableHeadersDisplay = append(tableHeadersDisplay, tableHeader.Index)
		}
	}

	tableConfig.TableHeaders = tableHeaders
	tableConfig.TableHeadersDisplay = tableHeadersDisplay
	tableConfig.TableActions = tableActions

	return tableConfig
}
