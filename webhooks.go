package main

import (
	"bt/cache"
	"bt/db"
	"bt/helpers"
	"bt/mailgun"
	"bt/trello"
	"bytes"
	"errors"
	"math/rand"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/context"
	"github.com/segmentio/analytics-go"
	gfm "github.com/shurcooL/github_flavored_markdown"

	goTrello "github.com/websitesfortrello/go-trello"
	goMailgun "github.com/websitesfortrello/mailgun-go"
)

func MailgunSuccess(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	logger := log.WithFields(log.Fields{"req-id": context.Get(r, "request-id")})

	r.ParseForm()
	cardId := strings.Trim(r.PostFormValue("card"), `"`)

	params, err := db.GetEmailParamsForCard(cardId)
	if err != nil {
		log.WithFields(log.Fields{
			"card": cardId,
			"err":  err.Error(),
		}).Warn("couldn't fetch mail params after mail success")
		return
	}

	logger.WithFields(log.Fields{
		"card":       cardId,
		"addReplier": params.AddReplier,
	}).Info("mailgun success")

	if params.AddReplier {
		card, err := trello.Client.Card(cardId)
		if err != nil {
			log.WithFields(log.Fields{
				"card": cardId,
				"err":  err.Error(),
			}).Warn("couldn't fetch card after mail success")
			return
		}

		commenterId := strings.Trim(r.PostFormValue("commenter"), `"`)

		card.AddMemberId(commenterId)
		if err != nil {
			log.WithFields(log.Fields{
				"card":    cardId,
				"replier": commenterId,
				"err":     err.Error(),
			}).Warn("couldn't add replier to card.")
		}
	}
}

func MailgunFailure(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	logger := log.WithFields(log.Fields{"req-id": context.Get(r, "request-id")})

	cardId := strings.Trim(r.FormValue("card"), `"`)
	log.Print("cardId: ", cardId)

	card, err := trello.Client.Card(cardId)
	if err != nil {
		log.WithFields(log.Fields{
			"card": cardId,
			"err":  err.Error(),
		}).Error("couldn't fetch card after mail failure")
		return
	}

	logger.WithFields(log.Fields{
		"card":      cardId,
		"recipient": r.FormValue("recipient"),
	}).Warn("mailgun failure")

	notice := fmt.Sprintf("**mail could not be delivered to %s**\n\n---\n\n%s", r.FormValue("recipient"), r.FormValue("description"))

	_, err = card.AddComment(notice)
	if err != nil {
		log.WithFields(log.Fields{
			"card":   cardId,
			"notice": notice,
			"err":    err.Error(),
		}).Error("couldn't add mailgun failure notice to card.")
	}

	logger.WithFields(log.Fields{
		"card":      cardId,
		"recipient": r.PostFormValue("recipient"),
	}).Debug("posted notice to the card")
}

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
	logger := log.WithFields(log.Fields{"req-id": context.Get(r, "request-id")})

	r.ParseForm()
	inboundAddr := r.PostFormValue("recipient")
	url := r.PostFormValue("message-url")

	logger.WithFields(log.Fields{
		"recipient": inboundAddr,
		"sender":    r.PostFormValue("from"),
		"url":       url,
	}).Info("got mail")

	// target list for this email
	listId, err := db.GetTargetListForEmailAddress(inboundAddr)
	if err != nil {
		sendJSONError(w, err, 500, logger)
		return
	}
	if listId == "" {
		// logger.Debug("no list registered for address " + inboundAddr)
		sendJSONError(w, errors.New("no list registered for address."), 406, logger)
		return
	}

	// fetch userId for this address
	userId, _ := db.GetUserForAddress(inboundAddr)

	// fetch entire email message
	// -- do not fetch message from mailgun, use what comes in the post body
	// message, err := mailgun.Client.GetStoredMessage(url)
	// if err != nil {
	//	logger.WithField("err", err).Warn("couldn't fetch message from mailgun")

	// --- build our own message ---
	// headers are very important, so we will fail without them:
	var headers [][]string
	err = json.Unmarshal([]byte(r.PostFormValue("message-headers")), &headers)
	if err != nil {
		logger.WithField("err", err).Warn("couldn't build the message from the post parameters also")
		sendJSONError(w, err, 503, logger)
		return
	}

	// attachments and content-id-map are not
	var attachments []goMailgun.StoredAttachment
	json.Unmarshal([]byte(r.PostFormValue("attachments")), &attachments)
	var contentidmap = map[string]string{}
	var cidstructs = make(map[string]struct {
		Url         string `json:"url"`
		ContentType string `json:"content-type"`
		Name        string `json:"name"`
		Size        int64  `json:"size"`
	})
	err = json.Unmarshal([]byte(r.PostFormValue("content-id-map")), &contentidmap)
	if err == nil {
		for cid, url := range contentidmap {
			cidstructs[cid] = struct {
				Url         string `json:"url"`
				ContentType string `json:"content-type"`
				Name        string `json:"name"`
				Size        int64  `json:"size"`
			}{url, "", "", 0}
		}
	}

	message := goMailgun.StoredMessage{
		Recipients:     r.PostFormValue("recipient"),
		Sender:         r.PostFormValue("sender"),
		From:           r.PostFormValue("from"),
		Subject:        r.PostFormValue("Subject"),
		StrippedText:   r.PostFormValue("stripped-text"),
		StrippedHtml:   r.PostFormValue("stripped-html"),
		BodyPlain:      r.PostFormValue("body-plain"),
		BodyHtml:       r.PostFormValue("body-html"),
		MessageHeaders: headers,
		ContentIDMap:   cidstructs,
		Attachments:    attachments,
	}
	//}

	// fetch preferences for dealing with this message on trello
	prefs, err := db.GetReceivingParams(inboundAddr)
	if err != nil {
		// an error is not that serious, let's just post the message with the defaults
		logger.WithField("err", err).Warn("couldn't fetch receiving preferences")
	}

	// card creation process
	createCard := func() *goTrello.Card {
		card, err := trello.CreateCardFromMessage(listId, message)
		if err != nil {
			sendJSONError(w, err, 503, logger)
			return nil
		}
		card, err = trello.Client.Card(card.Id)
		if err != nil {
			sendJSONError(w, err, 503, logger)
			return nil
		}

		webhookId, err := trello.CreateWebhook(card.Id, settings.WebhookHandler+"/webhooks/trello/card")
		if err != nil {
			sendJSONError(w, err, 503, logger)
			return nil
		}

		err = db.SaveCardWithEmail(inboundAddr, card.ShortLink, card.Id, webhookId)
		if err != nil {
			sendJSONError(w, err, 500, logger)
			return nil
		}

		return card
	}

	// get card for this mail message, if exists (and is valid)
	shortLink, err := db.GetCardForMessage(
		helpers.MessageHeader(message, "In-Reply-To"),
		message.Subject,
		helpers.ReplyToOrFrom(message),
		inboundAddr,
	)
	if err != nil {
		sendJSONError(w, err, 404, logger)
		return
	}

	var card *goTrello.Card
	if shortLink != "" {
		// card exists
		card, err = trello.Client.Card(shortLink)
		if err != nil {
			// card doesn't exist on trello, delete it from db
			log.WithFields(log.Fields{
				"card": shortLink,
				"err":  err.Error(),
			}).Warn("card doesn't exist on trello, deleting it from db")
			err = db.RemoveCard(shortLink)

			if err != nil {
				log.WithFields(log.Fields{
					"card": shortLink,
					"err":  err.Error(),
				}).Warn("couldn't delete card from db. proceeding to create anyway.")
			}

			// then proceed to the card creation process
			card = createCard()
		} else {
			// card exists on trello, revive it
			_, err = card.SendToBoard()
			if err != nil {
				sendJSONError(w, err, 503, logger)
				return
			}
			if prefs.MoveToTop {
				_, err = card.MoveToPos(0)
				if err != nil {
					// this error is not big enough to justify abandoning the request
					logger.WithFields(log.Fields{
						"err": err,
					}).Warn("couldn't move card to top of list")
				}
			}

			// in this case, we will not put the message on the card's description
			// no matter what (because the card is not new)
			// so we fake the prefs.MessageInDesc to reflect this
			prefs.MessageInDesc = false
		}
	} else {
		// card doesn't exist on our db, proceed to the card creation proccess
		card = createCard()
	}

	// if something fails during the card creation process `card` will be nil
	if card == nil {
		return
	}

	// if something fails during the card creation process `card` will be nil
	// now upload attachments
	logger.WithFields(log.Fields{"quantity": len(message.Attachments)}).Debug("uploading attachments")
	dir := filepath.Join("/tmp", "bt", message.From, message.Subject)
	os.MkdirAll(dir, 0777)
	attachmentUrls := make(map[string]string)

	for _, mailAttachment := range message.Attachments {
		if mailAttachment.Size < 100000000 {
			filedst := filepath.Join(dir, mailAttachment.Name)
			err = helpers.DownloadFile(filedst, mailAttachment.Url, "api", settings.MailgunAPIKey)
			if err != nil {
				logger.WithFields(log.Fields{
					"url":  mailAttachment.Url,
					"path": filedst,
					"err":  err.Error(),
					"card": card.ShortLink,
				}).Warn("attachment download failed")

				// if fail because target is a directory do something!
				if strings.Contains(err.Error(), "is a directory") {
					filedst = filepath.Join("/tmp/", randStringBytesMaskImprSrc(rand.NewSource(time.Now().UnixNano()), 6))
					err = helpers.DownloadFile(filedst, mailAttachment.Url, "api", settings.MailgunAPIKey)
					if err != nil {
						logger.WithFields(log.Fields{
							"url":  mailAttachment.Url,
							"path": filedst,
							"err":  err.Error(),
							"card": card.ShortLink,
						}).Warn("attachment download failed for the second time")
						continue
					}
				} else {
					continue
				}
			}

			// before uploading, check if file is already on this trello card
			if cache.Has(card.Id, filedst) {
				attachmentUrls[mailAttachment.Url] = cache.Url()
			} else {
				trelloAttachment, err := card.UploadAttachment(filedst)
				if err != nil {
					logger.WithFields(log.Fields{
						"path": filedst,
						"err":  err.Error(),
						"card": card.ShortLink,
					}).Warn("attachment upload failed")
					continue
				}
				attachmentUrls[mailAttachment.Url] = trelloAttachment.Url
				cache.Save(trelloAttachment.Url)
			}
		}
	}

	// upload the message as an attachment
	var attachedBody goTrello.Attachment
	var processedStrippedHTML string = message.StrippedHtml
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
					logger.Warn("didn't found the newUrl for a cid attachment.")
					continue
				}
				body = strings.Replace(body, "cid:"+cid, newUrl, -1)

				// use this opportunity to also replace in the strippedHTML that will be posted
				processedStrippedHTML = strings.Replace(processedStrippedHTML, "cid:"+cid, newUrl, -1)
			}
		}

		// save body-html as a temporary file then upload it
		messageFilename := helpers.ReplyToOrFrom(message)
		if len(messageFilename) > 80 {
			messageFilename = helpers.ReplyToOrFrom(message)[:79]
		}
		msgdst := filepath.Join(dir, messageFilename+"."+ext)
		err = ioutil.WriteFile(msgdst, []byte(body), 0644)
		if err != nil {
			break
		}
		attachedBody_, err := card.UploadAttachment(msgdst)
		if err != nil {
			break
		}
		attachedBody = *attachedBody_
		break
	}

	// get markdown from message HTML
	md := helpers.HTMLToMarkdown(processedStrippedHTML)
	if md == "" {
		// fallback
		if message.StrippedText != "" {
			md = message.StrippedText
		} else {
			md = message.StrippedHtml
		}
	}

	// post the message as comment
	logger.WithFields(log.Fields{"card": card.ShortLink}).Debug("posting comment")
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
		logger.WithFields(log.Fields{
			"card": card.ShortLink,
			"err":  err.Error(),
		}).Error("couldn't post the comment")
		return
		// do not return an error or the webhook will retry and more cards will be created
	}

	// if this setting is enabled, update the card description
	if prefs.MessageInDesc {
		err = trello.PutMessageBodyOnDesc(card, message, commentText)
		if err != nil {
			// we shouldn't care a lot for this error
			log.WithField("err", err).Warn("couldn't add message to card desc.")
		}
	}

	// save it on the database
	err = db.SaveEmailReceived(
		card.Id,
		card.ShortLink,
		helpers.MessageHeader(message, "Message-Id"),
		mailgun.TrimSubject(message.Subject),
		helpers.ReplyToOrFrom(message),
		comment.Id,
	)
	if err != nil {
		logger.WithFields(log.Fields{
			"email":   helpers.MessageHeader(message, "Message-Id"),
			"card":    card.ShortLink,
			"comment": comment.Id,
			"err":     err.Error(),
		}).Error("couldn't save the email received")
		return
		// do not return an error or the webhook will retry and more cards will be created
	}

	w.WriteHeader(200)

	// tracking
	segment.Track(&analytics.Track{
		Event:  "Received mail",
		UserId: userId,
		Properties: map[string]interface{}{
			"card":    card.Id,
			"from":    helpers.ReplyToOrFrom(message),
			"address": inboundAddr,
		},
	})
}

func TrelloCardWebhookCreation(w http.ResponseWriter, r *http.Request) {
	log.Debug("a Trello webhook was created.")
	w.WriteHeader(200)
}

func TrelloCardWebhook(w http.ResponseWriter, r *http.Request) {
	/*
	   filter out comments made by the bot
	   react to comment
	   find related email thread
	   send email
	*/
	logger := log.WithFields(log.Fields{"req-id": context.Get(r, "request-id")})

	var wh struct {
		Action goTrello.Action `json:"action"`
	}
	err := json.NewDecoder(r.Body).Decode(&wh)
	if err != nil {
		sendJSONError(w, err, 400, logger)
		return
	}

	var strippedText string

	// filter out bot actions
	if wh.Action.MemberCreator.Id == settings.TrelloBotId {
		goto abort
		return
	}

	switch wh.Action.Type {
	case "deleteCard":
		logger.WithFields(log.Fields{"type": wh.Action.Type, "card": wh.Action.Data.Card.Id}).Info("webhook")
		db.RemoveCard(wh.Action.Data.Card.Id)
		w.WriteHeader(202)
		return
	case "updateComment":
		logger.WithFields(log.Fields{"type": wh.Action.Type, "card": wh.Action.Data.Card.ShortLink}).Info("webhook")
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
			logger.WithFields(log.Fields{
				"err":     err,
				"comment": wh.Action.Data.Action.Id,
				"card":    wh.Action.Data.Card.ShortLink,
			}).Error("couldn't fetch email for comment id.")
			goto abort
		} else if email.Id == "" {
			// we couldn't find it, so let's send
			goto sendMail
		} else {
			// we found it
			goto abort
		}
	case "commentCard":
		logger.WithFields(log.Fields{"type": wh.Action.Type, "card": wh.Action.Data.Card.ShortLink}).Info("webhook")
		text := wh.Action.Data.Text
		strippedText = helpers.CommentStripPrefix(text)
		if text == strippedText {
			// comment doesn't have prefix
			goto abort
		}
		goto sendMail
	case "updateCard":
		logger.WithFields(log.Fields{"type": wh.Action.Type, "card": wh.Action.Data.Card.ShortLink}).Info("webhook")
		params, err := helpers.ParseCardDescription(wh.Action.Data.Card.Desc)
		if err != nil {
			logger.WithFields(log.Fields{
				"err":  err,
				"card": wh.Action.Data.Card.ShortLink,
				"desc": wh.Action.Data.Card.Desc,
			}).Info("desc change will not take effect.")
			goto abort
		}

		logger.WithFields(log.Fields{
			"params": params,
			"card":   wh.Action.Data.Card.ShortLink,
		}).Info("changing first email params.")
		err = db.ChangeThreadParams(wh.Action.Data.Card.Id, params)
		if err != nil {
			log.WithField("err", err).Error("couldn't change thread params.")
		}

		goto abort
	default:
		w.WriteHeader(202)
		return
	}
abort:
	w.WriteHeader(202)
	return
sendMail:
	params, err := db.GetEmailParamsForCard(wh.Action.Data.Card.ShortLink)
	if err != nil {
		logger.WithFields(log.Fields{
			"card": wh.Action.Data.Card.ShortLink,
			"text": strippedText,
		}).Warn("no card found in our database for this comment. will ignore it and cancel the webhook.")

		// post a comment on the card telling about the error
		card, perr := trello.Client.Card(wh.Action.Data.Card.ShortLink)
		if perr == nil {
			_, perr = card.AddComment("Due to a misterious error, replies in this card can't be send. Please report this issue.")
		}

		if err == nil {
			err = perr
		}

		sendJSONError(w, err, 404, logger)
		return
	}

	// check outbound email address validity
	sendingAddr := params.OutboundAddr
	if params.OutboundAddr == "" {
		sendingAddr = params.InboundAddr
	} else {
		domain := strings.Split(params.OutboundAddr, "@")[1]
		logger.Debug("trying to send from " + sendingAddr)
		if domain != settings.BaseDomain {
			canSend, canReceive := mailgun.DomainCanSendReceive(domain)
			if !canSend {
				sendingAddr = params.InboundAddr
			}
			if canReceive {
				params.ReplyTo = params.OutboundAddr
			}
		}
	}
	logger.WithFields(log.Fields{
		"to":   params.Recipients,
		"from": sendingAddr,
	}).Info("sending email")

	// the default, safe replyTo address
	if params.ReplyTo == "" || !isEmail(params.ReplyTo) {
		params.ReplyTo = params.InboundAddr
	}

	// add signature, if specified
	for {
		var err error
		var sign *template.Template
		var buf bytes.Buffer

		if params.SignatureTemplate != "" {
			sign, err = template.New("signature").Delims("{", "}").Funcs(template.FuncMap{
				"NAME":     func() string { return wh.Action.MemberCreator.FullName },
				"USERNAME": func() string { return wh.Action.MemberCreator.Username },
			}).Parse(params.SignatureTemplate)
			if err != nil {
				goto signatureError
			}

			err = sign.Execute(&buf, nil)
			if err != nil {
				goto signatureError
			}

			strippedText += "\n\n" + buf.String()
			break

		signatureError:
			// this error is not a big deal, we will ignore it
			logger.WithFields(log.Fields{
				"err":       err,
				"signature": params.SignatureTemplate,
			}).Warn("couldn't render signature.")
			break
		}
		break
	}

	// actually send
	messageId, err := mailgun.Send(mailgun.NewMessage{
		ApplyMetadata: true,
		HTML:          string(gfm.Markdown([]byte(strippedText))),
		Text:          strippedText,
		Recipients:    params.Recipients,
		FromName:      params.SenderName,
		From:          sendingAddr,
		Domain:        strings.Split(sendingAddr, "@")[1],
		Subject:       mailgun.TrimSubject(params.LastMailSubject),
		InReplyTo:     params.LastMailId,
		ReplyTo:       params.ReplyTo,
		CardId:        wh.Action.Data.Card.Id,
		CommenterId:   wh.Action.MemberCreator.Id,
	})
	if err != nil {
		sendJSONError(w, err, 503, logger)
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
		sendJSONError(w, err, 500, logger)
		return
	}

	w.WriteHeader(200)

	// tracking
	userId, _ := db.GetUserForAddress(params.InboundAddr)
	segment.Track(&analytics.Track{
		Event:  "Sent mail",
		UserId: userId,
		Properties: map[string]interface{}{
			"card":    wh.Action.Data.Card.Id,
			"to":      params.Recipients,
			"address": params.InboundAddr,
		},
	})
}
