package main

import (
	"flag"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mdmdirector/mdmdirector/db"
	"github.com/mdmdirector/mdmdirector/director"
	"github.com/mdmdirector/mdmdirector/log"
	"github.com/mdmdirector/mdmdirector/types"
	"github.com/mdmdirector/mdmdirector/utils"

	"github.com/micromdm/go4/env"
	"github.com/sirupsen/logrus"
)

// MicroMDMURL is the url for your MicroMDM server
var MicroMDMURL string

// MicroMDMAPIKey is the api key for your MicroMDM server
var MicroMDMAPIKey string

// Sign is whether profiles should be signed (they really should)
var Sign bool

// KeyPassword is the password for the private key
var KeyPassword string

// KeyPath is the path for the private key
var KeyPath string

// CertPath is the path for the signing cert or p12 file
var CertPath string

// PushNewBuild is whether to push all profiles if the device's build number changes
var PushNewBuild bool

// BasicAuthPass is the password used for basic auth
var BasicAuthPass string

// DBUsername is the account to connect to the database
var DBUsername string

// DBPassword is used to connect to the database
var DBPassword string

// DBName is used to connect to the database
var DBName string

// DBHost is used to connect to the database
var DBHost string

// DBPort is used to connect to the database
var DBPort string

// DBSSLMode is used to connect to the database
var DBSSLMode string

// LogLevel = log level
var LogLevel string

// ResetDeviceAtEnrollment
var ResetDeviceProfilesAtEnrollment bool

func main() {
	var port string
	var debugMode bool
	logrus.SetLevel(logrus.DebugLevel)
	flag.BoolVar(&debugMode, "debug", env.Bool("DEBUG", false), "Enable debug mode")
	flag.BoolVar(&PushNewBuild, "push-new-build", env.Bool("PUSH_NEW_BUILD", true), "Re-push profiles if the device's build number changes.")
	flag.BoolVar(&ResetDeviceProfilesAtEnrollment, "reset-device-profiles", env.Bool("RESET_DEVICE_PROFILES", false), "Reset device profiles when the device enrolls.")
	flag.StringVar(&port, "port", env.String("DIRECTOR_PORT", "8000"), "Port number to run mdmdirector on.")
	flag.StringVar(&MicroMDMURL, "micromdmurl", env.String("MICRO_URL", ""), "MicroMDM Server URL")
	flag.StringVar(&MicroMDMAPIKey, "micromdmapikey", env.String("MICRO_API_KEY", ""), "MicroMDM Server API Key")
	flag.BoolVar(&Sign, "sign", env.Bool("SIGN", false), "Sign profiles prior to sending to MicroMDM.")
	flag.StringVar(&KeyPassword, "key-password", env.String("SIGNING_PASSWORD", ""), "Password to encrypt/read the signing key(optional) or p12 file.")
	flag.StringVar(&KeyPath, "signing-private-key", env.String("SIGNING_KEY", ""), "Path to the signing private key. Don't use with p12 file.")
	flag.StringVar(&CertPath, "cert", env.String("SIGNING_CERT", ""), "Path to the signing certificate or p12 file.")
	flag.StringVar(&BasicAuthPass, "password", env.String("DIRECTOR_PASSWORD", ""), "Password used for basic authentication")
	flag.StringVar(&DBUsername, "db-username", "", "The username associated with the postgress instance")
	flag.StringVar(&DBPassword, "db-password", "", "The password of the db user account")
	flag.StringVar(&DBName, "db-name", "", "The name of the postgress database to use")
	flag.StringVar(&DBHost, "db-host", "", "The hostname or IP of the postgress instance")
	flag.StringVar(&DBPort, "db-port", "5432", "The port of the postgress instance")
	flag.StringVar(&DBSSLMode, "db-sslmode", "disable", "The SSL Mode to use to connect to postgres")
	flag.StringVar(&LogLevel, "loglevel", env.String("LOG_LEVEL", "warn"), "Log level. One of debug, info, warn, error")
	flag.Parse()

	logLevel, err := logrus.ParseLevel(LogLevel)
	if err != nil {
		log.Fatalf("Unable to parse the log level - %s \n", err)
	}
	logrus.SetLevel(logLevel)

	if MicroMDMURL == "" {
		log.Fatal("MicroMDM Server URL missing. Exiting.")
	}

	if MicroMDMAPIKey == "" {
		log.Fatal("MicroMDM API Key missing. Exiting.")
	}

	if BasicAuthPass == "" {
		log.Fatal("Basic Auth password missing. Exiting.")
	}

	if DBUsername == "" || DBPassword == "" || DBName == "" || DBHost == "" || DBPort == "" || DBSSLMode == "" {
		log.Fatal("Required database details missing, Exiting.")
	}

	if LogLevel != "debug" && LogLevel != "info" && LogLevel != "warn" && LogLevel != "error" {
		log.Fatal("loglevel value is not one of debug, info, warn or error.")
	}

	r := mux.NewRouter()
	r.HandleFunc("/webhook", director.WebhookHandler).Methods("POST")
	r.HandleFunc("/profile", utils.BasicAuth(director.PostProfileHandler)).Methods("POST")
	r.HandleFunc("/profile", utils.BasicAuth(director.DeleteProfileHandler)).Methods("DELETE")
	r.HandleFunc("/profile", utils.BasicAuth(director.GetSharedProfiles)).Methods("GET")
	r.HandleFunc("/profile/{udid}", utils.BasicAuth(director.GetDeviceProfiles)).Methods("GET")
	r.HandleFunc("/device", utils.BasicAuth(director.DeviceHandler)).Methods("GET")
	r.HandleFunc("/device/{udid}", utils.BasicAuth(director.SingleDeviceHandler)).Methods("GET")
	r.HandleFunc("/installapplication", utils.BasicAuth(director.PostInstallApplicationHandler)).Methods("POST")
	r.HandleFunc("/installapplication", utils.BasicAuth(director.GetSharedApplicationss)).Methods("GET")
	r.HandleFunc("/command/pending", utils.BasicAuth(director.GetPendingCommands)).Methods("GET")
	r.HandleFunc("/command/pending/delete", utils.BasicAuth(director.DeletePendingCommands)).Methods("GET")
	r.HandleFunc("/command/error", utils.BasicAuth(director.GetErrorCommands)).Methods("GET")
	r.HandleFunc("/command", utils.BasicAuth(director.GetAllCommands)).Methods("GET")
	r.HandleFunc("/health", director.HealthCheck).Methods("GET")
	http.Handle("/", r)

	if err := db.Open(); err != nil {
		log.Error(err)
		log.Fatal("Failed to open database")
	}
	defer db.Close()

	// db.DB.LogMode(true)
	db.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	db.DB.AutoMigrate(
		&types.Device{},
		&types.DeviceProfile{},
		&types.Command{},
		&types.SharedProfile{},
		&types.SecurityInfo{},
		&types.FirmwarePasswordStatus{},
		&types.ManagementStatus{},
		&types.OSUpdateSettings{},
		&types.SharedInstallApplication{},
		&types.DeviceInstallApplication{},
		&types.Certificate{},
		&types.ScheduledPush{},
		&types.ProfileList{},
	)

	log.Info("mdmdirector is running, hold onto your butts...")

	go director.FetchDevicesFromMDM()
	go director.ScheduledCheckin()
	go director.ProcessScheduledCheckinQueue()
	// go director.UnconfiguredDevices()
	// go director.RetryCommands()

	log.Info(http.ListenAndServe(":"+port, nil))
}
