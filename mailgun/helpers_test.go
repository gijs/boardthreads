package mailgun

import (
	"testing"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func TestDB(t *testing.T) {

	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("helper functions", func() {

		g.It("should trim subject", func() {
			Expect(TrimSubject("subject x")).To(Equal("subject x"))
			Expect(TrimSubject("fwd: subject x")).To(Equal("subject x"))
			Expect(TrimSubject("Fwd: subject x")).To(Equal("subject x"))
			Expect(TrimSubject("re: subject x")).To(Equal("subject x"))
			Expect(TrimSubject("RE: subject x")).To(Equal("subject x"))
			Expect(TrimSubject(" Fwd: subject x")).To(Equal("subject x"))
			Expect(TrimSubject("re: subject x ")).To(Equal("subject x"))
		})

	})
}
