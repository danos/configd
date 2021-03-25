// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

// Temporary yangd functionality attached to configd

package main

import (
	"log"

	"bytes"
	"fmt"

	"github.com/danos/config/schema"
	"github.com/danos/config/yangconfig"
	"github.com/danos/encoding/rfc7951"
	"github.com/danos/mgmterror"
	"github.com/danos/vci"
	"github.com/danos/vci/conf"
	"github.com/danos/yang/compile"
	"github.com/danos/yang/data/datanode"
	yangenc "github.com/danos/yang/data/encoding"
	yangschema "github.com/danos/yang/schema"
)

// Defines yangd VCI-accessible methods
type Yangd interface {
	ValidateRpcInput(input []byte) ([]byte, error)
	ValidateNotification(input []byte) ([]byte, error)
	LookupRpcDestinationByModuleName(input []byte) (*lookupDestinationByModuleOutput, error)
}

// Object implementing yangd methods
type yangd struct {
	st         schema.ModelSet
	stFull     schema.ModelSet
	compConfig []*conf.ServiceConfig
}

func NewYangd(st, stFull schema.ModelSet, cfg []*conf.ServiceConfig) Yangd {
	return &yangd{
		compConfig: cfg,
		st:         st,
		stFull:     stFull,
	}
}

// Lookup the Yang module-name for a Yang namespace
func (y *yangd) getModuleName(namespace string) string {
	for name, module := range y.st.Modules() {
		if namespace == module.Namespace() {
			return name
		}
	}
	return ""
}

func (y *yangd) findRpc(
	namespace, moduleName,
	name string,
) (schema.Rpc, bool, error) {

	if moduleName != "" {
		return y.findRpcByModuleName(moduleName, name)
	}
	return y.findRpcByNamespace(namespace, name)
}

func (y *yangd) findRpcByModuleName(
	moduleName, name string,
) (schema.Rpc, bool, error) {
	mod, ok := y.st.Modules()[moduleName]
	if !ok {
		return nil, false, fmt.Errorf(
			"Unable to find RPC '%s' for module '%s'",
			name, moduleName)
	}
	return y.findRpcByNamespace(mod.Namespace(), name)
}

func (y *yangd) findRpcByNamespace(
	namespace, name string,
) (schema.Rpc, bool, error) {

	allrpcs := y.st.Rpcs()
	mod_rpcs, ok := allrpcs[namespace]
	if !ok {
		return nil, false,
			fmt.Errorf("Unable to find namespace '%s' for RPC '%s'",
				namespace, name)

	}

	rpc, ok := mod_rpcs[name]
	if !ok || rpc.Input() == nil {
		return nil, false,
			fmt.Errorf("Unable to find RPC '%s' in namespace '%s'",
				name, namespace)
	}

	return rpc.(schema.Rpc), true, nil
}

func unmarshal(sch yangschema.Tree, data, format, mod, ns, name string) (datanode.DataNode, error) {
	var d datanode.DataNode
	var jsonErr error
	modns := mod
	if format == "xml" {
		modns = ns
	}

	switch format {
	case "json":
		d, jsonErr = yangenc.UnmarshalJSON(sch, []byte(data))
	case "rfc7951":
		d, jsonErr = yangenc.UnmarshalRFC7951(sch, []byte(data))
	case "xml":
		d, jsonErr = yangenc.UnmarshalXML(sch, []byte(data))
	default:
		cerr := mgmterror.NewOperationFailedApplicationError()
		cerr.Message = fmt.Sprintf("Unknown RPC encoding '%s' for RPC %s:%s", format, modns, name)
		return nil, cerr
	}

	if jsonErr != nil {
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = fmt.Sprintf(
			"Unable to validate RPC %s:%s - %s",
			modns, name, jsonErr.Error())
		return nil, err

	}

	return d, nil
}

func marshal(sch yangschema.Tree, data datanode.DataNode, format string) ([]byte, error) {
	var d []byte
	switch format {
	case "json":
		d = yangenc.ToJSON(sch, data)
	case "rfc7951":
		d = yangenc.ToRFC7951(sch, data)
	case "xml":
		d = yangenc.ToXML(sch, data)
	default:
		cerr := mgmterror.NewOperationFailedApplicationError()
		cerr.Message = fmt.Sprintf("Unknown RPC encoding '%s'", format)
		return nil, cerr
	}
	return d, nil
}

func (y *yangd) ConvertRpcOutput(input []byte) ([]byte, error) {
	genOut := func(dta string, err error) ([]byte, error) {
		result := struct {
			Data string `rfc7951:"yangd-v1:data"`
		}{Data: dta}

		out, errMarshal := rfc7951.Marshal(result)
		if errMarshal != nil {
			return nil, mgmterror.NewOperationFailedApplicationError()
		}
		return out, err
	}

	var in struct {
		Namespace    string `rfc7951:"yangd-v1:rpc-namespace"`
		ModuleName   string `rfc7951:"yangd-v1:module-name"`
		Name         string `rfc7951:"yangd-v1:rpc-name"`
		Data         string `rfc7951:"yangd-v1:data"`
		InputFormat  string `rfc7951:"yangd-v1:input-format"`
		OutputFormat string `rfc7951:"yangd-v1:output-format"`
	}
	if jsonErr := rfc7951.Unmarshal(input, &in); jsonErr != nil {
		err := mgmterror.NewMalformedMessageError()
		err.Message = fmt.Sprintf(
			"Unable to parse request (internal format error): %s",
			jsonErr.Error())
		return nil, err
	}

	sch, ok, rpcErr := y.findRpc(in.Namespace, in.ModuleName, in.Name)
	if !ok {
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = rpcErr.Error()
		return nil, err
	}

	data, err := unmarshal(sch.Output(), in.Data, in.InputFormat, in.Namespace, in.ModuleName, in.Name)
	if err != nil {
		return nil, err
	}

	d, err := marshal(sch.Output(), data, in.OutputFormat)
	if err != nil {
		return nil, err
	}

	return genOut(string(d), err)

}

func (y *yangd) ConvertRpcInput(input []byte) ([]byte, error) {
	genOut := func(dta string, err error) ([]byte, error) {
		result := struct {
			Data string `rfc7951:"yangd-v1:data"`
		}{Data: dta}

		out, errMarshal := rfc7951.Marshal(result)
		if errMarshal != nil {
			return nil, mgmterror.NewOperationFailedApplicationError()
		}
		return out, err
	}

	var in struct {
		Namespace    string `rfc7951:"yangd-v1:rpc-namespace"`
		ModuleName   string `rfc7951:"yangd-v1:module-name"`
		Name         string `rfc7951:"yangd-v1:rpc-name"`
		Data         string `rfc7951:"yangd-v1:data"`
		InputFormat  string `rfc7951:"yangd-v1:input-format"`
		OutputFormat string `rfc7951:"yangd-v1:output-format"`
	}
	if jsonErr := rfc7951.Unmarshal(input, &in); jsonErr != nil {
		err := mgmterror.NewMalformedMessageError()
		err.Message = fmt.Sprintf(
			"Unable to parse request (internal format error): %s",
			jsonErr.Error())
		return nil, err
	}

	sch, ok, rpcErr := y.findRpc(in.Namespace, in.ModuleName, in.Name)
	if !ok {
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = rpcErr.Error()
		return nil, err
	}

	data, err := unmarshal(sch.Input(), in.Data, in.InputFormat, in.Namespace, in.ModuleName, in.Name)
	if err != nil {
		return nil, err
	}

	d, err := marshal(sch.Input(), data, in.OutputFormat)
	if err != nil {
		return nil, err
	}

	return genOut(string(d), nil)
}

func (y *yangd) LookupModuleNameByNamespace(input []byte) ([]byte, error) {

	var in struct {
		Namespace string `rfc7951:"yangd-v1:namespace"`
	}

	if jsonErr := rfc7951.Unmarshal(input, &in); jsonErr != nil {
		err := mgmterror.NewMalformedMessageError()
		err.Message = fmt.Sprintf(
			"Unable to parse request (internal format error): %s",
			jsonErr.Error())
		return nil, err
	}

	moduleName := y.getModuleName(in.Namespace)

	result := struct {
		ModuleName string `rfc7951:"yangd-v1:module-name"`
	}{ModuleName: moduleName}

	return rfc7951.Marshal(result)
}

func (y *yangd) ValidateRpcInput(input []byte) ([]byte, error) {
	genOut := func(valid bool, err error) ([]byte, error) {
		result := struct {
			Valid bool `rfc7951:"yangd-v1:valid"`
		}{Valid: valid}

		out, errMarshal := rfc7951.Marshal(result)
		if errMarshal != nil {
			return []byte("{\"yangd-v1:valid\": false}"),
				mgmterror.NewOperationFailedApplicationError()
		}
		return out, err
	}

	var in struct {
		Namespace  string `rfc7951:"yangd-v1:rpc-namespace"`
		ModuleName string `rfc7951:"yangd-v1:rpc-module-name"`
		Name       string `rfc7951:"yangd-v1:rpc-name"`
		Input      string `rfc7951:"yangd-v1:rpc-input"`
	}
	if jsonErr := rfc7951.Unmarshal(input, &in); jsonErr != nil {
		err := mgmterror.NewMalformedMessageError()
		err.Message = fmt.Sprintf(
			"Unable to parse request (internal format error): %s",
			jsonErr.Error())
		return genOut(false, err)
	}

	//if in.ModelName != "" && in.Namespace != "" {
	// TODO: This can be removed when we figure out why
	// must expressions in RPC input statements aren't working.
	//	return genOut(false, vci.NewInvalidValueApplicationError())
	//}

	sch, ok, rpcErr := y.findRpc(in.Namespace, in.ModuleName, in.Name)
	if !ok {
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = rpcErr.Error()
		return genOut(false, err)
	}

	_, jsonErr := yangenc.UnmarshalRFC7951(sch.Input(), []byte(in.Input))
	if jsonErr != nil {
		moduleOrNs := in.ModuleName
		if moduleOrNs == "" {
			moduleOrNs = in.Namespace
		}
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = fmt.Sprintf(
			"Unable to validate RPC %s:%s - %s",
			moduleOrNs, in.Name, jsonErr.Error())
		return genOut(false, err)
	}

	return genOut(true, nil)
}

func (y *yangd) findNotification(
	namespace, moduleName,
	name string,
) (schema.Notification, bool, error) {

	if moduleName != "" {
		return y.findNotificationByModuleName(moduleName, name)
	}
	return y.findNotificationByNamespace(namespace, name)
}

func (y *yangd) findNotificationByModuleName(
	moduleName, name string,
) (schema.Notification, bool, error) {
	mod, ok := y.st.Modules()[moduleName]
	if !ok {
		return nil, false, fmt.Errorf(
			"Unable to find Notification '%s' for module '%s'",
			name, moduleName)
	}
	return y.findNotificationByNamespace(mod.Namespace(), name)
}

func (y *yangd) findNotificationByNamespace(
	namespace, name string,
) (schema.Notification, bool, error) {

	allnots := y.st.Notifications()
	mod_nots, ok := allnots[namespace]
	if !ok {
		return nil, false,
			fmt.Errorf("Unable to find namespace '%s' for Notification '%s'",
				namespace, name)

	}

	notification, ok := mod_nots[name]
	if !ok || notification == nil {
		return nil, false,
			fmt.Errorf("Unable to find Notification '%s' in namespace '%s'",
				name, namespace)
	}

	return notification.(schema.Notification), true, nil
}

func (y *yangd) ValidateNotification(input []byte) ([]byte, error) {
	genOut := func(output string, err error) ([]byte, error) {
		result := struct {
			Output string `rfc7951:"yangd-v1:output"`
		}{Output: output}

		out, errMarshal := rfc7951.Marshal(result)
		if errMarshal != nil {
			return []byte(""),
				mgmterror.NewOperationFailedApplicationError()
		}
		return out, err
	}

	var in struct {
		Namespace  string `rfc7951:"yangd-v1:namespace"`
		ModuleName string `rfc7951:"yangd-v1:module-name"`
		Name       string `rfc7951:"yangd-v1:name"`
		Input      string `rfc7951:"yangd-v1:input"`
	}
	if jsonErr := rfc7951.Unmarshal(input, &in); jsonErr != nil {
		err := mgmterror.NewMalformedMessageError()
		err.Message = fmt.Sprintf(
			"Unable to parse request (internal format error): %s",
			jsonErr.Error())
		return genOut("", err)
	}

	sch, ok, notErr := y.findNotification(in.Namespace, in.ModuleName, in.Name)
	if !ok {
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = notErr.Error()
		return genOut("", err)
	}

	dta, jsonErr := yangenc.UnmarshalRFC7951(sch.Schema(), []byte(in.Input))
	if jsonErr != nil {
		moduleOrNs := in.ModuleName
		if moduleOrNs == "" {
			moduleOrNs = in.Namespace
		}
		err := mgmterror.NewInvalidValueApplicationError()
		err.Message = fmt.Sprintf(
			"Unable to validate Notification %s:%s - %s",
			moduleOrNs, in.Name, jsonErr.Error())
		return genOut("", err)
	}

	json := string(yangenc.ToRFC7951(sch.Schema(), dta))
	return genOut(json, nil)
}

type lookupDestinationByModuleInput struct {
	ModuleName string `rfc7951:"yangd-v1:module-name"`
}
type lookupDestinationByModuleOutput struct {
	ModelName string `rfc7951:"yangd-v1:destination"`
}

func (y *yangd) LookupRpcDestinationByModuleName(
	input []byte,
) (*lookupDestinationByModuleOutput, error) {
	var in lookupDestinationByModuleInput
	if jsonErr := rfc7951.Unmarshal(input, &in); jsonErr != nil {
		err := mgmterror.NewMalformedMessageError()
		err.Message = fmt.Sprintf(
			"Unable to parse request (internal format error): %s",
			jsonErr.Error())
		return nil, err
	}

	mod, ok := y.st.Modules()[in.ModuleName]
	if !ok {
		return nil, fmt.Errorf("Unable to find model name for module '%s'", in.ModuleName)
	}
	modelName, ok := y.st.GetModelNameForNamespace(mod.Namespace())
	if !ok {
		return nil, fmt.Errorf("Unable to find model name for module '%s'", in.ModuleName)
	}
	return &lookupDestinationByModuleOutput{
		ModelName: modelName,
	}, nil
}

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

func startYangd() (st, stFull schema.ModelSet) {
	compConfig, err := conf.LoadComponentConfigDir(*compdir)
	if err != nil {
		// For now, failing to load component configuration
		// isn't fatal, so just log the error
		log.Println(err)
	}

	// TODO - what action should we take for any problem components?
	err = validateComponents(compConfig)
	if err != nil {
		log.Println(err)
	}

	ycfg := yangconfig.NewConfig().IncludeYangDirs(*yangdir).
		IncludeFeatures(*capabilities).SystemConfig()

	st, err = schema.CompileDir(
		&compile.Config{
			YangLocations: ycfg.YangLocator(),
			Features:      ycfg.FeaturesChecker(),
			Filter:        compile.IsConfig},
		&schema.CompilationExtensions{
			ComponentConfig: compConfig,
		})
	fatal(err)

	stFull, err = schema.CompileDir(
		&compile.Config{
			YangLocations: ycfg.YangLocator(),
			Features:      ycfg.FeaturesChecker(),
			Filter:        compile.IsConfigOrState()},
		&schema.CompilationExtensions{
			ComponentConfig: compConfig,
		})
	fatal(err)

	// Start up yangd
	yangd := NewYangd(st, stFull, compConfig)
	comp := vci.NewComponent("net.vyatta.vci.config.yangd")
	comp.Model("net.vyatta.vci.config.yangd.v1").
		RPC("yangd-v1", yangd)
	comp.Run()

	return st, stFull
}
