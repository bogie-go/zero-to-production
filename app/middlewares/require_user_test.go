package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"ecommerce/app/middlewares"
	"ecommerce/app/services/token"
)

var testTokens = token.Maker{Secret: []byte("a-test-secret-of-at-least-thirty-two-bytes"), Issuer: "test"}

func userRequest(authorization string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/me", middlewares.RequireUser(testTokens), func(c *gin.Context) { c.String(http.StatusOK, middlewares.Subject(c)) })
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

var _ = Describe("RequireUser", func() {
	var good, expired string

	BeforeEach(func() {
		var err error
		good, err = testTokens.Mint("user-42", time.Minute)
		Expect(err).NotTo(HaveOccurred())
		expired, err = testTokens.Mint("user-42", -time.Minute)
		Expect(err).NotTo(HaveOccurred())
	})

	It("lets a good token through with its subject", func() {
		rec := userRequest("Bearer " + good)
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(Equal("user-42"))
	})

	DescribeTable("answers 401",
		func(header func() string) {
			Expect(userRequest(header()).Code).To(Equal(http.StatusUnauthorized))
		},
		Entry("with no token", func() string { return "" }),
		Entry("with an expired token", func() string { return "Bearer " + expired }),
		Entry("with garbage", func() string { return "Bearer nope" }),
		Entry("with the wrong scheme", func() string { return "Basic " + good }),
	)
})
