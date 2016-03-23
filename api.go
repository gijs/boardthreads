package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gopkg.in/validator.v2"

	"github.com/MindscapeHQ/raygun4go"
	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/segmentio/analytics-go"

	"bt/db"
	"bt/mailgun"
	"bt/trello"
)

func SetSession(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	decoder := json.NewDecoder(r.Body)
	var data struct {
		TrelloToken string `json:"trello_token"`
	}
	err := decoder.Decode(&data)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}

	// who is this user in Trello?
	user, err := trello.UserFromToken(data.TrelloToken)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}

	// ensure we have it on our database
	logger.WithFields(log.Fields{
		"user": user.Id,
	}).Info("fetching/saving user on db")
	new, err := db.EnsureUser(user.Id)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 500, logger)
		return
	}

	// send the jwt token for this user
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["id"] = user.Id
	token.Claims["token"] = data.TrelloToken
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Second * 3600 * 24 * 365).Unix()
	jwtString, err := token.SignedString([]byte(settings.SessionSecret))
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 500, logger)
		return
	}
	logger.WithFields(log.Fields{
		"user": user.Id,
	}).Info("logged in")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"jwt": jwtString})

	// tracking
	if new {
		segment.Identify(&analytics.Identify{
			UserId: user.Id,
			Traits: map[string]interface{}{
				"username": user.Username,
				"name":     user.FullName,
				"avatar":   user.AvatarSource,
			},
		})
	}
}

func GetAccount(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})
	/* account information
	   for now this is just a {addresses: [...]}
	*/

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)

	// get addresses
	addresses, err := db.GetAddresses(userId)
	if err != nil {
		logger.WithFields(log.Fields{"err": err, "user": userId}).Warn("error fetching addresses")
		reportError(raygun, err, logger)
		sendJSONError(w, err, 500, logger)
		return
	}

	// making sure addresses is not nil
	if addresses == nil {
		addresses = make([]db.Address, 0)
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(db.Account{
		Addresses: addresses,
	})
}

func GetAddress(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})
	/*
	   address detailed information, includes domain status
	*/

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	// address data
	address, err := db.GetAddress(userId, vars["address"]+"@"+settings.BaseDomain)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 404, logger)
		return
	}
	MaybeFillDomainInformation(address)

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(address)
}

func SetAccount(w http.ResponseWriter, r *http.Request) {
}

func SetAddress(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})
	/* accepts only an email address as parameter
	       add address to db
		   add bot to board
		   send welcome message */
	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	trelloToken := context.Get(r, "user").(*jwt.Token).Claims["token"].(string)
	vars := mux.Vars(r)

	data := struct {
		ListId       string `json:"listId"`
		InboundAddr  string `json:"inboundAddr"  validate:"email"`
		OutboundAddr string `json:"outboundAddr" validate:"email"`
	}{
		InboundAddr: vars["address"] + "@" + settings.BaseDomain,
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}

	// outboundaddr defaults to inboundaddr
	if data.OutboundAddr == "" {
		data.OutboundAddr = data.InboundAddr
	}

	// validation
	if err := validator.Validate(data); err != nil {
		log.WithFields(log.Fields{
			"data": data,
			"err":  err.Error(),
		}).Error("validation error")
		sendJSONError(w, err, 400, logger)
		return
	}

	logger.WithFields(log.Fields{
		"user":         userId,
		"address":      data.InboundAddr,
		"outboundaddr": data.OutboundAddr,
		"list":         data.ListId,
	}).Info("creating address")

	// fetch board and ensure bot is on the board
	board, err := trello.EnsureBot(trelloToken, data.ListId)
	if err != nil && err.Error() == "no-permission" {
		sendJSONError(w, err, 403, logger)
		return
	} else if err != nil {
		logger.WithFields(log.Fields{
			"err":     err.Error(),
			"user":    userId,
			"token":   trelloToken,
			"address": data.InboundAddr,
			"list":    data.ListId,
		}).Error("couldn't fetch board or ensure the existence of the bot on the board")
		reportError(raygun, err, logger)
		sendJSONError(w, err, 502, logger)
		return
	}

	// first remove old domains and routes
	oldAddress, err := db.GetAddress(userId, data.InboundAddr)
	if err == nil {
		MaybeDeleteDomainAndRouteFlow(oldAddress, data.OutboundAddr)
	}

	// adding address to db
	new, actualOutbound, err := db.SetAddress(userId, board.ShortLink, data.ListId, data.InboundAddr, data.OutboundAddr)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 500, logger)
		return
	}

	logger.WithFields(log.Fields{
		"address":      data.InboundAddr,
		"outboundaddr": data.OutboundAddr,
		"list":         data.ListId,
	}).Info("saved to db")

	// create domain on mailgun (here we use the actual outboundaddr)
	if data.InboundAddr != actualOutbound && isEmail(actualOutbound) {
		logger.Debug("will create domain and route on mailgun")
		routeId, err := mailgun.PrepareExternalAddress(data.InboundAddr, actualOutbound)
		if err == nil && routeId != "" {
			err = db.SaveRouteId(data.OutboundAddr, routeId)
			if err != nil {
				reportError(raygun, err, logger)
			}
		} else {
			// if the domain could not be added to mailgun, set outboundaddr to inboundaddr
			_, _, err := db.SetAddress(userId, board.ShortLink, data.ListId, data.InboundAddr, data.InboundAddr)
			if err != nil {
				reportError(raygun, err, logger)
				sendJSONError(w, err, 500, logger)
				return
			}
		}
	}

	// release the old domain if we can
	if oldAddress != nil && oldAddress.DomainName != "" {
		err = db.MaybeReleaseDomainFromOwner(oldAddress.DomainName)
		if err != nil {
			logger.WithFields(log.Fields{
				"domain": oldAddress.DomainName,
				"err":    err.Error(),
			}).Warn("failed to release domain")
		}
	}

	// returning the address as response
	// address data
	newAddress, err := db.GetAddress(userId, vars["address"]+"@"+settings.BaseDomain)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 404, logger)
		return
	}
	MaybeFillDomainInformation(newAddress)

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newAddress)

	// tracking
	if new {
		segment.Track(&analytics.Track{
			Event:  "Created address",
			UserId: userId,
			Properties: map[string]interface{}{
				"listId":  data.ListId,
				"boardId": board.Id,
				"address": data.InboundAddr,
			},
		})

		mailgun.Send(mailgun.NewMessage{
			ApplyMetadata: false,
			Text: fmt.Sprintf(`
	Hello and welcome to BoardThreads. This is a test message with the sole purpose of showing you how emails sent to %s will appear to you. If you need any help or have anything to say to us, you can reply here.

	Remember: to send replies you can just write a normal comment, only prefixed with :email: or :e-mail:, any other comments will stay as comments.
	        `, data.InboundAddr),
			Recipients: []string{data.InboundAddr},
			From:       "welcome@boardthreads.com",
			Subject:    "Welcome to BoardThreads",
		})
	}
}

func ChangeAddressSettings(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	address := vars["address"] + "@" + settings.BaseDomain

	var params db.AddressSettings
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}

	logger.WithFields(log.Fields{
		"address": address,
		"user":    userId,
		"params":  params,
	}).Info("changing settings")

	err = db.ChangeAddressSettings(userId, address, params)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}
}

func DeleteAddress(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	logger := log.WithFields(log.Fields{"ip": r.RemoteAddr})
	/* remove address from the db
	   if there's a custom domain
	     and the custom domain isn't being used by another list
	       remove custom domain from mailgun
	*/
	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	logger.WithFields(log.Fields{
		"address": vars["address"] + "@" + settings.BaseDomain,
		"user":    userId,
	}).Info("deleting address")

	address, err := db.GetAddress(userId, vars["address"]+"@"+settings.BaseDomain)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 400, logger)
		return
	}

	// remove paypalProfileId and cancel subscription
	err = MaybeDowngradeAddress(address)
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 503, logger)
	}

	// remove old domains and routes
	MaybeDeleteDomainAndRouteFlow(address, "")

	// actually delete
	err = address.Delete()
	if err != nil {
		reportError(raygun, err, logger)
		sendJSONError(w, err, 500, logger)
		return
	}

	// release domain if we can
	if address.DomainName != "" {
		err = db.MaybeReleaseDomainFromOwner(address.DomainName)
		if err != nil {
			logger.WithFields(log.Fields{
				"domain": address.DomainName,
				"err":    err.Error(),
			}).Warn("failed to release domain")
		}
	}

	w.WriteHeader(200)

	// tracking
	segment.Track(&analytics.Track{
		Event:  "Deleted address",
		UserId: userId,
		Properties: map[string]interface{}{
			"address": vars["address"] + "@" + settings.BaseDomain,
		},
	})
}
