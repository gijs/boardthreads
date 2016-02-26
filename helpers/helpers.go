package helpers

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mailgun/mailgun-go"
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

func DownloadFile(path string, url string) (err error) {
	out, err := os.Create(path)
	if err != nil {
		return
	}
	defer out.Close()

	resp, err := http.Get(url)
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
	for _, header := range message.MessageHeaders {
		if header[0] == "Reply-To" {
			return header[1]
		}
	}
	return message.From
}
