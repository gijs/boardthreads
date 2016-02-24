package main

import "net/http"

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
}

func TrelloCardWebhookCreation(w http.ResponseWriter, r *http.Request) {}
func TrelloCardWebhook(w http.ResponseWriter, r *http.Request) {
	/*
	   filter out comments made by the bot
	   react to comment
	   find related email thread
	   send email
	*/
}

func SegmentTracking(w http.ResponseWriter, r *http.Request) {}
