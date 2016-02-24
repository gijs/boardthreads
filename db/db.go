package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "gopkg.in/cq.v1"
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
