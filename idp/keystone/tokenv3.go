// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package keystone

import (
	"net/http"
	"strings"

	"golang.org/x/net/context"
	"gopkg.in/errgo.v1"
	"gopkg.in/httprequest.v1"
	"gopkg.in/juju/idmclient.v1/params"

	"github.com/CanonicalLtd/blues-identity/config"
	"github.com/CanonicalLtd/blues-identity/idp"
	"github.com/CanonicalLtd/blues-identity/idp/idputil"
	"github.com/CanonicalLtd/blues-identity/idp/keystone/internal/keystone"
)

func init() {
	config.RegisterIDP("keystonev3_token", constructor(NewV3TokenIdentityProvider))
}

// NewV3TokenIdentityProvider creates a idp.IdentityProvider which will
// authenticate against a keystone (version 3) server using existing
// tokens.
func NewV3TokenIdentityProvider(p Params) idp.IdentityProvider {
	return &v3tokenIdentityProvider{
		identityProvider: newIdentityProvider(p),
	}
}

// v3tokenIdentityProvider is an identity provider that uses a configured
// keystone instance to authenticate against using an existing token to
// authenticate.
type v3tokenIdentityProvider struct {
	identityProvider
}

// Interactive implements idp.IdentityProvider.Interactive.
func (*v3tokenIdentityProvider) Interactive() bool {
	return false
}

// Handle implements idp.IdentityProvider.Handle.
func (idp *v3tokenIdentityProvider) Handle(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	var lr TokenLoginRequest
	if err := httprequest.Unmarshal(idputil.RequestParams(ctx, w, req), &lr); err != nil {
		idp.initParams.VisitCompleter.Failure(ctx, w, req, idputil.DischargeID(req), errgo.WithCausef(err, params.ErrBadRequest, "cannot unmarshal login request"))
		return
	}
	user, err := idp.doLoginV3(ctx, keystone.AuthV3{
		Identity: keystone.Identity{
			Methods: []string{"token"},
			Token: &keystone.IdentityToken{
				ID: lr.Token.Login.ID,
			},
		},
	})
	if err != nil {
		idp.initParams.VisitCompleter.Failure(ctx, w, req, idputil.DischargeID(req), err)
		return
	}
	if strings.TrimPrefix(req.URL.Path, idp.initParams.URLPrefix) == "/interact" {
		dt, err := idp.initParams.DischargeTokenCreator.DischargeToken(ctx, idputil.DischargeID(req), user)
		if err != nil {
			idp.initParams.VisitCompleter.Failure(ctx, w, req, idputil.DischargeID(req), err)
			return
		}
		httprequest.WriteJSON(w, http.StatusOK, TokenLoginResponse{
			DischargeToken: dt,
		})
	} else {
		idp.initParams.VisitCompleter.Success(ctx, w, req, idputil.DischargeID(req), user)
	}
}
