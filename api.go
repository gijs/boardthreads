package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MindscapeHQ/raygun4go"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"

	"bt/db"
	"bt/trello"
)

func SetSession(w http.ResponseWriter, r *http.Request) {
	raygun, err := raygun4go.New("boardthreads", settings.RaygunAPIKey)

	decoder := json.NewDecoder(r.Body)
	var data struct {
		TrelloToken string `json:"trello_token"`
	}
	err = decoder.Decode(&data)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 400)
		return
	}

	// who is this user in Trello?
	user, err := trello.UserFromToken(data.TrelloToken)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 400)
		return
	}

	// ensure we have it on our database
	err = db.EnsureUser(user.Id)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 500)
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
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"jwt": jwtString})
}

func GetAccount(w http.ResponseWriter, r *http.Request) {
	raygun, err := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	/* account information
	   for now this is just a list of lists
	*/

	userId := context.Get(r, "user").(*jwt.Token).Claims["id"]
	addresses, err := db.GetAddressesForUserId(userId.(string))
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Account{
		Addresses: addresses,
	})
}

func SetAccount(w http.ResponseWriter, r *http.Request) {
}

func UpgradeAccount(w http.ResponseWriter, r *http.Request) {
}

func DowngradeAccount(w http.ResponseWriter, r *http.Request) {
}

func DeleteAccount(w http.ResponseWriter, r *http.Request) {
}

func SetAddress(w http.ResponseWriter, r *http.Request) {
	/* add address to db
	   add bot to board
	   create failure and success labels */
}

func DeleteAddress(w http.ResponseWriter, r *http.Request) {
}
