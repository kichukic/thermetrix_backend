package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"
)

// swagger:model
type ResponseData struct {
	Status          int         `json:"status,omitempty"`
	Message         string      `json:"message,omitempty"`
	Detail          string      `json:"detail,omitempty"`
	Draw            int         `json:"draw,omitempty"`
	RecordsFiltered int         `json:"recordsFiltered,omitempty"`
	RecordsTotal    int         `json:"recordsTotal,omitempty"`
	Data            interface{} `json:"data,omitempty"`
	Paging          *Paging     `json:"paging,omitempty"`
	Sections        []Section   `json:"sections,omitempty"`
}

// swagger:model
type Filter struct {
	DateFrom NullTime        `json:"date_from,omitempty"`
	DateTo   NullTime        `json:"date_to,omitempty"`
	Limit    uint            `json:"limit,omitempty"`
	Filters  []CustomFilter  `json:"filters,omitempty"`
	OrderBy  []CustomOrderBy `json:"order_by,omitempty"`
	GroupBy  []CustomGroupBy `json:"group_by,omitempty"`
}

type FilterDefinition struct {
	Model
	FilterName string          `json:"filter_name,omitempty"`
	Limit      uint            `json:"limit,omitempty"`
	Filters    []CustomFilter  `json:"filters,omitempty"`
	OrderBy    []CustomOrderBy `json:"order_by,omitempty"`
	GroupBy    []CustomGroupBy `json:"group_by,omitempty"`
}

func (FilterDefinition) TableName() string {
	return "core_filter_definitions"
}

type CustomFilter struct {
	Model
	FilterDefinitionId uint        `json:"-"`
	Key                string      `json:"key"`
	DisplayValue       string      `json:"display_value"`
	Type               string      `json:"type"`
	Compare            string      `json:"compare"`
	Value              interface{} `json:"value" gorm:"TYPE:VARCHAR(255)"`
}

func (CustomFilter) TableName() string {
	return "core_custom_filters"
}

type CustomOrderBy struct {
	Model
	FilterDefinitionId uint   `json:"-"`
	Key                string `json:"key"`
	DisplayValue       string `json:"display_value"`
	IsDesc             bool   `json:"is_desc" gorm:"-"`
	IsActive           bool   `json:"is_active" gorm:"-"`
}

func (CustomOrderBy) TableName() string {
	return "core_custom_order_by"
}

type CustomGroupBy struct {
	Model
	FilterDefinitionId uint   `json:"-"`
	Key                string `json:"key"`
	DisplayValue       string `json:"display_value"`
	IsActive           bool   `json:"is_active" gorm:"-"`
}

func (CustomGroupBy) TableName() string {
	return "core_custom_group_by"
}

func (f Filter) ParseFilters() (string, []interface{}, error) {
	whereCond := ""
	whereCondValues := make([]interface{}, 0)

	for i, condition := range f.Filters {
		if i > 0 {
			whereCond += " AND "
		}
		//check for SQL injection
		err := condition.Validate()
		if err != nil {
			return "", nil, err
		}

		conditionAffix := ""
		conditionSuffix := ""
		if condition.Compare == "IN" || condition.Compare == "ANY" || condition.Compare == "ALL" || condition.Compare == "SOME" {
			conditionAffix = "("
			conditionSuffix = ")"
		}

		conditionTypeAffix := ""
		conditionTypeSuffix := ""
		if condition.Type == "date" {
			conditionTypeAffix = " DATE("
			conditionTypeSuffix = ")"
		}
		if condition.Type == "datetime" {
			conditionTypeAffix = " DATETIME("
			conditionTypeSuffix = ")"
		}

		whereCondValues = append(whereCondValues, condition.Value)

		whereCond += conditionTypeAffix + condition.Key + conditionTypeSuffix + " " + condition.Compare + conditionTypeAffix + conditionAffix + " ? " + conditionSuffix + conditionTypeSuffix
	}

	return whereCond, whereCondValues, nil
}

func (f CustomFilter) Validate() error {
	if !checkConditionKey(f.Key) {
		return errors.New("Key contains invalid characters")
	}
	if !checkConditionCompare(f.Compare) {
		return errors.New("Compare contains invalid characters")
	}
	return nil
}

func checkConditionKey(s string) bool {
	return regexp.MustCompile(`^[a-z0-9_]*$`).MatchString(s)
}

func checkConditionCompare(s string) bool {
	return (s == "=" || s == "<>" || s == "<" || s == "<=" || s == ">" || s == ">=" || s == "LIKE" || s == "IN" || s == "ANY" || s == "ALL" || s == "SOME")
}

// swagger:model
type FrontendFilter struct {
	Value   interface{} `json:"value"`
	Display string      `json:"display"`
}
type FrontendFilters []FrontendFilter

// swagger:model
type Section struct {
	Key   string `json:"key"`
	Index int    `json:"index"`
}

// swagger:model
type Model struct {
	ID        uint       `json:"id" gorm:"primary_key"`
	CreatedAt time.Time  `json:"-" `
	UpdatedAt time.Time  `json:"-" `
	DeletedAt *time.Time `json:"-" sql:"index"`
}

type CurrencyNumber float64

type Paging struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	//PageCount  int `json:"page_count"`
	TotalCount int `json:"total_count"`
	TotalPage  int `json:"total_page"`
	Offset     int `json:"offset"` // Helper
	Limit      int `json:"limit"`  // Helper

}

func (n CurrencyNumber) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.2f", n)), nil
}

// swagger:model
type HandleErrorData struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

type Email string

func (email *Email) Validate() bool {
	return true
}

type EncryptedValue string

func (eV EncryptedValue) String() string {
	return string(eV)
}

func (ev EncryptedValue) Value() (driver.Value, error) {
	key := []byte(Config.Customer.CustomerEncryptionKey)

	return []byte(encrypt(key, ev.String())), nil
}

func (ev *EncryptedValue) Scan(src interface{}) error {
	key := []byte(Config.Customer.CustomerEncryptionKey)
	var source []byte
	switch src.(type) {
	case string:
		source = []byte(src.(string))
	case []byte:
		source = src.([]byte)
	default:
		return errors.New("Incompatible type for encrypted value.")
	}
	blibb := EncryptedValue(decrypt(key, string(source)))

	*ev = blibb

	return nil
}

// encrypt string to base64 crypto using AES
func encrypt(key []byte, text string) string {
	// key := []byte(keyText)

	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return ""
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

// decrypt from base64 to decrypted string
func decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.x
	if len(ciphertext) < aes.BlockSize {
		return ""
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}

/*
func (g GzippedText) Value()	  (driver.Value, error)	  {
 	  	   b := make([]byte, 0, len(g))
 	  	   buf := bytes.NewBuffer(b)
 	  	   w := gzip.NewWriter(buf)
 	  	  	  w.Write(g)
 	  	  	  w.Close()
 	  	   return buf.Bytes(), nil
}
func (g *GzippedText) Scan(src interface{}) error {
	 	  	   var source []byte
	 	  	   //	  let's	  support	  string	  and	  []byte
	 	  	   switch src.(type)	  {
	 	  	   case string:
	 	  	  	  	  	  	  	  source	  =	  []byte(src.(string))
	 	  	   case []byte:
	 	  	  	  	  	  	  	  source	  =	  src.([]byte)
	 	  	   default:
	 	  	  	  	  	  	   return errors.New("Incompatible	  type	  for	  GzippedText")
	 	  	  	  }
	 	  	   reader, _ := gzip.NewReader(bytes.NewReader(source))
	 	  	   defer reader.Close()
	 	  	   b, err := ioutil.ReadAll(reader)
	 	  	   if err	  != nil {
		 	  	  	  	  	  	   return err
		 	  	  }
	 	  	  	  *g	  = GzippedText(b)
	 	  	   return nil
}
*/

const (
	Client_Portal     = "060a4e73-dcf5-4d6d-920a-6bee885806c9"
	Client_Admin      = "d539b1fe-ba20-11eb-8529-0242ac130003"
	Client_APK_Home   = "3efaccef-cdbf-4f9f-8973-1cb2243b33a6"
	Client_APK_Pro    = "501d7ff9-8a18-45fa-91a3-a554fc6144a9"
	CLIENT_APK_Remote = "46b09d33-fa57-4d7a-bc7f-2786ae791045"
)
