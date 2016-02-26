package mailgun

type NewMessage struct {
	HTML          string
	Text          string
	Recipients    []string
	From          string
	Subject       string
	InReplyTo     string
	ReplyTo       string
	CardShortLink string
	CommenterId   string
}
