package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MindscapeHQ/raygun4go"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	"bt/db"
	"bt/mailgun"
	"bt/trello"
)

func SetSession(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)

	decoder := json.NewDecoder(r.Body)
	var data struct {
		TrelloToken string `json:"trello_token"`
	}
	err := decoder.Decode(&data)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	// who is this user in Trello?
	user, err := trello.UserFromToken(data.TrelloToken)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	// ensure we have it on our database
	err = db.EnsureUser(user.Id)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 500)
		return
	}

	// send the jwt token for this user
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims["id"] = user.Id
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["exp"] = time.Now().Add(time.Second * 3600 * 24 * 365).Unix()
	jwtString, err := token.SignedString([]byte(settings.SessionSecret))
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"jwt": jwtString})
}

func GetAccount(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	/* account information
	   for now this is just a {addresses: [...]}
	*/

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	addresses, err := db.GetAddresses(userId)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Account{
		Addresses: addresses,
	})
}

func SetAccount(w http.ResponseWriter, r *http.Request) {
}

func SetAddress(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	/* accepts only an email address as parameter
	       add address to db
		   add bot to board
		   send welcome message
		   create failure and success labels */
	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	data := struct {
		BoardShortLink string `json:"boardShortLink"`
		ListId         string `json:"listId"`
		InboundAddr    string `json:"inboundAddr"`
	}{
		InboundAddr: vars["address"] + "@" + settings.BaseDomain,
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	// fetch board
	list, err := trello.Client.List(data.ListId)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}
	board, err := trello.Client.Board(list.IdBoard)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	p := strings.Split(board.ShortUrl, "/")
	data.BoardShortLink = p[len(p)-1]

	// adding address to db
	ok, err := db.SetupNewAddress(userId, data.BoardShortLink, data.ListId, data.InboundAddr)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 500)
		return
	}

	if !ok {
		sendJSONError(w, err, 401)
		return
	}

	// sending welcome message
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

	// returning response
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(db.Address{
		InboundAddr:    data.InboundAddr,
		OutboundAddr:   data.InboundAddr,
		ListId:         data.ListId,
		BoardShortLink: data.BoardShortLink,
	})
}

func DeleteAddress(w http.ResponseWriter, r *http.Request) {
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	/* remove address from the db
	   if there's a custom domain
	     and the custom domain isn't being used by another list
	       remove custom domain from mailgun
	*/
	userId := context.Get(r, "user").(*jwt.Token).Claims["id"].(string)
	vars := mux.Vars(r)

	address, err := db.GetAddress(userId, vars["address"]+"@"+settings.BaseDomain)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	err = address.Delete()
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 500)
		return
	}

	w.WriteHeader(200)
}
