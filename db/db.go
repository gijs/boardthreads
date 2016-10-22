package db

import (
	"bt/mailgun"
	"errors"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "gopkg.in/cq.v1"
	"gopkg.in/cq.v1/types"
)

var DB *sqlx.DB

type Settings struct {
	Neo4jURL   string `envconfig:"GRAPHSTORY_URL" default:"http://localhost:7474/"`
	BaseDomain string `envconfig:"BASE_DOMAIN"    default:"boardthreads.com"`
}

var settings Settings

func init() {
	var err error
	err = envconfig.Process("", &settings)
	if err != nil {
		log.Fatal(err.Error())
	}

	DB = sqlx.MustConnect("neo4j-cypher", settings.Neo4jURL)
}

func EnsureUser(id string) (new bool, err error) {
	err = DB.Get(&new, `
OPTIONAL MATCH (ou:User {id: {0}})
MERGE (nu:User {id: {0}})
WITH CASE WHEN ou.id = nu.id THEN false ELSE true END AS new
RETURN new
    `, id)
	if err != nil {
		return
	}
	return
}

func GetUserForAddress(address string) (userId string, err error) {
	err = DB.Get(&userId, `
MATCH (u:User)-[:CONTROLS]->(:EmailAddress {address: {0}})
RETURN u.id AS userId
    `, strings.ToLower(address))
	return
}

func GetAddress(userId, emailAddress string) (*Address, error) {
	emailAddress = strings.ToLower(emailAddress)

	address := Address{}
	err := DB.Get(&address, `
MATCH (out)<-[s:SENDS_THROUGH]-(addr:EmailAddress {address: {0}})<-[c:CONTROLS]-()
MATCH (addr)-[t:TARGETS]->(l:List)
OPTIONAL MATCH (addr)-[sends:SENDS_THROUGH]->(o) WHERE o.address <> addr.address
OPTIONAL MATCH (out)<-[:OWNS]-(d:Domain)<-[:OWNS]-(:User {id: {1}})
RETURN
  l.id AS listId,
  addr.date AS date,
  addr.address AS inboundaddr,
  CASE WHEN out.address IS NOT NULL THEN out.address ELSE addr.address END AS outboundaddr,
  CASE WHEN d.host IS NOT NULL THEN d.host ELSE "" END AS domain,
  CASE WHEN sends.routeId IS NOT NULL THEN sends.routeId ELSE "" END AS routeId,
  CASE WHEN c.paypalProfileId IS NOT NULL THEN c.paypalProfileId ELSE "" END AS paypalProfileId,
  CASE WHEN addr.replyTo IS NOT NULL THEN addr.replyTo ELSE addr.address END AS replyTo,
  CASE WHEN addr.senderName IS NOT NULL THEN addr.senderName ELSE "" END AS senderName,
  CASE WHEN addr.addReplier IS NOT NULL THEN addr.addReplier ELSE false END AS addReplier,
  CASE WHEN addr.messageInDesc IS NOT NULL THEN addr.messageInDesc ELSE false END AS messageInDesc,
  CASE WHEN addr.signatureTemplate IS NOT NULL THEN addr.signatureTemplate ELSE "" END AS signatureTemplate,
  CASE WHEN addr.moveToTop IS NOT NULL THEN addr.moveToTop ELSE false END AS moveToTop
LIMIT 1
`, emailAddress, userId)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			// a real error
			return nil, err
		} else {
			// nothing found
			return nil, nil
		}
	}

	// post processing
	address.UserId = userId
	address.PostProcess()

	return &address, nil
}

func GetAddresses(userId string) (addresses []Address, err error) {
	err = DB.Select(&addresses, `
MATCH (u:User {id: {0}})
MATCH (u)-[c:CONTROLS]->(addr:EmailAddress)-->(l:List)
OPTIONAL MATCH (addr)-[:SENDS_THROUGH]->(o) WHERE o.address <> addr.address
RETURN
  l.id AS listId,
  addr.date AS date,
  addr.address AS inboundaddr,
  CASE WHEN o.address IS NOT NULL THEN o.address ELSE addr.address END AS outboundaddr,
  CASE WHEN c.paypalProfileId IS NOT NULL THEN c.paypalProfileId ELSE "" END AS paypalProfileId
ORDER BY date
    `, userId)
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			// a real error
			return addresses, err
		} else {
			// nothing found
			return addresses, nil
		}
	}

	// post processing
	for i := range addresses {
		addresses[i].UserId = userId
		addresses[i].PostProcess()
	}

	return
}

func (address *Address) Delete() (err error) {
	_, err = DB.Exec(`
MATCH (l:List)<-[t:TARGETS]-(addr {address: {0}})<-[c:CONTROLS]-()
OPTIONAL MATCH ()<-[s:SENDS_THROUGH]-(addr)
OPTIONAL MATCH (addr)-[h]-(card:Card)
OPTIONAL MATCH (m:Mail)-[mr]-(card)
OPTIONAL MATCH ()-[cmm:COMMENTED]->(m)
DELETE s, t, addr, c, h, card, m, mr, cmm
    `, address.InboundAddr)
	return
}

func SetAddress(userId, boardShortLink, listId, address, outboundaddr string) (new bool, actualOutbound string, err error) {
	address = strings.ToLower(address)
	outboundaddr = strings.ToLower(outboundaddr)

	err = DB.Get(&new, `
OPTIONAL MATCH (oldaddress:EmailAddress {address: {3}})
OPTIONAL MATCH (oldaddress)-[t:TARGETS]->()
OPTIONAL MATCH (oldaddress)-[s:SENDS_THROUGH]->(oldsendingaddress)
OPTIONAL MATCH (olduser:User)-[c:CONTROLS]->(oldaddress)
MERGE (newuser:User {id: {0}})
MERGE (newaddr:EmailAddress {address: {3}})
  ON CREATE SET newaddr.date = TIMESTAMP(),
                newaddr.moveToTop = true,
                newaddr.addReplier = true
MERGE (newlist:List {id: {2}})
MERGE (board:Board {shortLink: {1}})

MERGE (board)-[:CONTAINS]->(newlist)
MERGE (board)-[:MEMBER {admin: true}]->(newuser)

WITH olduser, oldaddress, oldsendingaddress, t, s, c, newuser, newlist, newaddr

// if 
FOREACH (t IN CASE WHEN oldaddress IS NULL THEN [1] ELSE [] END |
  MERGE (newuser)-[:CONTROLS]->(newaddr)
  MERGE (newaddr)-[:TARGETS]->(newlist)
  MERGE (newaddr)-[:SENDS_THROUGH]->(newaddr) // send through itself initially
)
// else
FOREACH (oldaddress IN CASE WHEN oldaddress IS NULL THEN [] ELSE [1] END |
  // if olduser.id == newuser.id
  FOREACH (oldaddress IN CASE WHEN olduser.id = newuser.id THEN [1] ELSE [] END |
    DELETE t, s, c
    MERGE (newuser)-[:CONTROLS]->(newaddr)
    MERGE (newaddr)-[:TARGETS]->(newlist)
    MERGE (newaddr)-[:SENDS_THROUGH]->(oldsendingaddress) // preserve any previous sending configuration
  )
  // else do nothing
)

WITH CASE
  WHEN oldaddress IS NULL THEN true
  // WHEN olduser.id = newuser.id THEN true
  ELSE false
END as new
RETURN new
`, userId, boardShortLink, listId, address)
	if err != nil {
		return
	}

	// we assume the function will be able to set the outboundaddr as specified
	actualOutbound = outboundaddr

	if new && outboundaddr == address {
		return
	}

	var domainName string
	outbound := strings.Split(outboundaddr, "@")
	if len(outbound) == 2 && (domainName != settings.BaseDomain || outboundaddr == address) {
		domainName = outbound[1]
		log.WithFields(log.Fields{"domain": domainName}).Debug("will set domain")
	} else {
		actualOutbound = address
		log.WithFields(log.Fields{
			"address":      address,
			"outboundaddr": outboundaddr,
		}).Warn("outboundaddr being set is invalid")
		return
	}

	// other queries just to set the outbound address
	var authorized bool
	if domainName == settings.BaseDomain {
		authorized = false
	} else {
		err = DB.Get(&authorized, `
OPTIONAL MATCH (d:Domain {host: {1}})
OPTIONAL MATCH (owner:User)-[ownership:OWNS]->(d)

RETURN CASE
  WHEN owner.id = {0} THEN true
  WHEN d IS NULL THEN true
  WHEN ownership IS NULL THEN true
  ELSE false
END AS authorized
    `, userId, domainName)
		if err != nil {
			authorized = false
		}
	}

	if !authorized {
		actualOutbound = address
		_, err = DB.Exec(`
MATCH (e:EmailAddress {address: {0}})
OPTIONAL MATCH (e)-[sends:SENDS_THROUGH]->()
OPTIONAL MATCH (e)-[:SENDS_THROUGH]->(:External)<-[ec:CONTROLS]-()
DELETE sends, ec

WITH e
MERGE (e)-[:SENDS_THROUGH]->(e)
    `, address)
		if err != nil {
			log.WithFields(log.Fields{
				"inboundaddr": address,
				"err":         err.Error(),
			}).Warn("failed to sent SENDS_THROUGH to inboundaddr")
		}
	} else {
		_, err = DB.Exec(`
MATCH (e:EmailAddress {address: {1}})
OPTIONAL MATCH (e)-[sends:SENDS_THROUGH]->()
OPTIONAL MATCH (e)-[:SENDS_THROUGH]->(:External)<-[ec:CONTROLS]-()
DELETE sends, ec
  
WITH e
MERGE (u:User {id: {0}})

MERGE (d:Domain {host: {2}})
MERGE (o:EmailAddress:External {address: {3}})

MERGE (u)-[:OWNS]->(d)
MERGE (d)-[:OWNS]->(o)
MERGE (o)<-[:SENDS_THROUGH]-(e)
   `, userId, address, domainName, outboundaddr)
		if err != nil {
			log.WithFields(log.Fields{
				"address":      address,
				"outboundaddr": outboundaddr,
				"err":          err.Error(),
			}).Warn("failed to set outboundaddr")
		}
	}
	return
}

func ChangeAddressSettings(userId, address string, p AddressSettings) error {
	address = strings.ToLower(address)

	var tmp string
	err := DB.Get(&tmp, `
MATCH (addr:EmailAddress {address: {1}})<-[:CONTROLS]-(user:User {id: {0}})
SET addr.replyTo = {2}
SET addr.senderName = {3}
SET addr.addReplier = {4}
SET addr.messageInDesc = {5}
SET addr.signatureTemplate = {6}
SET addr.moveToTop = {7}
RETURN user.id // just to fail when "no rows..."
    `, userId, address,
		p.ReplyTo, p.SenderName, p.AddReplier, p.MessageInDesc, p.SignatureTemplate, p.MoveToTop)
	return err
}

func MaybeReleaseDomainFromOwner(domainName string) error {
	_, err := DB.Exec(`
MATCH (d:Domain {host: {0}})
OPTIONAL MATCH (d)<-[ownership:OWNS]-(:User)
OPTIONAL MATCH (d)-[:OWNS]->(:EmailAddress:External)<-[s:SENDS_THROUGH]-(:EmailAddress)

WITH ownership, s

// delete ownership from domain when no emailaddress it
// is being used in relation with a fundamental address
FOREACH (x IN CASE WHEN s IS NULL THEN [1] ELSE [] END |
  DELETE ownership
)
    `, domainName)
	return err
}

func SaveRouteId(outboundaddr, routeId string) (err error) {
	var found bool
	err = DB.Get(&found, `
MATCH (out:EmailAddress:External {address: {0}})<-[sends:SENDS_THROUGH]-()
SET sends.routeId = {1}
RETURN CASE WHEN out IS NOT NULL THEN true ELSE false END AS found
    `, outboundaddr, routeId)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("couldn't find address " + outboundaddr)
	}
	return nil
}

func GetTargetListForEmailAddress(address string) (listId string, err error) {
	err = DB.Get(&listId, `
MATCH (:EmailAddress {address: {0}})-[:TARGETS]->(l:List)
RETURN l.id AS listId
    `, strings.ToLower(address))
	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			// a real error
			return "", err
		} else {
			// nothing found
			return "", nil
		}
	}
	return listId, nil
}

func GetMainEmailAddressForList(listId string) (addr string, err error) {
	err = DB.Get(&addr, `
MATCH (l:List {id: {0}})<-[:TARGETS]-(e:EmailAddress)
RETURN e.address ORDER BY e.date LIMIT 1
	`, listId)
	return
}

func GetCardForMessage(messageId, rawSubject, senderAddress, recipientAddress string) (string, error) {
	var queryResult struct {
		ShortLink   string         `db:"cardShortLink"`
		Address     string         `db:"address"`
		LastMessage types.NullTime `db:"last"`
		Expired     bool           `db:"expired"`
	}
	err := DB.Get(&queryResult, `
MATCH (m:Mail) WHERE m.id = {0} OR
                     ((m.subject = {1} OR m.subject = {2}) AND m.from = {3})
MATCH (m)--(c:Card)--(addr:EmailAddress)

WITH addr, c, MAX(m.date) AS last

RETURN
 c.shortLink AS cardShortLink,
 addr.address AS address,
 last,
 (TIMESTAMP() - last > 1000*60*60*24*15) AS expired // expiration: 15 days
LIMIT 1
    `, messageId, rawSubject, mailgun.TrimSubject(rawSubject), strings.ToLower(senderAddress))

	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			// a real error
			return "", err
		} else {
			// nothing found
			return "", nil
		}
	}

	// old messages are ignored so that we create a new card
	if queryResult.Expired {
		return "", nil
	}

	// cards that are somehow tied to a different @boardthreads address are also ignored
	if queryResult.Address != recipientAddress {
		return "", nil
	}

	return queryResult.ShortLink, nil
}

func ListAddressesOnDomain(domainName string) (domains []string, err error) {
	err = DB.Select(&domains, `
MATCH (:Domain {host: {0}})-[:OWNS]->(e:EmailAddress)
  WHERE (e)<-[:SENDS_THROUGH]-()
RETURN LOWER(e.address)
    `, domainName)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return domains, nil
		} else {
			return nil, err
		}
	}
	return domains, nil
}

func LastMessagesForUser(userId string, quantity int) (messages []Email, err error) {
	messages = make([]Email, 0)
	err = DB.Select(&messages, `
MATCH (u:User {id: {0}})--(e:EmailAddress)--(c:Card)--(m:Mail)
RETURN
  e.address AS address,
  c.shortLink AS cardShortLink,
  CASE WHEN m.id IS NOT NULL THEN m.id ELSE "" END AS id,
  CASE WHEN m.from IS NOT NULL THEN m.from ELSE "" END AS from,
  CASE WHEN m.subject IS NOT NULL THEN m.subject ELSE "" END AS subject,
  CASE WHEN m.commentId IS NOT NULL THEN m.commentId ELSE "" END AS commentId,
  m.date AS date
ORDER BY m.date DESC LIMIT {1}
    `, userId, quantity)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return messages, nil
		} else {
			return nil, err
		}
	}
	return messages, nil
}

func SaveCardWithEmail(emailAddress, cardShortLink, cardId, webhookId string) (err error) {
	if cardShortLink == "" || cardId == "" || webhookId == "" {
		log.WithFields(log.Fields{
			"emailAddress":  emailAddress,
			"cardShortLink": cardShortLink,
			"cardId":        cardId,
			"webhookId":     webhookId,
		}).Error("missing arguments to SaveCardWithEmail")
		return errors.New("missing arguments")
	}

	emailAddress = strings.ToLower(emailAddress)

	_, err = DB.Exec(`
MERGE (addr:EmailAddress {address: {0}})
MERGE (c:Card {shortLink: {1}})
MERGE (c)-[:LINKED_TO]->(addr)
      
WITH c
SET c.id = {2}
SET c.webhookId = {3}
    `, emailAddress, cardShortLink, cardId, webhookId)
	return
}

func RemoveCard(id string) (err error) {
	_, err = DB.Exec(`
MATCH (:EmailAddress)-[l]-(c:Card) WHERE c.shortLink = {0} OR c.id = {0}
WITH l, c LIMIT 1

OPTIONAL MATCH (c)-[rl]-(linked:Mail) WHERE (linked)--(:Card)
OPTIONAL MATCH (c)-[ru]-(unlinked:Mail) WHERE NOT (unlinked)--(:Card)

OPTIONAL MATCH ()-[cr:COMMENTED]-(unlinked)

DELETE c, rl, ru, l, unlinked, cr
    `, id)
	return
}

func GetEmailFromCommentId(commentId string) (email Email, err error) {
	err = DB.Get(&email, `
MATCH (m:Mail {commentId: {0}})
RETURN
  m.id AS id,
  m.date AS date,
  CASE WHEN m.subject THEN m.subject ELSE '' END AS subject,
  CASE WHEN m.from THEN LOWER(m.from) ELSE '' END AS from,
  m.commentId AS commentId
    `, commentId)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return Email{}, nil
		} else {
			return Email{}, err
		}
	}
	return email, nil
}

func GetReceivingParams(address string) (params receivingParams, err error) {
	err = DB.Get(&params, `
MATCH (addr:EmailAddress {address: {0}})
RETURN
  CASE WHEN addr.messageInDesc IS NOT NULL THEN addr.messageInDesc ELSE false END AS messageInDesc,
  CASE WHEN addr.moveToTop IS NOT NULL THEN addr.moveToTop ELSE false END AS moveToTop
LIMIT 1
    `, address)
	return
}

func GetEmailParamsForCard(shortLink string) (params sendingParams, err error) {
	err = DB.Get(&params, `
MATCH (c:Card) WHERE c.shortLink = {0} OR c.id = {0}
MATCH (c)--(addr:EmailAddress)
MATCH (outbound:EmailAddress)<-[:SENDS_THROUGH]-(addr)
MATCH (c)-[:CONTAINS]->(m:Mail) WHERE m.subject IS NOT NULL

WITH
  c, outbound, addr,
  reduce(lastMail = {}, m IN collect(m) | CASE WHEN lastMail.date > m.date THEN lastMail ELSE m END) AS lastMail,
  filter(r IN collect(DISTINCT LOWER(m.from)) WHERE r <> "") AS recipients
        
RETURN
  lastMail.id AS lastMailId,
  lastMail.subject AS lastMailSubject,
  LOWER(addr.address) AS inbound,
  LOWER(outbound.address) AS outbound,
  CASE WHEN addr.replyTo IS NOT NULL THEN addr.replyTo ELSE addr.address END AS replyTo,
  CASE WHEN addr.senderName IS NOT NULL THEN addr.senderName ELSE "" END AS senderName,
  CASE WHEN addr.addReplier IS NOT NULL THEN addr.addReplier ELSE false END AS addReplier,
  CASE WHEN addr.signatureTemplate IS NOT NULL THEN addr.signatureTemplate ELSE "" END AS signatureTemplate,
  recipients
LIMIT 1`, shortLink)
	return
}

func SaveEmailReceived(cardId, cardShortLink, messageId, subject, from, commentId string) (err error) {
	_, err = DB.Exec(`
MERGE (c:Card {shortLink: {0}})
MERGE (m:Mail {id: {1}})
  ON CREATE SET
    m.subject = {2},
    m.from = {3},
    m.commentId = {4},
    m.date = TIMESTAMP()
  ON MATCH SET
    m.from = {3}
MERGE (c)-[:CONTAINS]->(m)

WITH c
  SET c.id = {5}
`, cardShortLink, messageId, subject, strings.ToLower(from), commentId, cardId)
	return
}

func ChangeThreadParams(cardId string, params ThreadParams) (err error) {
	_, err = DB.Exec(`
MATCH (c:Card) WHERE c.shortLink = {0} OR c.id = {0}
MATCH (c)-[:CONTAINS]->(m:Mail) WHERE m.subject IS NOT NULL

WITH m ORDER BY m.date LIMIT 1
SET m.subject = {1}
SET m.from = {2}

RETURN m // just to fail when "no rows..."
    `, cardId, params.Subject, params.ReplyTo)
	return err
}

func SaveCommentSent(cardShortLink, commenterId, messageId, commentId string) (err error) {
	_, err = DB.Exec(`
MATCH (card:Card {shortLink: {0}})
MATCH (card)-[:LINKED_TO]->(:EmailAddress)-[:TARGETS]->(:List)--(b:Board)
MERGE (commenter:User {id: {1}})
MERGE (m:Mail {id: {2}})
  ON CREATE SET m.date = TIMESTAMP()

WITH m, commenter, card, b
SET m.commentId = {3}

MERGE (b)-[:MEMBER]->(commenter)
MERGE (card)-[:CONTAINS]->(m)
MERGE (commenter)-[:COMMENTED]->(m)
    `, cardShortLink, commenterId, messageId, commentId)
	return
}

func SavePaypalProfileId(userId, address, paypalProfileId string) (err error) {
	_, err = DB.Exec(`
MATCH (addr:EmailAddress {address: {1}})
MATCH (u:User {id: {0}})
MERGE (addr)<-[c:CONTROLS]-(u)
WITH c
SET c.paypalProfileId = {2}
`, userId, address, paypalProfileId)
	return
}

func RemovePaypalProfileId(address string) (err error) {
	_, err = DB.Exec(`
MATCH (:EmailAddress {address: {0}})<-[c:CONTROLS]-(:User)
SET c.paypalProfileId = null
`, address)
	return
}
