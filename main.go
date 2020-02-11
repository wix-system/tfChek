package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"tfChek/api"
	"tfChek/github"
	"tfChek/launcher"
	"tfChek/misc"
)

const (
	MajorVersion = 0
	MinorVersion = 5
	Revision     = 0
)

func config() {
	flag.Int(misc.PortKey, misc.PORT, "Port application will listen to")
	flag.Bool(misc.DebugKey, false, "Print debug messages")
	flag.String(misc.OutDirKey, "/var/tfChek/out/", "Directory to save output of the task runs")
	flag.Bool(misc.DismissOutKey, true, "Save tasks output to the files in outdir")
	flag.String(misc.TokenKey, "", "GitHub API access token")
	flag.Bool(misc.VersionKey, false, "Show the version info")
	flag.Bool(misc.Fuse, false, "Prevent server from applying run.sh")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("Cannot bind flags. Error: %s", err)
	}
	viper.SetDefault(misc.QueueLengthKey, 10)
	viper.SetDefault(misc.TimeoutKey, 300)
	viper.SetDefault(misc.RepoOwnerKey, "wix-system")
	viper.SetDefault(misc.WebHookSecretKey, "notAsecretAtAll:)")
	viper.SetDefault(misc.RepoDirKey, "/var/tfChek/repos_by_state/")
	viper.SetDefault(misc.CertSourceKey, "")
	viper.SetDefault(misc.CertSourceKey, "")
	viper.SetDefault(misc.RunDirKey, "/var/run/tfChek/")
	viper.SetDefault(misc.AvatarDir, "/var/tfChek/avatars")
	viper.SetDefault(misc.GitHubClientId, "client_id_here")
	viper.SetDefault(misc.GitHubClientSecret, "client_secret_here")
	viper.SetDefault(misc.OAuthAppName, misc.APPNAME)
	viper.SetDefault(misc.OAuthEndpoint, "https://bo.wixpress.com/tfchek")
	viper.SetDefault(misc.JWTSecret, "secret")
	viper.SetEnvPrefix(misc.EnvPrefix)
	viper.AutomaticEnv()
	viper.SetConfigName(misc.APPNAME)
	viper.AddConfigPath("/configs")
	viper.AddConfigPath("/opt/wix/" + misc.APPNAME + "/etc/")
	viper.AddConfigPath("/etc/" + misc.APPNAME)
	viper.AddConfigPath("$HOME/." + misc.APPNAME)
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		log.Printf("Cannot read configuration. Error: %s", err)
	} else {
		if viper.GetBool(misc.DebugKey) {
			log.Printf("Configuration has been loaded")
		}
	}

}

func setupRoutes() *mux.Router {
	authService := api.GetAuthService()
	middleware := authService.Middleware()
	authRoutes, avatarRoutes := authService.Handlers()
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc(misc.WSRUNSH+"{Id}", api.RunShWebsocket).Name("Websocket").Methods("GET")
	router.Path(misc.APIRUNSHIDQ + "{Hash}").Methods("GET").Name("Query by hash").HandlerFunc(api.GetTaskIdByHash)
	//These 2 API endpoints are going to be removed
	//router.Path(misc.APIRUNSH + "{Env}/{Layer}").Methods("GET").Name("Env/Layer").HandlerFunc(api.RunShEnvLayer)
	//router.Path(misc.APIRUNSH + "{Env}").Methods("GET").Name("Env").HandlerFunc(api.RunShEnv)
	router.Path(misc.APIRUNSH).Methods("POST").Name("run.sh universal task accepting endpoint").HandlerFunc(api.RunShPost)
	router.Path(misc.APICANCEL + "{Id}").Methods("GET").Name("Cancel").HandlerFunc(api.Cancel)
	router.Path(misc.WEBHOOKRUNSH).Methods("POST").Name("GitHub web hook").HandlerFunc(api.RunShWebHook)

	router.Path(misc.HEALTHCHECK).HandlerFunc(api.HealthCheck)
	router.Path(misc.AUTHINFO + "{Provider}").Name("Authentication info endpoint").Methods("GET").Handler(api.GetAuthInfoHandler())
	router.PathPrefix(misc.AVATARS).Name("Avatars").Handler(avatarRoutes)
	router.PathPrefix(misc.AUTH).Name("Authentication endpoint").Handler(authRoutes)
	router.Path(misc.READINESSCHECK).HandlerFunc(api.ReadinessCheck)
	router.Path("/login").Methods("GET").Name("Login").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"login.html")
	})
	router.Path(misc.STATICDIR + "script/auth_provider.js").Methods("GET").Name("Login Script").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"script/auth_provider.js")
	})
	router.Path(misc.STATICDIR + "css/main.css").Methods("GET").Name("CSS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"css/main.css")
	})
	router.PathPrefix(misc.STATICDIR + "pictures").Name("Pictures").Methods("GET").Handler(http.StripPrefix(misc.STATICDIR+"pictures", http.FileServer(http.Dir("."+misc.STATICDIR+"pictures"))))
	router.Path("/favicon.ico").Name("Icon").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "."+misc.STATICDIR+"pictures/tfChek_logo.ico")
	})
	router.PathPrefix(misc.STATICDIR).Handler(middleware.Auth(http.StripPrefix(misc.STATICDIR, http.FileServer(http.Dir("."+misc.STATICDIR)))))
	router.Path("/").Handler(&api.IndexHandler{
		HandlerFunc: func(writer http.ResponseWriter, request *http.Request) {
			http.ServeFile(writer, request, "."+misc.STATICDIR+"index.html")
		},
	})

	return router

}

func initialize() {
	//Prepare configuration
	config()

	if viper.GetBool(misc.DebugKey) {
		misc.LogConfig()
	}
	//Start task manager
	tm := launcher.GetTaskManager()
	fmt.Println("Starting task manager")
	go tm.Start()
}

func showVersion() {
	fmt.Printf("%d.%d.%d", MajorVersion, MinorVersion, Revision)
}

func main() {
	initialize()
	defer launcher.GetTaskManager().Close()
	defer github.CloseAll()
	fmt.Println("Starting server")
	router := setupRoutes()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt(misc.PortKey)), router))
}
