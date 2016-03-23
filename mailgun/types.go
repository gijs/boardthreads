package mailgun

type NewMessage struct {
	ApplyMetadata bool
	HTML          string
	Text          string
	Recipients    []string
	From          string
	FromName      string
	Domain        string
	Subject       string
	InReplyTo     string
	ReplyTo       string
	CardId        string
	CommenterId   string
}

type Domain struct {
	Name string `json:"name"`
	DNS  *DNS   `json:"dns"`
}

type DNS struct {
	Include   DNSRecord   `json:"include"`
	DomainKey DNSRecord   `json:"domain_key"`
	Receive   []DNSRecord `json:"receive"`
}

type DNSRecordType string

const (
	TXT = "TXT"
	MX  = "MX"
)

type DNSRecord struct {
	Type     DNSRecordType `json:"type"`
	Name     string        `json:"name"`
	Value    string        `json:"value"`
	Priority string        `json:"priority"`
	Valid    bool          `json:"valid"`
}
