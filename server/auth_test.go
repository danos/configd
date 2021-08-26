// Copyright (c) 2017-2021 AT&T Intellectual Property
// All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file tests that authorization is correctly applied on the likes
// of set / delete / load / show operations.

package server_test

import (
	"fmt"
	"os/user"
	"strings"
	"testing"

	"github.com/danos/config/auth"
	. "github.com/danos/config/testutils"
	"github.com/danos/config/testutils/assert"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/server"
	"github.com/danos/utils/audit"
	"github.com/danos/utils/pathutil"
)

const authTestSchema = `
container interfaces {
    presence "For contrast with protocols";
	list dataplane {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type string;
		}
		leaf-list addressByUser {
			type string;
			ordered-by user;
		}
		leaf mtu {
			type uint16;
			default 1500;
		}
	}
}
container protocols {
	list bgp {
		key id;
		leaf id {
			type uint32;
		}
	}
}
container system {
	leaf substLeaf {
		type string;
		configd:subst "echo 'Running subst script'";
	}
	container login {
		list group {
			key tagnode;
			leaf tagnode {
				type string;
			}
		}
		list user {
			key tagnode;
			leaf tagnode {
				type string;
			}
			container authentication {
				leaf plaintext-password {
					type string;
				}
			}
		}
	}
	leaf host-name {
		type string;
	}
}
`

const accessDenied = "Access to the requested protocol operation or data model is denied because authorization failed."

func assertCmdAuthzRequests(t *testing.T, tAuth auth.TestAuther, expReqs auth.TestAutherRequests) {
	t.Helper()
	err := auth.CheckRequests(tAuth.GetCmdRequests(), expReqs)
	if err != nil {
		t.Fatal(err)
	}
	tAuth.ClearCmdRequests()
}

func assertCmdAcctRequests(t *testing.T, tAuth auth.TestAuther, expReqs auth.TestAutherRequests) {
	t.Helper()
	err := auth.CheckRequests(tAuth.GetCmdAcctRequests(), expReqs)
	if err != nil {
		t.Fatal(err)
	}
	tAuth.ClearCmdAcctRequests()
}

func clearAllCmdRequestsAndUserAuditLogs(tAuth auth.TestAuther) {
	tAuth.ClearCmdRequests()
	tAuth.ClearCmdAcctRequests()
	tAuth.GetAuditer().ClearUserLogs()
}

func genUserCmdLog(cmds ...string) audit.UserLogSlice {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	logs := audit.UserLogSlice{}
	for _, cmd := range cmds {
		logs = append(
			logs,
			audit.UserLog{
				Type:   audit.LOG_TYPE_USER_CMD,
				Msg:    fmt.Sprintf("run: %s, for user: %s", cmd, u.Uid),
				Result: 1,
			})
	}
	return logs
}

func getAuditerUserCmdLogs(a *audit.TestAudit) audit.UserLogSlice {
	logs := audit.UserLogSlice{}

	// We only care about LOG_TYPE_USER_CMD
	for _, log := range a.GetUserLogs() {
		if log.Type == audit.LOG_TYPE_USER_CMD {
			logs = append(logs, log)
		}
	}
	return logs
}

// Whitelist of commands which are not subject to command authorization
var noCmdAuthzWhitelist = map[string]struct{}{
	"commit":         struct{}{},
	"commit-confirm": struct{}{},
	"confirm":        struct{}{},
	"discard":        struct{}{},
	"validate":       struct{}{},
}

func assertCommandAaa(
	t *testing.T, a auth.TestAuther, cmd, rcmd []string,
	elemAttrs ...pathutil.PathElementAttrs,
) {
	t.Helper()

	attrs := pathutil.NewPathAttrs()
	attrs.Attrs = append(attrs.Attrs, elemAttrs...)

	expReqs := auth.NewTestAutherRequests(
		auth.NewTestAutherCommandRequest(auth.T_REQ_AUTH, cmd, &attrs))

	// If the command is on the "no authorization" whitelist then
	// verify that no command authorization requests were seen.
	// Otherwise verify that the expected requests were seen.
	if _, onWhitelist := noCmdAuthzWhitelist[cmd[0]]; onWhitelist {
		assertCmdAuthzRequests(t, a, auth.NewTestAutherRequests())
	} else {
		assertCmdAuthzRequests(t, a, expReqs)
	}

	// All commands are accounted
	expReqs = auth.NewTestAutherRequests(
		auth.NewTestAutherCommandRequest(auth.T_REQ_ACCT_START, cmd, &attrs),
		auth.NewTestAutherCommandRequest(auth.T_REQ_ACCT_STOP, cmd, &attrs))
	assertCmdAcctRequests(t, a, expReqs)

	// And all accounted commands get sent to the audit logs
	auditer := a.GetAuditer()
	expAuditLogs := genUserCmdLog(strings.Join(rcmd, " "))
	audit.AssertUserLogSliceEqual(t, expAuditLogs, getAuditerUserCmdLogs(auditer))
}

// Convenience function for the majority of test cases where no secrets
// are present in the command.
// If the command args under test contain secrets then assertCommandAaa()
// must be used directly, and passed appropriate PathElementAttrs
func assertCommandAaaNoSecrets(t *testing.T, a auth.TestAuther, cmd []string) {
	elemAttrs := []pathutil.PathElementAttrs{}
	for _, _ = range cmd {
		elemAttrs = append(elemAttrs, pathutil.PathElementAttrs{Secret: false})
	}
	assertCommandAaa(t, a, cmd, cmd, elemAttrs...)
}

func authorisedSetTest(t *testing.T) (auth.TestAuther, *server.Disp) {
	t.Helper()
	a := auth.NewTestAuther(
		auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces"))
	d := newTestDispatcherWithCustomAuth(
		t, a,
		authTestSchema, emptyconfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "interfaces/dataplane/dp0s3")
	return a, d
}

func TestAuthorisedSet(t *testing.T) {
	_, d := authorisedSetTest(t)
	dispTestValidate(t, d, testSID)
}

func TestAuthorisedSetCmdAaa(t *testing.T) {
	a, _ := authorisedSetTest(t)
	assertCommandAaaNoSecrets(t, a,
		[]string{"set", "interfaces", "dataplane", "dp0s3"})
}

func TestUnauthorisedSet(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, emptyconfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSetFails(t, d, testSID, "protocols/bgp/100",
		assert.NewExpectedMessages(accessDenied))
}

func TestUnauthorisedSetAsConfigdSuperuser(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, emptyconfig,
		true, /* configd user, so should always succeed */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "protocols/bgp/100")
	dispTestValidate(t, d, testSID)
}

// Need to make sure we don't bypass authentication due to configd:subst
// script.
func TestUnauthorisedSetWithSubst(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, emptyconfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSetFails(t, d, testSID, "system/substLeaf/aValue",
		assert.NewExpectedMessages(accessDenied))
}

var initDelConfig = Root(
	Cont("protocols",
		List("bgp",
			ListEntry("100"))))

func authorisedDeleteTest(t *testing.T) (auth.TestAuther, *server.Disp) {
	t.Helper()
	a := auth.NewTestAuther(
		auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
		auth.NewTestRule(auth.Allow, auth.AllOps, "/protocols"))
	d := newTestDispatcherWithCustomAuth(
		t, a,
		authTestSchema, initDelConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestDelete(t, d, testSID, "protocols/bgp/100")
	return a, d
}

func TestAuthorisedDelete(t *testing.T) {
	_, d := authorisedDeleteTest(t)
	dispTestValidate(t, d, testSID)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", emptyConfig)
}

func TestAuthorisedDeleteCmdAaa(t *testing.T) {
	a, _ := authorisedDeleteTest(t)
	assertCommandAaaNoSecrets(t, a,
		[]string{"delete", "protocols", "bgp", "100"})
}

func TestUnauthorisedDelete(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, initDelConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestDeleteFails(t, d, testSID, "protocols/bgp/100",
		assert.NewExpectedMessages(accessDenied))
}

// LOAD tests
//
// The 'load' operation works by removing any configuration which the user
// has permission to modify, then adding the loaded configuration.  We need
// to ensure that existing configuration that we are not permitted to change
// remains intact.
//
// The tests use the concept of different users:
//   - 'operator' (allowed to make certain changes then blocked from
//      anything else)
//   - 'administrator' (blocked from certain bits of config, then allowed
//     free rein elsewhere)
//   - 'vyatta' (superuser allowed to do anything)
var initLoadConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"))),
	Cont("protocols",
		List("bgp",
			ListEntry("100"))),
	Cont("system",
		Leaf("substLeaf", "aValue")))

var expLoadConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s2"))),
	Cont("protocols",
		List("bgp",
			ListEntry("100"))),
	Cont("system",
		Leaf("substLeaf", "aValue")))

var authLoadConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s2"))))

// No initial config, allowed to remove anything, so just expect to see
// loaded config present.
func TestLoadNoInitialConfigAllPerms(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authLoadConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", authLoadConfig)
}

// No initial config so while we aren't allowed to remove some things, there's
// nothing to remove so we just expect to see loaded config.
func TestLoadNoInitalConfigRestrictedPerms(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/protocols"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/system"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, emptyConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authLoadConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", authLoadConfig)
}

func TestLoadInitialConfigAllPerms(t *testing.T) {
	a := auth.NewTestAuther(
		auth.NewTestRule(auth.Allow, auth.AllOps, "*"))
	d := newTestDispatcherWithCustomAuth(
		t, a,
		authTestSchema, initLoadConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authLoadConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", authLoadConfig)
}

// Simple restricted permissions - just block top level protocols and system.
// We don't restrict READ so that we can easily view the final config in full!
func TestLoadInitialConfigRestrictedPerms(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/protocols"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/system"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, initLoadConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authLoadConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", expLoadConfig)
}

var initSystemConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"))),
	Cont("protocols",
		List("bgp",
			ListEntry("100"))),
	Cont("system",
		Leaf("host-name", "VR5600"),
		Cont("login",
			List("group",
				ListEntry("user")),
			List("user",
				ListEntry("admin",
					Cont("authentication",
						Leaf("plaintext-password", "admin"))),
				ListEntry("oper",
					Cont("authentication",
						Leaf("plaintext-password", "oper"))),
				ListEntry("root",
					Cont("authentication",
						Leaf("plaintext-password", "root")))))))

// Change host-name and oper password.  Group not specified, so will go.
var authOperConfig = Root(
	Cont("system",
		Leaf("host-name", "VR5600-oper"),
		Cont("login",
			List("user",
				ListEntry("oper",
					Cont("authentication",
						Leaf("plaintext-password", "oper-new")))))))

var expOperConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"))),
	Cont("protocols",
		List("bgp",
			ListEntry("100"))),
	Cont("system",
		Leaf("host-name", "VR5600-oper"),
		Cont("login",
			List("user",
				ListEntry("admin",
					Cont("authentication",
						Leaf("plaintext-password", "admin"))),
				ListEntry("oper",
					Cont("authentication",
						Leaf("plaintext-password", "oper-new"))),
				ListEntry("root",
					Cont("authentication",
						Leaf("plaintext-password", "root")))))))

var expOperConfigDefaults = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1",
				Leaf("mtu", "1500")))),
	Cont("protocols",
		List("bgp",
			ListEntry("100"))),
	Cont("system",
		Leaf("host-name", "VR5600-oper"),
		Cont("login",
			List("user",
				ListEntry("admin",
					Cont("authentication",
						Leaf("plaintext-password", "admin"))),
				ListEntry("oper",
					Cont("authentication",
						Leaf("plaintext-password", "oper-new"))),
				ListEntry("root",
					Cont("authentication",
						Leaf("plaintext-password", "root")))))))

// Operator: auth.Allow specific areas only; rest blocked
func TestLoadInitialConfigAllowThenBlockPerms(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "/system/login/group"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "/system/login/user/oper"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "/system/host-name")),
		authTestSchema, initSystemConfig,
		false,
		false)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authOperConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", expOperConfig)
	dispTestShowDefaults(t, d, rpc.CANDIDATE, testSID, "",
		expOperConfigDefaults)
}

// Change admin password.  Group not specified, so will go.  Likewise,
// interfaces (presence) and protocols (non-presence) will get nuked up
// the tree.
var authAdminConfig = Root(
	Cont("system",
		Cont("login",
			List("user",
				ListEntry("admin",
					Cont("authentication",
						Leaf("plaintext-password", "admin-new")))))))

var expAdminConfig = Root(
	Cont("system",
		Leaf("host-name", "VR5600"),
		Cont("login",
			List("user",
				ListEntry("admin",
					Cont("authentication",
						Leaf("plaintext-password", "admin-new"))),
				ListEntry("oper",
					Cont("authentication",
						Leaf("plaintext-password", "oper"))),
				ListEntry("root",
					Cont("authentication",
						Leaf("plaintext-password", "root")))))))

// Admin: This time we block specific areas then allow the rest
func TestLoadInitialConfigBlockThenAllowPerms(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/system/host-name"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/system/login/user/root"),
			auth.NewTestRule(auth.Deny, auth.AllOps, "/system/login/user/oper"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, initSystemConfig,
		false,
		false)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authAdminConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", expAdminConfig)
}

// Another permutation - check list entry for interface remains but we lose
// a couple of list entry leaves.
var initDataplaneConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1",
				Leaf("mtu", "1234"),
				LeafList("address",
					LeafListEntry("10101010"))))))

var authDataplaneConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"))))

func TestLoadInitialConfigDeleteTwoSiblings(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, initDataplaneConfig,
		false,
		false)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, authDataplaneConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", authDataplaneConfig)
}

// Defaults need to be treated with care.  We test we can change from a
// default to non-default value, from non-default to explicitly set
// default, and finally just delete a non-default value.  In all cases
// we verify that the 'show' and 'show all' outputs are correct.
var dataplaneNonDfltConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1",
				Leaf("mtu", "1600")))))

var dataplaneDfltConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1",
				Leaf("mtu", "1500")))))

var dataplaneConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1"))))

func TestLoadInitialConfigChangeFromDefault(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, dataplaneConfig,
		false,
		false)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, dataplaneNonDfltConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", dataplaneNonDfltConfig)
	dispTestShowDefaults(t, d, rpc.CANDIDATE, testSID, "",
		dataplaneNonDfltConfig)
}

// As we set the value explicitly to the default, it should be seen with
// 'show' as well as 'show all'.
func TestLoadInitialConfigChangeToDefaultExplicit(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, dataplaneNonDfltConfig,
		false,
		false)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, dataplaneDfltConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", dataplaneDfltConfig)
	dispTestShowDefaults(t, d, rpc.CANDIDATE, testSID, "",
		dataplaneDfltConfig)
}

// Default is implicitly set (deletion of non-default value) so should only
// be seen with 'show all'.
func TestLoadInitialConfigChangeToDefaultImplicit(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
		authTestSchema, dataplaneNonDfltConfig,
		false,
		false)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, dataplaneConfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", dataplaneConfig)
	dispTestShowDefaults(t, d, rpc.CANDIDATE, testSID, "",
		dataplaneDfltConfig)
}

var expEmptyLoadConfig = Root(
	Cont("system",
		Leaf("host-name", "VR5600"),
		Cont("login",
			Leaf("group", "user"),
			List("user",
				ListEntry("admin",
					Cont("authentication",
						Leaf("plaintext-password", "admin"))),
				ListEntry("oper",
					Cont("authentication",
						Leaf("plaintext-password", "oper"))),
				ListEntry("root",
					Cont("authentication",
						Leaf("plaintext-password", "root")))))))

// Check all works when the config we load is empty.
func TestLoadEmptyConfig(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces"),
			auth.NewTestRule(auth.Allow, auth.AllOps, "/protocols")),
		authTestSchema, initSystemConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoad(t, d, testSID, emptyconfig)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", expEmptyLoadConfig)
	dispTestShowDefaults(t, d, rpc.CANDIDATE, testSID, "",
		expEmptyLoadConfig)
}

// Next few tests check behaviour with unauthorised config being loaded.
// We test with and without initial config, and also check that config
// that would have worked before the configd:subst problem was fixed
// is also blocked.
var unauthLoadConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s2"))),
	Cont("protocols",
		List("bgp",
			ListEntry("111"))))

func TestUnauthorisedLoadInitialConfig(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, emptyconfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoadFails(t, d, testSID, unauthLoadConfig,
		assert.NewExpectedMessages(accessDenied))
}

func TestUnauthorisedLoadNoInitialConfig(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, initSystemConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoadFails(t, d, testSID, unauthLoadConfig,
		assert.NewExpectedMessages(accessDenied))
}

var unauthLoadWithSubstConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s2"))),
	Cont("system",
		Leaf("substLeaf", "someValue")))

func TestUnauthorisedLoadWithSubst(t *testing.T) {
	d := newTestDispatcherWithCustomAuth(
		t, auth.NewTestAuther(
			auth.NewTestRule(auth.Allow, auth.AllOps, "/interfaces")),
		authTestSchema, emptyconfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestLoadFails(t, d, testSID, unauthLoadWithSubstConfig,
		assert.NewExpectedMessages(accessDenied))
}

// Tests on ordered-by-user leaf-lists
//
// - ABC -> A / B / C
// - ABC -> all 2-entry combinations (6)
// - ABC -> all 3-entry combinations (6)
//
// Insert
//
// - ABC -> DABC
// - ABC -> ADBC
// - ABC -> ABDC
// - ABC -> ABCD
//
// Replace
//
// - ABC -> DBC
// - ABC -> ADC
// - ABC -> ABD

var leafListInitConfig = Root(
	Cont("interfaces",
		List("dataplane",
			ListEntry("dp0s1",
				LeafList("addressByUser",
					LeafListEntry("A"),
					LeafListEntry("B"),
					LeafListEntry("C"))))))

var rearrangedOptions = []string{
	"A", "B", "C",
	"AB", "AC", "BA", "BC", "CA", "CB",
	"ABC", "ACB", "BAC", "BCA", "CAB", "CBA"}

var insertOptions = []string{
	"DABC", "ADBC", "ABDC", "ABCD"}

var replaceOptions = []string{
	"D",
	"DA", "AD", "DB", "BD", "DC", "CD",
	"DAB", "DBA", "DAC", "DCA", "DBC", "DCB",
	"ADB", "BDA", "ADC", "CDA", "BDC", "CDB",
	"ABD", "BAD", "ACD", "CAD", "BCD", "CBD"}

var initLLConfig = `interfaces {
	dataplane dp0s1 {%s
	}
}
`

func generateLeafListCfg(entries string) string {
	var llCfg = ""
	for i := 0; i < len(entries); i++ {
		llCfg += fmt.Sprintf("\n\t\taddressByUser %s", string(entries[i]))
	}
	return fmt.Sprintf(initLLConfig, llCfg)
}

func TestLoadOrderedByUserLeafListsRearrange(t *testing.T) {
	for _, entries := range rearrangedOptions {
		d := newTestDispatcherWithCustomAuth(
			t, auth.NewTestAuther(
				auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
			authTestSchema, leafListInitConfig,
			false,
			false)

		genCfg := generateLeafListCfg(entries)
		dispTestSetupSession(t, d, testSID)
		dispTestLoad(t, d, testSID, genCfg)
		dispTestShow(t, d, rpc.CANDIDATE, testSID, "", genCfg)
	}
}

func TestLoadOrderedByUserLeafListsInsert(t *testing.T) {
	for _, entries := range insertOptions {
		d := newTestDispatcherWithCustomAuth(
			t, auth.NewTestAuther(
				auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
			authTestSchema, leafListInitConfig,
			false,
			false)

		genCfg := generateLeafListCfg(entries)
		dispTestSetupSession(t, d, testSID)
		dispTestLoad(t, d, testSID, genCfg)
		dispTestShow(t, d, rpc.CANDIDATE, testSID, "", genCfg)
	}
}

func TestLoadOrderedByUserLeafListsReplace(t *testing.T) {
	for _, entries := range replaceOptions {
		d := newTestDispatcherWithCustomAuth(
			t, auth.NewTestAuther(
				auth.NewTestRule(auth.Allow, auth.AllOps, "*")),
			authTestSchema, leafListInitConfig,
			false,
			false)

		genCfg := generateLeafListCfg(entries)
		dispTestSetupSession(t, d, testSID)
		dispTestLoad(t, d, testSID, genCfg)
		dispTestShow(t, d, rpc.CANDIDATE, testSID, "", genCfg)
	}
}

func TestShowCmdReqs(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		authTestSchema, initLoadConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "", initLoadConfig)

	assertCommandAaaNoSecrets(t, a, []string{"show"})
}

func TestShowCmdReqsPath(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcherWithCustomAuth(
		t, a,
		authTestSchema, initLoadConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestShow(t, d, rpc.CANDIDATE, testSID,
		"interfaces/dataplane/dp0s1", List("dataplane", ListEntry("dp0s1")))

	assertCommandAaaNoSecrets(t, a,
		[]string{"show", "interfaces", "dataplane", "dp0s1"})
}

const secretAuthTestSchema = `
container protocols {
	list bgp {
		key id;
		leaf id {
			type uint32;
		}
		leaf password {
			configd:secret "true";
			type string;
		}
	}
}
`

var secretAuthInitConfig = Root(
	Cont("protocols",
		List("bgp",
			ListEntry("50",
				Leaf("password", "bar")))))

func authorisedSecretSetTest(t *testing.T) (auth.TestAuther, *server.Disp) {
	t.Helper()
	a := auth.NewTestAuther(
		auth.NewTestRule(auth.Allow, auth.AllOps, "/protocols"))
	d := newTestDispatcherWithCustomAuth(
		t, a,
		secretAuthTestSchema, emptyconfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestSet(t, d, testSID, "protocols/bgp/100/password/foo")
	return a, d
}

func TestAuthorisedSecretSet(t *testing.T) {
	_, d := authorisedSecretSetTest(t)
	dispTestValidate(t, d, testSID)
}

func TestAuthorisedSecretSetCmdAaa(t *testing.T) {
	a, _ := authorisedSecretSetTest(t)

	cmd := []string{"set", "protocols", "bgp", "100", "password"}
	assertCommandAaa(t, a, append(cmd, "foo"), append(cmd, "**"),
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: true})
}

func authorisedSecretDeleteTest(t *testing.T) (auth.TestAuther, *server.Disp) {
	t.Helper()
	a := auth.NewTestAuther(
		auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
		auth.NewTestRule(auth.Allow, auth.AllOps, "/protocols"))
	d := newTestDispatcherWithCustomAuth(
		t, a,
		secretAuthTestSchema, secretAuthInitConfig,
		false, /* not configd user, so our auther gets used! */
		false /* not in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestDelete(t, d, testSID, "protocols/bgp/50/password/bar")
	return a, d
}

func TestAuthorisedSecretDelete(t *testing.T) {
	_, d := authorisedSecretDeleteTest(t)
	dispTestValidate(t, d, testSID)
	dispTestShow(t, d, rpc.CANDIDATE, testSID, "",
		Root(Cont("protocols", List("bgp", ListEntry("50")))))
}

func TestAuthorisedSecretDeleteCmdAaa(t *testing.T) {
	a, _ := authorisedSecretDeleteTest(t)

	cmd := []string{"delete", "protocols", "bgp", "50", "password"}
	assertCommandAaa(t, a, append(cmd, "bar"), append(cmd, "**"),
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: true})
}

func TestAuthorisedSecretShowConfigWithContextDiffs(t *testing.T) {
	a := auth.NewTestAuther(
		auth.NewTestRule(auth.Allow, auth.P_READ, "*"),
		auth.NewTestRule(auth.Allow, auth.AllOps, "/protocols"))
	d := newTestDispatcherWithCustomAuth(
		t, a,
		secretAuthTestSchema, secretAuthInitConfig,
		false, /* not configd user, so our auther gets used! */
		true /* in secrets group */)

	dispTestSetupSession(t, d, testSID)
	dispTestShowConfigWithContextDiffs(t, d, testSID, "protocols/bgp/50/password/bar",
		FormatAsDiffNoTrailingLine(Leaf("password", "bar")), false)

	cmd := []string{"show", "protocols", "bgp", "50", "password"}
	assertCommandAaa(t, a, append(cmd, "bar"), append(cmd, "**"),
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: false},
		pathutil.PathElementAttrs{Secret: true})
}
