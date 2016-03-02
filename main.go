package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/MindscapeHQ/raygun4go"
	"github.com/auth0/go-jwt-middleware"
	"github.com/carbocation/interpose"
	"github.com/carbocation/interpose/adaptors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/cors"
	"gopkg.in/tylerb/graceful.v1"
)

type Settings struct {
	Port           string `envconfig:"PORT"`
	WebhookHandler string `envconfig:"WEBHOOK_HANDLER"`
	BaseDomain     string `envconfig:"BASE_DOMAIN"`
	SessionSecret  string `envconfig:"SESSION_SECRET"`
	RaygunAPIKey   string `envconfig:"RAYGUN_API_KEY"`
	MailgunAPIKey  string `envconfig:"MAILGUN_API_KEY"`
	TrelloBotId    string `envconfig:"TRELLO_BOT_ID"`
}

var settings Settings

func main() {
	envconfig.Process("", &settings)

	jwtMiddle := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return []byte(settings.SessionSecret), nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})

	middle := interpose.New()

	middle.Use(func(next http.Handler) http.Handler {
		// clear context
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			context.Clear(r) // clears after handling everything.
		})
	})
	middle.Use(adaptors.FromNegroni(cors.New(cors.Options{
		// CORS
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders: []string{"Content-Type", "Accept", "Authorization"},
	})))

	router := mux.NewRouter()
	middle.UseHandler(router)

	router.Path("/api/session").Methods("POST").HandlerFunc(SetSession)
	router.Path("/api/account").Methods("GET").
		Handler(jwtMiddle.Handler(http.HandlerFunc(GetAccount)))
	router.Path("/api/account").Methods("PUT").
		Handler(jwtMiddle.Handler(http.HandlerFunc(SetAccount)))
	router.Path("/api/addresses/{address}").Methods("PUT").
		Handler(jwtMiddle.Handler(http.HandlerFunc(SetAddress)))
	router.Path("/api/addresses/{address}").Methods("DELETE").
		Handler(jwtMiddle.Handler(http.HandlerFunc(DeleteAddress)))

	router.Path("/webhooks/mailgun/email").Methods("POST").HandlerFunc(MailgunIncoming)
	router.Path("/webhooks/mailgun/success").Methods("POST").HandlerFunc(MailgunSuccess)
	router.Path("/webhooks/mailgun/failure").Methods("POST").HandlerFunc(MailgunFailure)
	router.Path("/webhooks/trello/card").Methods("HEAD", "GET").HandlerFunc(TrelloCardWebhookCreation)
	router.Path("/webhooks/trello/card").Methods("POST").HandlerFunc(TrelloCardWebhook)
	router.Path("/webhooks/trello/{card}").Methods("POST").HandlerFunc(TrelloCardWebhook)
	router.Path("/webhooks/segment/tracking").Methods("POST").HandlerFunc(SegmentTracking)

	server := &graceful.Server{
		Timeout: 2 * time.Second,
		Server: &http.Server{
			Addr:    ":" + settings.Port,
			Handler: middle,
		},
	}

	log.Print("Listening at " + settings.Port + "...")
	stop := server.StopChan()
	server.ListenAndServe()

	<-stop
	log.Print("Exiting...")
}

func reportError(raygun *raygun4go.Client, err error) {
	if raygun == nil {
		log.Print(err.Error())
	} else {
		raygun.CreateError(err.Error())
	}
}

func sendJSONError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err.Error())
}
