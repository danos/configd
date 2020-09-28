// Copyright (c) 2019-2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server

var productionCallerCmdSetPrivs = callerCmdSetPrivs
var productionTmpDir = tmpDir

func SetCallerCmdSetPrivs(set bool) {
	callerCmdSetPrivs = set
}

func GetProductionCallerCmdSetPrivs() bool {
	return productionCallerCmdSetPrivs
}

func SetTmpDir(dir string) {
	tmpDir = dir
}

func GetProductionTmpDir() string {
	return productionTmpDir
}
