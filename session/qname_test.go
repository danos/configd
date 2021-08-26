// Copyright (c) 2017-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests on XPATH 'QName' (qualified name) handling.
// These names may be prefixed (foo:bar / foo:*) or unprefixed (bar / *).
//
// Testing the expansion of unprefixed names, and mapping of prefixed
// names (prefixes only unique within a module) to module names (globally
// unique) is done in this file
//
// These tests on prefixed names (foo:bar in a path) all use a pair of
// schemas defined in the testdata/prefixedNames directory.

package session_test

import (
	"strings"
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// Basic tests
// -----------
//
// 'local' = within current module
// 'remote' = in different module
//
// NB: all import statements add a suffix to any prefix they define in an
//     import statement that is unique to that module.  This ensures that any
//     lookup of that prefix is matching from the correct module.  (If we
//     had 'remote' defined in ALL modules, then we aren't testing where it
//     gets looked up).
//
// - 'when' refers to unprefixed local node
// - 'when' refers to prefixed local node
// - 'when' refers to prefixed remote node
// - 'when' refers to remote path but not all elements are prefixed (CMT FAIL)
// - 'when' refers to unknown module name (COMPILE FAIL)

// More complex tests
// ------------------
//
// These use the 'grouping', 'uses' and 'augment' statements to verify that
// groupings with unprefixed when statement path elements pick up the correct
// namespace, ie the destination (location of uses) statement.  Note that
// if the uses is inside an augment, then it is the namespace where the
// augment is defined, not the namespace being augmented, that is used for
// unprefixed elements in the grouping's when statement, and of course for
// elements in the augment itself.
//
// Note that as we have already established that prefixed paths work, we
// don't need to run the following tests for the prefixed case; they are
// only run for the unprefixed case as that's the interesting one where
// module can change.
//
// Tests cover:
//
// 1) augment remote module with top-level 'when' with unprefixed path
//
// 2) augment remote module with unprefixed path in leaf when
//
// 3) imported grouping with unprefixed path in leaf when
//
// 4) imported grouping within grouping, each with unprefixed path in leaf
//    when, to verify nesting doesn't mess things up.
//
// 5) multiple use of single grouping, in different namespaces, to ensure
//    that grouping is cloned before 'when' statement is parsed.
//
// 6) grouping inside 'augment', both in different namespaces to namespace
//    being augmented.
//
// 7) Wildcard outside local module should expand to '*:*' not '<local>:*'

// BASIC TESTS

// 'when' contains local nodes, no prefix.
const implicitLocalSchema = `
	container localCont {
	    leaf refLeaf {
		    type string;
        }
	    leaf noPfxLeaf {
		    type string;
			must "../refLeaf = 'hello'";
        }
    }
`

func TestImplicitLocalPrefix(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set local refLeaf value so must checks fail.",
			"localCont/refLeaf/halloo", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/noPfxLeaf/byebye", SetPass),
	}

	testA_expOut := errtest.NewMustDefaultError(t,
		"/localCont/noPfxLeaf/byebye",
		"../refLeaf = 'hello'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set local refLeaf value so must checks pass.",
			"localCont/refLeaf/hello", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/noPfxLeaf/byebye", SetPass),
	}

	testB_expCfg := `localCont {
	noPfxLeaf byebye
	refLeaf hello
}
`

	implicitLocalTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				SchemaSnippet: implicitLocalSchema,
			},
		},
		emptyconfig, implicitLocalTests)
}

// 'when' contains prefixed local node name
const explicitLocalSchema = `
	container localCont {
	    leaf refLeaf {
		    type string;
        }
	    leaf localPfxLeaf {
		    type string;
			must "../local:refLeaf = 'hello'";
        }
    }
`

func TestExplicitLocalPrefix(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set local refLeaf value so must checks fail.",
			"localCont/refLeaf/halloo", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/localPfxLeaf/byebye", SetPass),
	}

	testA_expOut := errtest.NewMustDefaultError(t,
		"/localCont/localPfxLeaf/byebye",
		"../local:refLeaf = 'hello'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set local refLeaf value so must checks pass.",
			"localCont/refLeaf/hello", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/localPfxLeaf/byebye", SetPass),
	}

	testB_expCfg := `localCont {
	localPfxLeaf byebye
	refLeaf hello
}
`

	explicitLocalTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				SchemaSnippet: explicitLocalSchema,
			},
		},
		emptyconfig, explicitLocalTests)
}

// 'when' contains prefixed remote module name
const remotePrefixTestLocalSchema = `
	container localCont {
	    leaf remotePfxLeaf {
		    type string;
			must "/remoteLcl:remoteCont/remoteLcl:remoteLeaf = 'hello'";
        }
    }
`
const remotePrefixTestRemoteSchema = `
	container remoteCont {
		leaf remoteLeaf {
			type string;
		}
	}
`

func TestRemotePrefix(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remote refLeaf value so must checks fail.",
			"remoteCont/remoteLeaf/halloo", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/remotePfxLeaf/byebye", SetPass),
	}

	testA_expOut := errtest.NewMustDefaultError(t,
		"/localCont/remotePfxLeaf/byebye",
		"/remoteLcl:remoteCont/remoteLcl:remoteLeaf = 'hello'").
		RawErrorStrings()

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remote refLeaf value so must checks pass.",
			"remoteCont/remoteLeaf/hello", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/remotePfxLeaf/byebye", SetPass),
	}

	testB_expCfg := `localCont {
	remotePfxLeaf byebye
}
remoteCont {
	remoteLeaf hello
}
`

	explicitRemoteTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{
					Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-remote", Prefix: "remoteLcl"}},
				SchemaSnippet: remotePrefixTestLocalSchema,
			},
			{
				Name: NameDef{
					Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: remotePrefixTestRemoteSchema,
			},
		},
		emptyconfig, explicitRemoteTests)
}

// Tests that within a path that starts with a prefixed name for a remote
// module, a subsequent unprefixed name picks up the local module prefix
// and so cannot be found even though the unprefixed name exists in the
// remote module.

const noRemotePrefixLocalSchema = `
	container localCont {
	    leaf unknownPfxLeaf {
		    type string;
			must "/remoteLcl:remoteCont/remoteLeaf = 'hello'";
        }
	}
`
const noRemotePrefixRemoteSchema = `
	container remoteCont {
		leaf remoteLeaf {
			type string;
		}
    }
`

func TestNoRemotePrefixName(t *testing.T) {
	// Initially remoteLeaf doesn't exist
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Try unprefixed reference.",
			"localCont/unknownPfxLeaf/byebye", SetPass),
	}

	test_expOut := errtest.NewMustDefaultError(t,
		"/localCont/unknownPfxLeaf/byebye",
		"/remoteLcl:remoteCont/remoteLeaf = 'hello'").
		RawErrorStrings()

	// Should fail still as remoteLeaf in must should have picked up
	// local module name not remote
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Set remote refLeaf - should fail.",
			"remoteCont/remoteLeaf/hello", SetPass),
		createValOpTbl("Try unprefixed reference.",
			"localCont/unknownPfxLeaf/byebye", SetPass),
	}

	noRemotePrefixTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitFail, emptyconfig,
			test_expOut),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-remote", Prefix: "remoteLcl"}},
				SchemaSnippet: noRemotePrefixLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: noRemotePrefixRemoteSchema,
			},
		},
		emptyconfig, noRemotePrefixTests)
}

const unknownSchema = `
container testCont {
	leaf testLeaf {
		type string;
		must "unknown:foo";
	}
}`

// Check we get compilation failure for an unknown prefix.
func TestUnknownPrefix(t *testing.T) {
	var err error
	if _, _, err = ValidateTestSchemaSnippet(t, unknownSchema); err == nil {
		t.Fatalf("Compilation of 'unknown' prefix should have failed.")
	}

	if !strings.Contains(err.Error(),
		"unknown import unknown") {
		t.Logf("Wrong error:\n")
		t.Logf("Exp: 'leaf testLeaf: unknown import unknown'\n")
		t.Fatalf("Got: '%s'\n", err.Error())
	}
}

// MORE COMPLEX TESTS ...

// From RFC 6020 section 7.12:
//
// The "uses" statement is used to reference a "grouping" definition.
// It takes one argument, which is the name of the grouping.
//
// The effect of a "uses" reference to a grouping is that the nodes
// defined by the grouping are copied into the current schema tree, and
// then updated according to the "refine" and "augment" statements.
//
// The identifiers defined in the grouping are not bound to a namespace
// until the contents of the grouping are added to the schema tree via a
// "uses" statement that does not appear inside a "grouping" statement,
// at which point they are bound to the namespace of the current module.

// 1) Augment remote module with top-level 'when', unprefixed path.
//
// Need to verify that we can only configure noPfxLeaf in remoteCont
// when we've configured localLeaf in localCont, thus proving that
// the 'when' statement has picked up the namespace of the local module
// it's defined in, not the remote module it is augmenting.

const augmentTopLevelWhenLocalSchema = `
	container localCont {
	    leaf localLeaf {
		    type string;
        }
	}

	augment /remoteLcl:remoteCont {
		leaf noPfxLeaf {
			type string;
		}
		when "/localCont/localLeaf";
	}
`
const augmentTopLevelWhenRemoteSchema = `
    container remoteCont {
		leaf remoteLeaf {
			type string;
		}
    }
`

func TestAugmentTopLevelWhen(t *testing.T) {
	// Set to fail by not setting localCont:localLeaf value
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails as localLeaf isn't configured.",
			"remoteCont/noPfxLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewWhenDefaultError(t,
		"/remoteCont",
		"/localCont/localLeaf").
		RawErrorStrings()

	// Set to pass by setting localCont:localLeaf value to something.
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"remoteCont/noPfxLeaf/someValue", SetPass),
		createValOpTbl(".",
			"localCont/localLeaf/anythingWillDo", SetPass),
	}

	testB_expCfg := `localCont {
	localLeaf anythingWillDo
}
remoteCont {
	noPfxLeaf someValue
}
`
	augmentTopLevelWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports:       []NameDef{{Namespace: "prefix-remote", Prefix: "remoteLcl"}},
				SchemaSnippet: augmentTopLevelWhenLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: augmentTopLevelWhenRemoteSchema,
			},
		},
		emptyconfig, augmentTopLevelWhenTests)
}

// 2) Augment remote module with leaf 'when', unprefixed path.
const augmentLeafWhenLocalSchema = `
	container localCont {
	    leaf localLeaf {
		    type string;
        }
	}

	augment /remoteLcl:remoteCont {
		leaf whenLeaf {
			type string;
 		    when "/localCont/localLeaf";
		}
	}
`
const augmentLeafWhenRemoteSchema = `
    container remoteCont {
		leaf remoteLeaf {
			type string;
		}
    }
`

func TestAugmentLeafWhen(t *testing.T) {
	// Set to fail by not setting localCont:localLeaf value
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails as localLeaf isn't configured.",
			"remoteCont/whenLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewWhenDefaultError(t,
		"/remoteCont/whenLeaf/someValue",
		"/localCont/localLeaf").
		RawErrorStrings()

	// Set to pass by setting localCont:localLeaf value to something.
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"remoteCont/whenLeaf/someValue", SetPass),
		createValOpTbl(".",
			"localCont/localLeaf/anythingWillDo", SetPass),
	}

	testB_expCfg := `localCont {
	localLeaf anythingWillDo
}
remoteCont {
	whenLeaf someValue
}
`
	augmentLeafWhenTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports:       []NameDef{{Namespace: "prefix-remote", Prefix: "remoteLcl"}},
				SchemaSnippet: augmentLeafWhenLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: augmentLeafWhenRemoteSchema,
			},
		},
		emptyconfig, augmentLeafWhenTests)
}

// 3) Imported grouping with unprefixed leaf when
//
// Both local and ref1 contain a commonLeaf, albeit in different containers,
// but with same relative path.
//
//	container <name>Cont {
//	    leaf commonLeaf {
//		    type string;
//        }
//	}
//
const importedGroupingLocalSchema = `
	container localCont {
	    leaf localLeaf {
		    type string;
        }
        uses refLcl:refGrp;
    }`

const importedGroupingRefSchema = `
	grouping refGrp {
		leaf refGrpLeaf {
			type string;
			when "../localLeaf";
		}
	}`

func TestImportedGrouping(t *testing.T) {
	// Initially we try to configure refGrpLeaf but it fails as
	// localLeaf is not set ('when' failure)
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails on missing localLeaf.",
			"localCont/refGrpLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewWhenDefaultError(t,
		"/localCont/refGrpLeaf/someValue",
		"../localLeaf").
		RawErrorStrings()

	// Now configure localLeaf and 'when' should pass
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"localCont/localLeaf/anythingWillDo", SetPass),
		createValOpTbl(".",
			"localCont/refGrpLeaf/someValue", SetPass),
	}

	testB_expCfg := `localCont {
	localLeaf anythingWillDo
	refGrpLeaf someValue
}
`

	importedGroupingTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{
					{Namespace: "prefix-ref", Prefix: "refLcl"}},
				SchemaSnippet: importedGroupingLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-ref", Prefix: "ref"},
				SchemaSnippet: importedGroupingRefSchema,
			},
		},
		emptyconfig, importedGroupingTests)
}

// 4) Grouping inside grouping, each from different module, chained inclusion.
//    Make sure ref2 when statement acts on localCont/localLeaf
//
const groupingInGroupingLocalSchema = `
	container localCont {
	    leaf localLeaf {
		    type string;
        }
        uses refLcl:refGrp;
    }`

const groupingInGroupingRefSchema = `
	grouping refGrp {
		leaf refGrpLeaf {
			type string;
		}
        uses ref2Lcl:ref2Grp;
	}`

const groupingInGroupingRef2Schema = `
	grouping ref2Grp {
		leaf ref2GrpLeaf {
			type string;
			when "../localLeaf";
		}
	}`

func TestGroupingInGrouping(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails on missing ref1Leaf.",
			"localCont/ref2GrpLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewWhenDefaultError(t,
		"/localCont/ref2GrpLeaf/someValue",
		"../localLeaf").
		RawErrorStrings()

	// Now configure commonLeaf and 'when' should pass
	// local module name not remote
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"localCont/localLeaf/anythingWillDo", SetPass),
		createValOpTbl(".",
			"localCont/ref2GrpLeaf/someValue", SetPass),
	}

	testB_expCfg := `localCont {
	localLeaf anythingWillDo
	ref2GrpLeaf someValue
}
`
	groupingInGroupingTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports:       []NameDef{{Namespace: "prefix-ref", Prefix: "refLcl"}},
				SchemaSnippet: groupingInGroupingLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-ref", Prefix: "ref"},
				Imports:       []NameDef{{Namespace: "prefix-ref2", Prefix: "ref2Lcl"}},
				SchemaSnippet: groupingInGroupingRefSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-ref2", Prefix: "ref2"},
				SchemaSnippet: groupingInGroupingRef2Schema,
			},
		},
		emptyconfig, groupingInGroupingTests)
}

// 5) Multiple use of one grouping.  Check that use of the same grouping
//    in 2 different modules picks up the correct commonLeaf instance in
//    each case.
//
const multipleUseGroupingLocalSchema = `
	container localCont {
	    leaf commonLeaf {
		    type string;
        }
        uses refLcl:refGrp;
    }`

const multipleUseGroupingRemoteSchema = `
	container remoteCont {
	    leaf commonLeaf {
		    type string;
        }
        uses refRem:refGrp;
    }`

const multipleUseGroupingRefSchema = `
	grouping refGrp {
		leaf refGrpLeaf {
			type string;
			when "../commonLeaf";
		}
	}`

func TestMultipleUseOfGrouping(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails on missing ref1Leaf.",
			"localCont/refGrpLeaf/someValue", SetPass),
		createValOpTbl("'when' fails on missing ref1Leaf.",
			"remoteCont/refGrpLeaf/someValue", SetPass),
	}

	testA_expOut :=
		errtest.NewWhenDefaultError(t,
			"/localCont/refGrpLeaf/someValue",
			"../commonLeaf").
			RawErrorStrings()
	testA_expOut = append(testA_expOut,
		errtest.NewWhenDefaultError(t,
			"/remoteCont/refGrpLeaf/someValue",
			"../commonLeaf").
			RawErrorStrings()...)

	// Now configure remoteCont/commonLeaf.  Will still get error for
	// localCont
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"remoteCont/commonLeaf/anythingWillDo", SetPass),
		createValOpTbl(".",
			"localCont/refGrpLeaf/someValue", SetPass),
		createValOpTbl(".",
			"remoteCont/refGrpLeaf/someValue", SetPass),
	}

	testB_expOut := errtest.NewWhenDefaultError(t,
		"/localCont/refGrpLeaf/someValue",
		"../commonLeaf").
		RawErrorStrings()

	// Now configure localCont/commonLeaf.  Will still get error for
	// remoteCont
	testC_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"localCont/commonLeaf/anythingWillDo", SetPass),
		createValOpTbl(".",
			"localCont/refGrpLeaf/someValue", SetPass),
		createValOpTbl(".",
			"remoteCont/refGrpLeaf/someValue", SetPass),
	}

	testC_expOut := errtest.NewWhenDefaultError(t,
		"/remoteCont/refGrpLeaf/someValue",
		"../commonLeaf").
		RawErrorStrings()

	// Now configure local and remote commonLeaf.  Should now pass.
	testD_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails on missing ref1Leaf.",
			"localCont/refGrpLeaf/someValue", SetPass),
		createValOpTbl(".",
			"remoteCont/refGrpLeaf/someValue", SetPass),
		createValOpTbl("Remote common leaf so remote when passes",
			"remoteCont/commonLeaf/anythingWillDo", SetPass),
		createValOpTbl("Local common leaf so local when passes",
			"localCont/commonLeaf/anythingWillDo", SetPass),
	}

	testD_expCfg := `localCont {
	commonLeaf anythingWillDo
	refGrpLeaf someValue
}
remoteCont {
	commonLeaf anythingWillDo
	refGrpLeaf someValue
}
`
	multipleUseGroupingTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitFail, emptyconfig,
			testB_expOut),
		newXpathTestEntry(testC_setTbl, nil, CommitFail, emptyconfig,
			testC_expOut),
		newXpathTestEntry(testD_setTbl, nil, CommitPass, testD_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports:       []NameDef{{Namespace: "prefix-ref", Prefix: "refLcl"}},
				SchemaSnippet: multipleUseGroupingLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				Imports:       []NameDef{{Namespace: "prefix-ref", Prefix: "refRem"}},
				SchemaSnippet: multipleUseGroupingRemoteSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-ref", Prefix: "ref"},
				SchemaSnippet: multipleUseGroupingRefSchema,
			},
		},
		emptyconfig, multipleUseGroupingTests)
}

// 6)  Grouping inside augment used in third module.  Grouping inherits
//     namespace of local module where it is used in the augment, not the
//     remote namespace where the augment statement is adding to.
const groupingInAugmentLocalSchema = `
	container localCont {
	    leaf commonLeaf {
		    type string;
        }
    }
    augment /remoteLcl:remoteCont {
	    uses refLcl:refGrp;
    }`

const groupingInAugmentRemoteSchema = `
	container remoteCont {
	    leaf commonLeaf {
		    type string;
        }
    }`

const groupingInAugmentRefSchema = `
	grouping refGrp {
		leaf refGrpLeaf {
			type string;
			when "/localCont/commonLeaf";
		}
	}`

func TestGroupingInAugment(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("'when' fails on missing local:commonLeaf.",
			"remoteCont/refGrpLeaf/someValue", SetPass),
	}

	testA_expOut := errtest.NewWhenDefaultError(t,
		"/remoteCont/refGrpLeaf/someValue",
		"/localCont/commonLeaf").
		RawErrorStrings()

	// Now configure commonLeaf and 'when' should pass
	// local module name not remote
	testB_setTbl := []ValidateOpTbl{
		createValOpTbl(".",
			"localCont/commonLeaf/anythingWillDo", SetPass),
		createValOpTbl(".",
			"remoteCont/refGrpLeaf/someValue", SetPass),
	}

	testB_expCfg := `localCont {
	commonLeaf anythingWillDo
}
remoteCont {
	refGrpLeaf someValue
}
`
	groupingInAugmentTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, emptyconfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, testB_expCfg,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports: []NameDef{{Namespace: "prefix-ref", Prefix: "refLcl"},
					{Namespace: "prefix-remote", Prefix: "remoteLcl"}},
				SchemaSnippet: groupingInAugmentLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: groupingInAugmentRemoteSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-ref", Prefix: "ref"},
				SchemaSnippet: groupingInAugmentRefSchema,
			},
		},
		emptyconfig, groupingInAugmentTests)
}

// 7) Wildcard outside local module.  Check global wildcard, and prefixed
//    one as well.
//
const wildcardLocalSchema = `
	container localCont {
	    leaf existingIntfName {
		    type string;
            must "current() = /remoteLcl:interfaces/*/remoteLcl:name";
            must "current() = /remoteLcl:interfaces/remoteLcl:*/remoteLcl:name";
        }
    }`

const wildcardRemoteSchema = `
	container interfaces {
        list dataplane {
            key "name";
            leaf "name" {
                type string;
            }
        }
        list serial {
            key "name";
            leaf "name" {
                type string;
            }
        }
    }`

const initWildcardConfig = `
	interfaces {
	dataplane dp0s1
	dataplane dp0s2
	serial s3
}
`
const expWildcardConfig = `interfaces {
	dataplane dp0s1
	dataplane dp0s2
	serial s3
}
localCont {
	existingIntfName dp0s2
}
`

func TestRemoteWildcard(t *testing.T) {
	testA_setTbl := []ValidateOpTbl{
		createValOpTbl("Name doesn't match existing interface name.",
			"localCont/existingIntfName/bogus", SetPass),
	}

	testA_expOut :=
		errtest.NewMustDefaultError(t,
			"/localCont/existingIntfName/bogus",
			"current() = /remoteLcl:interfaces/*/remoteLcl:name").
			RawErrorStrings()
	testA_expOut = append(testA_expOut,
		errtest.NewMustDefaultError(t,
			"/localCont/existingIntfName/bogus",
			"current() = /remoteLcl:interfaces/remoteLcl:*/remoteLcl:name").
			RawErrorStrings()...)

	testB_setTbl := []ValidateOpTbl{
		createValOpTbl("Name matches existing interface name.",
			"localCont/existingIntfName/dp0s2", SetPass),
	}

	wildcardRemoteTests := []xpathTestEntry{
		newXpathTestEntry(testA_setTbl, nil, CommitFail, initWildcardConfig,
			testA_expOut),
		newXpathTestEntry(testB_setTbl, nil, CommitPass, expWildcardConfig,
			expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name:          NameDef{Namespace: "prefix-local", Prefix: "local"},
				Imports:       []NameDef{{Namespace: "prefix-remote", Prefix: "remoteLcl"}},
				SchemaSnippet: wildcardLocalSchema,
			},
			{
				Name:          NameDef{Namespace: "prefix-remote", Prefix: "remote"},
				SchemaSnippet: wildcardRemoteSchema,
			},
		},
		initWildcardConfig, wildcardRemoteTests)
}
