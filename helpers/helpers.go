package helpers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/websitesfortrello/mailgun-go"
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
	command := exec.Command(filepath.Join(here, "parseaddress"), from)

	output, err := command.CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err.Error(),
			"address": from,
			"stderr":  string(output),
		}).Warn("couldn't parse address")
		return from
	}

	return string(output)
}

func ParseMultipleAddresses(to string) ([]string, error) {
	command := exec.Command(filepath.Join(here, "parsemultipleaddresses"), to)

	output, err := command.CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{
			"err":       err.Error(),
			"addresses": to,
			"stderr":    string(output),
		}).Warn("couldn't parse")
		return nil, err
	}

	// the JS script will print the result as a list of comma-separated addresses
	addresses := strings.Split(string(output), ",")

	return addresses, nil
}

func ExtractSubject(subject string) string {
	subject = strings.Trim(subject, " ")
	for i := 0; i < 3; i++ {
		subject = strings.TrimPrefix(subject, "fw:")
		subject = strings.TrimPrefix(subject, "re:")
		subject = strings.TrimPrefix(subject, "fwd:")
		subject = strings.TrimPrefix(subject, "enc:")
		subject = strings.TrimPrefix(subject, "FW:")
		subject = strings.TrimPrefix(subject, "RE:")
		subject = strings.TrimPrefix(subject, "FWD:")
		subject = strings.TrimPrefix(subject, "ENC:")
		subject = strings.TrimPrefix(subject, "Fw:")
		subject = strings.TrimPrefix(subject, "Re:")
		subject = strings.TrimPrefix(subject, "Fwd:")
		subject = strings.TrimPrefix(subject, "Enc:")
	}
	return strings.TrimSpace(subject)
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

func ReplyToOrFrom(message mailgun.StoredMessage) string {
	replyto := MessageHeader(message, "Reply-To")
	if replyto != "" {
		return replyto
	}
	return ParseAddress(message.From)
}

func MessageHeader(message mailgun.StoredMessage, header string) string {
	for _, pair := range message.MessageHeaders {
		if pair[0] == header {
			return pair[1]
		}
	}
	return ""
}

func CommentStripPrefix(text string) string {
	text = strings.TrimPrefix(text, ":email: ")
	text = strings.TrimPrefix(text, ":e-mail: ")
	text = strings.TrimPrefix(text, ":envelope: ")
	return text
}

func MakeCardName(message mailgun.StoredMessage) string {
	return fmt.Sprintf("%s :: %s", ReplyToOrFrom(message), ExtractSubject(message.Subject))
}

func MakeCardDesc(message mailgun.StoredMessage) string {
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
