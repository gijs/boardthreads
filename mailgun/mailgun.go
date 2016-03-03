package mailgun

import (
	"log"
	"strings"

	"github.com/kelseyhightower/envconfig"
	gfm "github.com/shurcooL/github_flavored_markdown"
	"github.com/websitesfortrello/mailgun-go"
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

	var include bool
	var domainkey bool
	for _, dns := range sending {
		log.Print("  ", dns.RecordType, " ", dns.Name, " ", dns.Value, " ", dns.Valid)
		if dns.Valid == "valid" && dns.RecordType == "TXT" {
			if dns.Name == domain && strings.Contains(dns.Value, "include") {
				include = true
			} else if strings.HasSuffix(dns.Name, "domainkey."+domain) {
				domainkey = true
			}
		}
	}
	if include && domainkey {
		return true
	}
	return false
}

func Send(params NewMessage) (messageId string, err error) {
	message := Client.NewMessage(params.From, params.Subject, params.Text, params.Recipients...)
	if params.HTML != "" {
		params.HTML = string(gfm.Markdown([]byte(params.Text)))
	}
	message.SetHtml(params.HTML)
	if params.ApplyMetadata {
		message.AddHeader("Reply-To", params.ReplyTo)
		message.AddHeader("In-Reply-To", params.InReplyTo)
		message.AddTag(params.From)
		message.AddVariable("card", params.CardShortLink)
		message.AddVariable("commenter", params.CommenterId)
		message.SetTrackingClicks(false)
		message.SetTrackingOpens(false)
	}
	status, messageId, err := Client.Send(message)
	if err != nil {
		log.Print("error sending email: ", status)
		return "", err
	}
	return messageId, nil
}
