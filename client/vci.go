// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"github.com/danos/vci"
)

// Input to convert-rpc-input and convert-rpc-output RPCs
type convertRpcData struct {
	Namespace    string `rfc7951:"yangd-v1:rpc-namespace"`
	ModuleName   string `rfc7951:"yangd-v1:module-name"`
	RpcName      string `rfc7951:"yangd-v1:rpc-name"`
	Data         string `rfc7951:"yangd-v1:data"`
	InputFormat  string `rfc7951:"yangd-v1:input-format"`
	OutputFormat string `rfc7951:"yangd-v1:output-format"`
}

// Output of convert-rpc-input and convert-rpc-output RPCs
type convertRpcResult struct {
	Data string `rfc7951:"yangd-v1:data"`
}

type LookupModuleNameReq struct {
	Namespace string `rfc7951:"yangd-v1:namespace"`
}
type LookupModuleNameResult struct {
	ModuleName string `rfc7951:"yangd-v1:module-name"`
}

const yangdV1 = `yangd-v1`
const convertRpcInput = `convert-rpc-input`
const convertRpcOutput = `convert-rpc-output`
const lookupModuleNameByNamespace = `lookup-module-name-by-namespace`

func CallRpc(namespace, name, args, encoding string) (string, error) {
	var inputResult convertRpcResult
	var outputResult convertRpcResult

	moduleName := ""

	rpcResult := ""

	vciClient, err := vci.Dial()
	if err != nil {
		return "", err
	}
	defer vciClient.Close()

	if encoding == "xml" {
		lookupreq := &LookupModuleNameReq{Namespace: namespace}
		var lookupresult LookupModuleNameResult
		err = vciClient.Call(yangdV1, lookupModuleNameByNamespace, lookupreq).
			StoreOutputInto(&lookupresult)
		if err != nil {
			return "", err
		}
		moduleName = lookupresult.ModuleName
	} else {
		moduleName = namespace
	}

	input := &convertRpcData{RpcName: name, Data: args,
		InputFormat: encoding, OutputFormat: "rfc7951"}
	if encoding == "xml" {
		input.Namespace = namespace
	} else {
		input.ModuleName = namespace
	}

	err = vciClient.Call(yangdV1, convertRpcInput, input).StoreOutputInto(&inputResult)
	if err != nil {
		return "", err
	}

	err = vciClient.Call(moduleName, name, inputResult.Data).StoreOutputInto(&rpcResult)
	if err != nil {
		return "", err
	}

	output := &convertRpcData{RpcName: name, Data: rpcResult,
		InputFormat: "rfc7951", OutputFormat: encoding}
	if encoding == "xml" {
		output.Namespace = namespace
	} else {
		output.ModuleName = namespace
	}

	err = vciClient.Call(yangdV1, convertRpcOutput, output).StoreOutputInto(&outputResult)
	if err != nil {
		return "", err
	}

	return outputResult.Data, nil
}
