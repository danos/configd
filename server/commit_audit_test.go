// Copyright (c) 2019-2021, AT&T Intellectual Copyright. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package server_test

import (
	"os/user"
	"testing"

	"github.com/danos/config/auth"
	"github.com/danos/config/testutils"
	"github.com/danos/configd/server"
	"github.com/danos/utils/audit"
)

const commitAuditTestSchema = `
	container test-container {
		leaf secret-leaf {
			configd:secret "true";
			type string;
		}
		leaf non-secret-leaf {
			type string;
		}
		list test-list {
			key test-key;
			leaf test-key {
				type string;
			}
			leaf test-leaf {
				configd:priority 500;
				configd:secret "true";
				type string;
			}
		}
	}`

func genCommitAuditMsg(op, path string) string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return "configuration path [" + path + "] " + op + " by user " + u.Uid
}

func genCommitAuditLog(op, path string) audit.UserLog {
	return audit.UserLog{
		Type:   audit.LOG_TYPE_USER_CFG,
		Msg:    genCommitAuditMsg(op, path),
		Result: 1}
}

func commitAndAssertAuditLogs(t *testing.T, d *server.Disp, tAuth auth.TestAuther, expLogs ...audit.UserLog) {
	dispTestCommit(t, d, testSID)
	auditer := tAuth.GetAuditer()
	audit.AssertUserLogSliceEqualSort(t,
		append(audit.UserLogSlice{}, expLogs...), auditer.GetUserLogs())
	auditer.ClearUserLogs()
}

func TestCommitAuditLog(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcher(t, a, commitAuditTestSchema, emptyConfig)
	dispTestSetupSession(t, d, testSID)

	// Set some config
	dispTestSet(t, d, testSID,
		"test-container/secret-leaf/"+testutils.POISON_SECRETS[0])
	dispTestSet(t, d, testSID, "test-container/non-secret-leaf/bar")

	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("created", "test-container secret-leaf **"),
		genCommitAuditLog("created", "test-container non-secret-leaf bar"))

	// Then delete a secret
	dispTestDelete(t, d, testSID,
		"test-container/secret-leaf/"+testutils.POISON_SECRETS[0])

	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("deleted", "test-container secret-leaf"))

	// Then delete the rest
	dispTestDelete(t, d, testSID, "test-container/non-secret-leaf")

	commitAndAssertAuditLogs(t, d, a, genCommitAuditLog("deleted", "test-container"))
}

func TestMultiPriorityCommitAuditLog(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcher(t, a, commitAuditTestSchema, emptyConfig)
	dispTestSetupSession(t, d, testSID)

	// Set some config
	dispTestSet(t, d, testSID,
		"test-container/secret-leaf/"+testutils.POISON_SECRETS[0])
	dispTestSet(t, d, testSID,
		"test-container/test-list/bar/test-leaf/"+testutils.POISON_SECRETS[1])

	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("created", "test-container secret-leaf **"),
		genCommitAuditLog("created", "test-container test-list bar test-leaf **"))

	// Then delete a secret in one priority tree
	dispTestDelete(t, d, testSID,
		"test-container/test-list/bar/test-leaf/"+testutils.POISON_SECRETS[1])

	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("deleted", "test-container test-list bar test-leaf"))

	// Then delete everything else
	dispTestDelete(t, d, testSID, "test-container/test-list")
	dispTestDelete(t, d, testSID, "test-container/secret-leaf/"+testutils.POISON_SECRETS[0])

	commitAndAssertAuditLogs(t, d, a, genCommitAuditLog("deleted", "test-container"))
}

const commitAuditTestSchemaDefault = `
	container test-container {
		presence "true";
		leaf default-leaf {
			type string;
			default a-default-value;
		}
		leaf non-default-leaf {
			type string;
		}
	}`

func TestDefaultsCommitAuditLog(t *testing.T) {
	a := auth.TestAutherAllowAll()
	d := newTestDispatcher(t, a, commitAuditTestSchemaDefault, emptyConfig)
	dispTestSetupSession(t, d, testSID)

	// Set some config
	dispTestSet(t, d, testSID,
		"test-container")
	dispTestSet(t, d, testSID,
		"test-container/non-default-leaf/foo")

	// Check that:
	// default-leaf value created, with implicit default value
	// non-default leaf value created
	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("created", "test-container non-default-leaf foo"),
		genCommitAuditLog("created", "test-container default-leaf"))

	dispTestDelete(t, d, testSID,
		"test-container/non-default-leaf")
	dispTestSet(t, d, testSID,
		"test-container/default-leaf/bar")

	// Check that:
	// non-default leaf is deleted, the default-leaf value is deleted
	// and explicit value on default leaf is created
	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("deleted", "test-container non-default-leaf"),
		genCommitAuditLog("deleted", "test-container default-leaf a-default-value"),
		genCommitAuditLog("created", "test-container default-leaf bar"))

	dispTestDelete(t, d, testSID, "test-container/default-leaf/bar")

	// Check that:
	// default-leaf is updated, as explicit value removed
	// and updated with default value
	commitAndAssertAuditLogs(t, d, a,
		genCommitAuditLog("updated", "test-container default-leaf"),
		genCommitAuditLog("updated", "test-container"))
}
