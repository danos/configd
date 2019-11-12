// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// Testing to validate DotComponent files.  Tests here verify problems
// exposed with multiple component files that can be detected without
// needing to parse any YANG files.

package main

import (
	"github.com/danos/config/testutils/assert"
	"github.com/danos/vci/conf"
	"testing"
)

func getComponentConfigsCheckError(t *testing.T, dotCompFiles ...string,
) (configs []*conf.ServiceConfig) {

	configs, err := getComponentConfigs(dotCompFiles...)
	if err != nil {
		t.Fatalf("Unexpected component config parse failure:\n  %s\n\n",
			err.Error())
	}

	return configs
}

func getComponentConfigs(dotCompFiles ...string,
) (configs []*conf.ServiceConfig, err error) {
	for _, file := range dotCompFiles {
		cfg, err := conf.ParseConfiguration([]byte(file))
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

// Simple component, single module, model and modelset
const firstComp = `[Vyatta Component]
Name=net.vyatta.test.service.first
Description=First Component
ExecName=/opt/vyatta/sbin/first-service
ConfigFile=/etc/vyatta/first.conf

[Model net.vyatta.test.service.first]
Modules=vyatta-service-first-v1
ModelSets=vyatta-v1`

// Second simple component file, unique config file name
const secondComp = `[Vyatta Component]
Name=net.vyatta.test.service.second
Description=Second Component
ExecName=/opt/vyatta/sbin/second-service
ConfigFile=/etc/vyatta/second.conf

[Model net.vyatta.test.service.second]
Modules=vyatta-service-second-v1
ModelSets=vyatta-v1`

// Third component, with 2 models
const thirdComp = `[Vyatta Component]
Name=net.vyatta.test.service.third
Description=Third Component
ExecName=/opt/vyatta/sbin/third-service
ConfigFile=/etc/vyatta/third.conf

[Model net.vyatta.test.service.third.a]
Modules=vyatta-service-third-a-v1
ModelSets=open-v1

[Model net.vyatta.test.service.third.b]
Modules=vyatta-service-third-b-v1
ModelSets=vyatta-v1`

func TestMultipleComponentsDifferentNamePass(t *testing.T) {
	compConfigs := getComponentConfigsCheckError(t,
		firstComp,
		secondComp,
		thirdComp)

	if err := validateComponents(compConfigs); err != nil {
		t.Fatalf("Unexpected error parsing multiple components: %s",
			err.Error())
	}
}

func TestTwoComponentsSameNameFails(t *testing.T) {
	compConfigs := getComponentConfigsCheckError(t,
		firstComp,
		firstComp)

	if err := validateComponents(compConfigs); err != nil {
		expMsg := assert.NewExpectedMessages(
			"These components are duplicated",
			"net.vyatta.test.service.first")
		expMsg.ContainedIn(t, err.Error())
	} else {
		t.Fatalf("Unexpected success parsing 2 components with same name.")
	}
}

// Simple component that shares config file name with firstComp, as well as
// having its own unique one.
const firstCopyComp_firstConfFile = `[Vyatta Component]
Name=net.vyatta.test.service.firstCopy
Description=FirstCopy Component
ExecName=/opt/vyatta/sbin/firstCopy-service
ConfigFile=/etc/vyatta/firstUnique.conf,/etc/vyatta/first.conf

[Model net.vyatta.test.service.firstCopy]
Modules=vyatta-service-firstCopy-v1
ModelSets=vyatta-v1`

func TestTwoComponentsSameConfigFileFails(t *testing.T) {
	compConfigs := getComponentConfigsCheckError(t,
		firstComp,
		firstCopyComp_firstConfFile)

	if err := validateComponents(compConfigs); err != nil {
		expMsg := assert.NewExpectedMessages(
			"These components have duplicate config files:",
			"first.conf", "service.first ", "service.firstCopy")
		expMsg.ContainedIn(t, err.Error())
	} else {
		t.Fatalf("Unexpected success parsing 2 components with same cfg file.")
	}
}

// 2 components using same model name for same model set
const testComp1_modelX = `[Vyatta Component]
Name=net.vyatta.test.service.testComp1
Description=Test Component 1
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/testComp1.conf

[Model net.vyatta.test.service.X]
Modules=vyatta-service-test-X1-v1
ModelSets=vyatta-v1`

const testComp2_modelX = `[Vyatta Component]
Name=net.vyatta.test.service.testComp2
Description=Test Component 2
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/testComp2.conf

[Model net.vyatta.test.service.X]
Modules=vyatta-service-test-X2-v1
ModelSets=vyatta-v1`

func TestTwoModelsSameNameDifferentComponentFails(t *testing.T) {
	compConfigs := getComponentConfigsCheckError(t,
		testComp1_modelX,
		testComp2_modelX)

	if err := validateComponents(compConfigs); err != nil {
		expMsg := assert.NewExpectedMessages(
			"Model 'net.vyatta.test.service.X' duplicated in:",
			"net.vyatta.test.service.testComp1",
			"net.vyatta.test.service.testComp2")
		expMsg.ContainedIn(t, err.Error())
	} else {
		t.Fatalf("Unexpected success: Comp with 2 components using same model.")
	}
}

const testComp1_testModule = `[Vyatta Component]
Name=net.vyatta.test.service.testComp1
Description=TestComp1 Component
ExecName=/opt/vyatta/sbin/testComp1-service
ConfigFile=/etc/vyatta/testComp1.conf

[Model net.vyatta.test.service.testComp1]
Modules=vyatta-service-testComp-v1,vyatta-service-testComp1-v1
ModelSets=open-v1`

const testComp2_testModule = `[Vyatta Component]
Name=net.vyatta.test.service.testComp2
Description=TestComp2 Component
ExecName=/opt/vyatta/sbin/testComp2-service
ConfigFile=/etc/vyatta/testComp2.conf

[Model net.vyatta.test.service.testComp2]
Modules=vyatta-service-testComp2-v1,vyatta-service-testComp-v1
ModelSets=open-v1`

func TestTwoComponentsSameYangModuleAndModelSetFails(t *testing.T) {
	compConfigs := getComponentConfigsCheckError(t,
		testComp1_testModule,
		testComp2_testModule)

	if err := validateComponents(compConfigs); err != nil {
		expMsg := assert.NewExpectedMessages(
			"YANG module vyatta-service-testComp-v1 is in multiple models",
			"(m/set open-v1)",
			"net.vyatta.test.service.testComp1",
			"net.vyatta.test.service.testComp2")
		expMsg.ContainedIn(t, err.Error())
	} else {
		t.Fatalf("Unexpected success: 2 comps using same YANG.")
	}
}

const testComp3_testModule = `[Vyatta Component]
Name=net.vyatta.test.service.testComp3
Description=TestComp3 Component
ExecName=/opt/vyatta/sbin/testComp3-service
ConfigFile=/etc/vyatta/testComp3.conf

[Model net.vyatta.test.service.testComp3]
Modules=vyatta-service-testComp3-v1,vyatta-service-testComp-v1
ModelSets=ietf-v1`

func TestTwoComponentsSameYangModuleDifferentModelSetPasses(t *testing.T) {
	compConfigs := getComponentConfigsCheckError(t,
		testComp1_testModule,
		testComp3_testModule)

	if err := validateComponents(compConfigs); err != nil {
		t.Fatalf(
			"Unexpected failure: 2 comps using same YANG in diff modelsets.")
	}
}
