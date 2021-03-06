// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package httpauth_test

import (
	"encoding/base64"
	"net/http"

	"golang.org/x/net/context"
	"gopkg.in/CanonicalLtd/candidclient.v1/params"
	gc "gopkg.in/check.v1"
	"gopkg.in/errgo.v1"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/identchecker"
	"gopkg.in/macaroon-bakery.v2/httpbakery"

	"github.com/CanonicalLtd/candid/internal/auth"
	"github.com/CanonicalLtd/candid/internal/auth/httpauth"
	"github.com/CanonicalLtd/candid/internal/candidtest"
)

type authSuite struct {
	candidtest.StoreSuite
	oven *bakery.Oven
}

var _ = gc.Suite(&authSuite{})

const identityLocation = "https://identity.test/id"

func (s *authSuite) SetUpTest(c *gc.C) {
	s.StoreSuite.SetUpTest(c)
	key, err := bakery.GenerateKey()
	c.Assert(err, gc.Equals, nil)
	locator := bakery.NewThirdPartyStore()
	locator.AddInfo(identityLocation, bakery.ThirdPartyInfo{
		PublicKey: key.Public,
		Version:   bakery.LatestVersion,
	})
	s.oven = bakery.NewOven(bakery.OvenParams{
		Key:      key,
		Locator:  locator,
		Location: "identity",
	})
}

func (s *authSuite) TearDownTest(c *gc.C) {
	s.StoreSuite.TearDownTest(c)
}

func (s *authSuite) TestAuthorizeWithAdminCredentials(c *gc.C) {
	tests := []struct {
		about              string
		adminPassword      string
		header             http.Header
		expectErrorMessage string
	}{{
		about:         "good credentials",
		adminPassword: "open sesame",
		header: http.Header{
			"Authorization": []string{"Basic " + b64str("admin:open sesame")},
		},
	}, {
		about:         "bad username",
		adminPassword: "open sesame",
		header: http.Header{
			"Authorization": []string{"Basic " + b64str("xadmin:open sesame")},
		},
		expectErrorMessage: "could not determine identity: invalid credentials",
	}, {
		about:         "bad password",
		adminPassword: "open sesame",
		header: http.Header{
			"Authorization": []string{"Basic " + b64str("admin:open sesam")},
		},
		expectErrorMessage: "could not determine identity: invalid credentials",
	}, {
		about:         "empty password denies access",
		adminPassword: "",
		header: http.Header{
			"Authorization": []string{"Basic " + b64str("admin:")},
		},
		expectErrorMessage: "could not determine identity: invalid credentials",
	}}
	for i, test := range tests {
		c.Logf("test %d. %s", i, test.about)
		authorizer := httpauth.New(s.oven, auth.New(auth.Params{
			AdminPassword:    test.adminPassword,
			Location:         identityLocation,
			Store:            s.Store,
			MacaroonVerifier: s.oven,
		}))
		req, _ := http.NewRequest("GET", "/", nil)
		for attr, val := range test.header {
			req.Header[attr] = val
		}
		authInfo, err := authorizer.Auth(context.Background(), req, identchecker.LoginOp)
		if test.expectErrorMessage != "" {
			c.Assert(err, gc.ErrorMatches, test.expectErrorMessage)
			c.Assert(errgo.Cause(err), gc.Equals, params.ErrUnauthorized)
			continue
		}
		c.Assert(err, gc.Equals, nil)
		c.Assert(authInfo.Identity.Id(), gc.Equals, auth.AdminUsername)
	}
}

func (s *authSuite) TestAuthorizeMacaroonRequired(c *gc.C) {
	authorizer := httpauth.New(s.oven, auth.New(auth.Params{
		AdminPassword:    "open sesame",
		Location:         identityLocation,
		Store:            s.Store,
		MacaroonVerifier: s.oven,
	}))
	req, err := http.NewRequest("GET", "http://example.com/v1/test", nil)
	c.Assert(err, gc.IsNil)
	authInfo, err := authorizer.Auth(context.Background(), req, identchecker.LoginOp)
	c.Assert(err, gc.ErrorMatches, `macaroon discharge required: authentication required`)
	c.Assert(authInfo, gc.IsNil)
	c.Assert(errgo.Cause(err), gc.FitsTypeOf, (*httpbakery.Error)(nil))
	derr := errgo.Cause(err).(*httpbakery.Error)
	c.Assert(derr.Info.CookieNameSuffix, gc.Equals, "candid")
	c.Assert(derr.Info.MacaroonPath, gc.Equals, "../")
	c.Assert(derr.Info.Macaroon, gc.NotNil)
}

func b64str(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
