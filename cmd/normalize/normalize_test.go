// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"testing"
)

type testCase struct {
	input        string
	expected     string
	errorMessage string
}

var tests = []testCase{
	{"simple", "simple", "Normal strings should not be modified"},
	{"0a:0b:0C:0d:0e:0f", "0a:0b:0c:0d:0e:0f", "MAC-48 addresses should be lower case"},
	{"0a-0b-0C-0d-0e-0f", "0a:0b:0c:0d:0e:0f", "EUI-48 addresses should be lower case with colons"},
	{"00-11-0a-0b-0C-0d-0e-0f", "00:11:0a:0b:0c:0d:0e:0f", "EUI-64 addresses should be lower case with colons"},
	{"0011.0a0b.0C0d", "00:11:0a:0b:0c:0d", "Dot spaced mac addresses get converted"},
	{"0011.0a0b.0C0d.0e0f", "00:11:0a:0b:0c:0d:0e:0f", "Dot spaced mac addresses get converted"},
	{"a:b:c:d:e:f", "0a:0b:0c:0d:0e:0f", "MAC-48 addresses can have missing zeros"},
	{"192.168.01.01", "192.168.1.1", "IPv4 addresses should strip leading zeros"},
	{"192.168.256.01", "192.168.256.01", "Invalid IPv4 addresses are treated as strings"},
	{"FE80:1111::2222", "fe80:1111::2222", "IPv6 addresses should be lower case"},
	{"::FFFF:0:0", "::ffff:0:0", "Special case for mapped IPv4 addresses"},
	{"2001:db9::198.18.4.100/96", "2001:db9::c612:464/96", "Special format with IPv6 containing IPv4 addresses"},
	{"fe80:0:0:0:0:1:0:2", "fe80::1:0:2", "IPv6 addresses should compress out one run of zeros"},
	{"fe80:1:0::0:2", "fe80:1::2", "IPv6 addresses should compress out one run of zeros"},
	{"fe80:0001:0::02", "fe80:1::2", "IPv6 addresses should strip leading zeros"},
	{"192.168.000.001/024", "192.168.0.1/24", "CIDRs should strip leading zeros"},
	{"FE80:0:0:0:0::2/064", "fe80::2/64", "CIDRs should strip leading zeros, plus the addr stuff"},
	{"::FFFF:0:0/064", "::ffff:0:0/64", "Special case for mapped IPv4 addresses"},
}

func TestNormalizeString(t *testing.T) {
	for _, testcase := range tests {
		actual := NormalizeString(testcase.input)
		if testcase.expected != actual {
			t.Error(testcase.errorMessage, "\n",
				"For:     ", testcase.input,
				"Expected:", testcase.expected,
				"But got: ", actual)
		}
	}
}

func v4PrefixCheck(t *testing.T, norm func(string) string) {
	input := "192.168.000.001/024"
	expect := "192.168.0.0/24"
	actual := norm(input)

	if actual != expect {
		t.Errorf("Prefix doesn't match:\n    expect: %s\n    actual: %s",
			expect, actual)
	}
}

func TestNormalizeIpv4Prefix(t *testing.T) {
	v4PrefixCheck(t, NormalizeIPv4prefix)
}

func TestInvalidIPv4Mask(t *testing.T) {
	input := "1.1.1.1/999"
	expect := "1.1.1.1/999"
	actual := NormalizeIPv4prefix(input)

	if actual != expect {
		t.Errorf("Prefix doesn't match:\n    expect: %s\n    actual: %s",
			expect, actual)
	}
}

func TestInvalidIPv4Address(t *testing.T) {
	input := "100.200.300.400/24"
	expect := "100.200.300.400/24"
	actual := NormalizeIPv4prefix(input)

	if actual != expect {
		t.Errorf("Prefix doesn't match:\n    expect: %s\n    actual: %s",
			expect, actual)
	}
}

func v6PrefixCheck(t *testing.T, norm func(string) string) {
	input := "FE80:0000:0:0:ffff:ffff:ffff:ffff/96"
	expect := "fe80::ffff:ffff:0:0/96"
	actual := norm(input)

	if actual != expect {
		t.Errorf("Prefix doesn't match:\n    expect: %s\n    actual: %s",
			expect, actual)
	}
}

func TestNormalizeIpv6Prefix(t *testing.T) {
	v6PrefixCheck(t, NormalizeIPv6prefix)
}

func TestInvalidIPv6Mask(t *testing.T) {
	input := "1::1/129"
	expect := "1::1/129"
	actual := NormalizeIPv6prefix(input)

	if actual != expect {
		t.Errorf("Prefix doesn't match:\n    expect: %s\n    actual: %s",
			expect, actual)
	}
}

func TestInvalidIPv6Address(t *testing.T) {
	input := "1::ABCDE/64"
	expect := "1::ABCDE/64"
	actual := NormalizeIPv6prefix(input)

	if actual != expect {
		t.Errorf("Prefix doesn't match:\n    expect: %s\n    actual: %s",
			expect, actual)
	}
}

func TestNormalizeIpPrefix(t *testing.T) {

	v4PrefixCheck(t, NormalizeIPprefix)
	v6PrefixCheck(t, NormalizeIPprefix)
}

func assertMatch(t *testing.T, expect, actual, normalizer string) {
	if actual != expect {
		t.Errorf("%s doesn't match:\n    expect: %s\n    actual: %s",
			normalizer, expect, actual)
	}
}

func TestNegativeIpv4(t *testing.T) {

	input := "!192.168.000.001"
	expect := "!192.168.0.1"
	actual := NormalizeNegIPv4(input)

	assertMatch(t, expect, actual, "Negative IPv4")
}

func TestNegativeIpv4prefix(t *testing.T) {

	input := "!192.168.000.001/024"
	expect := "!192.168.0.0/24"
	actual := NormalizeNegIPv4prefix(input)

	assertMatch(t, expect, actual, "Negative IPv4 prefix")
}

func TestNegativeIpv6(t *testing.T) {

	input := "!FE80:0:0:0:0::2/064"
	expect := "!fe80::2/64"
	actual := NormalizeNegIPv6(input)

	assertMatch(t, expect, actual, "Negative IPv6")
}

func TestNegativeIp(t *testing.T) {

	input := "!FE80:0:0:0:0::2/064"
	expect := "!fe80::2/64"
	actual := NormalizeNegIP(input)

	assertMatch(t, expect, actual, "Negative IP")
}

func TestNegativeIpv6prefix(t *testing.T) {

	input := "!FE80:0:0:0:0::2/064"
	expect := "!fe80::/64"
	actual := NormalizeNegIPv6prefix(input)

	assertMatch(t, expect, actual, "Negative IPv6 prefix")
}

func TestNegativeIPprefix(t *testing.T) {

	input := "!FE80:0:0:0:0::2/064"
	expect := "!fe80::/64"
	actual := NormalizeNegIPprefix(input)

	assertMatch(t, expect, actual, "Negative IP prefix")
}
