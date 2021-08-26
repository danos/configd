// Copyright (c) 2019-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session_test

import (
	"testing"

	"github.com/danos/config/testutils"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

// Declarations required in addition to those in session_test.go
const subcontainer = "subcontainer"
const level3container = "level3container"
const level4container = "level4container"
const subtrue = "subtrue"
const subfalse = "subfalse"
const level3leaf = "level3leaf"
const level4leaf = "level4leaf"

var subcontainerpath = pathutil.CopyAppend(testcontainerpath, subcontainer)
var level3containerpath = pathutil.CopyAppend(subcontainerpath, level3container)
var level4containerpath = pathutil.CopyAppend(level3containerpath, level4container)
var subfalsepath = pathutil.CopyAppend(subcontainerpath, subfalse)
var subtruepath = pathutil.CopyAppend(subcontainerpath, subtrue)
var level3leafpath = pathutil.CopyAppend(level3containerpath, level3leaf)
var level4leafpath = pathutil.CopyAppend(level4containerpath, level4leaf)

// Check that "mandatory false" statement is correctly compiled as
// "mandatory false"
func TestMandatoryFalseIsFalse(t *testing.T) {
	const schemaMandFalse = `
container testcontainer {
	leaf testleaf {
		type string;
		mandatory false;
	}
	leaf teststring {
		type string;
		mandatory false;
	}
	leaf testboolean {
		type boolean;
		mandatory false;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
		leaf testleaf {
			type string;
			mandatory false;
		}
	}
	leaf-list testleaflistuser {
		type string;
	}
}
`

	tblSetMandFalse := []ValidateOpTbl{
		NewValOpTblEntry("Verify set of non-mandatory", testleafpath, "foo", true),
		NewValOpTblEntry("", testbooleanpath, "true", true),
		NewValOpTblEntry("", testleaflistuserpath, "foo", true),
		NewValOpTblEntry("", teststringpath, "foo", true),
	}
	tblDeleteMandFalse := []ValidateOpTbl{
		NewValOpTblEntry("Verify delete of non-mandatory", testbooleanpath, "true", true),
		NewValOpTblEntry("", testleafpath, "foo", true),
		NewValOpTblEntry("", teststringpath, "foo", true),
		NewValOpTblEntry("", testleaflistuserpath, "foo", true),
	}

	srv, sess := TstStartup(t, schemaMandFalse, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSetMandFalse,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandFalse,
		DELETE_AND_COMMIT)
	sess.Kill()

}

// Check that a mandatory node with only non-presence container
// ancestors, forces the container to exists with all
// mandatory nodes satisfied
func TestMandatoryInTopNonPresenceContainer(t *testing.T) {
	const schemaMandTrue = `
container testcontainer {
	leaf testleaf {
		type string;
		mandatory true;
	}
	leaf teststring {
		type string;
		mandatory true;
	}
	leaf testboolean {
		type boolean;
		mandatory false;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
		min-elements 1;
	}
	leaf-list testleaflistuser {
		type string;
		min-elements 1;
	}
	list listwithmand {
		key nodetag;
		leaf nodetag {
			type string;
		}
		leaf mandlistnode {
			description "The scope of this mandatory
			leaf is limited to the parent list \"listwithmand\"";
			type string;
			mandatory true;
		}
	}
}
`
	// Ensure that initial commit will fail unless all mandatory
	// nodes are present in the config.
	mandatoryNodesConfig := testutils.Root(
		testutils.Cont("testcontainer",
			testutils.Leaf("testleaf", "foo"),
			testutils.Leaf("testboolean", "true"),
			testutils.Leaf("teststring", "foo"),
			testutils.LeafList("testleaflistuser",
				testutils.LeafListEntry("foo")),
			testutils.List("testlist",
				testutils.ListEntry("bar"))))

	// Check that a non-mandatory node can be deleted if all mandatory
	// leafs are still present.
	// Delete will fail in the absence of any mandatory leafs.
	// Being a top level, non-presence container (parent is root),
	// if any mandatory nodes are missing, commit will fail.
	tblDeleteMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Verify delete of non-mandatory node with mandatory siblings",
			testbooleanpath, "true", true),
		NewValOpTblEntry("Delete of mandatory node is rejected", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", false),
	}

	srv, sess := TstStartup(t, schemaMandTrue, mandatoryNodesConfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandTrue,
		DELETE_AND_COMMIT)
	sess.Kill()
}

// Check that a mandatory node with a presence container
// as an ancestor, must exist whenever the presence container
// exists.
func TestMandatoryInPresenceContainer(t *testing.T) {
	const schemaMandTrue = `
container testcontainer {
	presence "To limit scope of the mandatory";
	leaf testleaf {
		type string;
		mandatory true;
	}
	leaf teststring {
		type string;
		mandatory true;
	}
	leaf testboolean {
		type boolean;
		mandatory false;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
		min-elements 1;
	}
	leaf-list testleaflistuser {
		type string;
		min-elements 1;
	}
}
`

	// Ensure that commit will fail unless all mandatory
	// nodes are present in the config
	tblSetMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Validate commit fails with missing mandatory nodes", testbooleanpath, "true", false),
		NewValOpTblEntry("", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("All mandatory nodes now satisfied", testlistpath, "bar", true),
	}

	// Check that a non-mandatory node can be deleted if all mandatory
	// leafs are still present.
	// Delete will fail in the absence of any
	// mandatory leafs until ALL nodes AND the presence container
	// are deleted.
	tblDeleteMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Verify delete of non-mandatrory node with mandatory children", testbooleanpath, "true", true),
		NewValOpTblEntry("", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", false),
		NewValOpTblEntry("", testcontainerpath, "", true),
	}

	srv, sess := TstStartup(t, schemaMandTrue, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSetMandTrue,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandTrue,
		DELETE_AND_COMMIT)
	sess.Kill()
}

// Check that mandatory nodes in a child, non-presence container,
// have to be present before commit of anything in it's parent
// presence container.
func TestMandatoryInNonPresenceContainer(t *testing.T) {
	const schemaMandTrue = `
container testcontainer {
	presence "To limit scope of the mandatory";
	leaf testleaf {
		type string;
		mandatory true;
	}
	leaf teststring {
		type string;
		mandatory true;
	}
	leaf testboolean {
		type boolean;
		mandatory false;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
		min-elements 1;
	}
	leaf-list testleaflistuser {
		type string;
		min-elements 1;
	}
	container subcontainer {
		leaf subtrue {
			type string;
			mandatory true;
		}
		leaf subfalse {
			type string;
			mandatory false;
		}
	}
}
`

	// Ensure that commit will fail unless all mandatory
	// nodes are present in the config
	tblSetMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Validate commit fails with missing mandatory nodes", testbooleanpath, "true", false),
		NewValOpTblEntry("", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", false),
		NewValOpTblEntry("All mandatory constraints satisfied", subtruepath, "true", true),
		NewValOpTblEntry("", subfalsepath, "false", true),
	}

	// Check that a non-mandatory node can be deleted if all mandatory
	// leafs are still present.
	// Delete will fail in the absence of any mandatory leafs until
	// the parent non-presence container is deleted
	tblDeleteMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Verify delete of non-mandatrory node with mandatory children", testbooleanpath, "true", true),
		NewValOpTblEntry("", subfalsepath, "false", true),
		NewValOpTblEntry("Fails the mandatory constraint", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", false),
		NewValOpTblEntry("", subtruepath, "true", false),
		NewValOpTblEntry("Verify that presence container delete is allowed", testcontainerpath, "", true),
	}

	srv, sess := TstStartup(t, schemaMandTrue, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSetMandTrue,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandTrue,
		DELETE_AND_COMMIT)
	sess.Kill()
}

// Check that mandatory nodes in a child, presence container,
// do not have to be present before commit of anything in it's parent
// presence container.
func TestMandatoryInPresenceSubContainer(t *testing.T) {
	const schemaMandTrue = `
container testcontainer {
	presence "To limit scope of the mandatory";
	leaf testleaf {
		type string;
		mandatory true;
	}
	leaf teststring {
		type string;
		mandatory true;
	}
	leaf testboolean {
		type boolean;
		mandatory false;
	}
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
		min-elements 1;
	}
	leaf-list testleaflistuser {
		type string;
		min-elements 1;
	}
	container subcontainer {
		presence "To limit scope of mandatory";
		leaf subtrue {
			type string;
			mandatory true;
		}
		leaf subfalse {
			type string;
			mandatory false;
		}
	}
}
`

	tblSetMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Validate commit fails with missing mandatory nodes", testbooleanpath, "true", false),
		NewValOpTblEntry("", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", true),
		NewValOpTblEntry("", subfalsepath, "false", false),
		NewValOpTblEntry("All mandatory constraints satisfied", subtruepath, "true", true),
	}

	// Check that a non-mandatory node can be deleted if all mandatory
	// leafs are still present.
	// Delete will fail in the absence of any
	// mandatory leafs until ALL container nodes are deleted
	tblDeleteMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Verify delete of non-mandatrory node with mandatory children", testbooleanpath, "true", true),
		NewValOpTblEntry("", subtruepath, "true", false),
		NewValOpTblEntry("", subfalsepath, "false", false),
		NewValOpTblEntry("Verify subcontainer can be deleted", subcontainerpath, "", true),
		NewValOpTblEntry("Fails the mandatory constraint", testleafpath, "foo", false),
		NewValOpTblEntry("", teststringpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", false),
		NewValOpTblEntry("Verify that presence container delete is allowed", testcontainerpath, "", true),
	}

	srv, sess := TstStartup(t, schemaMandTrue, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSetMandTrue,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandTrue,
		DELETE_AND_COMMIT)
	sess.Kill()
}

// Check that a mandatory node multiple levels down in a
// hierarchy of non-presence containers, prevents commit
// unless the mandatory node is present
func TestMandatoryInQuadNestedContainer(t *testing.T) {
	const schemaMandTrue = `
container testcontainer {
	presence "To limit scope of the mandatory";
	leaf testleaf {
		type string;
	}
	container subcontainer {
		leaf subfalse {
			type string;
		}
		container level3container {
			leaf level3leaf {
				type string;
			}
			container level4container {
				leaf level4leaf {
					type string;
					mandatory true;
				}
			}
		}
	}
}
`

	// Ensure that commit will fail unless the mandatory
	// node, four levels down, is present
	tblSetMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Reject commit due to mandatory contraint", testleafpath, "foo", false),
		NewValOpTblEntry("Reject commit due to mandatory constraint", subfalsepath, "false", false),
		NewValOpTblEntry("Reject commit due to mandatory constraint", level3leafpath, "level3", false),
		NewValOpTblEntry("Accept commit, mandatory constraints now ssatisfied", level4leafpath, "level4", true),
	}

	// Check that non-mandatory nodes in the hierarchy can be deleted
	// so long as the mandatory
	tblDeleteMandTrue := []ValidateOpTbl{
		NewValOpTblEntry("Commit allowed, mandatory still satisfied", subfalsepath, "false", true),
		NewValOpTblEntry("", level3leafpath, "level3", true),
		NewValOpTblEntry("", testleafpath, "foo", true),
		NewValOpTblEntry("Reject commit, mandatory constraint not satisfied", level4leafpath, "level4", false),
		NewValOpTblEntry("Commit allowed, presence container has been deleted", testcontainerpath, "", true),
	}

	// Do above tests in different order
	tblSetMandTrueReversed := []ValidateOpTbl{
		NewValOpTblEntry("Commit success, mandatory constraint satisfied", level4leafpath, "level4", true),
		NewValOpTblEntry("", level3leafpath, "level3", true),
		NewValOpTblEntry("", subfalsepath, "false", true),
		NewValOpTblEntry("", testleafpath, "foo", true),
	}

	tblDeleteMandTrueReversed := []ValidateOpTbl{
		NewValOpTblEntry("Reject Commit, mandatory contraint enforced", level4leafpath, "level4", false),
		NewValOpTblEntry("", testleafpath, "foo", false),
		NewValOpTblEntry("", level3leafpath, "level3", false),
		NewValOpTblEntry("", subfalsepath, "false", false),
		NewValOpTblEntry("Commit success, parent presence container deleted", testcontainerpath, "", true),
	}

	srv, sess := TstStartup(t, schemaMandTrue, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSetMandTrue,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandTrue,
		DELETE_AND_COMMIT)
	sess.Kill()

	srv, sess = TstStartup(t, schemaMandTrue, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSetMandTrueReversed,
		SET_AND_COMMIT)
	ValidateOperationTable(t, sess, srv.Ctx, tblDeleteMandTrueReversed,
		DELETE_AND_COMMIT)
	sess.Kill()
}

/*
 * Validate that min-elements and max-elements constraint is correctly
 * enforced for list and leaf-list yang elements
 */
func TestMinMaxElements(t *testing.T) {

	var testlistunbounded = "testlistunbounded"
	var testlistunboundedpath = pathutil.CopyAppend(testcontainerpath, testlistunbounded)
	const schema = `
container testcontainer {
	presence "A presence container";
	list testlist {
		key nodetag;
		leaf nodetag {
			type string;
		}
		min-elements 2;
		max-elements 3;
	}
	leaf-list testleaflistuser {
		type string;
		ordered-by user;
		min-elements 1;
	}
	list testlistunbounded {
		key nodetag;
		leaf nodetag {
			type string;
		}
		min-elements 0;
		max-elements unbounded;
	}
	leaf testboolean {
		type boolean;
	}
}
`

	const configDelete = `
testcontainer {
	testboolean true
	testlist foo
	testlist bar
	testlist foobar
	testleaflistuser foo
	testleaflistuser bar
	testleaflistuser foobar
	testleaflistuser baz
	testlistunbounded foo
	testlistunbounded bar
	testlistunbounded foobar
}
`

	tblSet := []ValidateOpTbl{
		NewValOpTblEntry("", testbooleanpath, "true", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "foo", false),
		NewValOpTblEntry("", testlistpath, "bar", true), // min-elements constraints now satisfied
		NewValOpTblEntry("", testleaflistuserpath, "bar", true),
		NewValOpTblEntry("", testleaflistuserpath, "foobar", true),
		NewValOpTblEntry("", testleaflistuserpath, "baz", true),
		NewValOpTblEntry("", testlistpath, "foobar", true),
		NewValOpTblEntry("", testlistunboundedpath, "foo", true),
		NewValOpTblEntry("", testlistunboundedpath, "bar", true),
		NewValOpTblEntry("", testlistunboundedpath, "foobar", true),
		NewValOpTblEntry("", testlistpath, "baz", false), // max-elements exceeded; fail
	}

	tblDelete := []ValidateOpTbl{
		NewValOpTblEntry("", testlistpath, "foo", true),
		NewValOpTblEntry("", testleaflistuserpath, "bar", true),
		NewValOpTblEntry("", testleaflistuserpath, "baz", true),
		NewValOpTblEntry("", testlistunboundedpath, "foo", true),
		NewValOpTblEntry("", testlistunboundedpath, "bar", true),
		NewValOpTblEntry("", testlistunboundedpath, "foobar", true),
		NewValOpTblEntry("", testlistpath, "bar", false), // min-elements constraint now prevents commit
		NewValOpTblEntry("", testlistpath, "foobar", false),
		NewValOpTblEntry("", testleaflistuserpath, "foo", false),
		NewValOpTblEntry("", testleaflistuserpath, "foobar", false),
		NewValOpTblEntry("", testbooleanpath, "true", false),
		NewValOpTblEntry("", testcontainerpath, "", true), // everything now gone, commit succeeds
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSet, SET_AND_COMMIT)
	sess.Kill()
	srv, sess = TstStartup(t, schema, configDelete)
	ValidateOperationTable(t, sess, srv.Ctx, tblDelete, DELETE_AND_COMMIT)
	sess.Kill()
}

/*
 * Validate that min-elements and max-elements constraint is correctly
 * enforced for list and leaf-list yang elements
 */
func TestChoice(t *testing.T) {

	var testone = "testone"
	var testonepath = pathutil.CopyAppend(testcontainerpath, testone)
	var testtwo = "testtwo"
	var testtwopath = pathutil.CopyAppend(testcontainerpath, testtwo)
	var notamandatorypath = pathutil.CopyAppend(testcontainerpath, "notamandatory")
	var isachoicemand = pathutil.CopyAppend(testcontainerpath, "isachoicemand")
	var deeponepath = pathutil.CopyAppend(testcontainerpath, "deep-one")
	var deeptwopath = pathutil.CopyAppend(testcontainerpath, "deep-two")
	var deepthreepath = pathutil.CopyAppend(deeptwopath, "deep-three")
	var deepfourpath = pathutil.CopyAppend(deepthreepath, "deep-four")
	var greekpath = pathutil.CopyAppend(testcontainerpath, "greek")
	var gammapath = pathutil.CopyAppend(greekpath, "gamma")
	var deltapath = pathutil.CopyAppend(greekpath, "delta")
	const schema = `
container testcontainer {
	presence "A presence container";

	leaf defdef {
		type string;
		default "non-choice-default";
	}
	choice testchoice {
		mandatory true;

		leaf testone {
			type string;
		}

		leaf testtwo {
			type string;
		}
	}
	choice book {
		case abook {
			leaf notamandatory {
				type string;
			}
			leaf isachoicemand {
				mandatory true;
				type string;
			}
		}
	}
	choice deep {
		case deep {
			leaf deep-one {
				type string;
			}
			container deep-two {
				container deep-three {
					leaf deep-four {
						mandatory true;
						type string;
					}
				}
			}
			container deep-five {
				presence "shield below mandatory";
				container deep-six {
					leaf deep-seven {
						mandatory true;
						type string;
					}
				}
			}
		}
		container implicitcase {
			leaf mandinimplcase {
				mandatory true;
				type string;
			}
		}
	}
	container blah {
		choice blah {
			container blah {
				leaf blah {
					mandatory true;
					type string;
				}
			}
		}
	}
	container greek {
		choice alpha {
			case beta {
				choice iota {
					leaf gamma {
						type string;
					}
					leaf delta {
						type string;
						mandatory true;
					}
				}
			}
		}
	}
}
`

	const configDelete = `
testcontainer {
	testone foo
}
`

	tblSet := []ValidateOpTbl{
		NewValOpTblEntry("", testcontainerpath, "", false),
		NewValOpTblEntry("", testtwopath, "foo", true),
		NewValOpTblEntry("", gammapath, "foo", true),
		NewValOpTblEntry("", deltapath, "foo", true),
		NewValOpTblEntry("", notamandatorypath, "foo", false),
		NewValOpTblEntry("", isachoicemand, "foo", true),
		NewValOpTblEntry("", deeponepath, "foo", false),
		NewValOpTblEntry("", deepfourpath, "foo", true),
	}

	tblDelete := []ValidateOpTbl{
		NewValOpTblEntry("", testonepath, "", false),
		NewValOpTblEntry("", testcontainerpath, "", true), // everything now gone, commit succeeds
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSet, SET_AND_COMMIT)
	sess.Kill()
	srv, sess = TstStartup(t, schema, configDelete)
	ValidateOperationTable(t, sess, srv.Ctx, tblDelete, DELETE_AND_COMMIT)
	sess.Kill()
}

func TestChoiceTwo(t *testing.T) {

	var testone = "test-one"
	var testonepath = []string{testone}
	var isachoicemandpath = pathutil.CopyAppend(testonepath, "isachoicemand")
	var notamandatorypath = pathutil.CopyAppend(testonepath, "notamandatory")
	var timepath = pathutil.CopyAppend(testonepath, "time")
	var newpath = pathutil.CopyAppend(testonepath, "new")
	var choochoopath = pathutil.CopyAppend(testonepath, "choochoo")
	var digitalpath = pathutil.CopyAppend(testonepath, "digital")
	var powerpath = pathutil.CopyAppend(testonepath, "power")
	var batterypath = pathutil.CopyAppend(powerpath, "battery")
	var voltagepath = pathutil.CopyAppend(powerpath, "voltage")
	const schema = `
		container test-one {
		presence "";
		choice book {
			mandatory true;
			container spiderman {
				leaf spidername {
					type string;
					default spring;
				}
			}
			case blank {
				leaf choochoo {
					type string;
					default "abc";
				}
				leaf ardvark {
					type string;
					default "xyz";
				}
				choice firmness {
					default softness;

					container softness {
						leaf value {
							type string;
							default grob;
						}
						leaf scrib {
							type string;
						}
						container blub {
							leaf groo {
								type string;
								default frop;
							}
						}
					}
					container hardness {
						leaf value {
							type string;
							default strib;
						}
						leaf scrib {
							type string;
						}
					}
				}
			}
			case abook {
				leaf notamandatory {
					type string;
				}
				leaf isachoicemand {
					mandatory true;
					type string;
				}
			}
			case clock {
				leaf ticktock {
					type string;
					default "yes";
				}
				leaf time {
					mandatory true;
					type string;
				}

				leaf new {
					type string;
				}

				choice type {
					mandatory true;

					container digital {
						presence "";
						leaf led-colour {
							type string;
							default red;
						}
						container blue {
							container red {
								leaf branch {
									type uint8;
									default 4;
								}
							}
						}
					}
					container style {
						leaf colour {
							type string;
							default black;
						}
						leaf face {
							type string;
							default analogue;
						}
					}
					container power {
						leaf voltage {
							mandatory true;
							type uint32;
						}
						choice source {
							mandatory true;
							leaf battery {
								type empty;
							}
							leaf mains {
								type empty;
							}
							leaf adefault {
								type string;
								default "defaultvaule";
							}
						}
					}
				}
			}
		}

	}
`

	const configDelete = `
	test-one {
		isachoicemand foo
		notamandatory bar
	}
	`

	tblSet := []ValidateOpTbl{
		NewValOpTblEntry("", testonepath, "", false),
		NewValOpTblEntry("", choochoopath, "chewchew", true),
		NewValOpTblEntry("", notamandatorypath, "foo", false),
		NewValOpTblEntry("", isachoicemandpath, "foo", true),
		NewValOpTblEntry("", timepath, "16:44", false),
		NewValOpTblEntry("", newpath, "16:44", false),
		NewValOpTblEntry("", digitalpath, "", true),
		NewValOpTblEntry("", voltagepath, "212", false),
		NewValOpTblEntry("", batterypath, "", true),
		NewValOpTblEntry("", notamandatorypath, "foo", false),
		NewValOpTblEntry("", isachoicemandpath, "foo", true),
	}

	tblDelete := []ValidateOpTbl{
		NewValOpTblEntry("", isachoicemandpath, "", false),
		NewValOpTblEntry("", testonepath, "", true), // everything now gone, commit succeeds
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateOperationTable(t, sess, srv.Ctx, tblSet, SET_AND_COMMIT)
	sess.Kill()
	srv, sess = TstStartup(t, schema, configDelete)
	ValidateOperationTable(t, sess, srv.Ctx, tblDelete, DELETE_AND_COMMIT)
	sess.Kill()
}

// TODO: (pac) Test anyxml once implemented.
