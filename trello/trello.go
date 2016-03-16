package trello

import (
	"bt/helpers"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	"github.com/websitesfortrello/go-trello"
	"github.com/websitesfortrello/mailgun-go"
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

	Client, err = trello.NewAuthClient(settings.ApiKey, settings.BotToken)
	if err != nil {
		log.Fatal(err.Error())
	}

	Bot, err := Client.Member("me")
	if err != nil || Bot.Id != settings.BotId {
		log.Fatal(err.Error())
	}
}

func UserFromToken(token string) (member *trello.Member, err error) {
	c, err := trello.NewAuthClient(settings.ApiKey, token)
	if err != nil {
		return
	}
	member, err = c.Member("me")
	return
}

func EnsureBot(token, listId string) (*trello.Board, error) {
	c, err := trello.NewAuthClient(settings.ApiKey, token)
	if err != nil {
		return nil, err
	}

	list, err := c.List(listId)
	if err != nil {
		return nil, err
	}

	board, err := c.Board(list.IdBoard)
	if err != nil {
		return nil, err
	}

	err = board.AddMemberId(settings.BotId, "normal")
	if err != nil {
		if strings.Contains(err.Error(), "401") {
			// no add-member permissions
			return nil, errors.New("no-permission")
		} else {
			// unknown error
			return nil, err
		}
	}

	return board, nil
}

func ReviveCard(card *trello.Card) (err error) {
	_, err = card.SendToBoard()
	if err != nil {
		return
	}
	// _, err = card.MoveToPos("top")
	// if err != nil {
	// 	return
	// }
	return nil
}

func CreateCardFromMessage(listId string, message mailgun.StoredMessage) (card *trello.Card, err error) {
	list, err := Client.List(listId)
	if err != nil {
		return nil, err
	}

	card, err = list.AddCard(trello.Card{
		IdList: listId,
		Name:   fmt.Sprintf("%s :: %s", helpers.ReplyToOrFrom(message), helpers.ExtractSubject(message.Subject)),
		Desc: fmt.Sprintf(`
---

to: %s
recipient: %s
from: %s
reply-to: %s
subject: %s

---
            `,
			helpers.MessageHeader(message, "To"),
			message.Recipients,
			message.From,
			helpers.ReplyToOrFrom(message),
			message.Subject,
		),
		Pos:       0,
		IdMembers: []string{settings.BotId},
	})

	return
}

func CreateWebhook(entityId, endpoint string) (string, error) {
	params := url.Values{}
	params.Add("idModel", entityId)
	params.Add("callbackURL", endpoint)

	body, err := Client.Put("/webhooks", params)
	if err != nil {
		return "", err
	}

	var data struct {
		Id string `json:"id"`
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	return data.Id, nil
}
