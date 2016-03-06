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

type Domain struct {
	Name       string     `json:"name"`
	SendingDNS SendingDNS `json:"sendingDNS"`
}

type SendingDNS struct {
	Include   DNSRecord `json:"include"`
	DomainKey DNSRecord `json:"domain_key"`
}

type DNSRecordType string

const (
	TXT = "TXT"
	MX  = "MX"
)

type DNSRecord struct {
	Type  DNSRecordType `json:"type"`
	Name  string        `json:"name"`
	Value string        `json:"value"`
	Valid bool          `json:"valid"`
}
