package mailgun

import "strings"

func TrimSubject(subject string) string {
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
