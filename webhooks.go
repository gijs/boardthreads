package main

import (
	"bt/db"
	"bt/helpers"
	"bt/mailgun"
	"bt/trello"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MindscapeHQ/raygun4go"
	gfm "github.com/shurcooL/github_flavored_markdown"

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

	r.ParseForm()
	recipient := r.PostFormValue("recipient")
	url := r.PostFormValue("message-url")

	// target list for this email
	listId, err := db.GetTargetListForEmailAddress(recipient)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 406)
		return
	}

	// fetch entire email message
	urlp := strings.Split(url, "/")
	message, err := mailgun.Client.GetStoredMessage(urlp[len(urlp)-1])
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 503)
		return
	}

	// card creation process
	createCard := func() *goTrello.Card {
		// card creation process
		card, err := trello.CreateCardFromMessage(listId, message)
		if err != nil {
			reportError(raygun, err)
			sendJSONError(w, err, 503)
			return nil
		}
		webhookId, err := trello.CreateWebhook(card.Id, settings.WebhookHandler)
		if err != nil {
			reportError(raygun, err)
			sendJSONError(w, err, 503)
			return nil
		}
		err = db.SaveCardWithEmail(recipient, card.ShortLink, webhookId)
		if err != nil {
			reportError(raygun, err)
			sendJSONError(w, err, 500)
			return nil
		}
		return card
	}

	// get card for this mail message, if exists (and is valid)
	shortLink, err := db.GetCardForMessage(
		helpers.MessageHeader(message, "Message-Id"), message.Subject, recipient)

	// "no rows" is an error. but that's actually a valid result for us
	if err != nil && !strings.HasPrefix(err.Error(), "sql: no rows in result set") {
		reportError(raygun, err)
		sendJSONError(w, err, 404)
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
				sendJSONError(w, err, 409)
				return
			}

			// then proceed to the card creation process
			card = createCard()
		} else {
			// card exists on trello, revive it
			err = trello.ReviveCard(card)
			if err != nil {
				reportError(raygun, err)
				sendJSONError(w, err, 503)
				return
			}
		}
	} else {
		// card doesn't exist on our db, proceed to the card creaion proccess
		card = createCard()
	}

	// if something fails during the card creation process `card` will be nil
	if card == nil {
		return
	}

	// now upload attachments
	dir := filepath.Join("/tmp", "bt", message.From, message.Subject)
	os.MkdirAll(dir, 0777)
	attachmentUrls := make(map[string]string)

	for _, mailAttachment := range message.Attachments {
		if mailAttachment.Size < 100000000 {
			filedst := filepath.Join(dir, mailAttachment.Name)
			err = helpers.DownloadFile(filedst, mailAttachment.Url, "api", settings.MailgunAPIKey)
			if err != nil {
				log.Print("download of " + mailAttachment.Url + " failed: " + err.Error())
				reportError(raygun, err)
				continue
			}
			trelloAttachment, err := card.UploadAttachment(filedst)
			if err != nil {
				log.Print("upload of " + mailAttachment.Url + " failed: " + err.Error())
				reportError(raygun, err)
				continue
			}
			attachmentUrls[mailAttachment.Url] = trelloAttachment.Url
		}
	}

	// upload the message as an attachment
	for {
		if message.BodyHtml == "" {
			// message body html is empty, no need to upload
			break
		}

		// replace content-id mapped images
		bodyHtml := message.BodyHtml
		for cid, meta := range message.ContentIDMap {
			cid = strings.Trim(cid, "<>")
			oldUrl := meta.Url
			var newUrl string
			var ok bool
			if newUrl, ok = attachmentUrls[oldUrl]; !ok {
				// something is wrong here, continue
				log.Print("didn't found the newUrl for a cid attachment.")
				continue
			}
			bodyHtml = strings.Replace(bodyHtml, "cid:"+cid, newUrl, -1)
		}

		// save body-html as a temporary file then upload it
		msgdst := filepath.Join(
			dir,
			time.Now().Format("2006-01-02T15:04:05Z-0700MST")+message.Sender+".html",
		)
		err = ioutil.WriteFile(msgdst, []byte(bodyHtml), 0644)
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
	comment, err := card.AddComment(
		fmt.Sprintf(":envelope_with_arrow: %s:\n\n> %s",
			helpers.ReplyToOrFrom(message),
			strings.Join(strings.Split(md, "\n"), "\n> "),
		),
	)
	if err != nil {
		reportError(raygun, err)
	}

	// save it on the database
	err = db.SaveEmailReceived(
		card.ShortLink,
		helpers.MessageHeader(message, "Message-Id"),
		helpers.ExtractSubject(message.Subject),
		helpers.ReplyToOrFrom(message),
		comment.Id,
	)
	if err != nil {
		reportError(raygun, err)
	}

	w.WriteHeader(200)
}

func TrelloCardWebhookCreation(w http.ResponseWriter, r *http.Request) {
	log.Print("a Trello webhook was created.")
	w.WriteHeader(200)
}

func TrelloCardWebhook(w http.ResponseWriter, r *http.Request) {
	/*
	   filter out comments made by the bot
	   react to comment
	   find related email thread
	   send email
	*/

	raygun, _ := raygun4go.New("boardthreads", settings.RaygunAPIKey)
	w.WriteHeader(200)

	var wh struct {
		Action goTrello.Action `json:"action"`
	}
	err := json.NewDecoder(r.Body).Decode(&wh)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	// filter out bot actions
	if wh.Action.MemberCreator.Id == settings.TrelloBotId {
		return
	}

	var strippedText string

	switch wh.Action.Type {
	case "deleteCard":
	// doesn't work because we are storing shortlinks only
	// db.RemoveCard(wh.Action.Data.Card.Id)
	case "updateComment":
		text := wh.Action.Data.Action.Text
		strippedText = helpers.CommentStripPrefix(text)
		if len(text) == len(strippedText) {
			// comment doesn't have prefix
			return
		}
		// see if we have already sent this message
		email, err := db.GetEmailFromCommentId(wh.Action.Data.Action.Id)
		if err != nil {
			reportError(raygun, err)
		} else if email.Id == "" {
			// we couldn't find it, so let's send
			goto sendMail
		}
	case "commentCard":
		text := wh.Action.Data.Action.Text
		strippedText = helpers.CommentStripPrefix(text)
		if len(text) == len(strippedText) {
			// comment doesn't have prefix
			return
		}
		goto sendMail
	}
sendMail:
	params, err := db.GetEmailParamsForCard(wh.Action.Data.Card.ShortLink)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 503)
		return
	}

	// check outbound email address validity
	sendingAddr := params.OutboundAddr
	if params.OutboundAddr == "" {
		sendingAddr = params.InboundAddr
	} else {
		domain := strings.Split(params.OutboundAddr, "@")[0]
		if domain != settings.BaseDomain {
			if !mailgun.DomainCanSend(domain) {
				sendingAddr = params.InboundAddr
			}
		}
	}

	// actually send
	messageId, err := mailgun.Send(mailgun.NewMessage{
		ApplyMetadata: true,
		HTML:          string(gfm.Markdown([]byte(strippedText))),
		Text:          strippedText,
		Recipients:    params.Recipients,
		From:          sendingAddr,
		Subject:       helpers.ExtractSubject(params.LastMail.Subject),
		InReplyTo:     params.LastMail.Id,
		ReplyTo:       params.InboundAddr,
		CardShortLink: wh.Action.Data.Card.ShortLink,
		CommenterId:   wh.Action.MemberCreator.Id,
	})
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 503)
		return
	}

	// save email sent
	commentId := wh.Action.Data.Action.Id
	if commentId == "" {
		commentId = wh.Action.Id
	}
	err = db.SaveCommentSent(
		wh.Action.Data.Card.ShortLink,
		wh.Action.MemberCreator.Id,
		messageId,
		commentId,
	)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 500)
		return
	}
}

func SegmentTracking(w http.ResponseWriter, r *http.Request) {}
