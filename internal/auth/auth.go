package auth

import (
	"encoding/base64"
	"net/http"
)

const proxyAuthorizationHeader = "Proxy-Authorization"

// Credentials represents a username/password pair.
type Credentials struct {
	Username string
	Password string
}

// IsValid returns true when both username and password are non-empty.
func (c Credentials) IsValid() bool {
	return c.Username != "" && c.Password != ""
}

// BasicHeader returns the value for a Proxy-Authorization header using basic auth.
func (c Credentials) BasicHeader() string {
	if !c.IsValid() {
		return ""
	}
	token := c.Username + ":" + c.Password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(token))
}

// SetProxyAuthorization sets the Proxy-Authorization header on the provided request if credentials are valid.
func SetProxyAuthorization(req *http.Request, cred Credentials) {
	if req == nil {
		return
	}
	if !cred.IsValid() {
		req.Header.Del(proxyAuthorizationHeader)
		return
	}
	req.Header.Set(proxyAuthorizationHeader, cred.BasicHeader())
}
