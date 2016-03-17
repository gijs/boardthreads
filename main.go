package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/MindscapeHQ/raygun4go"
	log "github.com/Sirupsen/logrus"
	"github.com/auth0/go-jwt-middleware"
	"github.com/carbocation/interpose"
	"github.com/carbocation/interpose/adaptors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/cors"
	"github.com/segmentio/analytics-go"
	"gopkg.in/tylerb/graceful.v1"
)

type Settings struct {
	Port           string `envconfig:"PORT"`
	WebhookHandler string `envconfig:"WEBHOOK_HANDLER"`
	ServiceURL     string `envconfig:"SERVICE_URL"`
	BaseDomain     string `envconfig:"BASE_DOMAIN"`
	SessionSecret  string `envconfig:"SESSION_SECRET"`
	RaygunAPIKey   string `envconfig:"RAYGUN_API_KEY"`
	MailgunAPIKey  string `envconfig:"MAILGUN_API_KEY"`
	TrelloBotId    string `envconfig:"TRELLO_BOT_ID"`
	SegmentioKey   string `envconfig:"SEGMENTIO_WRITE_KEY"`
}

var settings Settings
var router *mux.Router
var segment *analytics.Client

func main() {
	envconfig.Process("", &settings)

	segment = analytics.New(settings.SegmentioKey)
	setValidators()

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})

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
	middle.Use(func(next http.Handler) http.Handler {
		// request id
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var src = rand.NewSource(time.Now().UnixNano())
			context.Set(r, "request-id", randStringBytesMaskImprSrc(src, 6))
			next.ServeHTTP(w, r)
		})
	})

	router = mux.NewRouter()
	middle.UseHandler(router)

	router.Path("/api/session").Methods("POST").HandlerFunc(SetSession)
	router.Path("/api/account").Methods("GET").
		Handler(jwtMiddle.Handler(http.HandlerFunc(GetAccount)))
	router.Path("/api/account").Methods("PUT").
		Handler(jwtMiddle.Handler(http.HandlerFunc(SetAccount)))
	router.Path("/api/addresses/{address}").Methods("GET").
		Handler(jwtMiddle.Handler(http.HandlerFunc(GetAddress)))
	router.Path("/api/addresses/{address}").Methods("PUT").
		Handler(jwtMiddle.Handler(http.HandlerFunc(SetAddress)))
	router.Path("/api/addresses/{address}").Methods("DELETE").
		Handler(jwtMiddle.Handler(http.HandlerFunc(DeleteAddress)))

	router.Path("/billing/{address}/paypal").Methods("GET").
		Handler(http.HandlerFunc(UpgradeList))
	router.Path("/billing/{address}/paypal/success").Methods("GET").
		Handler(http.HandlerFunc(PaypalSuccess)).
		Name("paypal-success")
	router.Path("/billing/{address}/paypal/failure").Methods("GET").
		Handler(http.HandlerFunc(PaypalFailure)).
		Name("paypal-failure")

	router.Path("/webhooks/mailgun/email").Methods("POST").HandlerFunc(MailgunIncoming)
	router.Path("/webhooks/mailgun/success").Methods("POST").HandlerFunc(MailgunSuccess)
	router.Path("/webhooks/mailgun/failure").Methods("POST").HandlerFunc(MailgunFailure)
	router.Path("/webhooks/trello/card").Methods("HEAD", "GET").HandlerFunc(TrelloCardWebhookCreation)
	router.Path("/webhooks/trello/card").Methods("POST").HandlerFunc(TrelloCardWebhook)
	router.Path("/webhooks/trello/{card}").Methods("POST").HandlerFunc(TrelloCardWebhook)

	router.Path("/check").Methods("GET").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})

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

func reportError(raygun *raygun4go.Client, err error, logger *log.Entry) {
	logger.WithFields(log.Fields{"err": err.Error()}).Error("reportError")
	if raygun != nil {
		raygun.CreateError(err.Error())
	}
}

func sendJSONError(w http.ResponseWriter, err error, code int, logger *log.Entry) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	logger.WithFields(log.Fields{"code": code}).Error("returned JSON error")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err.Error())
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz0987654321"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randStringBytesMaskImprSrc(src rand.Source, n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
