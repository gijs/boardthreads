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

	g.Describe("db functions", func() {

		g.It("should connect and delete everything", func() {
			Expect(DB).ToNot(BeNil())
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
			Expect(SetAddress("maria", "b36437", "l43834", "maria@boardthreads.com", "maria@boardthreads.com")).To(Equal(true))
		})

		g.It("should get that address", func() {
			address := Address{
				ListId:       "l43834",
				InboundAddr:  "maria@boardthreads.com",
				OutboundAddr: "maria@boardthreads.com",
			}
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{address}))
			Expect(GetAddress("maria", "maria@boardthreads.com")).To(BeEquivalentTo(&address))
		})

		g.It("should change the target list", func() {
			Expect(SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "maria@boardthreads.com")).To(Equal(false))

			address := Address{
				ListId:       "l49983",
				InboundAddr:  "maria@boardthreads.com",
				OutboundAddr: "maria@boardthreads.com",
			}
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{address}))
			Expect(GetAddress("maria", "maria@boardthreads.com")).To(BeEquivalentTo(&address))
		})

		g.It("should change the outboundaddr", func() {
			new, err := SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "help@maria.com")
			Expect(err).To(BeNil())
			Expect(new).To(Equal(false))

			address := Address{
				ListId:       "l49983",
				InboundAddr:  "maria@boardthreads.com",
				OutboundAddr: "help@maria.com",
			}
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{address}))

			address.DomainName = "maria.com" // GetAddress returns this, GetAddresses don't
			Expect(GetAddress("maria", "maria@boardthreads.com")).To(BeEquivalentTo(&address))
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
			Expect(addresses).To(HaveLen(3))
			Expect(addresses).To(ContainElement("help@maria.com"))
			Expect(addresses).To(ContainElement("support@maria.com"))
			Expect(addresses).To(ContainElement("ana@maria.com"))
		})
	})
}
