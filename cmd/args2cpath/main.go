// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014, 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package main

import (
	"fmt"
	"os"

	"github.com/danos/utils/pathutil"
)

func main() {
	fmt.Println(pathutil.Pathstr(os.Args[1:]))
}
