package db

import "gopkg.in/cq.v1/types"

type Address struct {
	Start           types.NullTime `json:"start"                     db:"start"`
	BoardShortLink  string         `json:"boardShortLink"            db:"boardShortLink"`
	ListId          string         `json:"listId"                    db:"listId"`
	InboundAddr     string         `json:"inboundaddr"               db:"inboundaddr"`
	OutboundAddr    string         `json:"outboundaddr"              db:"outboundaddr"`
	PaypalProfileId string         `json:"paypalProfileId,omitempty" db:"paypalProfileId"`
	Domain          *Domain        `json:"domain"                    db:"domain"`
}

type Domain struct {
	Domain string `json:"domain" db:"domain"`
}

type Email struct {
	Id        string         `json:"id"        db:"id"`
	Date      types.NullTime `json:"date"      db:"date"`
	Subject   string         `json:"subject"   db:"subject"`
	From      string         `json:"from"      db:"from"`
	CommentId string         `json:"commentId" db:"commentId"`
}
