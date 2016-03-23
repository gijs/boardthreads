package main

import (
	"bt/db"
	"bt/paypal"
	"errors"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/segmentio/analytics-go"
)

func UpgradeList(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)
	emailAddress := vars["address"] + "@" + settings.BaseDomain

	addressOwnerId, err := db.GetUserForAddress(emailAddress)
	if addressOwnerId != userId {
		err = errors.New(userId + " doesn't own " + emailAddress + " because " + addressOwnerId + " does")
	}
	if err != nil {
		logger.WithFields(log.Fields{
			"err":     err.Error(),
			"address": emailAddress,
			"userId":  userId,
			"ownerId": addressOwnerId,
		}).Error("couldn't verify user ownership of this address")
		sendJSONError(w, err, 403, logger)
		return
	}

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
		logger.WithFields(log.Fields{
			"err":     err.Error(),
			"userId":  userId,
			"address": emailAddress,
			"stderr":  paypalPayURL,
		}).Error("couldn't get paypal auth url")
		sendJSONError(w, err, 500, logger)
		return
	}

	fmt.Fprintf(w, paypalPayURL)

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
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	address, err := db.GetAddress(userId, vars["address"]+"@"+settings.BaseDomain)
	if err != nil {
		sendJSONError(w, err, 400, logger)
		return
	}

	err = MaybeDowngradeAddress(address)
	if err != nil {
		sendJSONError(w, err, 500, logger)
	}

	w.WriteHeader(200)
}

func PaypalSuccess(w http.ResponseWriter, r *http.Request) {
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	vars := mux.Vars(r)
	emailAddress := vars["address"] + "@" + settings.BaseDomain

	query := r.URL.Query()
	userId := query.Get("userId")
	token := query.Get("token")
	payerId := query.Get("PayerID")

	profileId, err := paypal.CreateSubscription(userId, emailAddress, token, payerId)
	if err != nil {
		logger.WithFields(log.Fields{
			"err":     err.Error(),
			"userId":  userId,
			"address": emailAddress,
			"stderr":  profileId,
		}).Error("couldn't create subscription on paypal")
		http.Redirect(w, r, settings.DashboardURL+"#error=Couldn't create your subscription for some reason, please contact us.", http.StatusFound)
		return
	}

	err = db.SavePaypalProfileId(userId, emailAddress, profileId)
	if err != nil {
		logger.WithFields(log.Fields{
			"err":       err.Error(),
			"userId":    userId,
			"address":   emailAddress,
			"profileId": profileId,
		}).Error("couldn't save paypal subscription on db")
		http.Redirect(w, r, settings.DashboardURL+"#error=We have created your subscription, but due to an horrible error on our systems your list status couldn't be upgraded. Please contact us immediately and inform this number: "+profileId, http.StatusFound)
		return
	}

	http.Redirect(w, r, settings.DashboardURL+"#success=Your subscription has been successfully created.", http.StatusFound)

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
	query := r.URL.Query()
	userId := query.Get("userId")

	vars := mux.Vars(r)

	log.WithFields(log.Fields{
		"address": vars["address"] + "@" + settings.BaseDomain,
		"userId":  userId,
	}).Error("paypal failure")

	http.Redirect(w, r, settings.DashboardURL+"#error=Couldn't authorize the payment. That's all we know.", http.StatusFound)
}
