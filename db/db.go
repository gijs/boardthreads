package db

import (
	"bt/helpers"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "gopkg.in/cq.v1"
	"gopkg.in/cq.v1/types"
)

var DB *sqlx.DB

type Settings struct {
	Neo4jURL string `envconfig:"GRAPHSTORY_URL"`
}

func init() {
	var err error
	var settings Settings
	err = envconfig.Process("", &settings)
	if err != nil {
		log.Fatal(err.Error())
	}

	DB = sqlx.MustConnect("neo4j-cypher", settings.Neo4jURL)
}

func EnsureUser(id string) error {
	_, err := DB.Exec("MERGE (u:User {id: {0}})", id)
	if err != nil {
		return err
	}
	return nil
}

func GetAddressesForUserId(userId string) (addresses []Address, err error) {
	addresses = []Address{}
	err = DB.Select(&addresses, `
MATCH (u:User {id: {0}})
MATCH (u)-[c:CONTROLS]->(addr:EmailAddress)-->(l:List)
OPTIONAL MATCH (addr)-[:SENDS_THROUGH]->(o) WHERE o.address <> addr.address
RETURN
  l.id AS listId,
  addr.date AS start,
  addr.address AS inboundaddr,
  CASE WHEN o.address IS NOT NULL THEN o.address ELSE addr.address END AS outboundaddr,
  CASE WHEN c.paypalProfileId IS NOT NULL THEN c.paypalProfileId ELSE "" END AS paypalProfileId
ORDER BY start
    `, userId)
	return
}

func GetTargetListForEmailAddress(address string) (listId string, err error) {
	err = DB.Get(listId, `
MATCH (:EmailAddress {address: {0}})-[:TARGETS]->(l:List)
RETURN l.id AS listId
    `, address)
	return
}

func GetCardForMessage(
	messageId string,
	messageSubject string,
	currentAddress string,
) (string, error) {
	var queryResult struct {
		ShortLink   string         `db:"cardShortLink"`
		Address     string         `db:"address"`
		LastMessage types.NullTime `db:"last"`
		Expired     bool           `db:"expired"`
	}
	err := DB.Get(`
MATCH (m:Mail) WHERE m.id = {0} OR ((m.subject = {1} OR m.subject = {2}) AND m.from = {3})
MATCH (m)--(c:Card)--(addr:EmailAddress)

WITH addr, c, MAX(m.date) AS last

RETURN
 c.shortLink AS cardShortLink,
 addr.address AS address,
 last,
 (TIMESTAMP() - last > 1000*60*60*24*15) AS expired // expiration: 15 days
LIMIT 1
    `, messageId, messageSubject, helpers.ExtractSubject(messageSubject))
	if err != nil {
		return "", err
	}

	// old messages are ignored so that we create a new card
	if queryResult.Expired {
		return "", nil
	}

	// cards that are somehow tied to a different @boardthreads address are also ignored
	if queryResult.Address != currentAddress {
		return "", nil
	}

	return queryResult.ShortLink, nil
}

func SaveCardWithEmail(emailAddress string, cardShortLink string, webhookId string) (err error) {
	_, err = DB.Exec(`
MERGE (addr:EmailAddress {address: {0}})
MERGE (c:Card {shortLink: {1}})
MERGE (c)-[:LINKED_TO]->(addr)
      
WITH c
SET c.webhookId = {2}
    `, emailAddress, cardShortLink, webhookId)
	return
}

func RemoveCard(id string) (err error) {
	_, err = DB.Exec(`
MATCH (c:Card {shortLink: {0}})
MATCH (c)-[r]-(m:Mail)
MATCH (c)-[l]-(:EmailAddress)
DELETE c, r, l, m
    `, id)
	return
}

func SaveEmail(cardShortLink string, messageId string, subject string, from string) (err error) {
	_, err = DB.Exec(`
MERGE (c:Card {shortLink: {0}})
MERGE (m:Mail {
  id: {1},
  subject: {2},
  from: {3},
  date: TIMESTAMP()
})
MERGE (c)-[:CONTAINS]->(m)
`, cardShortLink, messageId, subject, from)
	return
}
