package db

import (
	"bt/mailgun"

	"gopkg.in/cq.v1/types"
)

type Account struct {
	Addresses []Address `json:"addresses"`
}

type Address struct {
	Start           types.NullTime  `json:"start"                     db:"start"`
	BoardShortLink  string          `json:"boardShortLink"            db:"boardShortLink"`
	ListId          string          `json:"listId"                    db:"listId"`
	InboundAddr     string          `json:"inboundaddr"               db:"inboundaddr"`
	OutboundAddr    string          `json:"outboundaddr"              db:"outboundaddr"`
	RouteId         string          `json:"-"                         db:"routeId"`
	PaypalProfileId string          `json:"paypalProfileId,omitempty" db:"paypalProfileId"`
	DomainName      string          `json:"-"                         db:"domain"`
	DomainStatus    *mailgun.Domain `json:"domain,omitempty"`
}

type Email struct {
	Id        string         `json:"id"        db:"id"`
	Date      types.NullTime `json:"date"      db:"date"`
	Subject   string         `json:"subject"   db:"subject"`
	From      string         `json:"from"      db:"from"`
	CommentId string         `json:"commentId" db:"commentId"`
}

type emailParams struct {
	LastMailId      string   `db:"lastMailId"`
	LastMailSubject string   `db:"lastMailSubject"`
	InboundAddr     string   `db:"inbound"`
	OutboundAddr    string   `db:"outbound"`
	ReplyTo         string   `db:"replyTo"`
	Recipients      []string `db:"recipients"`
}
