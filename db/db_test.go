package db

import (
	"testing"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func TestDB(t *testing.T) {

	if settings.Neo4jURL != "http://localhost:7474/" {
		panic("WRONG TEST DATABASE URL")
	}

	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("db", func() {

		g.Before(func() {
			DB.Exec(`
MATCH (n)
OPTIONAL MATCH (n)-[r]-()
DELETE n,r
            `)
		})

		g.It("should create a new user", func() {
			Expect(EnsureUser("u47284")).To(Equal(true))
		})

		g.It("should just ensure a created user", func() {
			Expect(EnsureUser("u47284")).To(Equal(false))
		})

		g.It("should not get an unexistent address", func() {
			address, err := GetAddress("u3298", "l23213")
			Expect(err).To(BeNil())
			Expect(address).To(BeNil())
		})

		g.It("should set new address", func() {
			_, _, err := SetAddress("maria", "b36437", "l43834", "maria@boardthreads.com", "maria@boardthreads.com")
			Expect(err).To(BeNil())
		})

		g.It("should get that address", func() {
			addr, err := GetAddress("maria", "maria@boardthreads.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr.UserId).To(Equal("maria"))
			Expect(addr.ListId).To(Equal("l43834"))
			Expect(addr.InboundAddr).To(Equal("maria@boardthreads.com"))
			Expect(addr.OutboundAddr).To(Equal("maria@boardthreads.com"))
			addr.Settings.ReplyTo = "" // GetAddress returns this, GetAddresses don't
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{*addr}))
		})

		g.It("should change the target list", func() {
			new, _, _ := SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "maria@boardthreads.com")
			Expect(new).To(Equal(false))

			addr, err := GetAddress("maria", "maria@boardthreads.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr.UserId).To(Equal("maria"))
			Expect(addr.ListId).To(Equal("l49983"))
			Expect(addr.InboundAddr).To(Equal("maria@boardthreads.com"))
			Expect(addr.OutboundAddr).To(Equal("maria@boardthreads.com"))
			addr.Settings.ReplyTo = "" // GetAddress returns this, GetAddresses don't
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{*addr}))
		})

		g.It("should change the outboundaddr", func() {
			new, outboundaddr, err := SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "help@maria.com")
			Expect(err).To(BeNil())
			Expect(new).To(Equal(false))
			Expect(outboundaddr).To(Equal("help@maria.com"))

			addr, err := GetAddress("maria", "maria@boardthreads.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(addr.UserId).To(Equal("maria"))
			Expect(addr.ListId).To(Equal("l49983"))
			Expect(addr.InboundAddr).To(Equal("maria@boardthreads.com"))
			Expect(addr.OutboundAddr).To(Equal("help@maria.com"))
			Expect(addr.DomainName).To(Equal("maria.com"))
			addr.DomainName = ""       // GetAddress returns this, GetAddresses don't
			addr.Settings.ReplyTo = "" // GetAddress returns this, GetAddresses don't
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{*addr}))
		})

		g.It("should get the owner of an address", func() {
			Expect(GetUserForAddress("maria@boardthreads.com")).To(Equal("maria"))
		})

		g.It("should set the route id for an outboundaddr", func() {
			Expect(SaveRouteId("help@maria.com", "r389472")).To(Succeed())
			Expect(SaveRouteId("help@ana.com", "r389472")).ToNot(Succeed())
		})

		g.It("should list correctly the addresses for a domain", func() {
			Expect(ListAddressesOnDomain("maria.com")).To(BeEquivalentTo([]string{"help@maria.com"}))
			Expect(ListAddressesOnDomain("donono.cy")).To(HaveLen(0))

			SetAddress("u2897321", "b3447", "l437734", "348956348956@boardthreads.com", "support@maria.com") // this fails
			addresses, _ := ListAddressesOnDomain("maria.com")
			Expect(addresses).To(HaveLen(1))
			Expect(addresses).To(ContainElement("help@maria.com"))

			SetAddress("maria", "b346847", "l4997734", "maria-support@boardthreads.com", "support@maria.com")
			addresses, _ = ListAddressesOnDomain("maria.com")
			Expect(addresses).To(HaveLen(2))
			Expect(addresses).To(ContainElement("help@maria.com"))
			Expect(addresses).To(ContainElement("support@maria.com"))
		})

		g.It("should delete a domain ownership when it has no email addresses in use", func() {
			Expect(MaybeReleaseDomainFromOwner("maria.com")).To(Succeed())
			// no effect
			SetAddress("u2897321", "b3447", "l437734", "324723y4782@boardthreads.com", "ana@maria.com") // should fail
			addresses, _ := ListAddressesOnDomain("maria.com")
			Expect(addresses).To(HaveLen(2))
			Expect(addresses).To(ContainElement("help@maria.com"))
			Expect(addresses).To(ContainElement("support@maria.com"))

			// remove the emails from the control of 'maria'
			SetAddress("maria", "b346847", "l4997734", "maria-support@boardthreads.com", "maria-support@boardthreads.com")
			SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "maria@boardthreads.com")
			Expect(MaybeReleaseDomainFromOwner("maria.com")).To(Succeed())

			// now another user can use the domain
			SetAddress("u2897321", "b3447", "l437734", "324723y4782@boardthreads.com", "ana@maria.com")
			addresses, _ = ListAddressesOnDomain("maria.com")
			Expect(addresses).To(HaveLen(1))
			Expect(addresses).To(ContainElement("ana@maria.com"))
		})

		g.Describe("billing", func() {

			g.It("should create an address with billing", func() {
				_, _, err := SetAddress("gorilla", "b96847", "l497814", "gorilla-support@boardthreads.com", "support@gorilla.com")
				Expect(err).ToNot(HaveOccurred())
				addr, _ := GetAddress("gorilla", "gorilla-support@boardthreads.com")
				Expect(addr.Status).To(Equal(TRIAL))

				Expect(SavePaypalProfileId("gorilla", "gorilla-support@boardthreads.com", "pay33746")).To(Succeed())

				addr, _ = GetAddress("gorilla", "gorilla-support@boardthreads.com")
				Expect(addr.PaypalProfileId).To(Equal("pay33746"))
				Expect(addr.Status).To(Equal(VALID))
			})

			g.It("should remove billing from an address", func() {
				Expect(RemovePaypalProfileId("gorilla-support@boardthreads.com")).To(Succeed())

				addr, _ := GetAddress("gorilla", "gorilla-support@boardthreads.com")
				Expect(addr.PaypalProfileId).To(Equal(""))
				Expect(addr.Status).To(Equal(TRIAL))
			})

		})

		g.Describe("creating, matching and deleting cards and messages", func() {

			g.It("should set a new user and address", func() {
				EnsureUser("bob")
				_, outboundaddr, _ := SetAddress("bob", "b34852", "l329847", "BOB@boardthreads.com", "emailTo@bob.com")
				Expect(outboundaddr).To(Equal("emailto@bob.com"))
				Expect(GetUserForAddress("bob@boardTHreads.com")).To(Equal("bob"))
			})

			g.It("should find a list for a fake received email", func() {
				Expect(GetTargetListForEmailAddress("bob@boardthreads.com")).To(Equal("l329847"))
			})

			g.It("should save new a card after failing to fetch one", func() {
				Expect(GetCardForMessage("", "this message", "frOM@someone.com", "bob@boardthreads.com")).To(Equal(""))
				Expect(SaveCardWithEmail("boB@boardthreads.com", "csl3739", "cid3739", "7676767")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (c:Card {id: "cid3739"})-[:LINKED_TO]-(e:EmailAddress {address: "bob@boardthreads.com"}) RETURN CASE WHEN c IS NOT NULL AND e IS NOT NULL THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))
			})

			g.It("should save the received email", func() {
				Expect(SaveEmailReceived("cid3739", "csl3739", "<mid3739>", "this message", "From@someone.com", "comm38754")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (c:Card {id: "cid3739"})-[:CONTAINS]->(m:Mail {id: "<mid3739>", subject: "this message", from: "from@someone.com", commentId: "comm38754"}) RETURN CASE WHEN c IS NOT NULL AND m IS NOT NULL THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))
			})

			g.It("should send a fake email from a fake comment", func() {
				Expect(GetEmailParamsForCard("csl3739")).To(BeEquivalentTo(sendingParams{
					LastMailId:      "<mid3739>",
					LastMailSubject: "this message",
					InboundAddr:     "bob@boardthreads.com",
					OutboundAddr:    "emailto@bob.com",
					Recipients:      []string{"from@someone.com"},
					ReplyTo:         "bob@boardthreads.com",
					AddReplier:      false,
				}))

				Expect(SaveCommentSent("csl3739", "bob", "<repl3739>", "32423432")).To(Succeed())
			})

			g.It("should delete the card", func() {
				Expect(RemoveCard("cid3739")).To(Succeed())
				var found bool
				err := DB.Get(&found, `MATCH (c:Card {id: "cid3739"})-[:CONTAINS]->(m:Mail {id: "<mid3739>", subject: "this message", from: "from@someone.com"}) RETURN c`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
			})
		})

		g.Describe("two equal (repeated) emails going to the same card", func() {
			// this is more a bug than a feature, two equal emails from the same person should be regarded as one
			g.It("should save a new card with two emails", func() {
				Expect(SaveCardWithEmail("bob@BOARDthreads.com", "csl8484", "cid8484", "7676767")).To(Succeed())
				Expect(SaveEmailReceived("cid8484", "csl8484", "<mid8484>", "repeated email", "from@someone.com", "comm84841")).To(Succeed())
				Expect(SaveCardWithEmail("bob@boardthreads.com", "csl8484", "cid8484", "7676767")).To(Succeed())
				Expect(SaveEmailReceived("cid8484", "csl8484", "<mid8484>", "repeated email", "FROM@someone.com", "comm84842")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (e:EmailAddress {address: "bob@boardthreads.com"}) RETURN CASE WHEN count(e) = 1 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))

				ok = false
				err = DB.Get(&ok, `MATCH (c:Card {id: "cid8484"})-[:CONTAINS]->(m:Mail {id: "<mid8484>", subject: "repeated email", from: "from@someone.com"}) RETURN CASE WHEN count(m) = 1 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))
			})

			g.It("should delete the card", func() {
				Expect(RemoveCard("cid8484")).To(Succeed())
				var found bool
				err := DB.Get(&found, `MATCH (c:Card {id: "cid8484"})-[:CONTAINS]->(m:Mail {id: "<mid8484>", subject: "repeated email", from: "from@someone.com"}) RETURN c`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
			})
		})

		g.Describe("deleting a card with emails linked to multiple cards", func() {
			// this happens when two different boardthreads accounts get the same email
			g.It("should save two cards out of the same email", func() {
				Expect(SaveCardWithEmail("bob@boardthreads.com", "csl5656", "cid5656", "7676767")).To(Succeed())
				Expect(SaveCardWithEmail("maria@boardthreads.com", "csl5757", "cid5757", "7676767")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (c:Card) WHERE c.id = "cid5656" OR c.id = "cid5757" RETURN CASE WHEN count(c) = 2 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))
			})

			g.It("should save two received emails with the same subject, but in different cards", func() {
				Expect(SaveEmailReceived("cid5656", "csl5656", "<mid55>", "multiple", "from@someone.com", "comm56561")).To(Succeed())
				Expect(SaveEmailReceived("cid5757", "csl5757", "<mid55>", "multiple", "from@someone.com", "comm57572")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (m:Mail {id: "<mid55>", subject: "multiple", from: "from@someone.com"}) RETURN CASE WHEN count(m) = 1 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))

				ok = false
				err = DB.Get(&ok, `MATCH (c:Card) WHERE c.id = "cid5656" OR c.id = "cid5757" RETURN CASE WHEN count(c) = 2 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))
			})

			g.It("should delete one of the cards", func() {
				Expect(RemoveCard("cid5656")).To(Succeed())
				var ok bool
				err := DB.Get(&ok, `MATCH (c:Card {id: "cid5656"})-[:CONTAINS]->(m:Mail {id: "<mid55>", subject: "multiple", from: "from@someone.com"}) RETURN c`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
				err = DB.Get(&ok, `MATCH (:Card {id: "cid5656"})-[con:CONTAINS]->(m) RETURN m`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
				err = DB.Get(&ok, `MATCH (c:Card {id: "cid5656"}) RETURN c`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
			})
		})

		g.Describe("again, now with more complication", func() {

			g.It("should setup everything", func() {
				// three cards
				Expect(SaveCardWithEmail("bob@boardthreads.com", "csl9696", "cid9696", "7676767")).To(Succeed())
				Expect(SaveCardWithEmail("maria@boardthreads.com", "csl9797", "cid9797", "7676767")).To(Succeed())
				Expect(SaveCardWithEmail("maria-support@boardthreads.com", "csl9898", "cid9898", "8686868")).To(Succeed())

				// one email for the three cards
				Expect(SaveEmailReceived("cid9696", "csl9696", "<mid99>", "it is complicated", "from@someone.com", "comm96961")).To(Succeed())
				Expect(SaveEmailReceived("cid9797", "csl9797", "<mid99>", "it is complicated", "from@someone.com", "comm97972")).To(Succeed())
				Expect(SaveEmailReceived("cid9898", "csl9898", "<mid99>", "it is complicated", "from@someone.com", "comm98982")).To(Succeed())

				// a different email, just for two cards
				Expect(SaveEmailReceived("cid9696", "csl9696", "<mid991>", "it is complicated", "from@someone.com", "comm96961")).To(Succeed())
				Expect(SaveEmailReceived("cid9797", "csl9797", "<mid991>", "it is complicated", "from@someone.com", "comm97972")).To(Succeed())

				// another, now just for one card
				Expect(SaveEmailReceived("cid9797", "csl9797", "<mid992>", "it is complicated", "from@someone.com", "comm97972")).To(Succeed())

				// some comments
				Expect(SaveCommentSent("csl9696", "u744763", "<replw6e4>", "324232")).To(Succeed())
				Expect(SaveCommentSent("csl9797", "u744863", "<replwew4>", "324432")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (m:Mail {from: "from@someone.com"})--(c:Card) WHERE c.id IN ["cid9797", "cid9898", "cid9696"] RETURN CASE WHEN count(DISTINCT m) = 3 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))

				ok = false
				err = DB.Get(&ok, `MATCH (c:Card) WHERE c.id = "cid9696" OR c.id = "cid9797" RETURN CASE WHEN count(c) = 2 THEN true ELSE false END AS ok`)
				Expect(err).To(BeNil())
				Expect(ok).To(Equal(true))
			})

			g.It("should delete one of the cards", func() {
				Expect(RemoveCard("cid9696")).To(Succeed())

				var ok bool
				err := DB.Get(&ok, `MATCH (c:Card {id: "cid9696"})-[:CONTAINS]->(m:Mail {id: "<mid99>", subject: "it is complicated", from: "from@someone.com"}) RETURN c`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
				err = DB.Get(&ok, `MATCH (:Card {id: "cid9696"})-[con:CONTAINS]->(m) RETURN m`)
				Expect(err).To(MatchError(`sql: no rows in result set`))
				err = DB.Get(&ok, `MATCH (c:Card {id: "cid9696"}) RETURN c`)
				Expect(err).To(MatchError(`sql: no rows in result set`))

				var ids []string
				DB.Select(&ids, `MATCH (c:Card {id: "cid9797"})--(m:Mail) RETURN m.id ORDER BY m.id`)
				Expect(ids).To(BeEquivalentTo([]string{"<mid991>", "<mid992>", "<mid99>", "<replwew4>"}))
			})
		})

		g.Describe("account info", func() {

			g.It("fetch last messages for user", func() {
				messages, err := LastMessagesForUser("bob", 20)
				Expect(err).ToNot(HaveOccurred())
				Expect(messages).To(HaveLen(0))

				messages, err = LastMessagesForUser("maria", 3)
				Expect(err).ToNot(HaveOccurred())
				Expect(messages).To(HaveLen(3))

				m := messages[0]

				messages, err = LastMessagesForUser("maria", 20)
				Expect(err).ToNot(HaveOccurred())
				Expect(messages).To(HaveLen(6))

				Expect(m.CommentId).To(Equal(messages[0].CommentId))
				Expect(messages[0].Id).To(Equal("<replwew4>"))
				Expect(messages[4].Address).To(Equal("maria-support@boardthreads.com"))
				Expect(messages[5].Address).To(Equal("maria@boardthreads.com"))

				messages, err = LastMessagesForUser("no-one", 20)
				Expect(err).ToNot(HaveOccurred())
				Expect(messages).To(HaveLen(0))
			})
		})

		g.Describe("custom params", func() {

			g.It("fetch default params for a card", func() {
				Expect(GetEmailParamsForCard("csl9797")).To(BeEquivalentTo(sendingParams{
					LastMailId:        "<mid992>",
					LastMailSubject:   "it is complicated",
					InboundAddr:       "maria@boardthreads.com",
					OutboundAddr:      "maria@boardthreads.com",
					Recipients:        []string{"from@someone.com"},
					ReplyTo:           "maria@boardthreads.com",
					SenderName:        "",
					SignatureTemplate: "",
					AddReplier:        false,
				}))
			})

			g.It("should set some params then fetch again", func() {
				Expect(ChangeAddressSettings("maria", "maria@boardthreads.com", AddressSettings{
					ReplyTo:           "cuisine@maria.com",
					SenderName:        "Marie",
					AddReplier:        true,
					MessageInDesc:     false,
					SignatureTemplate: "---\n\nThanks!\n{NAME}",
					MoveToTop:         true,
				})).To(Succeed())

				Expect(GetEmailParamsForCard("csl9797")).To(BeEquivalentTo(sendingParams{
					LastMailId:        "<mid992>",
					LastMailSubject:   "it is complicated",
					InboundAddr:       "maria@boardthreads.com",
					OutboundAddr:      "maria@boardthreads.com",
					Recipients:        []string{"from@someone.com"},
					ReplyTo:           "cuisine@maria.com",
					SenderName:        "Marie",
					SignatureTemplate: "---\n\nThanks!\n{NAME}",
					AddReplier:        true,
				}))

				Expect(GetReceivingParams("maria@boardthreads.com")).To(BeEquivalentTo(
					receivingParams{false, true},
				))

				addr, _ := GetAddress("maria", "maria@boardthreads.com")
				Expect(addr.Settings).To(BeEquivalentTo(
					AddressSettings{"Marie", "cuisine@maria.com", true, false, "---\n\nThanks!\n{NAME}", true}),
				)
			})

			g.It("set some params and unset others, then fetch", func() {
				Expect(ChangeAddressSettings("maria", "maria@boardthreads.com", AddressSettings{
					SenderName:    "Mariah",
					AddReplier:    false,
					MessageInDesc: true,
					ReplyTo:       "cuisine@maria.com", // all params must be set again everytime,
					// otherwise they are replaced with default values
					// it will happen with SignatureTemplate right now
					MoveToTop: false,
				})).To(Succeed())

				Expect(GetEmailParamsForCard("csl9797")).To(BeEquivalentTo(sendingParams{
					LastMailId:        "<mid992>",
					LastMailSubject:   "it is complicated",
					InboundAddr:       "maria@boardthreads.com",
					OutboundAddr:      "maria@boardthreads.com",
					Recipients:        []string{"from@someone.com"},
					ReplyTo:           "cuisine@maria.com",
					SenderName:        "Mariah",
					SignatureTemplate: "",
					AddReplier:        false,
				}))

				Expect(GetReceivingParams("maria@boardthreads.com")).To(BeEquivalentTo(
					receivingParams{true, false},
				))

				addr, _ := GetAddress("maria", "maria@boardthreads.com")
				Expect(addr.Settings).To(BeEquivalentTo(
					AddressSettings{"Mariah", "cuisine@maria.com", false, true, "", false}),
				)
			})

		})
	})
}
