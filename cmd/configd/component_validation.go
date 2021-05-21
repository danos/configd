// Copyright (c) 2017-2021, AT&T Intellectual Property.
// All rights reserved.
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"bytes"
	"fmt"

	"github.com/danos/vci/conf"
)

func checkForDuplicateComponentNames(compConfig []*conf.ServiceConfig) error {
	nameMap := make(map[string]bool)
	var errs bytes.Buffer

	for _, comp := range compConfig {
		if _, ok := nameMap[comp.Name]; ok {
			errs.WriteString(fmt.Sprintf("\t%s\n", comp.Name))
		}
		nameMap[comp.Name] = true
	}
	if errs.Len() != 0 {
		return fmt.Errorf("These components are duplicated:\n%s\n",
			errs.String())
	}
	return nil
}

func checkForDuplicateConfigFileNames(compConfig []*conf.ServiceConfig) error {
	cfgFileMap := make(map[string]string)
	var errs bytes.Buffer

	for _, comp := range compConfig {
		for _, cfgFile := range comp.ConfigFiles {
			if entry, ok := cfgFileMap[cfgFile]; ok {
				errs.WriteString(fmt.Sprintf("\t%s: %s and %s\n",
					cfgFile, entry, comp.Name))
			}
			cfgFileMap[cfgFile] = comp.Name
		}
	}
	if errs.Len() != 0 {
		return fmt.Errorf(
			"These components have duplicate config files:\n%s\n",
			errs.String())
	}
	return nil
}

// Assumptions for Models:
//
// - Only one component may provide a given model.  Two different components
//   may not provide the same model for different model sets (though it is
//   possible that at some point this may be a useful mechanism for upgrading
//   or other transitions so we may relax this in future)
//
// - A component may provide one model for multiple model sets, but all
//   must be declared in the same [Model] section, and thus share the same
//   modules.  The 'conf' package is responsible for detecting duplicate
//   Models.
//
func checkForDuplicateModelNames(compConfig []*conf.ServiceConfig) error {
	modelMap := make(map[string]string)
	var errs bytes.Buffer

	for _, comp := range compConfig {
		for _, model := range comp.ModelByName {
			if entry, ok := modelMap[model.Name]; ok {
				errs.WriteString(fmt.Sprintf(
					"Model '%s' duplicated in:\n\t'%s'\n\t'%s'\n",
					model.Name, entry, comp.Name))
			}
			modelMap[model.Name] = comp.Name
		}
	}
	if errs.Len() != 0 {
		return fmt.Errorf(errs.String())
	}
	return nil
}

func checkForDuplicateModuleReferences(compConfig []*conf.ServiceConfig) error {
	moduleMap := make(map[string]map[string]string)
	var errs bytes.Buffer

	for _, comp := range compConfig {
		for modelSetName, model := range comp.ModelByModelSet {
			if _, ok := moduleMap[modelSetName]; !ok {
				moduleMap[modelSetName] = make(map[string]string)
			}
			for _, module := range model.Modules {
				if _, ok := moduleMap[modelSetName][module]; ok {
					errs.WriteString(fmt.Sprintf(
						"YANG module %s is in multiple models (m/set %s)"+
							"\n\t%s\n\t%s\n",
						module, modelSetName,
						moduleMap[modelSetName][module], model.Name))
				} else {
					moduleMap[modelSetName][module] = model.Name
				}
			}
		}
	}
	if errs.Len() != 0 {
		return fmt.Errorf(errs.String())
	}
	return nil
}

// Validations that can be done prior to parsing YANG modules.
func validateComponents(compConfig []*conf.ServiceConfig) error {

	if err := checkForDuplicateComponentNames(compConfig); err != nil {
		return err
	}

	if err := checkForDuplicateConfigFileNames(compConfig); err != nil {
		return err
	}

	if err := checkForDuplicateModelNames(compConfig); err != nil {
		return err
	}

	if err := checkForDuplicateModuleReferences(compConfig); err != nil {
		return err
	}

	return nil
}
