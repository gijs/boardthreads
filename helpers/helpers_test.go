package helpers

import (
	"testing"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func TestDB(t *testing.T) {

	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("helper functions", func() {

		g.It("should convert html to markdown", func() {
			Expect(HTMLToMarkdown("<div><h1>Title</h1><p>content</p></div>")).To(Equal("# Title\n\ncontent"))
		})

		g.It("should parse a single email address", func() {
			Expect(ParseAddress("yu.ti@iuoe.com")).To(Equal("yu.ti@iuoe.com"))
			Expect(ParseAddress(" yu.ti@iuoe.com ")).To(Equal("yu.ti@iuoe.com"))
			Expect(ParseAddress("Yuti <yu.ti@iuoe.com> ")).To(Equal("yu.ti@iuoe.com"))
			Expect(ParseAddress("Yuti <yu.ti@iuoe.com> , Prili <tyu@weq.com>")).To(Equal("yu.ti@iuoe.com"))
		})

		g.It("should parse multiple email addresses", func() {
			Expect(ParseMultipleAddresses("ope@poe.eop")).To(BeEquivalentTo([]string{"ope@poe.eop"}))
			Expect(ParseMultipleAddresses("Opé <ope@poe.eop>, ytue@ut.ey")).To(BeEquivalentTo([]string{"ope@poe.eop", "ytue@ut.ey"}))
			Expect(ParseMultipleAddresses("pÉo <ope@poe.eop>, yy<ytue@ut.ey>")).To(BeEquivalentTo([]string{"ope@poe.eop", "ytue@ut.ey"}))
		})

	})
}
