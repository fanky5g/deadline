package deadline

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	client *http.Client
)

// IsUnixTimePast compares current time vs passed in reference time to see if time is past
func IsUnixTimePast(refTime int64) bool {
	return refTime-time.Now().Unix() <= 0
}

// GetClient returns an http client for transactions
func GetClient() *http.Client {
	createClient := func() *http.Client {
		var t = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   0,
				KeepAlive: 0,
			}).Dial,
			TLSHandshakeTimeout: 30 * time.Second,
		}

		var client = &http.Client{
			Transport: t,
		}

		return client
	}

	if client == nil {
		client = createClient()
		return client
	}

	return client
}
