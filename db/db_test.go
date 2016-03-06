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
			Expect(GetAddress("maria", "maria@boardthreads.com")).To(BeEquivalentTo(address))
		})

		g.It("should change the target list", func() {
			Expect(SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "maria@boardthreads.com")).To(Equal(true))

			address := Address{
				ListId:       "l49983",
				InboundAddr:  "maria@boardthreads.com",
				OutboundAddr: "maria@boardthreads.com",
			}
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{address}))
			Expect(GetAddress("maria", "maria@boardthreads.com")).To(BeEquivalentTo(address))
		})

		g.It("should change the outboundaddr", func() {
			ok, err := SetAddress("maria", "b77837", "l49983", "maria@boardthreads.com", "help@maria.com")
			Expect(err).To(BeNil())
			Expect(ok).To(Equal(true))

			address := Address{
				ListId:       "l49983",
				InboundAddr:  "maria@boardthreads.com",
				OutboundAddr: "help@maria.com",
			}
			Expect(GetAddresses("maria")).To(BeEquivalentTo([]Address{address}))

			address.DomainName = "maria.com" // GetAddress returns this, GetAddresses don't
			Expect(GetAddress("maria", "maria@boardthreads.com")).To(BeEquivalentTo(address))
		})
	})
}
