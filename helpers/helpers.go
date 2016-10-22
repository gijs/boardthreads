package helpers

import (
	"bt/db"
	"bt/mailgun"

	"gopkg.in/yaml.v2"

	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	mailgunGo "github.com/websitesfortrello/mailgun-go"
)

var here string

func init() {
	if os.Getenv("GOPATH") != "" {
		here = filepath.Join(os.Getenv("GOPATH"), "src/bt/helpers")
	} else {
		log.Info("no GOPATH found.")
		var err error
		here, err = filepath.Abs("./helpers")
		if err != nil {
			log.Fatal(err)
		}
	}
}

func HTMLToMarkdown(html string) string {
	command := exec.Command(filepath.Join(here, "html2markdown"), html)

	output, err := command.CombinedOutput()
	if err != nil {
		bound := len(html)
		if bound > 200 {
			bound = 200
		}
		log.WithFields(log.Fields{
			"err":    err.Error(),
			"html":   html[:bound],
			"stderr": string(output),
		}).Warn("couldn't convert html")
		return ""
	}

	return string(output)
}

func ParseAddress(from string) string {
	from = strings.Split(from, ",")[0]
	address, err := mail.ParseAddress(from)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err,
			"address": from,
		}).Warn("couldn't parse address")
		return from
	}
	return string(address.Address)
}

func ParseMultipleAddresses(to string) ([]string, error) {
	addresses, err := mail.ParseAddressList(to)
	if err != nil {
		log.WithFields(log.Fields{
			"err":       err,
			"addresses": to,
		}).Warn("couldn't parse multiple addresses")
		return nil, err
	}

	addrs := make([]string, len(addresses))
	for i, a := range addresses {
		addrs[i] = a.Address
	}

	return addrs, nil
}

func ParseCardDescription(desc string) (params db.ThreadParams, err error) {
	if desc == "" {
		return params, errors.New("description is blank.")
	}

	parts := strings.SplitN(desc, "---\n\n", 2)
	if len(parts) < 2 {
		return params, errors.New("no ---\\n\\n found.")
	}
	parts = strings.SplitN(parts[1], "\n\n---", 2)
	if len(parts) < 2 {
		return params, errors.New("no \\n\\n--- found.")
	}

	err = yaml.Unmarshal([]byte(parts[0]), &params)
	if err != nil {
		return params, err
	}
	if params.Subject == "" || params.ReplyTo == "" {
		return params, errors.New("yaml missing required params.")
	}

	return params, nil
}

func DownloadFile(path, url, authName, authPassword string) (err error) {
	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()

	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(authName, authPassword)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return
	}

	return
}

func ReplyToOrFrom(message mailgunGo.StoredMessage) string {
	replyto := MessageHeader(message, "Reply-To")
	if replyto != "" {
		return replyto
	}
	return ParseAddress(message.From)
}

func MessageHeader(message mailgunGo.StoredMessage, header string) string {
	for _, pair := range message.MessageHeaders {
		if pair[0] == header {
			return pair[1]
		}
	}
	return ""
}

func CommentEnvelopePrefix(text string) (len int) {
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, ":email:") {
		return 7
	} else if strings.HasPrefix(lower, ":e-mail:") {
		return 8
	} else if strings.HasPrefix(lower, ":envelope:") {
		return 10
	}
	return 0
}

func MakeCardName(message mailgunGo.StoredMessage) string {
	return fmt.Sprintf("%s {%s}", mailgun.TrimSubject(message.Subject), ReplyToOrFrom(message))
}

func ParseCardName(name string) (subject string, addr string, err error) {
	splitted := strings.Split(name, "{")
	if len(splitted) != 2 {
		err = errors.New("{ should have splitted in 2.")
		return
	}
	splitted2 := strings.Split(splitted[1], "}")
	if len(splitted2) != 2 {
		err = errors.New("} should have splitted in 2.")
		return
	}

	address, err := mail.ParseAddress(splitted2[0])
	if err != nil {
		return
	}

	subject = strings.TrimSpace(splitted[0])
	addr = address.Address

	return
}

func MakeCardDesc(message mailgunGo.StoredMessage) string {
	return fmt.Sprintf(`
---

to: %s
recipient: %s
from: %s
reply-to: %s
subject: %s

---
            `,
		MessageHeader(message, "To"),
		message.Recipients,
		message.From,
		ReplyToOrFrom(message),
		message.Subject,
	)
}
