package upstream

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsForDomain(t *testing.T) {
	exampleDomain := "example.com"

	tests := []struct {
		desc           string
		proxyDomain    string
		testDomain     string
		expectedResult bool
	}{
		{
			desc:           "when both domains are the same",
			proxyDomain:    exampleDomain,
			testDomain:     exampleDomain,
			expectedResult: true,
		},
		{
			desc:           "when domains are different",
			proxyDomain:    exampleDomain,
			testDomain:     "foobar.com",
			expectedResult: false,
		},
		{
			desc:           "when test domain is a subdomain",
			proxyDomain:    exampleDomain,
			testDomain:     "test." + exampleDomain,
			expectedResult: true,
		},
		{
			desc:           "when test domain is uppercase",
			proxyDomain:    exampleDomain,
			testDomain:     "test." + strings.ToUpper(exampleDomain),
			expectedResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "http://"+test.testDomain, nil)

			assert.Equal(t, test.expectedResult, isForDomain(test.proxyDomain)(r))
		})
	}
}
