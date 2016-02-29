package mailgun

type NewMessage struct {
	ApplyMetadata bool
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
