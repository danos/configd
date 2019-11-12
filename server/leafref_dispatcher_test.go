// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// These tests verify tab / '?' completion options are correctly generated
// for leafref statements.

package server_test

import (
	"strings"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils"
	"github.com/danos/configd/rpc"
	"github.com/danos/configd/server"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror"
)

const (
	testSID     = ""
	emptyconfig = ""
	treesDiffer = false
	treesMatch  = true
)

func getAllowedOptionsInternal(
	t *testing.T,
	d *server.Disp,
	path string,
	treesShouldMatch bool,
) []string {

	// Get initial config
	initCfgTree, _ := d.TreeGet(rpc.RUNNING, testSID, "", "json", nil)

	// Now get allowed values
	actual, err := d.TmplGetAllowed(testSID, path)
	if err != nil {
		t.Fatalf("Unable to get allowed options: %s\n", err.Error())
	}

	// Get final tree and check it matches - any dummy nodes created while
	// working out allowed options must have been removed again.
	finalCfgTree, _ := d.TreeGet(rpc.CANDIDATE, testSID, "", "json", nil)
	if treesShouldMatch && (initCfgTree != finalCfgTree) {
		t.Fatalf("Dummy node wasn't removed!")
	}

	return actual
}

func getAllowedOptions(
	t *testing.T,
	allowAuth bool,
	schema, initConfig, path string,
) (allowedVals []string) {

	d := newTestDispatcher(t, auth.TestAutherAllowOrDenyAll(allowAuth),
		schema, initConfig)
	return getAllowedOptionsInternal(t, d, path, treesMatch)
}

func getAllowedOptionsMultipleSchemas(
	t *testing.T,
	allowAuth bool,
	schemaDefs []sessiontest.TestSchema,
	initConfig, path string,
) (allowedVals []string) {

	d := newTestDispatcherWithMultipleSchemas(
		t, auth.TestAutherAllowOrDenyAll(allowAuth), schemaDefs, initConfig)
	return getAllowedOptionsInternal(t, d, path, treesMatch)
}

func checkAllowedOptionsError(
	t *testing.T,
	allowAuth bool,
	schema, initConfig, path, expectedError string,
) {

	d := newTestDispatcher(t, auth.TestAutherAllowOrDenyAll(allowAuth),
		schema, initConfig)
	actual, err := d.TmplGetAllowed(testSID, path)

	if len(actual) != 0 {
		t.Fatalf("Should not have received any allowed options for '%s'", path)
		return
	}
	if err == nil {
		t.Fatalf("Expected error getting allowed options for '%s'", path)
		return
	}

	if err.Error() != expectedError {
		t.Logf("Wrong error getting allowed options for '%s'\n", path)
		t.Fatalf("Exp: %s\nGot: %s\n", expectedError, err.Error())
	}
}

func checkLeafrefOptions(t *testing.T, expected, actual []string) {
	if len(expected) != len(actual) {
		t.Logf("Expected %d options; got %d",
			len(expected), len(actual))
		t.Fatalf("Exp: %v\nAct: %v\n", expected, actual)
		return
	}

	for index, exp := range expected {
		if actual[index] != exp {
			t.Fatalf("Expected: '%s'. Got: '%s'", exp, actual[index])
			return
		}
	}
}

func commitLeafref(t *testing.T, schema, config, path string) error {

	d := newTestDispatcher(
		t, auth.TestAutherAllowAll(), schema, config)
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
	}
	dispTestSet(t, d, testSID, path)

	_, err := d.Commit(testSID, "message", false /* debug */)
	return err
}

func checkLeafrefValidationPass(t *testing.T, schema, config, path string) {
	if err := commitLeafref(t, schema, config, path); err != nil {
		t.Fatalf("Unable to commit config: %s\n", err.Error())
		return
	}
}

func checkLeafrefValidationFail(
	t *testing.T,
	schema,
	config,
	path,
	expErr string,
) {
	err := commitLeafref(t, schema, config, path)
	if err == nil {
		t.Fatalf("Able to commit config.\n")
		return
	}
	// Until VRVDR-27906 is fixed, there are cases where the path printed
	// as 'missing' is incorrect as the code to determine this uses an
	// algorithm which doesn't cope with lists well.
	if !strings.Contains(err.Error(), expErr) {
		// strings.Replace just reduces 'height' of output so more fits on
		// screen - useful when debugging several tests at once.
		t.Logf("Warning:\n%s\nShould contain: %s\n",
			strings.Replace(err.Error(), "\n\n", "\n", -1), expErr)
	}
}

// ************************ TESTS ******************************

// Simple leaf in a container, no lists in sight.  No existing
// configured leafListLeafref
const noListSchema = `
container testCont {
    container subCont {
        leaf-list testLeaf {
            type string;
        }
    }

    // Test leaflist
    leaf leafListLeafref {
		type leafref {
			path "../subCont/testLeaf";
		}
	}
}`
const noListConfig = `
	testCont {
		subCont {
            testLeaf woohoo
            testLeaf woohoo2
            testLeaf woohoo3
        }
	}`

func TestLeafrefOptionsLeafInCont(t *testing.T) {
	actual := getAllowedOptions(t, true, noListSchema, noListConfig,
		"testCont/leafListLeafref")
	expected := []string{"woohoo", "woohoo2", "woohoo3"}
	checkLeafrefOptions(t, expected, actual)
}

// Non tagnode string leaf within a list.
const stringSchema = `
container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		leaf admin-status {
			type string;
		}
	}

    // Tests non-tagnode leaf in list.
    leaf mgmt-interface-status {
		type leafref {
			path "../interface/admin-status";
		}
	}
}`
const stringConfig = `
	testCont {
		interface dp0s2 {
            admin-status "ok"
        }
	}`

func TestLeafrefOptionsStringLeafInList(t *testing.T) {
	actual := getAllowedOptions(t, true, stringSchema, stringConfig,
		"testCont/mgmt-interface-status")
	expected := []string{"ok"}
	checkLeafrefOptions(t, expected, actual)
}

// Piggy-back on same config, but with no read permission.
func TestLeafrefOptionsPermission(t *testing.T) {
	expectedError := mgmterror.NewAccessDeniedApplicationError().Error()
	checkAllowedOptionsError(t, false, stringSchema, stringConfig,
		"testCont/mgmt-interface-status", expectedError)
}

// Uint leaf within a list.
const uintSchema = `
container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		leaf mtu {
			type int32;
		}
	}

    // Test populated uint leaf
    leaf mgmt-interface-mtu {
		type leafref {
			path "../interface/mtu";
		}
	}

	list testList {
		key "name";
		leaf name {
			type string;
		}
		leaf ref {
			type leafref {
				path "/testCont/interface/mtu";
			}
		}
	}
}`
const uintConfig = `
	testCont {
		interface dp0s2 {
            mtu 1500
        }
	}`

func TestLeafrefOptionsUintLeafInList(t *testing.T) {
	actual := getAllowedOptions(t, true, uintSchema, uintConfig,
		"testCont/mgmt-interface-mtu")
	expected := []string{"1500"}
	checkLeafrefOptions(t, expected, actual)
}

// Uint leaf configured in candidate config not committed config.
const uintCandidateConfig = `
	testCont {
		interface dp0s2 {
            admin-status "ok";
        }
	}`

const uintCandidateSchema = `
	container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		leaf admin-status {
			type string;
		}
		leaf mtu {
			type int32;
		}
	}

    // Test populated uint leaf
    leaf mgmt-interface-mtu {
		type leafref {
			path "../interface/mtu";
		}
	}
}`

// Check auto-completion for candidate (uncommitted) nodes works.
func TestLeafrefOptionsUintCandidate(t *testing.T) {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), uintCandidateSchema,
		uintCandidateConfig)
	ok, err := d.SessionSetup(testSID)
	if !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}

	// Add interface MTU to candidate config.
	_, err = d.Set(testSID, "testCont/interface/dp0s2/mtu/1500")
	if err != nil {
		t.Fatalf("Unable to configure session: %s\n", err.Error())
		return
	}

	// Now get allowed values
	actual := getAllowedOptionsInternal(t, d, "testCont/mgmt-interface-mtu",
		treesDiffer)
	expected := []string{"1500"}
	checkLeafrefOptions(t, expected, actual)
}

// If we are getting allowed options for a not-yet-created node, then
// we create a dummy node to act as the context node for XPATH evaluation.
// This test makes sure that it gets removed again.
func TestLeafrefOptionsVerifyDummyNodeRemoved(t *testing.T) {

	actual := getAllowedOptions(t, true, uintSchema, uintConfig,
		"testCont/mgmt-interface-mtu")
	expected := []string{"1500"}
	checkLeafrefOptions(t, expected, actual)
}

// Check we remove ALL dummy nodes we create
func TestLeafrefOptionsVerifyDummyNodeRemovedInsideList(t *testing.T) {

	actual := getAllowedOptions(t, true, uintSchema, uintConfig,
		"testCont/testList/entry1/ref")
	expected := []string{"1500"}
	checkLeafrefOptions(t, expected, actual)
}

// Conversely, in this case ensure existing list remains!
func TestLeafrefOptionsInsideExistingList(t *testing.T) {
	d := newTestDispatcher(t, auth.TestAutherAllowAll(), uintSchema, uintConfig)

	ok, err := d.SessionSetup(testSID)
	if !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}

	// Add list entry
	_, err = d.Set(testSID, "testCont/testList/entry1")
	if err != nil {
		t.Fatalf("Unable to configure session: %s\n", err.Error())
		return
	}
	_, err = d.Commit(testSID, "", false)
	if err != nil {
		t.Fatalf("Unable to commit session: %s\n", err.Error())
		return
	}

	actual := getAllowedOptionsInternal(t, d, "testCont/testList/entry1/ref",
		treesMatch)
	expected := []string{"1500"}
	checkLeafrefOptions(t, expected, actual)
}

// Unconfigured uint leaf within a list.
const unusedUintSchema = `
container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		leaf admin-status {
			type string;
		}
		leaf unused {
			type uint8;
		}
	}

    // Test unpopulated uint leaf
    leaf unusedUintLeafref {
		type leafref {
			path "../interface/unused";
		}
	}
}`
const unusedUintConfig = `
	testCont {
		interface dp0s2 {
            admin-status "ok"
        }
	}`

func TestLeafrefOptionsUnusedUintLeafInList(t *testing.T) {
	actual := getAllowedOptions(t, true, unusedUintSchema, unusedUintConfig,
		"testCont/unusedUintLeafref")
	expected := []string{}
	checkLeafrefOptions(t, expected, actual)
}

// Call out tagnode separately as it is handled differently to other
// leaves within a list.
//
// NB: In this case we have the leafref under test configured already,
//     to a non-existent value.  Just making sure we don't get this value
//     back!
const tagnodeSchema = `
container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		list address {
			key "ip";
			leaf ip {
				type string;
			}
		}
	}

    // Test tagnode
    leaf mgmt-interface {
		type leafref {
			path "../interface/name";
		}
	}
}`
const tagnodeConfig = `
	testCont {
		interface dp0s2 {
            address 1234
            address 5678
        }
		interface s2 {
            address 4321
            address 8765
        }
        mgmt-interface dp0s2
	}`

func TestLeafrefOptionsTagnodeInList(t *testing.T) {

	actual := getAllowedOptions(t, true, tagnodeSchema, tagnodeConfig,
		"testCont/mgmt-interface")
	expected := []string{"dp0s2", "s2"}
	checkLeafrefOptions(t, expected, actual)
}

const leafrefValueConfig = `
	testCont {
		interface dp0s2 {
            address 1234
            address 5678
        }
		interface s2 {
            address 4321
            address 8765
        }
        mgmt-interface dp0s2
	}`

// Check that when we have a configured leafref, and we then specify that
// actual value in the path, that we don't get options shown.  They should
// only appear when we query at the 'mgmt-interface' level.
func TestLeafrefOptionsNotShownWhenCfgedValueGiven(t *testing.T) {

	actual := getAllowedOptions(t, true, tagnodeSchema, leafrefValueConfig,
		"testCont/mgmt-interface/dp0s2")
	expected := []string{}
	checkLeafrefOptions(t, expected, actual)
}

// Check that when we have a configured leafref, and we then specify a
// different valid completion, we get no options shown.
func TestLeafrefOptionsNotShownWhenUnconfigdValueGiven(t *testing.T) {

	actual := getAllowedOptions(t, true, tagnodeSchema, leafrefValueConfig,
		"testCont/mgmt-interface/s2")
	expected := []string{}
	checkLeafrefOptions(t, expected, actual)
}

// Predicate leafref
const predSchema = `
container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		leaf mtu {
			type int32;
		}
		list address {
			key "ip";
			leaf ip {
				type string;
			}
		}
	}

    // Predicate testing
    container default-address {
		leaf dummy {
			type string;
		}
		leaf ifname {
			type leafref {
				path "../../interface/name";
			}
		}
		leaf address {
			type leafref {
				path "../../interface[name = current()/../ifname]/address/ip";
			}
		}
	}
}`

const predConfig = `
	testCont {
		interface lo666 {
            mtu 1500
            address 4321
            address 8888
        }
		interface dp0s2 {
            mtu 1500
            address 1234
            address 5678
        }
    	default-address {
	    	ifname dp0s2
        }
	}`

func TestLeafrefOptionsPredicate(t *testing.T) {

	actual := getAllowedOptions(t, true, predSchema, predConfig,
		"testCont/default-address/address")
	expected := []string{"1234", "5678"}
	checkLeafrefOptions(t, expected, actual)
}

// Uint leaf within a list.
const dfltSchema = `
container testCont {
    list interface {
		key "name";
		leaf name {
			type string;
		}
		leaf mtu {
			type int32;
            default 1496;
		}
	}

    // Test populated uint leaf
    leaf mgmt-interface-mtu {
		type leafref {
			path "../interface/mtu";
		}
	}
}`
const defaultConfig = `
	testCont {
		interface dp0s2 {
        }
	}`

func TestLeafrefOptionsDefault(t *testing.T) {

	actual := getAllowedOptions(t, true, dfltSchema, defaultConfig,
		"testCont/mgmt-interface-mtu")
	expected := []string{"1496"}
	checkLeafrefOptions(t, expected, actual)
}

// These schemas are added to verify a bugfix for a relative leafref under a
// list statement.  The problem was that the FindNode function wasn't using
// the correct paths for comparison.
//
// The multiple schema stuff is actually irrelevant, but was written before the
// bug's root cause was found.  It has been left in as it verifies leafrefs
// work fine with multiple schemas.

const protocolsSchema = `
    container protocols {
    }`

const msdpSchema = `
	grouping peer-list {
		leaf-list peer-relative {
			ordered-by "user";
			configd:help "IP address of a peer in the group";
			type leafref {
				path "../../peer/address";
			}
		}
		leaf-list peer-absolute {
			ordered-by "user";
			configd:help "IP address of a peer in the group";
			type leafref {
				path "/protocols:protocols/msdp/peer/address";
			}
		}
	}

	augment /protocols:protocols {
		container msdp {
			list peer {
				key "address";
				leaf address {
					type string;
				}
            }
			list peer-group {
				key "name";
				leaf name {
					type string;
				}
				uses peer-list;
			}
		}
	}`

const msdpConfig = `
	protocols {
		msdp {
            peer 10.10.10.10 {
            }
            peer 20.20.20.20 {
            }
        }
	}`

var msdpSchemas = []sessiontest.TestSchema{
	{
		Name: sessiontest.NameDef{
			Namespace: "prefix-msdp",
			Prefix:    "msdp",
		},
		Imports: []sessiontest.NameDef{
			{"prefix-protocols", "protocols"}},
		SchemaSnippet: msdpSchema,
	},
	{
		Name: sessiontest.NameDef{
			Namespace: "prefix-protocols",
			Prefix:    "protocols",
		},
		SchemaSnippet: protocolsSchema,
	},
}

func TestMultipleSchemasRelPathAllowedUnderList(t *testing.T) {

	// Test relative peer options
	actual := getAllowedOptionsMultipleSchemas(
		t, true, msdpSchemas, msdpConfig,
		"protocols/msdp/peer-group/pg1/peer-relative")
	expected := []string{"10.10.10.10", "20.20.20.20"}
	checkLeafrefOptions(t, expected, actual)
}

func TestMultipleSchemasCommitRelPath(t *testing.T) {
	// Configure relative peer value
	d := newTestDispatcherWithMultipleSchemas(
		t, auth.TestAutherAllowAll(), msdpSchemas, msdpConfig)
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	dispTestSet(t, d, testSID,
		"protocols/msdp/peer-group/pg1/peer-relative/10.10.10.10")

	_, err := d.Commit(testSID, "message", false /* debug */)
	if err != nil {
		t.Fatalf("Unable to commit config.\n")
		return
	}
}

func TestMultipleSchemasAbsPathAllowedUnderList(t *testing.T) {
	// Test absolute peer options
	actual := getAllowedOptionsMultipleSchemas(
		t, true, msdpSchemas, msdpConfig,
		"protocols/msdp/peer-group/group1/peer-absolute")
	expected := []string{"10.10.10.10", "20.20.20.20"}
	checkLeafrefOptions(t, expected, actual)
}

func TestMultipleSchemasCommitAbsPath(t *testing.T) {
	// Configure absolute peer value
	d := newTestDispatcherWithMultipleSchemas(
		t, auth.TestAutherAllowAll(), msdpSchemas, msdpConfig)
	if ok, err := d.SessionSetup(testSID); !ok {
		t.Fatalf("Unable to setup session: %s\n", err.Error())
		return
	}
	dispTestSet(t, d, testSID,
		"protocols/msdp/peer-group/pg1/peer-absolute/10.10.10.10")

	_, err := d.Commit(testSID, "message", false /* debug */)
	if err != nil {
		t.Fatalf("Unable to commit config.\n")
		return
	}
}

// The following set of tests are designed to make sure relative leafrefs
// inside lists work correctly, in response to VRVDR-42907.
const listLeafrefSchema = `
	container top {
	list reference {
		key "name";
		leaf name {
			type string;
		}
		leaf refValue {
			type uint16;
		}
	}
	list source {
		key "src-name";
		leaf src-name {
			type string;
		}
		leaf-list leaflist-leafref {
			type leafref {
				path "../../reference/name";
			}
		}
		list abs-ref-list {
			key "name";
			leaf name {
				type leafref {
					path "/top/reference/name";
				}
			}
		}
		list rel-ok-key {
			key "name";
			leaf name {
				type leafref {
					path "../../../reference/name";
				}
			}
		}
		list rel-ok {
			key "name";
			leaf name {
				type string;
			}
			leaf value {
				type leafref {
					path "../../../reference/refValue";
				}
			}
		}
	}
}`

var listConfig = testutils.Root(
	testutils.Cont("top",
		testutils.List("reference",
			testutils.ListEntry("refX",
				testutils.Leaf("refValue", "66")),
			testutils.ListEntry("refY",
				testutils.Leaf("refValue", "77")),
			testutils.ListEntry("refZ"))))

var leaflistLeafrefConfig = testutils.Root(
	testutils.Cont("top",
		testutils.List("source",
			testutils.ListEntry("validLeaflistLeafref",
				testutils.LeafList("leaflist-leafref",
					testutils.LeafListEntry("refX"))))))

// Relative path from leaf-list inside list.
func TestLeafrefListOptionsLeaflist(t *testing.T) {

	actual := getAllowedOptions(t, true, listLeafrefSchema, listConfig,
		"top/source/validLeaflistLeafref/leaflist-leafref")
	expected := []string{"refX", "refY", "refZ"}

	checkLeafrefOptions(t, expected, actual)
}

func TestLeafrefListValidateLeaflistPass(t *testing.T) {

	checkLeafrefValidationPass(t, listLeafrefSchema, listConfig,
		"top/source/validLeaflistLeafref/leaflist-leafref/refZ")
}

func TestLeafrefListValidateLeaflistFail(t *testing.T) {
	// Get top source reference name none.
	checkLeafrefValidationFail(t, listLeafrefSchema, listConfig,
		"top/source/validLeaflistLeafref/leaflist-leafref/none",
		"[top reference none]")
}

// These 3 repeat the previous 3 *but* with an existing valid leafref
// entry in the 'source' node.  We test this separately because the code
// is different here (we don't have to create a dummy source node).
func TestLeafrefListOptionsLeafListExistingListEntry(t *testing.T) {
	actual := getAllowedOptions(t, true, listLeafrefSchema,
		listConfig+leaflistLeafrefConfig,
		"top/source/validLeaflistLeafref/leaflist-leafref")
	expected := []string{"refX", "refY", "refZ"}
	checkLeafrefOptions(t, expected, actual)
}

func TestLeafrefListValidateLeaflistPassExistingListEntry(t *testing.T) {
	checkLeafrefValidationPass(t, listLeafrefSchema,
		listConfig+leaflistLeafrefConfig,
		"top/source/validLeaflistLeafref/leaflist-leafref/refZ")
}

func TestLeafrefListValidateLeaflistFailExistingListEntry(t *testing.T) {
	checkLeafrefValidationFail(t, listLeafrefSchema,
		listConfig+leaflistLeafrefConfig,
		"top/source/validLeaflistLeafref/leaflist-leafref/none",
		"[top reference none]")
}

// Absolute path from list inside list.
func TestLeafrefListOptionsAbsRefList(t *testing.T) {
	actual := getAllowedOptions(t, true, listLeafrefSchema, listConfig,
		"top/source/absRefListEntry/abs-ref-list")
	expected := []string{"refX", "refY", "refZ"}
	checkLeafrefOptions(t, expected, actual)
}

func TestLeafrefListValidateAbsRefList(t *testing.T) {
	checkLeafrefValidationPass(t, listLeafrefSchema, listConfig,
		"top/source/absRefListEntry/abs-ref-list/refX")
}

func TestLeafrefListValidateAbsRefListFail(t *testing.T) {
	checkLeafrefValidationFail(t, listLeafrefSchema, listConfig,
		"top/source/absRefListEntry/abs-ref-list/none",
		"[top reference none]")
}

// Relative path from list leaf (key) inside list, correct leafref path
func TestLeafrefListOptionsRelOkListKey(t *testing.T) {
	actual := getAllowedOptions(t, true, listLeafrefSchema, listConfig,
		"top/source/relOkListEntry/rel-ok-key")
	expected := []string{"refX", "refY", "refZ"}
	checkLeafrefOptions(t, expected, actual)
}

func TestLeafrefListValidateRelOkListKeyPass(t *testing.T) {
	checkLeafrefValidationPass(t, listLeafrefSchema, listConfig,
		"top/source/relOkListEntry/rel-ok-key/refZ")
}

func TestLeafrefListValidateRelOkListKeyFail(t *testing.T) {
	checkLeafrefValidationFail(t, listLeafrefSchema, listConfig,
		"top/source/relOkListEntry/rel-ok-key/none",
		"[top reference none]")
}

// Relative path from list leaf (not key) inside list, correct leafref path,
// pointing to list leaf (not key).
func TestLeafrefListOptionsRelOkList(t *testing.T) {
	actual := getAllowedOptions(t, true, listLeafrefSchema, listConfig,
		"top/source/relOkListEntry/rel-ok/listEntry/value")
	expected := []string{"66", "77"}
	checkLeafrefOptions(t, expected, actual)
}

func TestLeafrefListValidateRelOkListPass(t *testing.T) {
	checkLeafrefValidationPass(t, listLeafrefSchema, listConfig,
		"top/source/relOkListEntry/rel-ok/listEntry/value/66")
}

func TestLeafrefListValidateRelOkListFail(t *testing.T) {
	checkLeafrefValidationFail(t, listLeafrefSchema, listConfig,
		"top/source/relOkListEntry/rel-ok/listEntry/value/88",
		"[top reference none]")
}

// Checking the path returned as 'missing' is ok when the list containing
// the reference leaves doesn't exist (previous test had list, just not
// matching entry for 'none')
func TestLeafrefListValidateRelOkListFailNoRefList(t *testing.T) {
	checkLeafrefValidationFail(t, listLeafrefSchema, emptyConfig,
		"top/source/relOkListEntry/rel-ok/listEntry/value/99",
		"[top reference none]")
}
