package ooobot_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dtimm/ooobot/pkg/ooobot"
)

var _ = Describe("Ooobot", func() {
	pacificTime, _ := time.LoadLocation("America/Los_Angeles")
	var (
		o                 *ooobot.Ooobot
		startDateFixture  = time.Date(2020, 1, 1, 0, 0, 0, 0, pacificTime)
		endDateFixture    = time.Date(2020, 1, 1, 23, 59, 59, 0, pacificTime)
		activeTimeFixture = time.Date(2020, 1, 1, 12, 0, 0, 0, pacificTime)
	)

	Describe("New", func() {
		It("returns a new instance of Ooobot", func() {
			Expect(ooobot.New()).To(BeAssignableToTypeOf(&ooobot.Ooobot{}))
		})
	})

	BeforeEach(func() {
		o = ooobot.New()
	})

	Describe("HandleSlackRequest", func() {
		Context("when given a valid request", func() {
			var rr *httptest.ResponseRecorder
			BeforeEach(func() {
				b := bytes.NewBuffer([]byte(`{"slack_name":"test","start_date":"2020-01-01","end_date":"2020-01-01"}`))

				rr = httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/v1/outofoffice", b)

				o.HandleSlackRequest(rr, req)
			})

			It("returns a 200", func() {
				Expect(rr.Code).To(Equal(200))
			})

			It("stores the request", func() {
				Expect(o.GetOut(activeTimeFixture)).To(HaveExactElements(ooobot.Out{
					SlackName: "test",
					Start:     startDateFixture,
					End:       endDateFixture,
				}))
			})
		})

		Context("with no body", func() {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/v1/outofoffice", nil)

			o.HandleSlackRequest(rr, req)

			It("returns a 400", func() {
				Expect(rr.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("Out", func() {
		Context("when given a valid Out", func() {
			BeforeEach(func() {
				o.AddOut("test", "2020-01-01", "2020-01-01")
			})

			It("stores the Out", func() {

				Expect(o.GetOut(activeTimeFixture)).To(HaveExactElements(ooobot.Out{
					SlackName: "test",
					Start:     startDateFixture,
					End:       endDateFixture,
				}))
			})
		})

		Context("when there are no outs covering the active time", func() {
			BeforeEach(func() {
				o.AddOut("test", "2020-01-02", "2020-01-02")
				o.AddOut("test", "2020-01-03", "2020-01-03")
			})

			It("returns an empty slice", func() {
				Expect(o.GetOut(activeTimeFixture)).To(BeEmpty())
			})
		})

		Context("with bad dates", func() {
			It("returns an error", func() {
				err := o.AddOut("test", "not-a-date", "2020-01-02")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with empty dates", func() {
			It("returns an error", func() {
				err := o.AddOut("test", "", "")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
