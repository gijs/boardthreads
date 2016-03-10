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
		message.AddVariable("card", params.CardId)
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

func DomainCanSend(domain string) bool {
	_, _, sending, err := Client.GetSingleDomain(domain)
	if err != nil {
		return false
	}

	sendingDNS := ExtractDNS(domain, sending)
	return sendingDNS.Include.Valid && sendingDNS.Include.Valid
}

func GetDomain(name string) (*Domain, error) {
	domain, _, sendingRecords, err := Client.GetSingleDomain(name)
	if err != nil {
		if strings.Contains(err.Error(), "got=404") {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &Domain{
		Name:       domain.Name,
		SendingDNS: ExtractDNS(name, sendingRecords),
	}, nil
}

func ExtractDNS(domain string, records []mailgun.DNSRecord) SendingDNS {
	s := SendingDNS{}
	for _, dns := range records {
		if dns.RecordType == "TXT" {
			if dns.Name == domain && strings.Contains(dns.Value, "include") {
				s.Include = DNSRecord{"TXT", dns.Name, dns.Value, isValid(dns.Valid)}
			} else if strings.HasSuffix(dns.Name, "domainkey."+domain) {
				s.DomainKey = DNSRecord{"TXT", dns.Name, dns.Value, isValid(dns.Valid)}
			}
		}
	}
	return s
}

func isValid(s string) bool {
	if s == "valid" {
		return true
	}
	return false
}
