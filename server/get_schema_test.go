// Copyright (c) 2017-2019, AT&T Intellectual Property Inc. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"testing"

	"github.com/danos/configd/session/sessiontest"
)

const parentSchema = `
container parentCont {
	leaf parentLeaf {
		type string;
	}
}`

const childSchema = `
container childCont {
	leaf childLeaf {
		type string;
	}
}`

const grandchildSchema = `
augment /childCont {
	container gcCont {
		leaf gcLeaf {
			type string;
		}
	}
}`

var parentSchemaDef = []*sessiontest.TestSchema{
	sessiontest.NewTestSchema("vyatta-test-parent-v1", "parent").
		AddSchemaSnippet(parentSchema),
}

func TestSchemaGetForModuleWithNoSubmodules(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefsByRef(parentSchemaDef).
		init()

	oc.getSchemaAndVerify("vyatta-test-parent-v1", parentSchema)
}

var submoduleSchemas = []*sessiontest.TestSchema{
	sessiontest.NewTestSchema("vyatta-test-parent-v1", "parent").
		AddInclude("vyatta-test-child-v1").
		AddInclude("vyatta-test-grandchild-v1").
		AddSchemaSnippet(parentSchema),
	sessiontest.NewTestSchema("vyatta-test-child-v1", submoduleHasNoPrefix).
		AddBelongsTo("vyatta-test-parent-v1", "parent").
		AddSchemaSnippet(childSchema),
	sessiontest.NewTestSchema(
		"vyatta-test-grandchild-v1", submoduleHasNoPrefix).
		AddBelongsTo("vyatta-test-parent-v1", "parent").
		AddInclude("vyatta-test-child-v1").
		AddSchemaSnippet(grandchildSchema),
}

func TestSchemaGetForSubmodule(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefsByRef(submoduleSchemas).
		init()

	oc.getSchemaAndVerify("vyatta-test-child-v1", childSchema,
		parentSchema, grandchildSchema)
}

func TestSchemaGetForModuleWithSubmodules(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefsByRef(submoduleSchemas).
		init()

	oc.getSchemaAndVerify("vyatta-test-parent-v1", parentSchema,
		childSchema, grandchildSchema)
}

func TestSchemaGetForNonExistentModule(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefsByRef(submoduleSchemas).
		init()

	oc.getNonExistentSchema("vyatta-test-nonexistent-v1")
}

func TestGetAllSchemas(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefsByRef(submoduleSchemas).
		init()

	oc.checkSchemasGettable(
		true, /* include submodules */
		"vyatta-test-parent-v1",
		"vyatta-test-child-v1",
		"vyatta-test-grandchild-v1")
}

func TestGetModuleSchemas(t *testing.T) {

	oc := newOutputChecker(t).
		setSchemaDefsByRef(submoduleSchemas).
		init()

	oc.checkSchemasGettable(
		false, /* exclude submodules */
		"vyatta-test-parent-v1")
	oc.checkSchemasNotGettable(
		false, /* exclude submodules */
		"vyatta-test-child-v1",
		"vyatta-test-grandchild-v1")
}
