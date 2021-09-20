// Copyright (c) 2019-2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
//
// This file tests basic type validation within the context of a running
// session, eg range, pattern, fraction-digits etc.

package session_test

import (
	"testing"

	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/utils/pathutil"
)

func TestValidateSetEmptyLeaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testempty {
		type empty;
	}
}
`
	srv, sess := TstStartup(t, schema, emptyconfig)

	ValidateSetPath(t, sess, srv.Ctx, testemptypath, false)
	path := pathutil.CopyAppend(testemptypath, "foo")
	ValidateSetPath(t, sess, srv.Ctx, path, true)

	sess.Kill()
}

func TestValidateSetBooleanLeaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testboolean {
		type boolean;
	}
}
`
	tbl := []ValidateOpTbl{
		NewValOpTblEntry("Validate set boolean without a value", testbooleanpath, "", SetFail),
		NewValOpTblEntry("Validate set boolean true", testbooleanpath, "true", SetPass),
		NewValOpTblEntry("Validate set boolean false", testbooleanpath, "false", SetPass),
		NewValOpTblEntry("Validate set boolean invalid value", testbooleanpath, "foo", SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

// Need to ensure we properly check the range, including the 'gaps', and
// overshooting the max value as well.
func TestValidateSetDec64Leaf(t *testing.T) {
	const schema = `
	container testcontainer {
		leaf testdec64 {
			type decimal64 {
				fraction-digits 3;
			}
		}
		leaf testdec64range {
			type decimal64 {
				fraction-digits 3;
				range "-50..50 | 51..60 | 70..80";
			}
		    default 42;
		}
	}
	`
	// The way decimal numbers are represented means that some numbers can't
	// actually quite be represented, and we also need to remember that unlike
	// (u)int64, some bits are needed for the exponent, so we can't have the
	// same precision as (u)int64.
	const dec64min = "-9223372036854775.808"
	const dec64min_minus2 = "-9223372036854777.808"
	const dec64max = "+9223372036854775.807"
	const dec64max_plus2 = "+9223372036854777.807"

	const dec64min_dropFractionDigit = "-9223372036854775.80"
	const dec64min_minus2_dropFractionDigit = "-9223372036854777.80"
	const dec64max_dropFractionDigit = "+9223372036854775.80"
	const dec64max_plus2_dropFractionDigit = "+9223372036854777.80"

	var testdec64path = pathutil.CopyAppend(testcontainerpath, "testdec64")
	var testdec64rangepath = pathutil.CopyAppend(
		testcontainerpath, "testdec64range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testdec64path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testdec64path, dec64min_minus2, SetFail),
		NewValOpTblEntry(validatesetminvalue, testdec64path, dec64min, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testdec64path, dec64max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testdec64path, dec64max_plus2, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testdec64rangepath, "-51", SetFail),
		NewValOpTblEntry(validatesetminrange1, testdec64rangepath, "-50", SetPass),
		NewValOpTblEntry("Validate set inner range value", testdec64rangepath, "52.0", SetPass),
		NewValOpTblEntry(validatesetbetweenrange2_3, testdec64rangepath, "65.999", SetFail),
		NewValOpTblEntry(validatesetmaxrange3, testdec64rangepath, "80", SetPass),
		NewValOpTblEntry(validatesetabovemaxrange3, testdec64rangepath, "81", SetFail),

		// Check the case of fewer digits used
		NewValOpTblEntry(validatesettoosmall, testdec64path, dec64min_minus2_dropFractionDigit, SetFail),
		NewValOpTblEntry(validatesettoolarge, testdec64path, dec64max_plus2_dropFractionDigit, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()

	// Same test using fraction-digits 2
	const schemaFd2 = `
	container testcontainer {
		leaf testdec64 {
			type decimal64 {
				fraction-digits 2;
			}
		}
		leaf testdec64range {
			type decimal64 {
				fraction-digits 2;
				range "-50..50 | 51..60 | 70..80";
			}
		    default 42;
		}
	}
	`
	// The way decimal numbers are represented means that some numbers can't
	// actually quite be represented, and we also need to remember that unlike
	// (u)int64, some bits are needed for the exponent, so we can't have the
	// same precision as (u)int64.
	const dec64Fd2min = "-92233720368547758.08"
	const dec64Fd2min_minus2 = "-92233720368547778.08"
	const dec64Fd2max = "+92233720368547758.08"
	const dec64Fd2max_plus2 = "+92233720368547778.08"
	var testdec64Fd2path = pathutil.CopyAppend(testcontainerpath, "testdec64")
	var testdec64Fd2rangepath = pathutil.CopyAppend(
		testcontainerpath, "testdec64range")
	tblFd2 := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testdec64Fd2path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testdec64Fd2path, dec64Fd2min_minus2, SetFail),
		NewValOpTblEntry(validatesetminvalue, testdec64Fd2path, dec64Fd2min, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testdec64Fd2path, dec64Fd2max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testdec64Fd2path, dec64Fd2max_plus2, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testdec64Fd2rangepath, "-51", SetFail),
		NewValOpTblEntry(validatesetminrange1, testdec64Fd2rangepath, "-50", SetPass),
		NewValOpTblEntry("Validate set inner range value", testdec64Fd2rangepath, "52.0", SetPass),
		NewValOpTblEntry(validatesetbetweenrange2_3, testdec64Fd2rangepath, "65.99", SetFail),
		NewValOpTblEntry(validatesetmaxrange3, testdec64Fd2rangepath, "80", SetPass),
		NewValOpTblEntry(validatesetabovemaxrange3, testdec64Fd2rangepath, "81", SetFail),
	}

	srv, sess = TstStartup(t, schemaFd2, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tblFd2)
	sess.Kill()
}

func TestValidateSetInt8Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testint8 {
		type int8;
	}
	leaf testint8range {
		type int8 {
			range "-50..50 | 52 .. 60 | 70.. 80";
		}
		default 42;
	}
}
`
	const int8min_minus1 = "-129"
	const int8min = "-128"
	const int8max = "127"
	const int8max_plus1 = "128"
	var testint8path = pathutil.CopyAppend(testcontainerpath, "testint8")
	var testint8rangepath = pathutil.CopyAppend(
		testcontainerpath, "testint8range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testint8path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testint8path, int8min_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testint8path, int8min, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testint8path, int8max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testint8path, int8max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testint8rangepath, intrange1min_minus1,
			SetFail),
		NewValOpTblEntry(validatesetminrange1, testint8rangepath, intrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testint8rangepath, intrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testint8rangepath, intrange1max_plus1,
			SetFail),
		NewValOpTblEntry(validatesetminrange2, testint8rangepath, intrange2min, SetPass),
		NewValOpTblEntry(validatesetbetweenrange2_3, testint8rangepath, intrangebetween2and3,
			SetFail),
		NewValOpTblEntry(validatesetmaxrange3, testint8rangepath, intrange3max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange3, testint8rangepath, intrange3maxplus1,
			SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetInt16Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testint16 {
		type int16;
	}
	leaf testint16range {
		type int16 {
			range "-50..50 |52 ..60 | 70.. 80";
		}
		default 42;
	}
}
`
	const int16min = "-32768"
	const int16min_minus1 = "-32769"
	const int16max = "32767"
	const int16max_plus1 = "32768"
	var testint16path = pathutil.CopyAppend(testcontainerpath, "testint16")
	var testint16rangepath = pathutil.CopyAppend(
		testcontainerpath, "testint16range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(
			validatesetnovalue, testint16path, "", SetFail),
		NewValOpTblEntry(
			validatesettoosmall, testint16path, int16min_minus1, SetFail),
		NewValOpTblEntry(
			validatesetminvalue, testint16path, int16min, SetPass),
		NewValOpTblEntry(
			validatesetmaxvalue, testint16path, int16max, SetPass),
		NewValOpTblEntry(
			validatesettoolarge, testint16path, int16max_plus1, SetFail),
		NewValOpTblEntry(
			validatesetbelowminrange1, testint16rangepath, intrange1min_minus1, SetFail),
		NewValOpTblEntry(
			validatesetminrange1, testint16rangepath, intrange1min, SetPass),
		NewValOpTblEntry(
			validatesetmaxrange1, testint16rangepath, intrange1max, SetPass),
		NewValOpTblEntry(
			validatesetabovemaxrange1, testint16rangepath, intrange1max_plus1, SetFail),
		NewValOpTblEntry(
			validatesetbetweenrange2_3, testint16rangepath, intrangebetween2and3, SetFail),
		NewValOpTblEntry(
			validatesetmaxrange3, testint16rangepath, intrange3max, SetPass),
		NewValOpTblEntry(
			validatesetabovemaxrange3, testint16rangepath, intrange3maxplus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetInt32Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testint32 {
		type int32;
	}
	leaf testint32range {
		type int32 {
			range -50..50;
		}
		default 42;
	}
}
`
	const int32min_minus1 = "-2147483649"
	const int32min = "-2147483648"
	const int32max = "2147483647"
	const int32max_plus1 = "2147483648"
	var testint32path = pathutil.CopyAppend(testcontainerpath, "testint32")
	var testint32rangepath = pathutil.CopyAppend(
		testcontainerpath, "testint32range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testint32path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testint32path, int32min_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testint32path, int32min, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testint32path, int32max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testint32path, int32max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testint32rangepath, intrange1min_minus1, SetFail),
		NewValOpTblEntry(validatesetminrange1, testint32rangepath, intrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testint32rangepath, intrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testint32rangepath, intrange1max_plus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetInt64Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testint64 {
		type int64;
	}
	leaf testint64range {
		type int64 {
			range -50..50;
		}
		default 42;
	}
}
`
	const int64min_minus1 = "-9223372036854775809"
	const int64min = "-9223372036854775808"
	const int64max = "9223372036854775807"
	const int64max_plus1 = "9223372036854775808"
	var testint64path = pathutil.CopyAppend(testcontainerpath, "testint64")
	var testint64rangepath = pathutil.CopyAppend(
		testcontainerpath, "testint64range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testint64path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testint64path, int64min_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testint64path, int64min, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testint64path, int64max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testint64path, int64max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testint64rangepath, intrange1min_minus1, SetFail),
		NewValOpTblEntry(validatesetminrange1, testint64rangepath, intrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testint64rangepath, intrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testint64rangepath, intrange1max_plus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetUint8Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testuint8 {
		type uint8;
	}
	leaf testuint8range {
		type uint8 {
			range "1..100 | 150 .. 199 | 220 ..240";
		}
		default 42;
	}
}
`
	const uint8max = "255"
	const uint8max_plus1 = "256"
	var testuint8path = pathutil.CopyAppend(testcontainerpath, "testuint8")
	var testuint8rangepath = pathutil.CopyAppend(
		testcontainerpath, "testuint8range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testuint8path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testuint8path, uintmin_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testuint8path, uintmin, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testuint8path, uint8max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testuint8path, uint8max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testuint8rangepath, uintrange1min_minus1, SetFail),
		NewValOpTblEntry(validatesetminrange1, testuint8rangepath, uintrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testuint8rangepath, uintrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testuint8rangepath, uintrange1max_plus1, SetFail),
		NewValOpTblEntry(validatesetbetweenrange2_3, testuint8rangepath, uintrangebetween2and3, SetFail),
		NewValOpTblEntry(validatesetmaxrange3, testuint8rangepath, uintrange3max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange3, testuint8rangepath, uintrange3maxplus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetUint16Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testuint16 {
		type uint16;
	}
	leaf testuint16range {
		type uint16 {
			range 1..100;
		}
		default 42;
	}
}
`
	const uint16max = "65535"
	const uint16max_plus1 = "65536"
	var testuint16path = pathutil.CopyAppend(testcontainerpath, "testuint16")
	var testuint16rangepath = pathutil.CopyAppend(
		testcontainerpath, "testuint16range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testuint16path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testuint16path, uintmin_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testuint16path, uintmin, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testuint16path, uint16max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testuint16path, uint16max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testuint16rangepath, uintrange1min_minus1, SetFail),
		NewValOpTblEntry(validatesetminrange1, testuint16rangepath, uintrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testuint16rangepath, uintrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testuint16rangepath, uintrange1max_plus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetUint32Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testuint32 {
		type uint32;
	}
	leaf testuint32range {
		type uint32 {
			range 1..100;
		}
		default 42;
	}
}
`
	const uint32max = "4294967295"
	const uint32max_plus1 = "4294967296"
	var testuint32path = pathutil.CopyAppend(testcontainerpath, "testuint32")
	var testuint32rangepath = pathutil.CopyAppend(
		testcontainerpath, "testuint32range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testuint32path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testuint32path, uintmin_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testuint32path, uintmin, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testuint32path, uint32max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testuint32path, uint32max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testuint32rangepath, uintrange1min_minus1, SetFail),
		NewValOpTblEntry(validatesetminrange1, testuint32rangepath, uintrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testuint32rangepath, uintrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testuint32rangepath, uintrange1max_plus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetUint64Leaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf testuint64 {
		type uint64;
	}
	leaf testuint64range {
		type uint64 {
			range 1..100;
		}
		default 42;
	}
}
`
	const uint64max = "18446744073709551615"
	const uint64max_plus1 = "18446744073709551616"
	var testuint64path = pathutil.CopyAppend(testcontainerpath, "testuint64")
	var testuint64rangepath = pathutil.CopyAppend(
		testcontainerpath, "testuint64range")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, testuint64path, "", SetFail),
		NewValOpTblEntry(validatesettoosmall, testuint64path, uintmin_minus1, SetFail),
		NewValOpTblEntry(validatesetminvalue, testuint64path, uintmin, SetPass),
		NewValOpTblEntry(validatesetmaxvalue, testuint64path, uint64max, SetPass),
		NewValOpTblEntry(validatesettoolarge, testuint64path, uint64max_plus1, SetFail),
		NewValOpTblEntry(validatesetbelowminrange1, testuint64rangepath, uintrange1min_minus1, SetFail),
		NewValOpTblEntry(validatesetminrange1, testuint64rangepath, uintrange1min, SetPass),
		NewValOpTblEntry(validatesetmaxrange1, testuint64rangepath, uintrange1max, SetPass),
		NewValOpTblEntry(validatesetabovemaxrange1, testuint64rangepath, uintrange1max_plus1, SetFail),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

func TestValidateSetStringLeaf(t *testing.T) {
	const schema = `
container testcontainer {
	leaf teststring {
		type string {
			length 3..5|7;
		}
	}
	leaf teststringpattern {
		type string {
			pattern "[a-zA-Z]+[1-9][0-9]*";
		}
	}
}
`
	var teststringpatternpath = pathutil.CopyAppend(
		testcontainerpath, "teststringpattern")
	tbl := []ValidateOpTbl{
		NewValOpTblEntry(validatesetnovalue, teststringpath, "", SetFail),
		NewValOpTblEntry("Validate set string 1", teststringpath, "12", SetFail),
		NewValOpTblEntry("Validate set string 2", teststringpath, "123", SetPass),
		NewValOpTblEntry("Validate set string 3", teststringpath, "1234", SetPass),
		NewValOpTblEntry("Validate set string 4", teststringpath, "12345", SetPass),
		NewValOpTblEntry("Validate set string 5", teststringpath, "123456", SetFail),
		NewValOpTblEntry("Validate set string 6", teststringpath, "1234567", SetPass),
		NewValOpTblEntry("Validate set string 7", teststringpath, "12345678", SetFail),
		NewValOpTblEntry("Validate set pattern 1", teststringpatternpath, "1", SetFail),
		NewValOpTblEntry("Validate set pattern 2", teststringpatternpath, "a", SetFail),
		NewValOpTblEntry("Validate set pattern 3", teststringpatternpath, "a0", SetFail),
		NewValOpTblEntry("Validate set pattern 4", teststringpatternpath, "a1", SetPass),
		NewValOpTblEntry("Validate set pattern 5", teststringpatternpath, "a12", SetPass),
	}

	srv, sess := TstStartup(t, schema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, tbl)
	sess.Kill()
}

// Check pattern matching with multiple branches correctly anchors the
// start and end of each part of the pattern.  First example would match
// higher than 65535 before bug in regexp compile was fixed to encompass
// complete expression in parentheses before then wrapping in ^ ... $.
func TestValidateSet4BytePattern(t *testing.T) {
	const patternSchema = `
	container testCont {
		leaf testLeaf {
			type string {
				// Match 1 - 65535.
				pattern '([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|'
				+ '(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5])';
			}
		}
	}`

	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Invalid low value",
			"testCont/testLeaf/0", SetFail),
		createValOpTbl("Lowest valid value",
			"testCont/testLeaf/1", SetPass),
		createValOpTbl("Highest valid value",
			"testCont/testLeaf/65535", SetPass),
		createValOpTbl("Invalid high value",
			"testCont/testLeaf/65536", SetFail),
	}

	srv, sess := TstStartup(t, patternSchema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, test_setTbl)
	sess.Kill()
}

// Second example is the bandwidth-suffix type used by QoS / NPF.  Prior to
// fixing the bug it was possible to add text either in front or after the
// bandwidth, depending on which unit was being used, eg:
//
//   - 1Gbit_oops
//   - oops_1Kbit
//
// This test check that in the original expression it's no longer possible to
// add this text, then, for backwards compatibility, that if we add '.*' in
// the appropriate place, we still can.
func TestValidateSetBWPattern(t *testing.T) {
	const patternSchema = `
	// Original expression which should no longer allow pfx/sfx.
	typedef bandwidth-suffix-orig {
		type string {   /* Number followed by suffix */
			pattern '((10|([0-9](\.([0-9]+))?))[gG][iI]?(bit|bps)?)|'
			+ '([0-9]+(\.([0-9]+))?(([KMBkm][iI]?)?(bit|bps))?)';
		}
	}

	// New expression which maintains original (wrong) behaviour by explicitly
    // unanchoring end of first branch and start of last (second) branch.
	typedef bandwidth-suffix-new {
		type string {   /* Number followed by suffix */
			pattern '((10|([0-9](\.([0-9]+))?))[gG][iI]?(bit|bps)?).*|'
			+ '.*([0-9]+(\.([0-9]+))?(([KMBkm][iI]?)?(bit|bps))?)';
		}
	}

	container testCont {
		leaf origBW {
			type bandwidth-suffix-orig;
		}
		leaf newBW {
			type bandwidth-suffix-new;
		}
	}`

	// 'branch' refers to a part of the regexp separated by '|' from other
	// parts. See XSD-Types spec for further information.
	testOrig_setTbl := []ValidateOpTbl{
		createValOpTbl("Orig: Invalid, prefixed BW (first branch)",
			"testCont/origBW/oops_1Gbit", SetFail),
		createValOpTbl("Orig: Invalid, suffixed BW (first branch)",
			"testCont/origBW/1Gbit_oops", SetFail),
		createValOpTbl("Orig: Invalid, prefixed BW (second branch)",
			"testCont/origBW/oops_1Kbps", SetFail),
		createValOpTbl("Orig: Invalid, suffixed BW (second branch)",
			"testCont/origBW/1Kbps_oops", SetFail),
		createValOpTbl("Orig: Valid value, first branch",
			"testCont/origBW/1Gbit", SetPass),
		createValOpTbl("Orig: Valid value, second branch",
			"testCont/origBW/666Kbit", SetPass),
	}

	testNew_setTbl := []ValidateOpTbl{
		createValOpTbl("New: Invalid, prefixed BW (first branch)",
			"testCont/newBW/oops_1Gbit", SetFail),
		createValOpTbl("New: Wrongly valid, suffixed BW (first branch)",
			"testCont/newBW/1Gbit_oops", SetPass),
		createValOpTbl("New: Wrongly valid, prefixed BW (second branch)",
			"testCont/newBW/oops_1Kbps", SetPass),
		createValOpTbl("New: Invalid, suffixed BW (second branch)",
			"testCont/newBW/1Kbps_oops", SetFail),
		createValOpTbl("New: Valid value, first branch",
			"testCont/newBW/1Gbit", SetPass),
		createValOpTbl("New: Valid value, second branch",
			"testCont/newBW/666Kbit", SetPass),
	}

	srv, sess := TstStartup(t, patternSchema, emptyconfig)
	ValidateSetPathTable(t, sess, srv.Ctx, testOrig_setTbl)
	ValidateSetPathTable(t, sess, srv.Ctx, testNew_setTbl)
	sess.Kill()
}
