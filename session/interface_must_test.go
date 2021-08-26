// Copyright (c) 2017-2021, AT&T Intellectual Property Inc.
// All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file contains tests to check XPATH functionality and performance of
// the must statement used as the equivalent of the leafref union we are not
// allowed until YANG 1.1 which supports such things.  Sigh.

package session_test

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/danos/config/testutils"
	. "github.com/danos/configd/session/sessiontest"
	"github.com/danos/mgmterror/errtest"
)

// Helper functions for generating configuration and errors.

func genIntfConfig(numInts int, refType, refName string) string {
	var b bytes.Buffer

	b.WriteString("management {\n")
	for i := 1; i <= numInts; i++ {
		b.WriteString(fmt.Sprintf("\tintf%s int%d {\n\t\tref %s\n\t}\n",
			refType, i, refName))
	}
	b.WriteString("}\n")
	return b.String()
}

func genIntfErr(t *testing.T, numInts int, refType, refName string) []string {

	var intfErrs []string
	for i := 1; i <= numInts; i++ {
		intfErrs = append(intfErrs,
			errtest.NewInterfaceMustExistError(t,
				fmt.Sprintf("management/intf%s/int%d/ref/%s",
					refType, i, refName)).
				RawErrorStrings()...)
	}
	return intfErrs
}

func genIntfSet(intfNum int, refType, refName string) string {
	return fmt.Sprintf("management/intf%s/int%d/ref/%s",
		refType, intfNum, refName)
}

// Test schemas need to span multiple modules, to mimic the vRouter schemas.
// We create all current interface types, which get augmented to the
// interfaces container.  We have a custom 'management' container that
// is used for testing, and which contains our various XPATH expressions.

const intfSchema = `
	typedef ipv4-address {
    type string {
		pattern
        '(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}'
		+  '([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])';
    }
}
typedef address-dhcp {
	type union {
		type enumeration {
			enum dhcp;
			enum dhcpv6;
		}
		type ipv4-address;
		// Ignore IPv6 for now - just need one non-dhcp option.
	}
}

container interfaces;
`

const mgmtSchema = `
container management {
	list intfOld {
		description
		    "For checking performance of old expression.  This explicitly
             named each interface, so is unwieldy, hard to read and maintain,
             can't cope with new interface types w/o modification, and is
             inefficient in terms of XPATH evaluation.";
		key name;
		leaf name {
			type string;
		}
		leaf ref {
			type string;
			must "(current() = /if:interfaces/if-dataplane:dataplane/if-dataplane:tagnode)"
			+ "or (substring-after(current(), '.') = /if:interfaces/if-dataplane:dataplane"
			+ "[if-dataplane:tagnode = substring-before(current(), '.')]/if-dataplane:vif/if-dataplane:tagnode)"
			+ "or (current() = /if:interfaces/if-bridge:bridge/if-bridge:tagnode)"
			+ "or (current() = /if:interfaces/if-erspan:erspan/if-erspan:ifname)"
			+ "or (current() = /if:interfaces/if-l2tpeth:l2tpeth/if-l2tpeth:tagnode)"
			+ "or (substring-after(current(), '.') = /if:interfaces/if-l2tpeth:l2tpeth"
			+ "[if-l2tpeth:tagnode = substring-before(current(), '.')]/if-l2tpeth:vif/if-l2tpeth:tagnode)"
			+ "or (current() = /if:interfaces/if-loopback:loopback/if-loopback:tagnode)"
			+ "or (current() = /if:interfaces/if-openvpn:openvpn/if-openvpn:tagnode)"
			+ "or (current() = /if:interfaces/if-tunnel:tunnel/if-tunnel:tagnode)"
			+ "or (current() = /if:interfaces/if-vti:vti/if-vti:tagnode)"
			+ "or (current() = /if:interfaces/if-bonding:bonding/if-bonding:tagnode)"
			+ "or (substring-after(current(), '.') = /if:interfaces/if-bonding:bonding"
			+ "[if-bonding:tagnode = substring-before(current(), '.')]/if-bonding:vif/if-bonding:tagnode)" {
				error-message "Interface must exist.";
			}
		}
	}
	list intfNew {
		description
		    "For checking all valid intf types are recognised.  This new
             expression will support any new interfaces that use 'tagnode'
             for their name, both at top level and at VIF sub-node level.
             'erspan' is called out separately due to the different node
             name (ifname vs tagnode).";
		key name;
		leaf name {
			type string;
		}
		leaf ref {
			type string;
			must "(current() = /if:interfaces/*/*[local-name(.) = 'tagnode'])"
			+ " or "
			+ "(current() = /if:interfaces/if-erspan:erspan/if-erspan:ifname)"
			+ " or "
			+ "/if:interfaces/*/*[local-name(.) = 'vif']"
			+ "[./../* = substring-before(current(), '.')]"
			+ "/*[local-name(.) = 'tagnode']"
			+ "[. = substring-after(current(), '.')]" {
				error-message "Interface must exist.";
			}
		}
	}
	leaf-list newIntfs {
		description "For checking all valid intf types are recognised";
		type string;
		must "(current() = /if:interfaces/*/*[local-name(.) = 'tagnode'])"
		+ " or "
        + "(current() = /if:interfaces/if-erspan:erspan/if-erspan:ifname)"
	    + " or "
		+ "/if:interfaces/*/*[local-name(.) = 'vif']"
		+ "[./../* = substring-before(current(), '.')]"
		+ "/*[local-name(.) = 'tagnode']"
		+ "[. = substring-after(current(), '.')]" {
			error-message "Interface must exist.";
		}
	}
	leaf-list intfNewDhcp {
		description
		    "For checking match against interface with DHCP configured.
             This is a variant of the new expression above that first
             finds the right interface tagnode, then looks for any address
             nodes for that list entry that are set to 'dhcp'";
		type string;
		must "(/if:interfaces/*/*[local-name(.) = 'tagnode']"
		+ "[. = current()]/../*[local-name(.) = 'address'][. = 'dhcp'])"
		+ " or "
		+ "(/if:interfaces/if-erspan:erspan/if-erspan:ifname[. = current()]"
		+ "/../if-erspan:address = 'dhcp')"
		+ " or "
		+ "(/if:interfaces/*/*[local-name(.) = 'vif']"
		+ "[./../* = substring-before(current(), '.')]"
		+ "/*[local-name(.) = 'tagnode']"
		+ "[. = substring-after(current(), '.')]"
		+ "/../*[local-name(.) = 'address'][. = 'dhcp'])" {
			error-message "Interface must exist, and have DHCP address.";
		}
	}
}`

const dataplaneSchema = `
augment /if:interfaces {
	list dataplane {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
		list vif {
			key tagnode;
			leaf tagnode {
				type uint32 {
					range 1..99999;
				}
			}
			leaf-list address {
				type if:address-dhcp;
			}
		}
	}
}`

const l2tpethSchema = `
augment /if:interfaces {
	list l2tpeth {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
		list vif {
			key tagnode;
			leaf tagnode {
				type uint32 {
					range 1..99999;
				}
			}
			leaf-list address {
				type if:address-dhcp;
			}
		}
	}
}`

const bondingSchema = `
augment /if:interfaces {
	list bonding {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
		list vif {
			key tagnode;
			leaf tagnode {
				type uint32 {
					range 1..99999;
				}
			}
			leaf-list address {
				type if:address-dhcp;
			}
		}
	}
}`

const bridgeSchema = `
augment /if:interfaces {
	list bridge {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
	}
}`

const erspanSchema = `
augment /if:interfaces {
	list erspan {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
	}
}`

const loopbackSchema = `
augment /if:interfaces {
	list loopback {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
	}
}`

const openvpnSchema = `
augment /if:interfaces {
	list openvpn {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
	}
}`

const switchSchema = `
augment /if:interfaces {
	list switch {
		key name;
		leaf name {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
		list vif {
			key tagnode;
			leaf tagnode {
				type uint32 {
					range 1..99999;
				}
			}
			leaf-list address {
				type if:address-dhcp;
			}
		}
	}
}`

const tunnelSchema = `
augment /if:interfaces {
	list tunnel {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
	}
}`

const vtiSchema = `
augment /if:interfaces {
	list vti {
		key tagnode;
		leaf tagnode {
			type string;
		}
		leaf-list address {
			type if:address-dhcp;
		}
	}
}`

var intfTestSchemas = []TestSchema{
	{
		Name:          NameDef{Namespace: "vyatta-dataplane", Prefix: "dataplane"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: dataplaneSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-l2tpeth", Prefix: "l2tpeth"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: l2tpethSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-bonding", Prefix: "bonding"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: bondingSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-bridge", Prefix: "bridge"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: bridgeSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-erspan", Prefix: "erspan"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: erspanSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-loopback", Prefix: "loopback"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: loopbackSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-openvpn", Prefix: "openvpn"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: openvpnSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-switch", Prefix: "switch"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: switchSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-tunnel", Prefix: "tunnel"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: tunnelSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-vti", Prefix: "vti"},
		Imports:       []NameDef{{Namespace: "vyatta-interfaces", Prefix: "if"}},
		SchemaSnippet: vtiSchema,
	},
	{
		Name:          NameDef{Namespace: "vyatta-interfaces", Prefix: "interfaces"},
		SchemaSnippet: intfSchema,
	},
	{
		Name: NameDef{Namespace: "vyatta-mgmt", Prefix: "mgmt"},
		Imports: []NameDef{
			{Namespace: "vyatta-interfaces", Prefix: "if"},
			{Namespace: "vyatta-dataplane", Prefix: "if-dataplane"},
			{Namespace: "vyatta-l2tpeth", Prefix: "if-l2tpeth"},
			{Namespace: "vyatta-bonding", Prefix: "if-bonding"},
			{Namespace: "vyatta-bridge", Prefix: "if-bridge"},
			{Namespace: "vyatta-erspan", Prefix: "if-erspan"},
			{Namespace: "vyatta-loopback", Prefix: "if-loopback"},
			{Namespace: "vyatta-openvpn", Prefix: "if-openvpn"},
			{Namespace: "vyatta-switch", Prefix: "if-switch"},
			{Namespace: "vyatta-tunnel", Prefix: "if-tunnel"},
			{Namespace: "vyatta-vti", Prefix: "if-vti"},
		},
		SchemaSnippet: mgmtSchema,
	},
}

var baseConfig = Cont("interfaces",
	List("bonding",
		ListEntry("bond1",
			List("vif",
				ListEntry("1"))),
		ListEntry("bond2")),
	List("bridge",
		ListEntry("br1")),
	List("dataplane",
		ListEntry("dp0s1",
			List("vif",
				ListEntry("1"),
				ListEntry("2"))),
		ListEntry("dp0s2",
			List("vif",
				ListEntry("2")))),
	List("erspan",
		ListEntry("erspan1")),
	List("l2tpeth",
		ListEntry("l2tpeth1"),
		ListEntry("l2tpeth2",
			List("vif",
				ListEntry("1")))),
	List("loopback",
		ListEntry("lo1")),
	List("openvpn",
		ListEntry("ov1")),
	List("switch",
		ListEntry("sw1",
			List("vif",
				ListEntry("1")))),
	List("tunnel",
		ListEntry("tun1")),
	List("vti",
		ListEntry("vti1")))

var newConfig = Cont("management",
	LeafList("newIntfs",
		LeafListEntry("bond1.1"),
		LeafListEntry("bond2"),
		LeafListEntry("br1"),
		LeafListEntry("dp0s1"),
		LeafListEntry("dp0s2.2"),
		LeafListEntry("erspan1"),
		LeafListEntry("l2tpeth1"),
		LeafListEntry("l2tpeth2.1"),
		LeafListEntry("lo1"),
		LeafListEntry("ov1"),
		LeafListEntry("sw1.1"), // Parent sw1 will fail must
		LeafListEntry("tun1"),
		LeafListEntry("vti1")))

// Ensure interfaces with tagnode/ifname keys, and all VIFS, pass the interface
// must statement.
func TestInterfaceMustNew(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Create bonding VIF management interface ref",
			"management/newIntfs/bond1.1", SetPass),
		createValOpTbl("Create bonding management interface ref",
			"management/newIntfs/bond2", SetPass),
		createValOpTbl("Create bridge management interface ref",
			"management/newIntfs/br1", SetPass),
		createValOpTbl("Create dataplane management interface ref",
			"management/newIntfs/dp0s1", SetPass),
		createValOpTbl("Create dataplane VIF management interface ref",
			"management/newIntfs/dp0s2.2", SetPass),
		createValOpTbl("Create erspan management interface ref",
			"management/newIntfs/erspan1", SetPass),
		createValOpTbl("Create l2tpeth management interface ref",
			"management/newIntfs/l2tpeth1", SetPass),
		createValOpTbl("Create l2tpeth VIF management interface ref",
			"management/newIntfs/l2tpeth2.1", SetPass),
		createValOpTbl("Create loopback management interface ref",
			"management/newIntfs/lo1", SetPass),
		createValOpTbl("Create ov management interface ref",
			"management/newIntfs/ov1", SetPass),
		createValOpTbl("Create sw VIF management interface ref",
			"management/newIntfs/sw1.1", SetPass),
		createValOpTbl("Create tunnel management interface ref",
			"management/newIntfs/tun1", SetPass),
		createValOpTbl("Create VTI management interface ref",
			"management/newIntfs/vti1", SetPass),
	}

	intfMustTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass,
			baseConfig+newConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		intfTestSchemas, baseConfig, intfMustTests)
}

// Check plain switch interface fails.
func TestInterfaceMustNewSwitchFail(t *testing.T) {
	t.Skipf("TBD")
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Create switch management interface ref",
			"management/newIntfs/sw1", SetPass),
	}

	intfMustTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass,
			baseConfig+newConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		intfTestSchemas, baseConfig, intfMustTests)
}

var baseDhcpConfig = Cont("interfaces",
	List("bonding",
		ListEntry("bond1",
			LeafList("address",
				LeafListEntry("1.2.3.4"))),
		ListEntry("bond2",
			LeafList("address",
				LeafListEntry("5.6.7.8"),
				LeafListEntry("dhcp")))),
	List("dataplane",
		ListEntry("dp0p1",
			List("vif",
				ListEntry("1",
					LeafList("address",
						LeafListEntry("dhcp"))))),
		ListEntry("dp0s1",
			List("vif",
				ListEntry("1"),
				ListEntry("2")))),
	List("erspan",
		ListEntry("erspan3",
			LeafList("address",
				LeafListEntry("dhcp")))),
	List("vti",
		ListEntry("vti1")))

var newDhcpConfig = Cont("management",
	LeafList("intfNewDhcp",
		LeafListEntry("bond2"),
		LeafListEntry("dp0p1.1"),
		LeafListEntry("erspan3")))

// These tests were created mainly to work out the correct XPATH expression
// for this scenario, ie finding the interface list entry matching the given
// 'leafref' name, and then determining if that entry has a DHCP address on
// it.  This XPATH expression builds on the syntax for the plain interface-
// name matching, as having found the interface, it goes back to the list
// entry to find any address children of value 'dhcp'.

func TestInterfaceDhcpPass(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Create bonding ref",
			"management/intfNewDhcp/bond2", SetPass),
		createValOpTbl("Create dataplane VIF ref",
			"management/intfNewDhcp/dp0p1.1", SetPass),
		createValOpTbl("Create erspan ref",
			"management/intfNewDhcp/erspan3", SetPass),
	}

	intfDhcpTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitPass,
			baseDhcpConfig+newDhcpConfig, expOutAllOK),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		intfTestSchemas, baseDhcpConfig, intfDhcpTests)
}

func TestInterfaceDhcpFail(t *testing.T) {
	test_setTbl := []ValidateOpTbl{
		createValOpTbl("Create bonding ref",
			"management/intfNewDhcp/bond1", SetPass),
		createValOpTbl("Create dataplane VIF ref",
			"management/intfNewDhcp/dp0p1.2", SetPass),
		createValOpTbl("Create VTI ref",
			"management/intfNewDhcp/vti1", SetPass),
	}

	test_expOut :=
		errtest.NewMustCustomError(t,
			"/management/intfNewDhcp/bond1",
			"Interface must exist, and have DHCP address.").
			RawErrorStrings()
	test_expOut = append(test_expOut,
		errtest.NewMustCustomError(t,
			"/management/intfNewDhcp/dp0p1.2",
			"Interface must exist, and have DHCP address.").
			RawErrorStrings()...)
	test_expOut = append(test_expOut,
		errtest.NewMustCustomError(t,
			"/management/intfNewDhcp/vti1",
			"Interface must exist, and have DHCP address.").
			RawErrorStrings()...)

	intfDhcpTests := []xpathTestEntry{
		newXpathTestEntry(test_setTbl, nil, CommitFail,
			baseDhcpConfig+newDhcpConfig, test_expOut),
	}

	runXpathTestsCheckOutputMultipleSchemas(t,
		intfTestSchemas, baseDhcpConfig, intfDhcpTests)
}

func testInterfaceMustCommon(
	t *testing.T,
	numInts int,
	refType,
	refName string,
	pass bool,
) {
	test_setTbl := []ValidateOpTbl{}
	for i := 1; i <= numInts; i++ {
		test_setTbl = append(test_setTbl,
			createValOpTbl("Create leafref management interface",
				genIntfSet(i, refType, refName), SetPass))
	}

	var intfMustTests = []xpathTestEntry{}
	if pass == CommitPass {
		intfMustTests = []xpathTestEntry{
			newXpathTestEntry(test_setTbl, nil, pass,
				baseConfig+genIntfConfig(numInts, refType, refName),
				expOutAllOK),
		}
	} else {
		intfMustTests = []xpathTestEntry{
			newXpathTestEntry(test_setTbl, nil, pass,
				baseConfig, genIntfErr(t, numInts, refType, refName)),
		}
	}
	runXpathTestsCheckOutputMultipleSchemas(t,
		intfTestSchemas, baseConfig, intfMustTests)
}

// For normal testing, keep the number of interfaces small.
var NumInterfaces = 50
var RefIntName = "dp0s2.2"
var RefInvalidIntName = "dp0s22.2"

// Test  Iterations   Old   New
//
// Pass   500         0.79   0.65
// Fail   500         0.75   0.55
//
// Pass  5000         8.35   6.67
// Fail  5000         8.67   7.83
//
// So, new expression is a bit faster, but not by a lot.

func TestInterfaceMustOldPass(t *testing.T) {
	testInterfaceMustCommon(t, NumInterfaces, "Old", RefIntName, CommitPass)
}

func TestInterfaceMustOldFail(t *testing.T) {
	testInterfaceMustCommon(t, NumInterfaces, "Old", RefInvalidIntName,
		CommitFail)
}

func TestInterfaceMustNewPass(t *testing.T) {
	testInterfaceMustCommon(t, NumInterfaces, "New", RefIntName, CommitPass)
}

func TestInterfaceMustNewFail(t *testing.T) {
	testInterfaceMustCommon(t, NumInterfaces, "New", RefInvalidIntName,
		CommitFail)
}
