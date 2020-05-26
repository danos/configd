// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/danos/mgmterror"
)

type testResult struct {
	TestStr string `json:"test-string"`
	TestInt int    `json:"test-int"`
}

func checkErrorEncoding(
	t *testing.T,
	err error,
	errJson, mgmtErrListJson string,
) {
	result := testResult{
		TestStr: "some string",
		TestInt: 222,
	}

	encJSON, encErr := json.Marshal(newResponse(result, err, 111))

	if encErr != nil {
		t.Fatalf("Error encoding response")
		return
	}

	r := strings.NewReplacer("\n", "", "\t", "")
	expJSON := r.Replace(`
	{
		"result":null,
		"error":` + errJson + `,
		"mgmterrorlist":{"error-list":[` + mgmtErrListJson + `]},
		"id":111
	}`)

	if string(encJSON) != expJSON {
		t.Fatalf("\nExp: %s\nGot: %s\n", expJSON, string(encJSON))
	}
}

func TestBasicErrorEncoding(t *testing.T) {
	err := fmt.Errorf("Plain old error.")

	expErrJson := `"Plain old error."`
	expMgmtErrListJson := ""

	checkErrorEncoding(t, err, expErrJson, expMgmtErrListJson)
}

func TestMgmtErrorEncodingWithInfo(t *testing.T) {

	err := mgmterror.NewUnknownElementApplicationError("bad element")

	expErrJson := "null"
	expMgmtErrListJson := `{
		"error-type":"application",
		"error-tag":"unknown-element",
		"error-severity":"error",
		"error-message":"An unexpected element is present.",
		"error-info":[
			{"bad-element":"bad element"}
		]
	}`

	checkErrorEncoding(t, err, expErrJson, expMgmtErrListJson)
}

func TestMgmtErrorEncodingWithAppTagAndPath(t *testing.T) {

	err := mgmterror.NewExecError(
		[]string{"path", "to", "element"},
		"some error message ...")

	expErrJson := "null"
	expMgmtErrListJson := `{
			"error-type":"application",
			"error-tag":"operation-failed",
			"error-severity":"error",
			"error-app-tag":"exec-failed",
			"error-path":"/path/to/element",
			"error-message":"some error message ..."
		}`

	checkErrorEncoding(t, err, expErrJson, expMgmtErrListJson)
}

func TestMgmtErrorEncodingErrorList(t *testing.T) {

	err1 := mgmterror.NewExecError(
		[]string{"path", "to", "element"},
		"some error message ...")
	err2 := mgmterror.NewExecError(
		[]string{"path", "to", "another", "element"},
		"another error message ...")

	var errList mgmterror.MgmtErrorList
	errList.MgmtErrorListAppend(err1, err2)

	expErrJson := "null"
	expMgmtErrListJson := `{
		"error-type":"application",
		"error-tag":"operation-failed",
		"error-severity":"error",
		"error-app-tag":"exec-failed",
		"error-path":"/path/to/element",
		"error-message":"some error message ..."
	},
	{
		"error-type":"application",
		"error-tag":"operation-failed",
		"error-severity":"error",
		"error-app-tag":"exec-failed",
		"error-path":"/path/to/another/element",
		"error-message":"another error message ..."
	}`

	checkErrorEncoding(t, errList, expErrJson, expMgmtErrListJson)
}
