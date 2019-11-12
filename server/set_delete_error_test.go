// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// These tests verify the error message format for set and delete errors
// that appear as soon as you hit return, ie you don't need to wait for
// validation.  They are typically formatted differently to validation
// errors, where more than one error can be shown at the same time.
//
// NB: to try to make these tests slightly less brittle, we check for
//     presence of each line in the expected output separately, and
//     ignore newlines.

// TODO: test mgmterror types, and use 'correctly' named New...Error functions
//       in all cases for clarity!

package server_test

import (
	"testing"

	"github.com/danos/config/testutils"
	"github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

const (
	setFailedStr        = "Set failed"
	validationFailedStr = "Value validation failed"
	// For Expand in particular, the way errors are formatted meant we were
	// seeing this at one point, so ensure we test!!
	doubleIsntValidStr = "is not valid is not valid"
)

func genSetTestSchema(input string) []sessiontest.TestSchema {
	return []sessiontest.TestSchema{
		{
			Name: sessiontest.NameDef{
				Namespace: "vyatta-test-validation-v1",
				Prefix:    "validation",
			},
			SchemaSnippet: input,
		},
	}
}

var setTestSchemaSnippet = `
container top {
	leaf intLeaf {
		type uint32;
	}
	leaf emptyLeaf {
		type empty;
	}
	list aList {
		key name;
		leaf name {
			type string;
		}
		leaf notName {
			type string;
		}
	}
	container sub {
		leaf subLeaf {
			type string;
		}
	}
}`

var setIntLeafConfig = testutils.Root(
	testutils.Cont("top",
		testutils.Leaf("intLeaf", "123")))

// Expected: *mgmterror.DataExistsError
//
// --- START ---
// [NL]
//   Configuration path: path to [cmd] is not valid
// [NL]
//   Node exists
// [NL]
// ---  END  ---
//
func TestSetExistingNode(t *testing.T) {

	testPath := "/top/intLeaf/123"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(setIntLeafConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewNodeExistsError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.DataMissingError
//
// --- START ---
// [NL]
//   Configuration path: path to [cmd] is not valid
// [NL]
//   Node does not exist
// [NL]
// ---  END  ---
//
func TestDeleteNonExistentNode(t *testing.T) {

	testPath := "/top/intLeaf/456"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(setIntLeafConfig)

	oc.delete(testPath)

	oc.setExpErr(errtest.NewNodeDoesntExistError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.MissingElementApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [cmd] is not valid
// [NL]
//   Node requires a child
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
func TestSetContainerRequiringChild(t *testing.T) {

	testPath := "/top"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewNodeRequiresChildError(t, testPath)).
		addExtraErrs(setFailedStr).
		setUnexpErrs(validationFailedStr)

	oc.verifyCLIError()
}

func TestSetNodeMissingKey(t *testing.T) {
	t.Skipf("TBD")
	// May not be possible from CLI
}

// TestSetInvalidPathTopLevel - check set error for top-level invalid path
//
// We check at top, 1-down, and 2-or-more-down, as the code for each is
// different.  1 test function for each variant.
//
// Expected: *mgmterror.UnknownElementApplicationError
//
// --- START ---
// [NL]
//   Configuration path: [cmd] is not valid
// [NL]
// ---  END  ---
//
func TestSetInvalidPathTopLevel(t *testing.T) {

	testPath := "/nonExistent"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidNodeError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.UnknownElementApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path-to [cmd] is not valid
// [NL]
// ---  END  ---
//
func TestSetInvalidPathNamedNodeExpected1LevelDown(t *testing.T) {

	testPath := "/top/nonExistent"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidNodeError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.UnknownElementApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path down-to [cmd] is not valid
// [NL]
// ---  END  ---
//
func TestSetInvalidPathNamedNodeExpected2LevelsDown(t *testing.T) {

	testPath := "/top/sub/nonExistent"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidNodeError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.UnknownElementApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path down-to [extra-cmd] is not valid
// [NL]
// ---  END  ---
//
func TestSetPathWithUnexpectedExtraElement(t *testing.T) {

	testPath := "/top/sub/subLeaf/subLeafValue/unexpectedExtra"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidNodeError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.UnknownElementApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to empty leaf [value] is not valid
// [NL]
// ---  END  ---
//
func TestSetValueForEmptyLeaf(t *testing.T) {

	testPath := "/top/emptyLeaf/unexpectedLeafValue"

	oc := newOutputChecker(t).
		setSchema(setTestSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewUnknownElementError(t, testPath)).
		setUnexpErrs(setFailedStr, validationFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [out-of-range-value] is not valid
// [NL]
//   Must have value between <lowest> and <highest>
// [NL]
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//

var rangeSchemaSnippet = `
	container top {
	leaf intWithRange {
		type int16 {
			range 1..1000;
		}
	}
}`

func TestSetOutOfRangeDefaultError(t *testing.T) {

	testPath := "/top/intWithRange/1001"

	oc := newOutputChecker(t).
		setSchema(rangeSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidRangeError(t, testPath, 1, 1000)).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [out-of-range-value] is not valid
// [NL]
//   <Custom error message>
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//

var rangeCustomErrorSchemaSnippet = `
	container top {
	leaf intWithRange {
		type int16 {
			range 1..1000 {
				error-message "Custom range error";
			}
		}
	}
}`

func TestSetOutOfRangeCustomError(t *testing.T) {

	testPath := "/top/intWithRange/1001"

	oc := newOutputChecker(t).
		setSchema(rangeCustomErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidRangeCustomError(
		t, testPath, "Custom range error")).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [invalid-type-value] is not valid
// [NL]
//   <invalid-type-value> is not an <type>
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
var invalidTypeErrorSchemaSnippet = `
	container top {
	leaf intLeaf {
		type uint16;
	}
}`

func TestSetInvalidTypeError(t *testing.T) {

	testPath := "/top/intLeaf/-1"

	oc := newOutputChecker(t).
		setSchema(invalidTypeErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidTypeError(t, testPath, "an uint16")).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [illegal-length-value] is not valid
// [NL]
//   Must have length between <lowest> and <highest>
// [NL]
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
var invalidLengthErrorSchemaSnippet = `
	container top {
	leaf stringLeaf {
		type string {
			length 1..6;
		}
	}
}`

func TestSetInvalidLengthDefaultError(t *testing.T) {

	testPath := "/top/stringLeaf/tooLong"

	oc := newOutputChecker(t).
		setSchema(invalidLengthErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidLengthError(t, testPath, 1, 6)).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [illegal-length-value] is not valid
// [NL]
//   <Custom error message>
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
var invalidLengthCustomErrorSchemaSnippet = `
	container top {
	leaf stringLeaf {
		type string {
			length 1..6 {
				error-message "Custom Length error";
			}
		}
	}
}`

func TestSetInvalidLengthCustomError(t *testing.T) {

	testPath := "/top/stringLeaf/tooLong"

	oc := newOutputChecker(t).
		setSchema(invalidLengthCustomErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidLengthCustomError(
		t, testPath, "Custom Length error")).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [illegal-length-value] is not valid
// [NL]
//   Does not match pattern <pattern>
// [NL]
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
// TODO:
//   - old format was slightly different for 'Does not match' message
//   - multiple patterns may be displayed differently
//
var invalidPatternErrorSchemaSnippet = `
	container top {
	leaf stringLeaf {
		type string {
			pattern '[a-z][A-Z]*';
		}
	}
}`

func TestSetInvalidPatternDefaultError(t *testing.T) {

	testPath := "/top/stringLeaf/tooLong"

	oc := newOutputChecker(t).
		setSchema(invalidPatternErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidPatternError(
		t, testPath, "[a-z][A-Z]*")).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [illegal-pattern-value] is not valid
// [NL]
//   <Custom error message>
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
var invalidPatternCustomErrorSchemaSnippet = `
	container top {
	leaf stringLeaf {
		type string {
			pattern '[a-z][A-Z]*' {
				error-message "Custom Pattern error";
			}
		}
	}
}`

func TestSetInvalidPatternCustomError(t *testing.T) {

	testPath := "/top/stringLeaf/Aa"

	oc := newOutputChecker(t).
		setSchema(invalidPatternCustomErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidPatternCustomError(
		t, testPath, "Custom Pattern error")).
		addExtraErrs(validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

// Expected: *mgmterror.InvalidValueApplicationError
//
// --- START ---
// [NL]
//   Configuration path: path to [illegal-pattern-value] is not valid
// [NL]
//   <Custom error message>
//   Value validation failed
// [NL]
//   Set failed
// [NL]
// ---  END  ---
//
var ipv6PatternErrorSchemaSnippet = `
	typedef ipv4-prefix {
		type string {
			pattern '(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}'
				+  '([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'
				+ '/(([0-9])|([1-2][0-9])|(3[0-2]))';
			configd:pattern-help "<x.x.x.x/x>";
			configd:help "IPv4 Prefix";
			configd:normalize "normalize ipv4";
		}
	}
	typedef ipv6-prefix {
		type string {
			pattern '((:|[0-9a-fA-F]{0,4}):)([0-9a-fA-F]{0,4}:){0,5}'
				+ '((([0-9a-fA-F]{0,4}:)?(:|[0-9a-fA-F]{0,4}))|'
				+ '(((25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])\.){3}'
				+ '(25[0-5]|2[0-4][0-9]|[01]?[0-9]?[0-9])))'
				+ '(/(([0-9])|([0-9]{2})|(1[0-1][0-9])|(12[0-8])))';
			pattern '(([^:]+:){6}(([^:]+:[^:]+)|(.*\..*)))|'
				+ '((([^:]+:)*[^:]+)?::(([^:]+:)*[^:]+)?)'
				+ '(/.+)';
			configd:pattern-help "<h:h:h:h:h:h:h:h/x>";
			configd:help "IPv6 Prefix";
			configd:normalize "normalize ipv6";
		}
	}
	typedef address {
		type union {
			type ipv4-prefix {
				configd:help "IPv4 Address";
				configd:syntax "valid_address $VAR(@)";
			}
			type ipv6-prefix {
				configd:help "IPv6 Address";
				configd:syntax "valid_address $VAR(@)";
			}
		}
	}
	container top {
	leaf address {
		type address;
	}
}`

func TestSetIpv6PatternError(t *testing.T) {

	testPath := "/top/address/10::1:1:1"

	oc := newOutputChecker(t).
		setSchema(ipv6PatternErrorSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.set(testPath)

	oc.setExpErr(errtest.NewInvalidPatternCustomError(
		t, testPath, "Must have one of the following values:")).
		addExtraErrs("<h:h:h:h:h:h:h:h/x>", validationFailedStr, setFailedStr)

	oc.verifyCLIError()
}

var ambiguousPathSchemaSnippet = `
	container cont1 {
	configd:help "First container";
	leaf path1 {
		type string;
	}
}
container cont2 {
	configd:help "Second container";
	leaf path2 {
		type string;
	}
}`

func TestExpandAmbiguousPathError(t *testing.T) {

	testPath := "/cont"

	oc := newOutputChecker(t).
		setSchema(ambiguousPathSchemaSnippet).
		setInitConfig(emptyConfig)

	oc.expand(testPath)

	oc.setExpErr(errtest.NewPathAmbiguousError(
		t, testPath)).
		addExtraErrs(
			"cont1\tFirst container",
			"cont2\tSecond container")

	oc.verifyCLIError()
}
