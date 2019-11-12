// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/danos/config/parse"
)

func main() {
	text, err := ioutil.ReadAll(os.Stdin)
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	t, err := parse.Parse("stdin", string(text))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(t.Root)
}
