package main

import (
	"bt/db"
	"bt/helpers"
	"bt/mailgun"
	"bt/trello"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MindscapeHQ/raygun4go"

	goTrello "github.com/websitesfortrello/go-trello"
)

func MailgunSuccess(w http.ResponseWriter, r *http.Request) {}
func MailgunFailure(w http.ResponseWriter, r *http.Request) {}

func MailgunIncoming(w http.ResponseWriter, r *http.Request) {
	/* parse email message
	   verify if it belongs to some card
	     unarchive the card and put it on the top of the list
	     save email to db as related to the card
	   or
	     create the card
	     set webhook for the card
	     create the card and the email on the db
	   post attachments to the card
	   replace message <img> with the trello references for the attachments
	   post message html as attachment to the card
	   post message as markdown as a comment to the card
	*/
	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)

	var data struct {
		Recipient string `json:"recipient"`
		Id        string `json:"id"`
	}
	err = json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 400)
		return
	}

	// target list for this email
	listId, err := db.GetTargetListForEmailAddress(data.Recipient)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 406)
		return
	}

	// fetch entire email message
	message, err := mailgun.Client.GetStoredMessage(data.Id)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 503)
		return
	}

	// card creation process
	createCard := func() *goTrello.Card {
		// card creation process
		card, err := trello.CreateCardFromMessage(listId, message)
		if err != nil {
			reportError(raygun, err)
			http.Error(w, err.Error(), 503)
			return nil
		}
		webhookId, err := trello.CreateWebhook(card.Id, settings.WebhookHandler)
		if err != nil {
			reportError(raygun, err)
			http.Error(w, err.Error(), 503)
			return nil
		}
		err = db.SaveCardWithEmail(data.Recipient, card.ShortLink, webhookId)
		if err != nil {
			reportError(raygun, err)
			http.Error(w, err.Error(), 500)
			return nil
		}
		return card
	}

	// get card for this mail message, if exists (and is valid)
	shortLink, err := db.GetCardForMessage(data.Id, message.Subject, data.Recipient)
	if err != nil {
		reportError(raygun, err)
		http.Error(w, err.Error(), 404)
		return
	}

	var card *goTrello.Card
	if shortLink != "" {
		// card exists
		c, err := trello.Client.Card(shortLink)
		card = c
		if err != nil {
			// card doesn't exist on trello, delete it from db
			err = db.RemoveCard(shortLink)

			if err != nil {
				reportError(raygun, err)
				http.Error(w, err.Error(), 409)
				return
			}

			// then proceed to the card creation process
			card = createCard()
		} else {
			// card exists on trello, revive it
			err = trello.ReviveCard(card)
			if err != nil {
				reportError(raygun, err)
				http.Error(w, err.Error(), 503)
				return
			}
		}
	} else {
		// card doesn't exist on our db, proceed to the card creaion proccess
		card = createCard()
	}

	// now upload attachments
	dir := filepath.Join("/tmp", "bt", message.From, message.Subject)
	os.MkdirAll(dir, 0777)
	attachmentUrls = make(map[string]string)

	for _, mailAttachment := range message.Attachments {
		if mailAttachment.Size < 100000000 {
			filedst := filepath.Join(dir, mailAttachment.Name)
			err = helpers.DownloadFile(filedst, mailAttachment.Url)
			if err != nil {
				reportError(raygun, err)
				continue
			}
			trelloAttachment, err := card.UploadAttachment(filedst)
			if err != nil {
				reportError(raygun, err)
				continue
			}
			attachmentUrls[mailAttachment.Url] = trelloAttachment.Url
		}
	}

	// upload the message as an attachment
	for {
		msgdst := filepath.Join(
			dir,
			time.Now().Format("2006-01-02T15:04:05Z-0700MST")+message.Sender+".html",
		)
		out, err := os.Open(msgdst)
		if err != nil {
			reportError(raygun, err)
			break
		}
		_, err = io.Copy(out, message.BodyHtml)
		if err != nil {
			reportError(raygun, err)
			break
		}
		_, err = card.UploadAttachment(msgdst)
		if err != nil {
			reportError(raygun, err)
			break
		}
		break
	}

	// get markdown from message HTML
	md, err := helpers.HTMLToMarkdown(message.StrippedHtml)
	if err != nil || md == "" {
		if message.StrippedText != "" {
			md = message.StrippedText
		} else {
			md = message.StrippedHtml
		}
	}

	// post the message as comment
	err = card.AddComment(
		fmt.Sprintf(":envelope_with_arrow: %s:\n\n> %s",
			helpers.ReplyToOrFrom(message),
			strings.Join(strings.Split(md, "\n"), "\n> "),
		),
	)
	if err != nil {
		reportError(raygun, err)
	}

	// save it on the database
	err = db.SaveEmail(
		card.ShortLink,
		message.Id,
		helpers.Extractsubject(message.Subject),
		helpers.ReplyToOrFrom(message),
	)
	if err != nil {
		reportError(raygun, err)
	}

	w.WriteHeader(200)
}

func TrelloCardWebhookCreation(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func TrelloCardWebhook(w http.ResponseWriter, r *http.Request) {
	/*
	   filter out comments made by the bot
	   react to comment
	   find related email thread
	   send email
	*/
}

func SegmentTracking(w http.ResponseWriter, r *http.Request) {}
