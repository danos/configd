// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2014-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/danos/config/data"
	"github.com/danos/config/schema"
	"github.com/danos/config/union"
	"github.com/danos/mgmterror"
	"github.com/danos/utils/pathutil"
	"github.com/danos/yang/data/encoding"
	"strings"
)

func isEmptyJson(jsonState []byte) bool {
	jsonStr := strings.TrimSpace(string(jsonState))
	return (jsonStr == "" || jsonStr == "{}")
}

func setOperState(
	ut union.Node,
	path []string,
	logger schema.StateLogger,
) []error {
	var warnings []error

	context := ut
	switch ut.GetSchema().(type) {
	case schema.Leaf, schema.LeafList:
		// When running a script on a leaf or leaflist, then we only expect
		// a value back, but since we are returned a JSON object we can only
		// decode at the parent level.
		context = ut.Parent()

	case schema.LeafValue, schema.ListEntry:
		// Special case for list and leaf, because script is on both
		// List and ListEntry and Leaf and LeafValue. We just want to
		// run on list and leaf.
		return nil
	}

	json_state, warns := ut.GetStateJsonWithWarnings(path, logger)
	if len(warns) > 0 {
		warnings = append(warnings, warns...)
	}

	for _, v := range json_state {

		if isEmptyJson(v) {
			continue
		}
		if ok := json.Valid(v); !ok {
			cerr := mgmterror.NewOperationFailedApplicationError()
			cerr.Path = pathutil.Pathstr(path)
			cerr.Message = fmt.Sprintf(
				"Invalidly formatted data returned for (%s)%s: %s",
				reflect.TypeOf(ut.GetSchema()), ut.GetSchema().Name(),
				string(v))
			warnings = append(warnings, cerr)
			continue
		}

		// We avoid higher-level YANG validation (scripts, mandatory, must
		// etc) that is done via the 'validate' command, but we do do basic
		// individual node checks (eg empty non-presence container, list key,
		// or missing leaf values etc).  That stops complete nonsense from
		// getting into the merged tree as Set calls will fail-safe here,
		// ignoring invalid nodes, but we will log warnings for them.
		//
		// Overall validation is done on whole tree as any individual
		// validation could reference any other part of the tree.
		start := time.Now()
		warn := union.UnmarshalJSONIntoNodeWithoutValidation(
			context, encoding.State, v)
		logStateTime(logger,
			fmt.Sprintf("%v Unmarshal w/o validation", path),
			start)
		if warn != nil {
			cerr := mgmterror.NewOperationFailedApplicationError()
			cerr.Path = pathutil.Pathstr(path)
			cerr.Message = fmt.Sprintf(
				"Failed to process returned data for (%s)%s: %s\n%s",
				reflect.TypeOf(ut.GetSchema()), ut.GetSchema().Name(),
				warn.Error(), string(v))
			warnings = append(warnings, cerr)
		}
	}
	return warnings
}

func getOperChild(ut union.Node, name string) union.Node {

	child := ut.Child(name)
	if child != nil {
		return child
	}

	// Get state for any state only children of this node
	for _, v := range ut.GetSchema().(schema.ExtendedNode).StateChildren() {
		if v.Name() == name {
			ut.Data().AddChild(data.New(v.Name()))
			return ut.Child(v.Name())
		}
	}
	return nil
}

func setAllOperState(
	ut union.Node,
	path []string,
	logger schema.StateLogger,
) []error {
	var warnings []error

	if warns := setOperState(ut, path, logger); len(warns) > 0 {
		warnings = append(warnings, warns...)

	}
	warns := setChildrenOperState(ut, path, logger)
	return append(warnings, warns...)
}

func setChildrenOperState(
	ut union.Node,
	path []string,
	logger schema.StateLogger,
) []error {
	var warnings []error

	has_run := make(map[string]bool)

	// Get state for any active children of this node
	for _, v := range ut.Children() {
		has_run[v.Name()] = true
		if warns := setAllOperState(v,
			append(path, v.Name()), logger); len(warns) > 0 {
			warnings = append(warnings, warns...)
		}
	}

	// Get state for any state only children of this node
	// Skip lists, as we don't want to run on the raw list, but only
	// active list entries
	if _, ok := ut.GetSchema().(schema.List); ok {
		return warnings
	}
	for _, v := range ut.GetSchema().(schema.ExtendedNode).StateChildren() {
		if _, ok := has_run[v.Name()]; !ok {
			if sn, ok := ut.GetSchema().(schema.ListEntry); ok {
				// Operational lists might not have keys
				keys := sn.Keys()
				if len(keys) > 0 && v.Name() == keys[0] {
					continue
				}
			}
			ut.Data().AddChild(data.New(v.Name()))
			warns := setAllOperState(ut.Child(v.Name()),
				append(path, v.Name()), logger)
			if len(warns) > 0 {
				warnings = append(warnings, warns...)
			}
		}
	}
	return warnings
}

type errorAndWarnings struct {
	err   error
	warns []error
}

// addStateToTree - adds relevant state information into the tree.
//
// Returns error if path can't be found (this might be because the node
// could exist but doesn't right now), plus list of warnings.
func addStateToTree(
	ut union.Node,
	path []string,
	logger schema.StateLogger,
) errorAndWarnings {
	var errAndWarns errorAndWarnings

	// Walk down to target node calling any scripts that are part of the path
	for i, p := range path {

		current := path[:i]

		// Create the child if necessary and then call any script
		// This means state scripts can populate the list before we
		// get children of the list
		ut = getOperChild(ut, p)
		if ut == nil {
			err := mgmterror.NewUnknownElementApplicationError(p)
			err.Path = pathutil.Pathstr(current)
			errAndWarns.err = err
			return errAndWarns
		}
		warns := setOperState(ut, pathutil.CopyAppend(current, ut.Name()),
			logger)
		if len(warns) > 0 {
			errAndWarns.warns = append(errAndWarns.warns, warns...)
		}

	}
	warns := setChildrenOperState(ut, path, logger)
	if len(warns) > 0 {
		errAndWarns.warns = append(errAndWarns.warns, warns...)
	}
	return errAndWarns
}

func validateFullTree(ut union.Node) error {

	_, errs, ok := schema.ValidateSchema(ut.GetSchema(), ut, false)
	if !ok {
		var out bytes.Buffer
		for _, err := range errs {
			if err == nil {
				continue
			}
			out.WriteString(err.Error())
			out.WriteString("\n")
		}

		return fmt.Errorf("%s", out.String())
	}

	return nil
}
