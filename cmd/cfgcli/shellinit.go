// Copyright (c) 2018-2019 AT&T Intellectual Property.
// All rights reserved.
// Copyright (c) 2015 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"bytes"
	"fmt"
	"strings"
)

func initShell() {
	buf := new(bytes.Buffer)
	fmt.Fprintln(buf, "complete -E -F vyatta_config_complete")
	fmt.Fprintln(buf, "complete -I -F vyatta_config_default_complete")
	m := make(map[string]bool)
	for cmd, _ := range CommandHelps() {
		for pos, _ := range cmd {
			switch cmd[0:pos] {
			case "", "for", "do", "done", "if", "fi", "case", "while", "tr":
				continue
			}
			m[cmd[0:pos]] = true
		}
		m[cmd] = true
	}
	for k, _ := range m {
		if strings.HasPrefix("run", k) {
			fmt.Fprintf(buf, "complete -F vyatta_run_complete %s\n", k)
		} else {
			fmt.Fprintf(buf, "complete -F vyatta_config_complete %s\n", k)
		}

		fmt.Fprintf(buf, "alias %s='vyatta_cfg_run %[1]s'\n", k)
	}
	fmt.Fprintln(buf, "shopt -s histverify")
	fmt.Printf("%s", buf)
}
