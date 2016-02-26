package mailgun

import (
	"log"

	"github.com/kelseyhightower/envconfig"
	"github.com/mailgun/mailgun-go"
)

type Settings struct {
	ApiKey string `envconfig:"MAILGUN_API_KEY"`
	Domain string `envconfig:"BASE_DOMAIN"`
}

var Client mailgun.Mailgun

func init() {
	var err error
	var settings Settings
	err = envconfig.Process("", &settings)
	if err != nil {
		log.Fatal(err.Error())
	}

	Client = mailgun.NewMailgun(settings.Domain, settings.ApiKey, "")
}

func DomainCanSend(domain string) bool {
	_, _, sending, err := Client.GetSingleDomain(domain)
	if err != nil {
		return false
	}
	log.Print("domain ", domain)
	for _, dns := range sending {
		log.Print("  ", dns.RecordType, " ", dns.Name, " ", dns.Value, " ", dns.Valid)
	}
	return false
}

func Send(params NewMessage) (messageId string, err error) {
	message := Client.NewMessage(params.From, params.Subject, params.Text, params.Recipients...)
	message.AddHeader("Reply-To", params.ReplyTo)
	message.AddHeader("In-Reply-To", params.InReplyTo)
	message.AddTag(params.From)
	message.AddVariable("card", params.CardShortLink)
	message.AddVariable("commenter", params.CommenterId)
	message.SetHtml(params.HTML)
	message.SetTrackingClicks(false)
	message.SetTrackingOpens(false)
	status, messageId, err := Client.Send(message)
	if err != nil {
		log.Print("error sending email: ", status)
		return "", err
	}
	return messageId, nil
}
