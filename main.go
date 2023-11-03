// Package classification thermetrix API
//
// Thermetrix API
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     Host: thermetrix.works4dev.de
//     BasePath: /api/v1
//     Version: 0.0.1
//      Contact: symblCrowd <apps@symblcrowd.de> https://www.symblcrowd.de
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package main

//swagger generate spec --scan-models -o ./swagger.json
//go install github.com/go-swagger/go-swagger/cmd/swagger

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"thermetrix_backend/app/tableconfig"
	"encoding/json"
	"flag"
	"github.com/jinzhu/gorm"
	"io"
	"math/rand"
	"os"
	"strings"
	"thermetrix_backend/app/core"
	"thermetrix_backend/app/podiumbundle"
	"thermetrix_backend/app/systembundle"
	"time"
)


var (
	name    = "works4"
	v       = "undefined"
	address string // address of the server
	ormDB   *gorm.DB
	Users   map[string]core.User
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("----")
	startServer()
	log.Println("----")

}

func initBundles(users *map[string]core.User) []core.Bundle {
	return []core.Bundle{
		systembundle.NewSystemBundle(ormDB, users),
		podiumbundle.NewPodiumBundle(ormDB, users),
		tableconfig.NewTableConfigBundle(ormDB, users),
	}
}

// Server starten mit: works4_backend -configFile=/var/works4/symblcrowd/config.json
func startServer() error {

	



	f, err := os.OpenFile(fmt.Sprintf("logs/log_runtime_%s", time.Now().String()), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	log.Println(err)
	_, err = f.WriteString("Start\n")
	log.Println(err)
	defer f.Close()

	log.Println("----")
	configFile := ""
	flag.StringVar(&configFile, "configFile", "config.json", "a string")
	flag.Parse()

	if configFile == "" {
		configFile = "config.json"
	}
	log.Println("using configfile: ", configFile)
	log.Println("----")

	file, _ := os.Open(configFile)
	decoder := json.NewDecoder(file)
	core.Config = core.Configuration{}
	err = decoder.Decode(&core.Config)
	if err != nil {
		log.Println("error: ", err)
	}

	core.GetEnvironmentConfig(&core.Config)

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", core.Config.Database.User, core.Config.Database.Password, core.Config.Database.Host, core.Config.Database.Port, core.Config.Database.Database)
	log.Println(">>>>>>>>>>>>>>>",core.Config.Database.User, core.Config.Database.Password, core.Config.Database.Host, core.Config.Database.Port, core.Config.Database.Database)
	log.Print("connecting to database... ")
	ormdb, err := gorm.Open("mysql", dataSourceName)
	for err != nil {
		log.Println(err)
		ormdb, err = gorm.Open("mysql", dataSourceName)
		time.Sleep(3)
	}
	log.Println("done")

	ormdb.Debug().Exec("SET NAMES utf8")
	ormdb.Debug().Exec("SET time_zone = \"+00:00\"")
	ormdb.Debug().Exec("SET @@session.time_zone = \"+00:00\"")
	ormDB = ormdb
	ormDB.LogMode(false)

	Users = make(map[string]core.User)

	accountsSessions := systembundle.SystemAccountsSessions{}
	ormdb.Preload("Account").Find(&accountsSessions)

	log.Print("reading account sessions tokens... ")
	for _, session := range accountsSessions {
		session.Account.Token = session.SessionToken
		Users[session.SessionToken] = session.Account
	}
	log.Println("done")


	r := mux.NewRouter()
	s := r.Host(core.Config.Server.Hostname).PathPrefix("/api/v1/").Subrouter()

	log.Print("Adding routes... ")
	for _, b := range initBundles(&Users) {
		for _, route := range b.GetRoutes() {
			//s.HandleFunc(route.Path, route.Handler).Methods(route.Method)
			s.Handle(route.Path, middleWare(f, route.Handler)).Methods(route.Method)

		}
	}
	log.Println("done")



	// Routes handling
	//http.Handle("/", r)

	if  core.Config.Server.DeliverAdminFrontEnd &&
		core.Config.Server.DeliverFrontEnd &&
		core.Config.Server.FrontendHostname ==
			core.Config.Server.AdminFrontendHostname {
		log.Fatal("Frontend hostname and Admin frontend may not be the same if both are to be served!")	
	}

	if core.Config.Server.DeliverAdminFrontEnd {
		r_admin_portal := r.Host(core.Config.Server.AdminFrontendHostname)
		deliverFrontEnd(core.Config.Server.AdminFrontEndPath, r_admin_portal)
	}

	if core.Config.Server.DeliverFrontEnd {
		r_portal := r.Host(core.Config.Server.FrontendHostname)
		deliverFrontEnd(core.Config.Server.FrontEndPath, r_portal)
	} 

	address := fmt.Sprintf(":%d", core.Config.Server.InternalPort)
	log.Println(address)

	if core.Config.Server.WithSSL {
		log.Fatal(http.ListenAndServeTLS(address, core.Config.Server.SSLCertFile, core.Config.Server.SSLKeyFile, r))
	} else {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", core.Config.Server.InternalPort), r))
	}

	

	return nil
}


// hello world





//TODO TEST ACCOUNT_LOCKED WEBSOCKET
func middleWare(f *os.File, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now().UnixNano()

		//log.Println("Before")

		auth := r.Header.Get("Authorization")
		//log.Println("auth: ", auth)
		user := core.User{}
		ok := false
		var userId uint = 0
		tmp := strings.Split(auth, " ")
		if len(tmp) == 2 {
			if user, ok = Users[tmp[1]]; ok {
				userId = user.ID
			}
		}

		log.Println(r.Method)
		log.Println(r.RequestURI)

		if userId == 0 && !(r.RequestURI == "/api/v1/system/login") && (r.RequestURI == "devices/system/version/portal-last") && !(r.RequestURI == "system/frontend-translations") && !(strings.Contains(r.RequestURI, "/api/v1/system/logo/")) && !(r.RequestURI == "/api/v1/system/login?practice=true") && !(r.RequestURI == "/api/v1/system/login?practice=false") && !(r.RequestURI == "/api/v1/patients/register") && !(r.RequestURI == "/api/v1/questionnaires/daily") && !(r.RequestURI == "/api/v1/register/practice") && !(r.RequestURI == "/api/v1/register/practice?doctor_account=true") && !(r.RequestURI == "/api/v1/register/practice?doctor_account=false") && !(r.RequestURI == "/api/v1/register/patient") && !(r.RequestURI == "/api/v1/system/user/password/request") && !(r.RequestURI == "/api/v1/system/user/password/reset") && r.Method != http.MethodOptions && !strings.Contains(r.RequestURI, "/api/v1/ws/") { //r.RequestURI == "/api/v1/ws/ticket" ||
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusUnauthorized)
			msg := core.ResponseData{
				Status:  997,
				Message: "You are not authorized, please login!",
			}
			b, _ := json.Marshal(msg)
			io.WriteString(w, string(b))
			return
		}

		if userId > 0 && !user.IsActive {
			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusUnauthorized)
			msg := core.ResponseData{
				Status:  core.Account_Locked,
				Message: "Account locked",
			}

			b, _ := json.Marshal(msg)
			io.WriteString(w, string(b))
			return
		}
		//log.Println("authorized")

		sqlCmd := `INSERT INTO system_log (user_id, log_type, log_date, log_title, log_text) VALUES (?, ?, NOW(), ?, ?)`
		_, err := ormDB.DB().Exec(sqlCmd, userId, 1, "open Route", r.Header.Get("Client")+" "+r.Method+" "+r.RequestURI)
		if err != nil {
			log.Println(err)
		}

		h.ServeHTTP(w, r) // call original

		ende := time.Now().UnixNano()
		dauer := ende - start
		//log.Println("After")

		text := fmt.Sprintf("Time: %s - Dauer: %f - Route: %s\n", time.Now().Format("2006-01-02 15:04:05"), float64(dauer)/1000000000.0, r.RequestURI)
		_, err = f.WriteString(text)
		if err != nil {
			//log.Println(err)
		}

	})
}

func deliverFrontEnd(frontendOSPath string, r *mux.Route) {
	s0 := r.PathPrefix("/").Subrouter()
	if frontendOSPath == "" {
		frontendOSPath = "./"
	}
	s0.HandleFunc("/{rest:.*}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//r.URL.Path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/%s"))
		log.Println("r.URL.Path", r.URL.Path)

		r.URL.Path = frontendOSPath + "/" + r.URL.Path;
		r.URL.Path = strings.Replace(r.URL.Path, "..", "", -1)
		r.URL.Path = strings.Replace(r.URL.Path, "//", "/", -1)
		log.Println("r.URL.Path", r.URL.Path)

		if _, err := os.Stat(r.URL.Path); err != nil {
			http.ServeFile(w, r, fmt.Sprintf("%s/index.html", frontendOSPath))
			return
		}
		http.ServeFile(w, r, r.URL.Path)
	}, )).Methods(http.MethodGet)
}