// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests to check XPATH performance with a large number
// of nodes with 'when' statements on them.

package session_test

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	. "github.com/danos/configd/session/sessiontest"
)

const npfSchema = `
	grouping action-fw-pbr {
		leaf action {
			type enumeration {
				enum "accept";
				enum "drop";
			}
			mandatory true;
		}
    }

    grouping rule-common {
	    leaf protocol {
			type string;
		}

		container destination {
			leaf port {
				type int16;
				when	"(../../protocol = 'tcp') or (../../protocol = 6) or ../../tcp or " +
					"(../../protocol = 'udp') or (../../protocol = 17)";
			}
		}
	}
`

const fwSchema = `
	grouping rule-fw {
	    uses npf:rule-common;
        uses npf:action-fw-pbr;
    }

	augment /security:security {
		container firewall {
			list name {
				key "ruleset-name";
				leaf ruleset-name {
					type string;
				}
				list rule {
					key "tagnode";
					leaf tagnode {
						type int16;
					}
					uses rule-fw;
				}
				leaf default-action {
					type enumeration {
						enum "drop";
						enum "accept";
					}
				}
			}
		}
	}`

const securitySchema = `
    container security {
	}
`

// XPATH performance tests
//
// Table below shows 5 values, for:
//
// - Original time (prior to fixing XIndex())
// - Original time / (num_entries)^2 * 1000 (to show time is O(n^2))
// - Time with checkMachine() returning without running check
// - Improved time (XIndex() fixed)
// - Improved / no check (to show O(n) relationship now)
//
// Entries   OrigTime   SquareLaw   NoCheck   Improved   LinearLaw
//
// 100     0.4          0.4
// 200     1.7 			0.43        0.15
// 300     4.7 			0.52        0.23
// 400     7.2 			0.45        0.31      0.35       1.13
// 1000                             0.78      0.86       1.10
// 10000                            8.5       9.8        1.15
//
// As can be seen, we've moved from O(n^2) to O(n).
const NumFWRules = 400

func TestFWPerformance(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skip FW Performance test for 'short' tests")
	}

	test_setTbl := []ValidateOpTbl{
		createValOpTbl("description",
			"security/firewall/name/TEST/default-action/drop", SetPass),
	}
	for i := 1; i <= NumFWRules; i++ {
		test_setTbl = append(test_setTbl, createFwRuleEntry(i)...)
	}

	expConfig := genExpFWConfig(NumFWRules)

	fwPerfTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		[]TestSchema{
			{
				Name: NameDef{"vyatta-firewall", "fw"},
				Imports: []NameDef{
					{"vyatta-npf", "npf"}, {"vyatta-security", "security"}},
				SchemaSnippet: fwSchema,
			},
			{
				Name:          NameDef{"vyatta-security", "security"},
				SchemaSnippet: securitySchema,
			},
			{
				Name:          NameDef{"vyatta-npf", "npf"},
				SchemaSnippet: npfSchema,
			},
		},
		emptyconfig, fwPerfTests)
}

func createFwRuleEntry(ruleNum int) []ValidateOpTbl {
	var retTbl []ValidateOpTbl

	retTbl = append(retTbl, createValOpTbl("dummy description",
		fmt.Sprintf("security/firewall/name/TEST/rule/%d/action/accept",
			ruleNum),
		SetPass))
	retTbl = append(retTbl, createValOpTbl("dummy description",
		fmt.Sprintf("security/firewall/name/TEST/rule/%d/destination/port/%d",
			ruleNum, ruleNum),
		SetPass))
	retTbl = append(retTbl, createValOpTbl("dummy description",
		fmt.Sprintf("security/firewall/name/TEST/rule/%d/protocol/tcp",
			ruleNum),
		SetPass))

	return retTbl
}

func genExpFWConfig(numRules int) string {
	var b bytes.Buffer

	b.WriteString("security {\n")
	b.WriteString("\tfirewall {\n")
	b.WriteString("\t\tname TEST {\n")
	b.WriteString("\t\t\tdefault-action drop\n")
	for i := 1; i <= numRules; i++ {
		b.WriteString(getRuleCfg(i))
	}
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func getRuleCfg(ruleNum int) string {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("\t\t\trule %d {\n", ruleNum))
	b.WriteString("\t\t\t\taction accept\n")
	b.WriteString("\t\t\t\tdestination {\n")
	b.WriteString(fmt.Sprintf("\t\t\t\t\tport %d\n", ruleNum))
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t\tprotocol tcp\n")
	b.WriteString("\t\t\t}\n")

	return b.String()
}

// Predicate / static interface next-hop performance.
//
// Schema is pretty close to original, but all typedefs moved into same
// schema, and ipv4-prefix converted to ipv4-address to save working out how
// to deal with a '/' in the value for a node mid-way through a config path
// as normally '/' is the divider between elements.
//
const staticNHSchema = `
typedef ipv4-address {
    type string {
		pattern
        '(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}'
		+  '([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])'
		+ '(%[\p{N}\p{L}]+)?';
    }
}

typedef interface-ifname {
	type string {
		length 1..15;
		pattern '[A-Za-z][-_.0-9A-Za-z]*' {
			error-message "Only alpha-numeric name etc...";
		}
	}
}

grouping static-route-distance {
	leaf distance {
		type uint32 {
			range 1..255;
		}
		default "1";
	}
}

grouping static-route-interface {
	leaf interface {
		type string;
	}
}

grouping static-route-disable {
	leaf disable {
		type empty;
	}
}

grouping static-route-ipv4-next-hop {
	list next-hop {
		must "disable or " +
			"(not(distance = ../blackhole/distance) and " +
			"not(distance = ../unreachable/distance) and " +
			"count(../../interface-route[tagnode = current()/../tagnode]/next-hop-interface[distance = current()/distance]) =" +
			"count(../../interface-route[tagnode = current()/../tagnode]/next-hop-interface[distance = current()/distance]/disable))" {
			error-message "Must not configure same distance for next-hop, interface-route and blackhole/unreachable";
		}
		key "tagnode";
		leaf tagnode {
			type ipv4-address;
			// Not loopback multicast or broadcast.
			must "(not(starts-with(., '127.'))) and " +
				"((substring-before(., '.') < 224) or " +
				"(substring-before(., '.') >=240)) and " +
				"(not(starts-with(., '255.255.255.255')))" {
				error-message "next-hop shouldn't be a loopback, multicast " +
					"or broadcast address.";
			}
		}
		uses static-route-disable;
		uses static-route-interface;
		uses static-route-distance;
	}
}

container protocols {
	container static {
		list route {
			must "count(./*) > 1" {
				error-message "Must add next-hop or blackhole/unreachable.";
			}
			key "tagnode";
			leaf tagnode {
				type ipv4-address; // Should be prefix but have to deal with /
			}
			uses static-route-ipv4-next-hop;
			container blackhole {
				presence "Indicates a blackhole route";
				must "not(../unreachable)" {
					error-message "Must not configure both blackhole and " +
						"unreachable";
				}
				uses static-route-distance;
			}
			container unreachable {
				presence "Indicates an unreachable route";
				uses static-route-distance;
			}
		}

		list interface-route {
			configd:help "Interface based static route";
			must "next-hop-interface" {
				error-message "Must add a next-hop-interface";
			}
			key "tagnode";
			leaf tagnode {
				type ipv4-address;
			}
			list next-hop-interface {
				must "disable or " +
					"(not(distance = ../../route[tagnode = current()/../tagnode]/blackhole/distance) and " +
					"not(distance = ../../route[tagnode = current()/../tagnode]/unreachable/distance))" {
					error-message "Must not configure same distance for interface-route and blackhole/unreachable";
				}
				key "tagnode";
				leaf tagnode {
					type interface-ifname;
				}
				leaf disable {
					type empty;
				}
				uses static-route-distance;
			}
		}
	}
}`

const NumRoutes = 100
const NHAddr = "16.1.6.2" // Same as reported bug used ...
const NHIntf = "dp0s3"
const IPAddrBase = 0x3D010100 // 61.1.1.0

// Test where we have lots of nodes with the problematic must statement, but
// no 'target' nodes.  Both 'context' and target are sibling lists, so when
// locating 'target' nodes we have to efficiently filter out the 'context'
// nodes.
func TestStaticNHPerformance(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skip Static NH Performance test for 'short' tests")
	}

	test_setTbl := []ValidateOpTbl{}
	for i := 1; i <= NumRoutes; i++ {
		test_setTbl = append(test_setTbl,
			createStaticNHEntry(i, IPAddrBase, NHAddr)...)
	}

	expConfig := genExpNHConfig(NumRoutes, IPAddrBase, NHAddr)

	staticNHPerfTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, staticNHSchema,
		emptyconfig, staticNHPerfTests)
}

// Test where we have lots of nodes with the problematic must statement, and
// the same number of target nodes.  Here we are just looking for any speed
// gain we can find, as we have a large number of target nodes to consider.
func TestStaticNHWithIntfRoutesPerformance(t *testing.T) {
	if testing.Short() {
		t.Skipf("Skip StaticNH (IntfRoutes) Performance test for 'short' tests")
	}

	test_setTbl := []ValidateOpTbl{}
	for i := 1; i <= NumRoutes; i++ {
		test_setTbl = append(test_setTbl,
			createStaticNHEntry(i, IPAddrBase, NHAddr)...)
		test_setTbl = append(test_setTbl,
			createStaticIntfRouteEntry(i, IPAddrBase, NHIntf)...)
	}

	expConfig := genExpNHAndIntfRouteConfig(
		NumRoutes, IPAddrBase, NHAddr, NHIntf)

	staticNHWithIntfRoutesPerfTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass, expConfig, expOutAllOK),
	}

	runXpathTestsCheckOutput(t, staticNHSchema,
		emptyconfig, staticNHWithIntfRoutesPerfTests)
}

func genExpNHAndIntfRouteConfig(
	numRoutes, ipAddrBase int,
	nhAddr, nhIntf string,
) string {

	var b bytes.Buffer

	b.WriteString("protocols {\n")
	b.WriteString("\tstatic {\n")
	b.WriteString(genIntfRteConfig(numRoutes, ipAddrBase, nhIntf))
	b.WriteString(genRteConfig(numRoutes, ipAddrBase, nhAddr))
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func genExpNHConfig(numRoutes, ipAddrBase int, nhAddr string) string {
	var b bytes.Buffer

	b.WriteString("protocols {\n")
	b.WriteString("\tstatic {\n")
	b.WriteString(genRteConfig(numRoutes, ipAddrBase, nhAddr))
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func genRteConfig(numRoutes, ipAddrBase int, nhAddr string) string {
	var b bytes.Buffer

	for i := 1; i <= numRoutes; i++ {
		b.WriteString(fmt.Sprintf("\t\troute %s {\n",
			createIPAddr(i, ipAddrBase)))
		b.WriteString(fmt.Sprintf("\t\t\tnext-hop %s {\n", nhAddr))
		b.WriteString("\t\t\t\tdistance 10\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
	}
	return b.String()
}

func genIntfRteConfig(numRoutes, ipAddrBase int, nhIntf string) string {
	var b bytes.Buffer

	for i := 1; i <= numRoutes; i++ {
		b.WriteString(fmt.Sprintf("\t\tinterface-route %s {\n",
			createIPAddr(i, ipAddrBase)))
		b.WriteString(fmt.Sprintf("\t\t\tnext-hop-interface %s {\n", nhIntf))
		b.WriteString("\t\t\t\tdistance 20\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t}\n")
	}
	return b.String()
}

func createStaticNHEntry(
	addrSeed, addrBase int,
	nhAddr string,
) []ValidateOpTbl {

	var retTbl []ValidateOpTbl

	ipAddr := createIPAddr(addrSeed, addrBase)

	retTbl = append(retTbl,
		createValOpTbl(fmt.Sprintf("Static route nexthop %s", ipAddr),
			fmt.Sprintf("protocols/static/route/%s/next-hop/%s/distance/10",
				ipAddr, nhAddr),
			SetPass))

	return retTbl
}

func createStaticIntfRouteEntry(
	addrSeed, addrBase int,
	nhIntf string,
) []ValidateOpTbl {

	var retTbl []ValidateOpTbl

	ipAddr := createIPAddr(addrSeed, addrBase)

	retTbl = append(retTbl,
		createValOpTbl(fmt.Sprintf("Static route nexthop %s", ipAddr),
			fmt.Sprintf(
				"protocols/static/interface-route/%s/next-hop-interface/%s"+
					"/distance/20",
				ipAddr, nhIntf),
			SetPass))

	return retTbl
}

func createIPAddr(seed, base int) string {
	remA := int(math.Mod(float64((seed+base)>>24), 256))
	remB := int(math.Mod(float64(((seed+base)>>16)&0xFF), 256))
	remC := int(math.Mod(float64(((seed+base)>>8)&0xFF), 256))
	remD := int(math.Mod(float64((seed+base)&0xFF), 256))

	return fmt.Sprintf("%d.%d.%d.%d", remA, remB, remC, remD)
}
