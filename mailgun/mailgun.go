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
