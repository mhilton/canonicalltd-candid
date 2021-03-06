// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package admincmd_test

import (
	"encoding/json"
	"path/filepath"

	jc "github.com/juju/testing/checkers"
	"gopkg.in/CanonicalLtd/candidclient.v1"
	"gopkg.in/CanonicalLtd/candidclient.v1/params"
	gc "gopkg.in/check.v1"
	"gopkg.in/macaroon-bakery.v2/httpbakery/agent"

	"github.com/CanonicalLtd/candid/cmd/candid/internal/admincmd"
)

type createAgentSuite struct {
	commandSuite
}

var _ = gc.Suite(&createAgentSuite{})

var createAgentUsageTests = []struct {
	about       string
	args        []string
	expectError string
}{{
	about:       "agent file and agent key specified together",
	args:        []string{"-k", "S2oglf2m3F7oN6o4d517Y/aRjObgw/S7ZNevIIp+NnQ=", "-f", "foo", "bob"},
	expectError: `cannot specify public key and an agent file`,
}, {
	about:       "empty public key",
	args:        []string{"-k", "", "bob"},
	expectError: `invalid value "" for flag -k: wrong length for key, got 0 want 32`,
}, {
	about:       "invalid public key",
	args:        []string{"-k", "xxx", "bob"},
	expectError: `invalid value "xxx" for flag -k: wrong length for key, got 2 want 32`,
}}

func (s *createAgentSuite) TestUsage(c *gc.C) {
	for i, test := range createAgentUsageTests {
		c.Logf("test %d: %v", i, test.about)
		CheckError(c, 2, test.expectError, s.Run, append([]string{"create-agent"}, test.args...)...)
	}
}

func (s *createAgentSuite) TestCreateAgentWithGeneratedKeyAndAgentFileNotSpecified(c *gc.C) {
	var calledReq *params.CreateAgentRequest
	runf := s.RunServer(c, &handler{
		createAgent: func(req *params.CreateAgentRequest) (*params.CreateAgentResponse, error) {
			calledReq = req
			return &params.CreateAgentResponse{
				Username: "a-foo@candid",
			}, nil
		},
	})
	out := CheckSuccess(c, runf, "create-agent", "--name", "agentname", "-a", "admin.agent")
	c.Assert(calledReq, gc.NotNil)
	// The output should be valid input to an agent.Visitor unmarshal.
	var v agent.AuthInfo
	err := json.Unmarshal([]byte(out), &v)
	c.Assert(err, gc.Equals, nil)

	// Check that the public key looks right.
	agents := v.Agents
	c.Assert(agents, gc.HasLen, 1)
	c.Assert(calledReq.PublicKeys, gc.HasLen, 1)
	c.Assert(&v.Key.Public, gc.DeepEquals, calledReq.PublicKeys[0])
	c.Assert(agents[0].URL, gc.Matches, "https://.*")
	c.Assert(agents[0].Username, gc.Matches, "a-.+@candid")

	calledReq.PublicKeys = nil
	c.Assert(calledReq, jc.DeepEquals, &params.CreateAgentRequest{
		CreateAgentBody: params.CreateAgentBody{
			FullName: "agentname",
		},
	})
}

func (s *createAgentSuite) TestCreateAgentWithNonExistentAgentsFileSpecified(c *gc.C) {
	var calledReq *params.CreateAgentRequest
	runf := s.RunServer(c, &handler{
		createAgent: func(req *params.CreateAgentRequest) (*params.CreateAgentResponse, error) {
			calledReq = req
			return &params.CreateAgentResponse{
				Username: "a-foo@candid",
			}, nil
		},
	})
	agentFile := filepath.Join(c.MkDir(), ".agents")
	out := CheckSuccess(c, runf, "create-agent", "-a", "admin.agent", "-f", agentFile)
	c.Assert(calledReq, gc.NotNil)
	c.Assert(out, gc.Matches, `added agent a-foo@candid for https://.* to .+\n`)

	v, err := admincmd.ReadAgentFile(agentFile)
	c.Assert(err, gc.Equals, nil)

	agents := v.Agents
	c.Assert(agents, gc.HasLen, 1)
	c.Assert(calledReq.PublicKeys, gc.HasLen, 1)
	c.Assert(&v.Key.Public, gc.DeepEquals, calledReq.PublicKeys[0])
	c.Assert(agents[0].URL, gc.Matches, "https://.*")
	c.Assert(agents[0].Username, gc.Equals, "a-foo@candid")

	calledReq.PublicKeys = nil
	c.Assert(calledReq, jc.DeepEquals, &params.CreateAgentRequest{
		CreateAgentBody: params.CreateAgentBody{},
	})
}

func (s *createAgentSuite) TestCreateAgentWithExistingAgentsFile(c *gc.C) {
	var calledReq *params.CreateAgentRequest
	runf := s.RunServer(c, &handler{
		createAgent: func(req *params.CreateAgentRequest) (*params.CreateAgentResponse, error) {
			calledReq = req
			return &params.CreateAgentResponse{
				Username: "a-foo@candid",
			}, nil
		},
	})
	out := CheckSuccess(c, runf, "create-agent", "-a", "admin.agent", "-f", "admin.agent", "somegroup")
	c.Assert(calledReq, gc.NotNil)
	c.Assert(out, gc.Matches, `added agent a-foo@candid for https://.* to .+\n`)

	v, err := admincmd.ReadAgentFile(filepath.Join(s.Dir, "admin.agent"))
	c.Assert(err, gc.Equals, nil)

	agents := v.Agents
	c.Assert(agents, gc.HasLen, 2)
	c.Assert(calledReq.PublicKeys, gc.HasLen, 1)
	c.Assert(&v.Key.Public, gc.DeepEquals, calledReq.PublicKeys[0])
	c.Assert(agents[1].URL, gc.Matches, "https://.*")
	c.Assert(agents[1].Username, gc.Equals, "a-foo@candid")

	calledReq.PublicKeys = nil
	c.Assert(calledReq, jc.DeepEquals, &params.CreateAgentRequest{
		CreateAgentBody: params.CreateAgentBody{
			Groups: []string{"somegroup"},
		},
	})
}

func (s *createAgentSuite) TestCreateAgentWithAdminFlag(c *gc.C) {
	// With the -n flag, it doesn't contact the candid server at all.
	out := CheckSuccess(c, s.Run, "create-agent", "--admin")
	var v agent.AuthInfo
	err := json.Unmarshal([]byte(out), &v)
	c.Assert(err, gc.Equals, nil)
	agents := v.Agents
	c.Assert(agents, gc.HasLen, 1)
	c.Assert(agents[0].Username, gc.Equals, "admin@candid")
	c.Assert(agents[0].URL, gc.Equals, candidclient.Production)
}
