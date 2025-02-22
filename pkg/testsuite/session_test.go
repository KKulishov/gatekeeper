/*
Copyright 2015 All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testsuite

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gogatekeeper/gatekeeper/pkg/apperrors"
	"github.com/gogatekeeper/gatekeeper/pkg/constant"
	"github.com/gogatekeeper/gatekeeper/pkg/keycloak/config"
	"github.com/gogatekeeper/gatekeeper/pkg/keycloak/proxy"
	"github.com/gogatekeeper/gatekeeper/pkg/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestGetIndentity(t *testing.T) {
	testCases := []struct {
		Request       func(token string) *http.Request
		Ok            bool
		ProxySettings func(c *config.Config)
	}{
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %s", token)},
					},
				}
			},
			Ok: true,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = false
			},
		},
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{"Basic QWxhZGRpbjpPcGVuU2VzYW1l"},
					},
				}
			},
			Ok: false,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = false
			},
		},
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{fmt.Sprintf("Test %s", token)},
					},
				}
			},
			Ok: false,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = false
			},
		},
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{},
				}
			},
			Ok: false,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = false
			},
		},
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{},
				}
			},
			Ok: false,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = true
			},
		},
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %s", token)},
					},
				}
			},
			Ok: false,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = true
			},
		},
		{
			Request: func(token string) *http.Request {
				return &http.Request{
					Header: http.Header{
						"Authorization": []string{"Basic QWxhZGRpbjpPcGVuU2VzYW1l"},
					},
				}
			},
			Ok: false,
			ProxySettings: func(c *config.Config) {
				c.SkipAuthorizationHeaderIdentity = true
			},
		},
	}

	for idx, testCase := range testCases {
		cfg := newFakeKeycloakConfig()
		testCase := testCase
		testCase.ProxySettings(cfg)

		p, idp, _ := newTestProxyService(cfg)
		token, err := newTestToken(idp.getLocation()).getToken()
		assert.NoError(t, err)

		user, err := p.GetIdentity(testCase.Request(token), cfg.CookieAccessName, "")

		if err != nil && testCase.Ok {
			t.Errorf("test case %d should not have errored", idx)
			continue
		}

		if err == nil && !testCase.Ok {
			t.Errorf("test case %d should not have errored", idx)
			continue
		}

		if err != nil && !testCase.Ok {
			continue
		}

		if user.RawToken != token {
			t.Errorf("test case %d the tokens are not the same", idx)
		}
	}
}

func TestGetTokenInRequest(t *testing.T) {
	defaultName := config.NewDefaultConfig().CookieAccessName
	token, err := newTestToken("test").getToken()
	assert.NoError(t, err)
	testCases := []struct {
		Token                           string
		AuthScheme                      string
		Error                           error
		SkipAuthorizationHeaderIdentity bool
	}{
		{
			Token:                           "",
			AuthScheme:                      "",
			Error:                           apperrors.ErrSessionNotFound,
			SkipAuthorizationHeaderIdentity: false,
		},
		{
			Token:                           token,
			AuthScheme:                      "",
			Error:                           nil,
			SkipAuthorizationHeaderIdentity: false,
		},
		{
			Token:                           token,
			AuthScheme:                      "Bearer",
			Error:                           nil,
			SkipAuthorizationHeaderIdentity: false,
		},
		{
			Token:                           "",
			AuthScheme:                      "",
			Error:                           apperrors.ErrSessionNotFound,
			SkipAuthorizationHeaderIdentity: true,
		},
		{
			Token:                           token,
			AuthScheme:                      "Bearer",
			Error:                           apperrors.ErrSessionNotFound,
			SkipAuthorizationHeaderIdentity: true,
		},
		{
			Token:                           token,
			AuthScheme:                      "",
			Error:                           nil,
			SkipAuthorizationHeaderIdentity: true,
		},
		{
			Token:                           "QWxhZGRpbjpPcGVuU2VzYW1l",
			AuthScheme:                      "Basic",
			Error:                           apperrors.ErrSessionNotFound,
			SkipAuthorizationHeaderIdentity: false,
		},
		{
			Token:                           token,
			AuthScheme:                      "Test",
			Error:                           apperrors.ErrSessionNotFound,
			SkipAuthorizationHeaderIdentity: false,
		},
	}
	for idx, testCase := range testCases {
		req := newFakeHTTPRequest(http.MethodGet, "/")
		if testCase.Token != "" {
			if testCase.AuthScheme != "" {
				req.Header.Set(constant.AuthorizationHeader, testCase.AuthScheme+" "+testCase.Token)
			} else {
				req.AddCookie(&http.Cookie{
					Name:   defaultName,
					Path:   req.URL.Path,
					Domain: req.Host,
					Value:  testCase.Token,
				})
			}
		}
		access, bearer, err := utils.GetTokenInRequest(req, defaultName, testCase.SkipAuthorizationHeaderIdentity, "")
		switch testCase.Error {
		case nil:
			assert.NoError(t, err, "case %d should not have thrown an error", idx)
			assert.Equal(t, testCase.AuthScheme == "Bearer", bearer)
			assert.Equal(t, token, access)
		default:
			assert.Equal(t, testCase.Error, err, "case %d, expected error: %s", idx, testCase.Error)
		}
	}
}

func TestIsExpired(t *testing.T) {
	user := &proxy.UserContext{
		ExpiresAt: time.Now(),
	}
	time.Sleep(1 * time.Millisecond)
	if !user.IsExpired() {
		t.Error("we should have been false")
	}
}

func TestGetUserContext(t *testing.T) {
	realmRoles := []string{"realm:realm"}
	clientRoles := []string{"client:client"}
	token := newTestToken("test")
	token.addRealmRoles(realmRoles)
	token.addClientRoles("client", []string{"client"})
	jwtToken, err := token.getToken()
	assert.NoError(t, err)
	webToken, err := jwt.ParseSigned(jwtToken)
	assert.NoError(t, err)
	context, err := proxy.ExtractIdentity(webToken)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, "1e11e539-8256-4b3b-bda8-cc0d56cddb48", context.ID)
	assert.Equal(t, "gambol99@gmail.com", context.Email)
	assert.Equal(t, "rjayawardene", context.PreferredName)
	assert.Equal(t, append(realmRoles, clientRoles...), context.Roles)
}

func TestGetUserRealmRoleContext(t *testing.T) {
	roles := []string{"dsp-dev-vpn", "vpn-user", "dsp-prod-vpn", "openvpn:dev-vpn"}
	token := newTestToken("test")
	token.addRealmRoles(roles)
	jwtToken, err := token.getToken()
	assert.NoError(t, err)
	webToken, err := jwt.ParseSigned(jwtToken)
	assert.NoError(t, err)
	context, err := proxy.ExtractIdentity(webToken)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, "1e11e539-8256-4b3b-bda8-cc0d56cddb48", context.ID)
	assert.Equal(t, "gambol99@gmail.com", context.Email)
	assert.Equal(t, "rjayawardene", context.PreferredName)
	// we have "defaultclient:default" in default test claims
	roles = append(roles, "defaultclient:default")
	assert.Equal(t, roles, context.Roles)
}

func TestUserContextString(t *testing.T) {
	token := newTestToken("test")
	jwtToken, err := token.getToken()
	assert.NoError(t, err)
	webToken, err := jwt.ParseSigned(jwtToken)
	assert.NoError(t, err)
	context, err := proxy.ExtractIdentity(webToken)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.NotEmpty(t, context.String())
}
