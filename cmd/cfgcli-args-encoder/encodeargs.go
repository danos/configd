// Copyright (c) 2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"unsafe"
)

func clearArgv() {
	for i := 1; i < len(os.Args); i++ {
		argvNstr := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[i]))
		argvN := (*[1 << 30]byte)(unsafe.Pointer(argvNstr.Data))[:argvNstr.Len]
		for j := 0; j < argvNstr.Len; j++ {
			argvN[j] = '*'
		}
	}
}

func copyArgs(inArgs []string) []string {
	args := make([]string, len(inArgs))
	for i, arg := range inArgs {
		buf := make([]byte, len(arg))
		copy(buf, arg)
		args[i] = string(buf)
	}
	return args
}

type argsObject struct {
	Args []string `json:"args"`
}

func encodeArgs(argsToEncode []string) string {
	args := argsObject{Args: argsToEncode}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.Encode(&args)
	return buf.String()
}

func main() {
	args := copyArgs(os.Args[1:])
	clearArgv()
	fmt.Println(encodeArgs(args))
}
