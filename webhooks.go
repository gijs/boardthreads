package main

import (
	"bt/db"
	"bt/helpers"
	"bt/mailgun"
	"bt/trello"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MindscapeHQ/raygun4go"
	log "github.com/Sirupsen/logrus"
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

	log.Info("mail to ", recipient, " ", url)

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
		card, err = trello.Client.Card(card.Id)

		webhookId, err := trello.CreateWebhook(card.Id, settings.WebhookHandler+"/webhooks/trello/card")
		if err != nil {
			reportError(raygun, err)
			sendJSONError(w, err, 503)
			return nil
		}

		err = db.SaveCardWithEmail(recipient, card.ShortLink, card.Id, webhookId)
		if err != nil {
			reportError(raygun, err)
			sendJSONError(w, err, 500)
			return nil
		}
		return card
	}

	// get card for this mail message, if exists (and is valid)
	shortLink, err := db.GetCardForMessage(
		helpers.MessageHeader(message, "In-Reply-To"),
		message.Subject,
		helpers.ReplyToOrFrom(message),
		recipient,
	)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 404)
		return
	}

	var card *goTrello.Card
	if shortLink != "" {
		// card exists
		card, err = trello.Client.Card(shortLink)
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

	// if something fails during the card creation process `card` will be nil
	// now upload attachments
	log.Info("--> uploading attachments... ", len(message.Attachments))
	dir := filepath.Join("/tmp", "bt", message.From, message.Subject)
	os.MkdirAll(dir, 0777)
	attachmentUrls := make(map[string]string)

	for _, mailAttachment := range message.Attachments {
		if mailAttachment.Size < 100000000 {
			filedst := filepath.Join(dir, mailAttachment.Name)
			err = helpers.DownloadFile(filedst, mailAttachment.Url, "api", settings.MailgunAPIKey)
			if err != nil {
				log.Warn("download of " + mailAttachment.Url + " failed: " + err.Error())
				reportError(raygun, err)
				continue
			}
			trelloAttachment, err := card.UploadAttachment(filedst)
			if err != nil {
				log.Warn("upload of " + mailAttachment.Url + " failed: " + err.Error())
				reportError(raygun, err)
				continue
			}
			attachmentUrls[mailAttachment.Url] = trelloAttachment.Url
		}
	}

	// upload the message as an attachment
	var attachedBody goTrello.Attachment
	for {
		ext := ".txt"
		body := message.BodyPlain // default case, 'body-plain' is always present.
		if message.BodyHtml != "" {
			// the normal case, almost always there is a body-html
			ext = "html"
			body = message.BodyHtml

			// replace content-id mapped images
			for cid, meta := range message.ContentIDMap {
				cid = strings.Trim(cid, "<>")
				oldUrl := meta.Url
				var newUrl string
				var ok bool
				if newUrl, ok = attachmentUrls[oldUrl]; !ok {
					// something is wrong here, continue
					log.Warn("didn't found the newUrl for a cid attachment.")
					continue
				}
				body = strings.Replace(body, "cid:"+cid, newUrl, -1)
			}
		}

		// save body-html as a temporary file then upload it
		msgdst := filepath.Join(
			dir,
			time.Now().Format("2006-01-02T15:04:05Z-0700MST")+message.Sender+"."+ext,
		)
		err = ioutil.WriteFile(msgdst, []byte(body), 0644)
		if err != nil {
			reportError(raygun, err)
			break
		}
		attachedBody_, err := card.UploadAttachment(msgdst)
		if err != nil {
			reportError(raygun, err)
			break
		}
		attachedBody = *attachedBody_
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
	log.Info("--> posting comment")
	prefix := fmt.Sprintf("[:envelope_with_arrow:](%s)", attachedBody.Url)
	commentText := fmt.Sprintf("%s %s:\n\n> %s",
		prefix,
		helpers.ReplyToOrFrom(message),
		strings.Join(strings.Split(md, "\n"), "\n> "),
	)

	if len(commentText) > 15000 {
		commentText = commentText[:1500] +
			fmt.Sprintf("\n\n---\n\nMESSAGE TRUNCATED, see [attachment](%s).", attachedBody.Url)
	}

	comment, err := card.AddComment(commentText)
	if err != nil {
		log.WithFields(log.Fields{
			"card": card.ShortLink,
			"err":  err.Error(),
		}).Error("couldn't post the comment")
		reportError(raygun, err)
		return
		// do not return an error or the webhook will retry and more cards will be created
	}

	// save it on the database
	err = db.SaveEmailReceived(
		card.Id,
		card.ShortLink,
		helpers.MessageHeader(message, "Message-Id"),
		helpers.ExtractSubject(message.Subject),
		helpers.ReplyToOrFrom(message),
		comment.Id,
	)
	if err != nil {
		log.WithFields(log.Fields{
			"email":   helpers.MessageHeader(message, "Message-Id"),
			"card":    card.ShortLink,
			"comment": comment.Id,
			"err":     err.Error(),
		}).Error("couldn't save the email received")
		reportError(raygun, err)
		return
		// do not return an error or the webhook will retry and more cards will be created
	}

	w.WriteHeader(200)
}

func TrelloCardWebhookCreation(w http.ResponseWriter, r *http.Request) {
	log.Info("a Trello webhook was created.")
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

	var wh struct {
		Action goTrello.Action `json:"action"`
	}
	err := json.NewDecoder(r.Body).Decode(&wh)
	if err != nil {
		reportError(raygun, err)
		sendJSONError(w, err, 400)
		return
	}

	var strippedText string

	// filter out bot actions
	if wh.Action.MemberCreator.Id == settings.TrelloBotId {
		goto abort
		return
	}

	log.Info("webhook ", wh.Action.Type, " for ", wh.Action.Data.Card.Id)

	switch wh.Action.Type {
	case "deleteCard":
		db.RemoveCard(wh.Action.Data.Card.Id)
		w.WriteHeader(202)
		return
	case "updateComment":
		text := wh.Action.Data.Action.Text
		strippedText = helpers.CommentStripPrefix(text)
		if text == strippedText {
			// comment doesn't have prefix
			goto abort
		}
		// see if we have already sent this message
		email, err := db.GetEmailFromCommentId(wh.Action.Data.Action.Id)
		if err != nil {
			// a real error
			reportError(raygun, err)
			goto abort
		} else if email.Id == "" {
			// we couldn't find it, so let's send
			goto sendMail
		} else {
			// we found it
			goto abort
		}
	case "commentCard":
		text := wh.Action.Data.Text
		strippedText = helpers.CommentStripPrefix(text)
		if text == strippedText {
			// comment doesn't have prefix
			goto abort
		}
		goto sendMail
	default:
		w.WriteHeader(202)
		return
	}
abort:
	log.Info("webhook handling aborted.")
	w.WriteHeader(202)
	return
sendMail:
	params, err := db.GetEmailParamsForCard(wh.Action.Data.Card.ShortLink)
	if err != nil {
		log.Info("no card found in our database for this comment reply. we will ignore it and cancel the webhook.")
		reportError(raygun, err)
		sendJSONError(w, err, 404)
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
		Subject:       helpers.ExtractSubject(params.LastMailSubject),
		InReplyTo:     params.LastMailId,
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

	w.WriteHeader(200)
}

func SegmentTracking(w http.ResponseWriter, r *http.Request) {}
