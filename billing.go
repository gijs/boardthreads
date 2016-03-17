package main

import (
	"bt/db"
	"bt/paypal"
	"net/http"

	"github.com/MindscapeHQ/raygun4go"
	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/segmentio/analytics-go"
)

func UpgradeList(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	referrer := r.Referer()
	if referrer == "" {
		referrer = "https://" + settings.BaseDomain
	}

	vars := mux.Vars(r)
	emailAddress := vars["address"] + "@" + settings.BaseDomain

	query := r.URL.Query()
	userId := query.Get("userId")

	successURL, _ := router.Get("paypal-success").URL("address", vars["address"])
	successURL.RawQuery = "userId=" + userId
	failureURL, _ := router.Get("paypal-failure").URL("address", vars["address"])
	failureURL.RawQuery = "userId=" + userId

	paypalPayURL, err := paypal.GetAuthURL(userId,
		emailAddress,
		settings.ServiceURL+successURL.String(),
		settings.ServiceURL+failureURL.String(),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err.Error(),
			"userId":  userId,
			"address": emailAddress,
			"stderr":  paypalPayURL,
		}).Error("couldn't get paypal auth url")
		reportError(raygun, err, logger)
		http.Redirect(w, r, referrer+"#error=Misterious error.", http.StatusFound)
		return
	}

	http.Redirect(w, r, paypalPayURL, http.StatusFound)

	// tracking
	segment.Track(&analytics.Track{
		Event:  "Started subscription creation",
		UserId: userId,
		Properties: map[string]interface{}{
			"address":  emailAddress,
			"provider": "Paypal",
		},
	})
}

func DowngradeAddress(w http.ResponseWriter, r *http.Request) {
	/*
	   this is the only handler from this file that doesn't return
	   a redirect, also it expects a JWT.
	*/
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	referrer := r.Referer()
	if referrer == "" {
		referrer = "https://" + settings.BaseDomain
	}

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	address, err := db.GetAddress(userId, vars["address"]+"@"+settings.BaseDomain)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}

	err = MaybeDowngradeAddress(address)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 500, logger)
	}

	w.WriteHeader(200)
}

func PaypalSuccess(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	referrer := r.Referer()
	if referrer == "" {
		referrer = "https://" + settings.BaseDomain
	}

	vars := mux.Vars(r)
	emailAddress := vars["address"] + "@" + settings.BaseDomain

	query := r.URL.Query()
	userId := query.Get("userId")
	token := query.Get("token")
	payerId := query.Get("PayerID")

	profileId, err := paypal.CreateSubscription(userId, emailAddress, token, payerId)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err.Error(),
			"userId":  userId,
			"address": emailAddress,
			"stderr":  profileId,
		}).Error("couldn't create subscription on paypal")
		reportError(raygun, err, logger)
		http.Redirect(w, r, referrer+"#error=Couldn't create your subscription for some reason, please contact us.", http.StatusFound)
		return
	}

	err = db.SavePaypalProfileId(userId, emailAddress, profileId)
	if err != nil {
		log.WithFields(log.Fields{
			"err":       err.Error(),
			"userId":    userId,
			"address":   emailAddress,
			"profileId": profileId,
		}).Error("couldn't save paypal subscription on db")
		reportError(raygun, err, logger)
		http.Redirect(w, r, referrer+"#error=We have created your subscription, but due to an horrible error on our systems your list status couldn't be upgraded. Please contact us immediately and inform this number: "+profileId, http.StatusFound)
		return
	}

	http.Redirect(w, r, referrer+"#success=Your subscription has been successfully created.", http.StatusFound)

	// tracking
	segment.Track(&analytics.Track{
		Event:  "Created subscription",
		UserId: userId,
		Properties: map[string]interface{}{
			"address":  emailAddress,
			"value":    10,
			"provider": "Paypal",
		},
	})
}

func PaypalFailure(w http.ResponseWriter, r *http.Request) {
	referrer := r.Referer()
	if referrer == "" {
		referrer = "https://" + settings.BaseDomain
	}

	query := r.URL.Query()
	userId := query.Get("userId")

	vars := mux.Vars(r)

	log.WithFields(log.Fields{
		"address": vars["address"] + "@" + settings.BaseDomain,
		"userId":  userId,
	}).Error("paypal failure")

	http.Redirect(w, r, referrer+"#error=Couldn't authorize the payment. That's all we know.", http.StatusFound)
}
