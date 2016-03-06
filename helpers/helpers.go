package helpers

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/websitesfortrello/mailgun-go"
)

func HTMLToMarkdown(html string) string {
	abspath, _ := filepath.Abs("./helpers/html2markdown")
	command := exec.Command(abspath, html)

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
	abspath, _ := filepath.Abs("./helpers/parseaddress")
	command := exec.Command(abspath, from)

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
