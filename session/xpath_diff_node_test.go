// Copyright (c) 2019,2021, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2015-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// This file implements tests for the diff.Node Xpath functionality.
// It is in configd/session as it is easier access all the
// infrastructure that these tests use. Ideally it should be
// elsewhere.
//
// Testing is done as follows.
//
// First, we verify assumptions about the schema types we will be presented
// with for each YANG node type.
//
// Next, we take each of the 4 node types (container, list, leaf-list and leaf)
// together with 'root', and our special list-key node type, and run them
// through the checkChildren() function that verifies:
//
// - XChildren() are as expected (number, XName / XPath / XValue)
// - XRoot() returns correct value
// - XParent() on each child correctly links to parent
//
// By starting with root (which has container children) and moving onto
// 'interfaces' (container with listEntry children), then onto 'dataplane'
// (container with Leaf and LeafList children) then finally a Leaf, we
// cover all nodes types as both child and parent.

package session_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/danos/config/data"
	"github.com/danos/config/diff"
	"github.com/danos/config/testutils"
	"github.com/danos/config/union"
	"github.com/danos/utils/natsort"
	"github.com/danos/utils/pathutil"
	"github.com/danos/yang/data/datanode"
	yang "github.com/danos/yang/schema"
	"github.com/danos/yang/xpath/xutils"
)

// From an XPATH perspective, a putative config tree might be as follows,
// which includes a container, lists, leaf-lists and leaves:
//
// interface {
//     dataplane dp0s1 {
//         address 1234
//         address 1235
//         address 4444
//     }
//     dataplane dp0s2 {
//         address 6666
//     }
//     serial s1 {
//         address 5555
//     }
//     loopback lo2 {
//     }
// }
//
// This maps to an XPATH tree as follows (dataplane and serial have key
// named 'name' which can only be interrogated by using predicates on the
// listEntry, or value query on 'name' leaf)
//
// Tree                  Schema      	XName       Value     Key

// interfaces            container   	interfaces  (none)
//     dataplane         listEntry   	dataplane   (none)    dp0s1
//         name          leafValue   	name        dp0s1
//         address       leafListValue  address     1234
//         address       leafListValue  address     1235
//         address       leafListValue  address     4444
//     dataplane         listEntry      dataplane   (none)    dp0s2
//         name          leafValue   	name        dp0s2
//         address       leafListValue  address     6666
//     serial            listEntry      serial      (none)    s1
//         name          leafValue   	name        s1
//         address       leafListValue  address     5555
//     loopback          listEntry      loopback    (none)    lo2
//         name          leafValue   	name        lo2
//
// protocols             container      protocols   (none)
//     mpls              container      mpls        (none)
//         min-label     leafValue      min-label   16
//
// Above validated using Perl XPATH tool on vRouter to query 'local-name()'
// and value ('.') on equivalent nodes on real router with config.
//
// Schema type validated by test function here.
//
const xpathSchemaTemplate = `
	module test-yang-xpath {
	namespace "urn:vyatta.com:test:yang-xpath";
	prefix xpath;
	organization "Brocade Communications Systems, Inc.";
	revision 2015-07-28 {
		description "Test schema for XPATH";
	}
` + intfSchemaSnippet + protocolsSchemaSnippet

const intfSchemaSnippet = `
    typedef ipaddress {
        type string;
    }

	container interfaces {
		list dataplane {
			key "name";
			leaf name {
				type string;
			}
			leaf-list address {
				type ipaddress;
				ordered-by user;
			}
		}
		list serial {
			key "name";
			leaf name {
				type string;
			}
			leaf-list address {
				type ipaddress;
			}
		}
		list loopback {
			key "name";
			leaf name {
				type string;
			}
			leaf-list address {
				type ipaddress;
			}
		}
	}
`

const protocolsSchemaSnippet = `
	container protocols {
		container mpls {
			leaf min-label {
				type uint32 {
					range "1 .. 1000000";
				}
			}
			leaf max-label {
				type uint32 {
					range "1 .. 1000000";
				}
			}
			leaf debug {
				type empty;
			}
		}
	}
}
`

func getDataTree(t *testing.T, config [][]string) union.Node {
	sch := bytes.NewBufferString(xpathSchemaTemplate)
	compiledSchema, err := testutils.GetConfigSchema(sch.Bytes())
	if err != nil {
		t.Fatalf("Unable to compile schema: %s", err.Error())
		return nil
	}

	can, run := data.New("root"), data.New("root")
	ut := union.NewNode(can, run, compiledSchema, nil, 0)
	if ut == nil {
		t.Fatalf("Unable to create diff tree.")
		return nil
	}

	for _, cfg := range config {
		ut.Set(nil, cfg)
	}

	return ut
}

// Standard configuration we use for tests that has containers, lists,
// leaf-lists and leaves.
func getConfigTree(t *testing.T) xutils.XpathNode {
	configTree := getDataTree(t,
		[][]string{
			{"interfaces", "dataplane", "dp0s1", "address", "1234"},
			{"interfaces", "dataplane", "dp0s1", "address", "1235"},
			{"interfaces", "dataplane", "dp0s1", "address", "4444"},
			{"interfaces", "dataplane", "dp0s2"},
			{"interfaces", "dataplane", "dp0s2", "address", "6666"},
			{"interfaces", "serial", "s1", "address", "5555"},
			{"interfaces", "loopback", "lo2"},
			{"protocols", "mpls", "min-label", "16"},
			{"protocols", "mpls", "debug"},
		})

	diffTree := diff.NewNode(configTree.Data(), nil,
		configTree.GetSchema(), nil)
	return yang.ConvertToXpathNode(diffTree, diffTree.Schema())
}

type expNodeKey struct {
	name  string
	value string
}

type expNode struct {
	path       xutils.PathType
	value      string
	nodeString string
	keys       []expNodeKey
}

type expNodeSet []expNode

func checkNodeVsExpected(
	t *testing.T,
	act xutils.XpathNode,
	exp expNode,
) {
	var expName string
	if len(exp.path) > 0 {
		expName = exp.path[len(exp.path)-1]
		if expName != act.XName() {
			t.Fatalf("Names mismatched: exp '%s', got '%s'.",
				expName, act.XName())
		}
	} else {
		t.Fatalf("Cannot get expected node name.")
	}
	if exp.path.EqualTo(act.XPath()) == false {
		t.Fatalf("Paths for '%s' mismatched: exp '%s', got '%s'.",
			expName, exp.path, act.XPath())
	}
	if exp.value != act.XValue() {
		t.Fatalf("Values for '%s' mismatched: exp '%s', got '%s'.",
			expName, exp.value, act.XValue())
	}
	if exp.nodeString != xutils.NodeString(act) {
		t.Fatalf("NodeStrings for '%s' mismatched: exp '%s', got '%s'.",
			expName, exp.nodeString, xutils.NodeString(act))
	}
}

func checkNodesEqual(
	t *testing.T,
	testNode, refNode xutils.XpathNode,
	refName string,
) {
	if testNode == nil {
		t.Fatalf("%s of %s doesn't exist.",
			refName, testNode.XName())
	}

	if err := xutils.NodesEqual(testNode, refNode); err != nil {
		t.Fatalf("%s of '%s' doesn't match. %s",
			refName, testNode.XName(),
			err.Error())
	}
}

func checkChildren(
	t *testing.T,
	root xutils.XpathNode,
	parent xutils.XpathNode,
	filter string,
	expResult expNodeSet,
) {
	// Unprefixed filter is fine here.
	children := parent.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: filter},
			xutils.FullTree),
		xutils.Sorted)
	if len(children) != len(expResult) {
		t.Fatalf("Expected %d child(ren) for '%s/%s'.  Got %d.",
			len(expResult), parent.XName(), filter, len(children))
	}

	for ix, child := range children {
		checkNodeVsExpected(t, child, expResult[ix])
		checkNodesEqual(t, child.XRoot(), root, "Root")
		checkNodesEqual(t, child.XParent(), parent, "Parent")
	}
}

func TestDiffXpathInvalidChild(t *testing.T) {
	diffTree := getConfigTree(t)

	checkChildren(t, diffTree, diffTree,
		"something else",
		expNodeSet{})
}

func TestDiffXpathRoot(t *testing.T) {
	diffTree := getConfigTree(t)

	checkChildren(t, diffTree, diffTree,
		"*",
		expNodeSet{
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces"}),
				value:      "",
				nodeString: "/interfaces"},
			expNode{
				path:       xutils.PathType([]string{"/", "protocols"}),
				value:      "",
				nodeString: "/protocols"}})
}

func TestDiffContainer(t *testing.T) {
	diffTree := getConfigTree(t)

	interfaceNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "interfaces"},
			xutils.FullTree),
		xutils.Sorted)
	if len(interfaceNodes) != 1 {
		t.Fatalf("Expected single interface child.")
	}

	// All children
	checkChildren(t, diffTree, interfaceNodes[0],
		"*",
		expNodeSet{
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces", "dataplane"}),
				value:      "dp0s1",
				nodeString: "/interfaces/dataplane[name='dp0s1']"},
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces", "dataplane"}),
				value:      "dp0s2",
				nodeString: "/interfaces/dataplane[name='dp0s2']"},
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces", "loopback"}),
				value:      "lo2",
				nodeString: "/interfaces/loopback[name='lo2']"},
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces", "serial"}),
				value:      "s1",
				nodeString: "/interfaces/serial[name='s1']"}})

	// Filtered set
	checkChildren(t, diffTree, interfaceNodes[0],
		"dataplane",
		expNodeSet{
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces", "dataplane"}),
				value:      "dp0s1",
				nodeString: "/interfaces/dataplane[name='dp0s1']"},
			expNode{
				path:       xutils.PathType([]string{"/", "interfaces", "dataplane"}),
				value:      "dp0s2",
				nodeString: "/interfaces/dataplane[name='dp0s2']"}})

	// Empty set
	checkChildren(t, diffTree, interfaceNodes[0],
		"something else",
		expNodeSet{})
}

func TestDiffList(t *testing.T) {
	diffTree := getConfigTree(t)

	xutils.ValidateTree(diffTree)
	interfaceNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "interfaces"},
			xutils.FullTree),
		xutils.Sorted)
	if len(interfaceNodes) != 1 {
		t.Fatalf("Expected single interface child.")
	}

	// Get dp0s1 list entry.  We know it will be first ...
	intfChildNodes := interfaceNodes[0].XChildren(
		xutils.AllChildren, xutils.Sorted)
	if len(intfChildNodes) == 0 {
		t.Fatalf("Cannot find any nodes.")
	}
	// ... but might as well check
	if (intfChildNodes[0].XName() != "dataplane") ||
		(intfChildNodes[0].XValue() != "dp0s1") {

		t.Fatalf("Cannot locate dp0s1 node.")
	}

	// All children
	checkChildren(t, diffTree, intfChildNodes[0],
		"*",
		expNodeSet{
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "address"}),
				value: "1234",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"address (1234)"},
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "address"}),
				value: "1235",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"address (1235)"},
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "address"}),
				value: "4444",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"address (4444)"},
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "name"}),
				value: "dp0s1",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"name (dp0s1)"},
		})

	// 'address' only (leaf-list)
	checkChildren(t, diffTree, intfChildNodes[0],
		"address",
		expNodeSet{
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "address"}),
				value: "1234",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"address (1234)"},
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "address"}),
				value: "1235",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"address (1235)"},
			expNode{
				path: xutils.PathType([]string{
					"/", "interfaces", "dataplane", "address"}),
				value: "4444",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"address (4444)"}})

	// 'name' node (key)
	checkChildren(t, diffTree, intfChildNodes[0],
		"name",
		expNodeSet{
			expNode{
				path:  xutils.PathType([]string{"/", "interfaces", "dataplane", "name"}),
				value: "dp0s1",
				nodeString: "/interfaces/dataplane[name='dp0s1']/" +
					"name (dp0s1)"}})

	// 'debug' node (empty leaf)
	protocolsNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "protocols"},
			xutils.FullTree),
		xutils.Sorted)
	mplsNodes := protocolsNodes[0].XChildren(xutils.AllChildren, xutils.Sorted)
	checkChildren(t, diffTree, mplsNodes[0],
		"debug",
		expNodeSet{
			expNode{
				path:       xutils.PathType([]string{"/", "protocols", "mpls", "debug"}),
				nodeString: "/protocols/mpls/debug ()"}})
}

func TestDiffKeyLeaf(t *testing.T) {
	diffTree := getConfigTree(t)

	interfaceNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "interfaces"},
			xutils.FullTree),
		xutils.Sorted)
	dataplaneNodes := interfaceNodes[0].XChildren(
		xutils.AllChildren, xutils.Sorted)
	nameNodes := dataplaneNodes[0].XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "name"},
			xutils.FullTree),
		xutils.Sorted)

	checkChildren(t, diffTree, nameNodes[0],
		"*",
		expNodeSet{})
}

func TestDiffNonKeyLeaf(t *testing.T) {
	diffTree := getConfigTree(t)

	protocolsNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "protocols"},
			xutils.FullTree),
		xutils.Sorted)
	mplsNodes := protocolsNodes[0].XChildren(
		xutils.AllChildren, xutils.Sorted)
	minLabelNodes := mplsNodes[0].XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "min-label"},
			xutils.FullTree),
		xutils.Sorted)

	checkChildren(t, diffTree, minLabelNodes[0],
		"*",
		expNodeSet{})
}

func TestDiffEmptyLeaf(t *testing.T) {
	diffTree := getConfigTree(t)

	protocolsNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "protocols"},
			xutils.FullTree),
		xutils.Sorted)
	mplsNodes := protocolsNodes[0].XChildren(
		xutils.AllChildren, xutils.Sorted)
	debugNodes := mplsNodes[0].XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "debug"},
			xutils.FullTree),
		xutils.Sorted)

	checkChildren(t, diffTree, debugNodes[0],
		"*",
		expNodeSet{})
}

func TestDiffLeafList(t *testing.T) {
	diffTree := getConfigTree(t)

	interfaceNodes := diffTree.XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "interfaces"},
			xutils.FullTree),
		xutils.Sorted)
	dataplaneNodes := interfaceNodes[0].XChildren(
		xutils.AllChildren, xutils.Sorted)
	addressNodes := dataplaneNodes[0].XChildren(
		xutils.NewXFilter(xml.Name{Space: "", Local: "address"},
			xutils.FullTree),
		xutils.Sorted)

	checkChildren(t, diffTree, addressNodes[0],
		"*",
		expNodeSet{})
}

// Further validation function in common code we might as well run.
func TestValidateTree(t *testing.T) {
	if err := xutils.ValidateTree(getConfigTree(t)); err != nil {
		t.Fatalf("Validate Tree failed: %s\n", err.Error())
	}
}

// Verify that the YangData functions for the union.Node object work
// as expected - here we test on a list entry as these are the most
// complex objects, having keys/tagnodes to deal with.
type dataplanePath []string

func (dpp dataplanePath) Generate(rand *rand.Rand, size int) reflect.Value {
	dpp = pathutil.CopyAppend([]string{"interfaces", "dataplane"},
		fmt.Sprintf("dp0s%d", rand.Intn(999)+1))
	return reflect.ValueOf(dpp)
}

func createCfgDataplanePaths(paths []dataplanePath) []string {
	cfgPaths := make([]string, len(paths))
	cfgPathMap := make(map[string]bool)
	index := 0
	for _, path := range paths {
		cfgPath := pathutil.Pathstr(path)
		if _, ok := cfgPathMap[cfgPath]; !ok {
			cfgPaths[index] = cfgPath
			index++
			cfgPathMap[cfgPath] = true
		}
	}

	cfgPaths = cfgPaths[:index]
	natsort.Sort(cfgPaths)
	return cfgPaths
}

func getDataplanePathsAndNames(paths []dataplanePath) ([][]string, []string) {

	cfgPaths := createCfgDataplanePaths(paths)
	cfgPathsForDataTree := make([][]string, len(cfgPaths))
	expDataplaneNames := make([]string, len(cfgPaths))
	for i, path := range cfgPaths {
		ps := pathutil.Makepath(path)
		cfgPathsForDataTree[i] = ps
		expDataplaneNames[i] = ps[len(ps)-1]
	}

	return cfgPathsForDataTree, expDataplaneNames
}

type genIPAddress string

func (addr genIPAddress) Generate(rand *rand.Rand, size int) reflect.Value {
	addrStr := fmt.Sprintf("%d.%d.%d.%d",
		rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	addr = genIPAddress(addrStr)
	return reflect.ValueOf(addr)
}

// uniquifyAddrs - remove duplicates, and convert to string type.
// No guarantee rand() will give us a unique set, so deal with that here.
// genIPAddress type is only useful while generating as we need a unique
// type for that.
func uniquifyAddrs(addrs []genIPAddress) []string {
	cfgAddrs := make([]string, len(addrs))
	cfgAddrMap := make(map[string]bool)
	index := 0
	for _, addr := range addrs {
		addrStr := string(addr)
		if _, ok := cfgAddrMap[addrStr]; !ok {
			cfgAddrs[index] = addrStr
			index++
			cfgAddrMap[addrStr] = true
		}
	}

	return cfgAddrs[:index]
}

// uniquifyAddrsAndGenPaths - return 'config' paths with matching exp addrs
// Takes randomly generated addresses and converts them into string slices
// that can be used to mock up configuration, using the given prefix.  Also
// returns a matching slice of the addresses for use when checking results
// of a test are as expected.
func uniquifyAddrsAndGenPaths(
	prefix []string, addrs []genIPAddress) ([][]string, []string) {

	cfgAddrs := uniquifyAddrs(addrs)
	cfgPathsForDataTree := make([][]string, len(cfgAddrs))
	expAddresses := make([]string, len(cfgAddrs))
	for i, addr := range cfgAddrs {
		cfgPathsForDataTree[i] = pathutil.CopyAppend(prefix, addr)
		expAddresses[i] = addr
	}

	return cfgPathsForDataTree, expAddresses
}

func TestDiffYangDataChildrenOrderedBySystem(t *testing.T) {

	check := func(paths []dataplanePath) bool {

		cfgPaths, expDataplaneNames := getDataplanePathsAndNames(paths)

		configUTree := getDataTree(t, cfgPaths)

		diffTree := diff.NewNode(configUTree.Data(), nil,
			configUTree.GetSchema(), nil)

		containerNodes := diffTree.Children()
		if len(containerNodes) == 0 {
			return true
		}
		interfaceNodes := containerNodes[0].Children()
		return checkYangDataFunctions(t, interfaceNodes[0], expDataplaneNames)
	}

	cfg, seed := getCheckCfg()
	if err := quick.Check(check, cfg); err != nil {
		t.Logf("Seed: %d\n", seed)
		t.Fatal(err)
	}
}

func TestUnionYangDataChildrenOrderedBySystem(t *testing.T) {

	check := func(paths []dataplanePath) bool {

		cfgPaths, expDataplaneNames := getDataplanePathsAndNames(paths)

		configUTree := getDataTree(t, cfgPaths)

		containerNodes := configUTree.Children()
		if len(containerNodes) == 0 {
			return true
		}
		interfaceNodes := containerNodes["interfaces"].Children()
		return checkYangDataFunctions(
			t, interfaceNodes["dataplane"], expDataplaneNames)
	}

	cfg, seed := getCheckCfg()
	if err := quick.Check(check, cfg); err != nil {
		t.Logf("Seed: %d\n", seed)
		t.Fatal(err)
	}
}

func TestDiffYangDataChildrenOrderedByUser(t *testing.T) {

	check := func(addrs []genIPAddress) bool {

		cfgPaths, expAddresses := uniquifyAddrsAndGenPaths(
			[]string{"interfaces", "dataplane", "dp0s1", "address"},
			addrs)

		configUTree := getDataTree(t, cfgPaths)

		diffTree := diff.NewNode(configUTree.Data(), nil,
			configUTree.GetSchema(), nil)

		containerNodes := diffTree.Children()
		if len(containerNodes) == 0 {
			return true
		}

		interfaceNodes := containerNodes[0].Children()
		dataplaneNodes := interfaceNodes[0].Children()
		return checkYangDataFunctions(
			t, dataplaneNodes[0].Children()[0],
			expAddresses)
	}

	cfg, seed := getCheckCfg()
	if err := quick.Check(check, cfg); err != nil {
		t.Logf("Seed: %d\n", seed)
		t.Fatal(err)
	}
}

func TestUnionYangDataChildrenOrderedByUser(t *testing.T) {

	check := func(addrs []genIPAddress) bool {

		cfgPaths, expAddresses := uniquifyAddrsAndGenPaths(
			[]string{"interfaces", "dataplane", "dp0s1", "address"},
			addrs)

		configUTree := getDataTree(t, cfgPaths)

		containerNodes := configUTree.Children()
		if len(containerNodes) == 0 {
			return true
		}

		interfaceNodes := containerNodes["interfaces"].Children()
		dataplaneNodes := interfaceNodes["dataplane"].Children()
		return checkYangDataFunctions(
			t, dataplaneNodes["dp0s1"].Children()["address"],
			expAddresses)
	}

	cfg, seed := getCheckCfg()
	if err := quick.Check(check, cfg); err != nil {
		t.Logf("Seed: %d\n", seed)
		t.Fatal(err)
	}
}

func getCheckCfg() (*quick.Config, int64) {
	seed := time.Now().UTC().UnixNano()
	src := rand.NewSource(seed)
	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(src),
	}
	return cfg, seed
}

func checkYangDataFunctions(
	t *testing.T,
	dn datanode.DataNode,
	expChildren []string,
) bool {
	if !checkYangDataChildren(t, dn, expChildren) {
		return false
	}
	return checkYangDataValues(t, dn, expChildren)
}

func checkYangDataChildren(
	t *testing.T,
	dn datanode.DataNode,
	expChildren []string,
) bool {
	actChildren := dn.YangDataChildren()
	if len(actChildren) != len(expChildren) {
		t.Logf("YangDataChildren: exp %d, got %d\n",
			len(expChildren), len(actChildren))
		return false
	}

	for index, child := range actChildren {
		if child.YangDataName() != expChildren[index] {
			t.Logf("Wrong order for YangDataChildren()\n")
			for ix, ch := range actChildren {
				t.Logf(" %d: exp %s, got %s\n", ix,
					expChildren[ix], ch.YangDataName())
			}
			t.Logf("\n")
			return false
		}
	}

	return true
}

func checkYangDataValues(
	t *testing.T,
	dn datanode.DataNode,
	expValues []string,
) bool {
	actValues := dn.YangDataValues()
	if len(actValues) != len(expValues) {
		t.Logf("YangDataValues: exp %d, got %d\n",
			len(expValues), len(actValues))
		return false
	}

	for index, child := range actValues {
		if child != expValues[index] {
			t.Logf("Wrong order for YangDataValues()\n")
			for ix, ch := range actValues {
				t.Logf(" %d: exp %s, got %s\n", ix,
					expValues[ix], ch)
			}
			t.Logf("\n")
			return false
		}
	}

	return true
}
