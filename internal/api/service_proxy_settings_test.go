package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func service(url string) *ServiceProxySettings {
	return &ServiceProxySettings{
		Url: url,
	}
}

func serviceCa(service *ServiceProxySettings) *ServiceProxySettings {
	service = service.Clone()
	service.CAPem = "Valid CA data"

	return service
}

func serviceHeader(service *ServiceProxySettings, values ...string) *ServiceProxySettings {
	if len(values) == 0 {
		values = []string{"Dummy Value"}
	}

	service = service.Clone()
	service.Header = http.Header{
		"Header": values,
	}

	return service
}

func TestServiceClone(t *testing.T) {
	a := serviceCa(serviceHeader(service("http:")))
	b := a.Clone()

	if a == b {
		t.Fatalf("Address of cloned build service didn't change")
	}

	if &a.Header == &b.Header {
		t.Fatalf("Address of cloned header didn't change")
	}
}

func TestServiceValidate(t *testing.T) {
	for _, tc := range []struct {
		service *ServiceProxySettings
		valid   bool
		msg     string
	}{
		{nil, false, "nil build service"},
		{service(""), false, "empty URL"},
		{service("ws:"), false, "websocket URL"},
		{service("wss:"), false, "secure websocket URL"},
		{service("http:"), true, "HTTP URL"},
		{service("https:"), true, "HTTPS URL"},
		{serviceCa(service("http:")), true, "any CA pem"},
		{serviceHeader(service("http:")), true, "any headers"},
		{serviceCa(serviceHeader(service("http:"))), true, "PEM and headers"},
	} {
		if tc.valid {
			assert.NoError(t, tc.service.Validate())
		} else {
			assert.Error(t, tc.service.Validate())
		}
	}
}
