package db

import "gopkg.in/cq.v1/types"

type Address struct {
	Start           types.NullTime `json:"start"                     db:"start"`
	ListId          string         `json:"listId"                    db:"listId"`
	Inboundaddr     string         `json:"inboundaddr"               db:"inboundaddr"`
	Outboundaddr    string         `json:"outboundaddr"              db:"outboundaddr"`
	PaypalProfileId string         `json:"paypalProfileId,omitempty" db:"paypalProfileId"`
}

type Email struct {
	Date      types.NullTime `json:"date"      db:"date"`
	Id        string         `json:"id"        db:"id"`
	Subject   string         `json:"subject"   db:"subject"`
	From      string         `json:"from"      db:"from"`
	CommentId string         `json:"commentId" db:"commentId"`
}
