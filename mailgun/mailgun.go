package mailgun

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/kelseyhightower/envconfig"
	gfm "github.com/shurcooL/github_flavored_markdown"
	"github.com/websitesfortrello/mailgun-go"
)

type Settings struct {
	ApiKey         string `envconfig:"MAILGUN_API_KEY"`
	BaseDomain     string `envconfig:"BASE_DOMAIN"`
	WebhookHandler string `envconfig:"WEBHOOK_HANDLER"`
	Secret         string `envconfig:"SESSION_SECRET"`
}

var Client mailgun.Mailgun
var settings Settings

func init() {
	var err error
	err = envconfig.Process("", &settings)
	if err != nil {
		log.Fatal(err.Error())
	}

	Client = mailgun.NewMailgun(settings.BaseDomain, settings.ApiKey, "")
}

func Send(params NewMessage) (messageId string, err error) {
	// we need a different client if we are to use an external domain
	localClient := Client
	if params.Domain != settings.BaseDomain {
		localClient = mailgun.NewMailgun(params.Domain, settings.ApiKey, "")
	}

	from := params.From
	if params.FromName != "" {
		from = fmt.Sprintf("%s <%s>", params.FromName, params.From)
	}
	message := localClient.NewMessage(from, params.Subject, params.Text, params.Recipients...)
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
	status, messageId, err := localClient.Send(message)
	if err != nil {
		log.Print("error sending email: ", status)
		return "", err
	}
	return messageId, nil
}

func DomainCanSendReceive(domain string) (canSend bool, canReceive bool) {
	_, receivingRecords, sendingRecords, err := Client.GetSingleDomain(domain)
	if err != nil {
		return false, false
	}

	DNS := ExtractDNS(domain, append(sendingRecords, receivingRecords...))
	canSend = DNS.Include.Valid && DNS.Include.Valid

	if DNS.Receive != nil && len(DNS.Receive) == 2 {
		canReceive = DNS.Receive[0].Valid && DNS.Receive[1].Valid
	}

	return
}

func PrepareExternalAddress(inbound, outbound string) (routeId string, err error) {
	// outbound must be previously verified to be a valid email address
	domain := strings.Split(outbound, "@")[1]

	err = Client.CreateDomain(domain, settings.Secret, "Delete", false)
	if err != nil {
		// verify if the domain isn't already there
		_, _, _, err = Client.GetSingleDomain(domain)
		if err != nil {
			// no, so it is a real problem
			log.WithFields(log.Fields{
				"err":    err.Error(),
				"domain": domain,
			}).Warn("failed to add domain to mailgun")
			return
		}
	}

	// create webhooks for bounce and success
	Client.CreateWebhook("deliver", settings.WebhookHandler+"/webhooks/mailgun/success")
	Client.CreateWebhook("bounce", settings.WebhookHandler+"/webhooks/mailgun/failure")
	// for now we don't care about the result

	route, err := Client.CreateRoute(mailgun.Route{
		Priority:    64,
		Description: "External inbound address.",
		Expression:  `match_recipient("` + outbound + `")`,
		Actions:     []string{`forward("` + inbound + `")`, "stop()"},
	})
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err.Error(),
			"route": `match_recipient("` + outbound + `")`,
		}).Warn("failed to add route to mailgun")
		return
	}

	return route.ID, nil
}

func GetDomain(name string) (*Domain, error) {
	domain, receivingRecords, sendingRecords, err := Client.GetSingleDomain(name)
	if err != nil {
		if strings.Contains(err.Error(), "Got=404") {
			return &Domain{Name: name}, nil
		} else {
			return nil, err
		}
	}

	return &Domain{
		Name: domain.Name,
		DNS:  ExtractDNS(name, append(sendingRecords, receivingRecords...)),
	}, nil
}

func ExtractDNS(domain string, records []mailgun.DNSRecord) *DNS {
	s := DNS{}
	for _, dns := range records {
		if dns.RecordType == "TXT" {
			if dns.Name == domain && strings.Contains(dns.Value, "include") {
				v := strings.Replace(dns.Value, "mailgun.org", "boardthreads.com", 1)
				s.Include = DNSRecord{"TXT", dns.Name, v, "", isValid(dns.Valid)}
			} else if strings.HasSuffix(dns.Name, "domainkey."+domain) {
				s.DomainKey = DNSRecord{"TXT", dns.Name, dns.Value, "", isValid(dns.Valid)}
			}
		} else if dns.RecordType == "MX" {
			s.Receive = append(s.Receive, DNSRecord{"MX", "", dns.Value, dns.Priority, isValid(dns.Valid)})
		}
	}
	return &s
}

func VerifyDNS(domain string) {
	Client.VerifyDomainDNS(domain)
}

func isValid(s string) bool {
	if s == "valid" {
		return true
	}
	return false
}
