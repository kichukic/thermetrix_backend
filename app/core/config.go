package core

import (
	"os"
	"strconv"
	"strings"
)

/*
// Config struct
type Config struct {
	DBConnection string
	DBType       string
}

// Fetch from env variables
func (c *Config) Fetch() {
	c.DBConnection = os.Getenv("DB_CONNECTION")
	c.DBType = os.Getenv("DB_TYPE")
}
*/

func GetEnvironmentConfig(c *Configuration) {
	if(os.Getenv("DATABASE_HOST") != "") {
		c.Database.Host = os.Getenv("DATABASE_HOST")
	}
	if(os.Getenv("DATABASE_DATABASE") != "") {
		c.Database.Database = os.Getenv("DATABASE_DATABASE")
	}
	if(os.Getenv("DATABASE_USER") != "") {
		c.Database.User = os.Getenv("DATABASE_USER")
	}
	if(os.Getenv("DATABASE_SERVER") != "") {
		c.Database.Server = os.Getenv("DATABASE_SERVER")
	}
	if(os.Getenv("DATABASE_PASSWORD") != "") {
		c.Database.Password = os.Getenv("DATABASE_PASSWORD")
	}
	if(os.Getenv("DATABASE_PORT") != "") {
		c.Database.Port, _ = strconv.Atoi(os.Getenv("DATABASE_PORT"))
	}
	if(os.Getenv("DATABASE_DO_AUTO_MIGRATE") != "") {
		c.Database.DoAutoMigrate, _ = strconv.ParseBool(os.Getenv("DATABASE_DO_AUTO_MIGRATE"))
	}
	if(os.Getenv("DATABASE_DO_INSERT") != "") {
		c.Database.DoInsert, _ = strconv.ParseBool(os.Getenv("DATABASE_DO_INSERT"))
	}
	if(os.Getenv("DATABASE_DELETE_DATA") != "") {
		c.Database.DeleteData, _ = strconv.ParseBool(os.Getenv("DATABASE_DELETE_DATA"))
	}
	if(os.Getenv("DATABASE_USERS_TO_KEEP") != "") {
		c.Database.UsersToKeep = strings.Split(os.Getenv("DATABASE_USERS_TO_KEEP"), ",")
	}
	if(os.Getenv("DATABASE_DEBUG") != "") {
		c.Database.Debug, _ = strconv.ParseBool(os.Getenv("DATABASE_DEBUG"))
	}
	if(os.Getenv("DATABASE_INIT_SET_IS_TEN_MINS_APART") != "") {
		c.Database.InitSetIsTenMinsApart, _ = strconv.ParseBool(os.Getenv("DATABASE_INIT_SET_IS_TEN_MINS_APART"))
	}

	if(os.Getenv("SERVER_HOSTNAME") != "") {
		c.Server.Hostname = os.Getenv("SERVER_HOSTNAME")
	}
	if(os.Getenv("SERVER_FRONTEND_HOSTNAME") != "") {
		c.Server.AdminFrontendHostname = os.Getenv("SERVER_FRONTEND_HOSTNAME")
	}
	if(os.Getenv("SERVER_ADMIN_FRONTEND_HOSTNAME") != "") {
		c.Server.AdminFrontendHostname = os.Getenv("SERVER_ADMIN_FRONTEND_HOSTNAME")
	}
	if(os.Getenv("SERVER_INTERNAL_PORT") != "") {
		c.Server.InternalPort, _ = strconv.Atoi(os.Getenv("SERVER_INTERNAL_PORT"))
	}
	if(os.Getenv("SERVER_EXTERNAL_PORT") != "") {
		c.Server.ExternalPort, _ = strconv.Atoi(os.Getenv("SERVER_EXTERNAL_PORT"))
	}
	if(os.Getenv("SERVER_PATH") != "") {
		c.Server.Path = os.Getenv("SERVER_PATH")
	}
	if(os.Getenv("SERVER_UPDATE_PATH") != "") {
		c.Server.UpdatePath = os.Getenv("SERVER_UPDATE_PATH")
	}
	if(os.Getenv("SERVER_WITH_SSL") != "") {
		c.Server.WithSSL, _ = strconv.ParseBool(os.Getenv("SERVER_WITH_SSL"))
	}
	if(os.Getenv("SERVER_SSL_CERT_FILE") != "") {
		c.Server.SSLCertFile = os.Getenv("SERVER_SSL_CERT_FILE")
	}
	if(os.Getenv("SERVER_SSL_KEY_FILE") != "") {
		c.Server.SSLKeyFile = os.Getenv("SERVER_SSL_KEY_FILE")
	}
	if(os.Getenv("SERVER_TABLE_CONFIG_PATH") != "") {
		c.Server.TableConfigPath = os.Getenv("SERVER_TABLE_CONFIG_PATH")
	}
	if(os.Getenv("SERVER_UPLOAD_FILEPATH") != "") {
		c.Server.UploadFilepath = os.Getenv("SERVER_UPLOAD_FILEPATH")
	}
	if(os.Getenv("SERVER_DELIVER_FRONT_END") != "") {
		c.Server.DeliverFrontEnd, _ = strconv.ParseBool(os.Getenv("SERVER_DELIVER_FRONT_END"))
	}
	if(os.Getenv("SERVER_FRONT_END_PATH") != "") {
		c.Server.FrontEndPath = os.Getenv("SERVER_FRONT_END_PATH")
	}
	if(os.Getenv("SERVER_DELIVER_ADMIN_FRONT_END") != "") {
		c.Server.DeliverAdminFrontEnd, _ = strconv.ParseBool(os.Getenv("SERVER_DELIVER_ADMIN_FRONT_END"))
	}
	if(os.Getenv("SERVER_ADMIN_FRONT_END_PATH") != "") {
		c.Server.AdminFrontEndPath = os.Getenv("SERVER_ADMIN_FRONT_END_PATH")
	}
	if(os.Getenv("SERVER_CUSTOM_NAME") != "") {
		c.Server.CustomName = os.Getenv("SERVER_CUSTOM_NAME")
	}
	if(os.Getenv("SERVER_CUSTOMER_SERVER_KEY") != "") {
		c.Server.CustomerServerKey = os.Getenv("SERVER_CUSTOMER_SERVER_KEY")
	}
	if(os.Getenv("SERVER_TMP_PATH") != "") {
		c.Server.TmpPath = os.Getenv("SERVER_TMP_PATH")
	}
	
	if(os.Getenv("MAIL_SERVER_SMTP_HOST") != "") {
		c.MailServer.SmtpHost = os.Getenv("MAIL_SERVER_SMTP_HOST")
	}
	if(os.Getenv("MAIL_SERVER_SMTP_PORT") != "") {
		c.MailServer.SmtpPort, _ = strconv.Atoi(os.Getenv("MAIL_SERVER_SMTP_PORT"))
	}
	if(os.Getenv("MAIL_SERVER_SMTP_USERNAME") != "") {
		c.MailServer.SmtpUsername = os.Getenv("MAIL_SERVER_SMTP_USERNAME")
	}
	if(os.Getenv("MAIL_SERVER_SMTP_PASSWORD") != "") {
		c.MailServer.SmtpPassword = os.Getenv("MAIL_SERVER_SMTP_PASSWORD")
	}

	if(os.Getenv("CUSTOMER_ENCRYPTION_KEY") != "") {
		c.Customer.CustomerEncryptionKey = os.Getenv("CUSTOMER_ENCRYPTION_KEY")
	}
	if(os.Getenv("CUSTOMER_BANKING_API") != "") {
		c.Customer.CustomerBankingApi = os.Getenv("CUSTOMER_BANKING_API")
	}

	// if(os.Getenv("CUSTOMER_BANK_BANK_NAME") != "") {
	// 	c.Customer.BankAccounts.BankName = os.Getenv("CUSTOMER_BANK_BANK_NAME")
	// }
	// if(os.Getenv("CUSTOMER_BANK_ACCOUNT_NUMBER") != "") {
	// 	c.Customer.BankAccounts.BankAccountNumber = os.Getenv("CUSTOMER_BANK_ACCOUNT_NUMBER")
	// }
	// if(os.Getenv("CUSTOMER_BANK_BANK_NUMBER") != "") {
	// 	c.Customer.BankAccounts.BankNumber = os.Getenv("CUSTOMER_BANK_BANK_NUMBER")
	// }
	// if(os.Getenv("CUSTOMER_BANK_IBAN") != "") {
	// 	c.Customer.BankAccounts.BankIban = os.Getenv("CUSTOMER_BANK_IBAN")
	// }
	// if(os.Getenv("CUSTOMER_BANK_BIC") != "") {
	// 	c.Customer.BankAccounts.BankBic = os.Getenv("CUSTOMER_BANK_BIC")
	// }

	// if(os.Getenv("CUSTOMER_ADDRESS_NAME") != "") {
	// 	c.Customer.CustomerAddress.Name = os.Getenv("CUSTOMER_ADDRESS_NAME")
	// }
	// if(os.Getenv("CUSTOMER_ADDRESS_STREET") != "") {
	// 	c.Customer.CustomerAddress.Street = os.Getenv("CUSTOMER_ADDRESS_STREET")
	// }
	// if(os.Getenv("CUSTOMER_ADDRESS_POSTAL_CODE") != "") {
	// 	c.Customer.CustomerAddress.PostalCode = os.Getenv("CUSTOMER_ADDRESS_POSTAL_CODE")
	// }
	// if(os.Getenv("CUSTOMER_ADDRESS_CITY") != "") {
	// 	c.Customer.CustomerAddress.City = os.Getenv("CUSTOMER_ADDRESS_CITY")
	// }
	// if(os.Getenv("CUSTOMER_ADDRESS_COUNTRY") != "") {
	// 	c.Customer.CustomerAddress.Country = os.Getenv("CUSTOMER_ADDRESS_COUNTRY")
	// }
	// if(os.Getenv("CUSTOMER_ADDRESS_ADDITIONAL_ADDRESS") != "") {
	// 	c.Customer.CustomerAddress.AdditionalAddress = os.Getenv("CUSTOMER_ADDRESS_ADDITIONAL_ADDRESS")
	// }
	// if(os.Getenv("CUSTOMER_ADDRESS_ADDRESS_FIELD") != "") {
	// 	c.Customer.CustomerAddress.AddressField = os.Getenv("CUSTOMER_ADDRESS_ADDRESS_FIELD")
	// }

	// if(os.Getenv("CUSTOMER_CONTACT_PHONE") != "") {
	// 	c.Customer.CustomerContact.Phone = os.Getenv("CUSTOMER_CONTACT_PHONE")
	// }
	// if(os.Getenv("CUSTOMER_CONTACT_FAX") != "") {
	// 	c.Customer.CustomerContact.Fax = os.Getenv("CUSTOMER_CONTACT_FAX")
	// }
	// if(os.Getenv("CUSTOMER_CONTACT_EMAIL") != "") {
	// 	c.Customer.CustomerContact.Email = os.Getenv("CUSTOMER_CONTACT_EMAIL")
	// }
	// if(os.Getenv("CUSTOMER_CONTACT_WEBSITE") != "") {
	// 	c.Customer.CustomerContact.Website = os.Getenv("CUSTOMER_CONTACT_WEBSITE")
	// }

	if(os.Getenv("DMS_URL") != "") {
		c.DMS.Url = os.Getenv("DMS_URL")
	}
	if(os.Getenv("DMS_USERNAME") != "") {
		c.DMS.Username = os.Getenv("DMS_USERNAME")
	}
	if(os.Getenv("DMS_PASSWORD") != "") {
		c.DMS.Password = os.Getenv("DMS_PASSWORD")
	}

	if(os.Getenv("PORTAL_ADDRESS") != "") {
		c.Portal.Address = os.Getenv("PORTAL_ADDRESS")
	}
}