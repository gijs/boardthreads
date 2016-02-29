package helpers

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/websitesfortrello/mailgun-go"
)

func HTMLToMarkdown(html string) (md string, err error) {
	command := exec.Command("./html2markdown")
	stdin, err := command.StdinPipe()
	stdin.Write([]byte(html))
	mdBytes, err := command.Output()
	return string(mdBytes), err
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
	}
	return subject
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
	return message.From
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
