package db

import (
	"bt/mailgun"
	"time"

	"gopkg.in/cq.v1/types"
)

type Account struct {
	Addresses []Address `json:"addresses"`
}

type addressStatus string

const (
	VALID    addressStatus = "VALID"
	TRIAL    addressStatus = "TRIAL"
	DISABLED addressStatus = "DISABLED"
)

type Address struct {
	Start           int64           `json:"-"                db:"date"`
	UserId          string          `json:"-"                db:"userId"`
	BoardShortLink  string          `json:"boardShortLink"   db:"boardShortLink"`
	ListId          string          `json:"listId"           db:"listId"`
	InboundAddr     string          `json:"inboundaddr"      db:"inboundaddr"`
	OutboundAddr    string          `json:"outboundaddr"     db:"outboundaddr"`
	RouteId         string          `json:"-"                db:"routeId"`
	DomainName      string          `json:"-"                db:"domain"`
	DomainStatus    *mailgun.Domain `json:"domain,omitempty"`
	PaypalProfileId string          `json:"-"                db:"paypalProfileId"`
	Status          addressStatus   `json:"status"`
}

func (addr *Address) StartTime() time.Time {
	return time.Unix(addr.Start/1000, 0)
}

func (addr *Address) SetStatus() {
	if addr.PaypalProfileId != "" {
		addr.Status = VALID
	} else if time.Since(addr.StartTime()).Hours() > 1488 {
		addr.Status = DISABLED
	} else {
		addr.Status = TRIAL
	}
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
	AddReplier      bool     `db:"addReplier"`
	SenderName      string   `db:"senderName"`
}
