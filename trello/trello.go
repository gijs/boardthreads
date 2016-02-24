package trello

import (
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/websitesfortrello/go-trello"
)

var Client *trello.Client
var Bot *trello.Member
var settings Settings

type Settings struct {
	ApiKey    string `envconfig:"TRELLO_API_KEY"`
	ApiSecret string `envconfig:"TRELLO_API_SECRET"`
	BotToken  string `envconfig:"TRELLO_BOT_TOKEN"`
	BotId     string `envconfig:"TRELLO_BOT_ID"`
}

func init() {
	err := envconfig.Process("", &settings)
	if err != nil {
		log.Fatal(err.Error())
	}

	Client, err = trello.NewAuthClient(settings.ApiKey, &settings.BotToken)
	if err != nil {
		log.Fatal(err.Error())
	}

	Bot, err := Client.Member("me")
	if err != nil || Bot.Id != settings.BotId {
		log.Fatal(err.Error())
	}
}

func UserFromToken(token string) (member *trello.Member, err error) {
	c, err := trello.NewAuthClient(settings.ApiKey, &token)
	if err != nil {
		return
	}
	member, err = c.Member("me")
	return
}

// func reviveCard(card *Card) error {
// 	_, err = card.SendToBoard()
// 	if err != nil {
// 		return err
// 	}
// 	_, err = card.MoveToPos("top")
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
