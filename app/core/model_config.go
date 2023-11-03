package core

// swagger:model
type Configuration struct {
	Database   ConfigurationDatabase   `json:"database"`
	Server     ConfigurationServer     `json:"server"`
	MailServer ConfigurationMailServer `json:"mail_server"`
	Customer   ConfigurationCustomer   `json:"customer"`
	DMS        ConfigurationDMS        `json:"dms"`
	Portal     ConfigurationPortal     `json:"portal"`
}

// swagger:model
type ConfigurationDatabase struct {
	Host                  string   `json:"host"`
	Database              string   `json:"database"`
	User                  string   `json:"user"`
	Server                string   `json:"server"`
	Password              string   `json:"password"`
	Port                  int      `json:"port"`
	DoAutoMigrate         bool     `json:"do_auto_migrate"`
	DoInsert              bool     `json:"do_insert"`
	DeleteData            bool     `json:"delete_data"`
	UsersToKeep           []string `json:"users_to_keep"`
	Debug                 bool     `json:"debug"`
	InitSetIsTenMinsApart bool     `json:"init_set_is_ten_mins_apart"`
}

// swagger:model
type ConfigurationServer struct {
	Hostname          string `json:"hostname"`
	FrontendHostname  string `json:"frontend_hostname"`
	AdminFrontendHostname     string `json:"admin_frontend_hostname"`
	InternalPort      int    `json:"internal_port"`
	ExternalPort      int    `json:"external_port"`
	Path              string `json:"path"`
	UpdatePath        string `json:"update_path"`
	WithSSL           bool   `json:"with_ssl"`
	SSLCertFile       string `json:"ssl_cert_file"`
	SSLKeyFile        string `json:"ssl_key_file"`
	TableConfigPath   string `json:"table_config_path"`
	UploadFilepath    string `json:"upload_filepath"`
	DeliverFrontEnd   bool   `json:"deliver_front_end"`
	FrontEndPath      string `json:"front_end_path"`
	DeliverAdminFrontEnd bool `json:"deliver_admin_front_end"`
	AdminFrontEndPath string `json:"admin_front_end_path"`
	CustomName        string `json:"custom_name"`
	CustomerServerKey string `json:"customer_server_key"`
	TmpPath           string `json:"tmp_path"`
}

// swagger:model
type ConfigurationMailServer struct {
	SmtpHost     string `json:"smtp_host"`
	SmtpPort     int    `json:"smtp_port"`
	SmtpUsername string `json:"smtp_username"`
	SmtpPassword string `json:"smtp_password"`
}

// swagger:model
type ConfigurationCustomer struct {
	/*CustomerName			string		`json:"name"`
	CustomerFullName		string		`json:"full_name"`
	CustomerAddress			ConfigurationCustomerAddress		`json:"address"`
	CustomerContact			ConfigurationCustomerContact	`json:"contact"`

	CustomerRegisterCourt	string		`json:"register_court"`
	CustomerCompanyRegister	string		`json:"company_register"`
	CustomerVatNumber		string		`json:"vat_number"`
	CustomerTaxNumber		string		`json:"tax_number"`

	CeoName					string		`json:"ceo_name"`
	BankAccounts			[]ConfigurationCustomerBankAccount		`json:"bank_accounts"`
	*/
	CustomerEncryptionKey string `json:"encryption_key"`
	CustomerBankingApi    string `json:"banking_api"`
}

// swagger:model
type ConfigurationCustomerBankAccount struct {
	BankName          string `json:"bank_name"`
	BankAccountNumber string `json:"account_number"`
	BankNumber        string `json:"bank_number"`
	BankIban          string `json:"iban"`
	BankBic           string `json:"bic"`
}

// swagger:model
type ConfigurationCustomerAddress struct {
	Name              string `json:"name"`
	Street            string `json:"street"`
	PostalCode        string `json:"postal_code"`
	City              string `json:"city"`
	Country           string `json:"country"`
	AdditionalAddress string `json:"additional_address"`
	AddressField      string `json:"address_field"`
}

// swagger:model
type ConfigurationCustomerContact struct {
	Phone   string `json:"phone"`
	Fax     string `json:"fax"`
	Email   string `json:"email"`
	Website string `json:"website"`
}

type ConfigurationPortal struct {
	Address string `json:"address"`
}

/*
{
      "provider": "finapi",
      "client_id": "7f881ec3-d954-4867-af22-f272d5f5c774",
      "shared_secret": "78411396-56f6-40f1-9d9a-c9a91071de24",
      "admin_id": "ad42a961-6be3-4f21-910f-c4cb874bb91b",
      "admin_secret": "41ad0bdd-4aa1-4bc9-b5b3-ac5337ee820a"
    }


/ / swagger:model
type ConfigurationCustomerBankApi struct {
	provider			string		`json:"provider"`
	client_id			string		`json:"client_id"`
	shared_secret		string		`json:"shared_secret"`
	admin_id			string		`json:"admin_id"`
	admin_secret		string		`json:"admin_secret"`
}
*/

// swagger:model
type ConfigurationDMS struct {
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}
