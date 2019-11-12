// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2014-2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	client "github.com/danos/configd/client"
)

type Schemas struct {
	XMLName xml.Name  `xml:"schemas"`
	Schema  []*Schema `xml:"schema"`
}

type Schema struct {
	XMLName    xml.Name `xml:"schema"`
	Id         string   `xml:"identifier"`
	Ver        string   `xml:"version"`
	Ns         string   `xml:"namespace"`
	Features   []string `xml:"-"`
	Deviations []string `xml:"-"`
}

func (s *Schema) String() string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(fmt.Sprintf("%s?module=%s", s.Ns, s.Id)))
	if len(s.Ver) > 0 {
		xml.EscapeText(&buf, []byte(fmt.Sprintf("&revision=%s", s.Ver)))
	}
	if len(s.Features) > 0 {
		xml.EscapeText(&buf, []byte(fmt.Sprintf("&features=%s",
			strings.Join(s.Features, ","))))
	}
	if len(s.Deviations) > 0 {
		xml.EscapeText(&buf, []byte(fmt.Sprintf("&deviations=%s",
			strings.Join(s.Deviations, ","))))
	}
	return buf.String()
}

func (s *Schemas) setFeatures(features map[string]string) {
	for id, list := range features {
		if len(list) == 0 {
			continue
		}
		for _, schema := range s.Schema {
			if schema.Id == id {
				schema.Features = strings.Split(list, ",")
				break
			}
		}
	}
}

func (s *Schemas) setDeviations(deviations map[string]string) {
	for id, list := range deviations {
		if len(list) == 0 {
			continue
		}
		for _, schema := range s.Schema {
			if schema.Id == id {
				schema.Deviations = strings.Split(list, ",")
				break
			}
		}
	}
}

func main() {
	configd, err := client.Dial("unix", "/run/vyatta/configd/main.sock", "")
	defer configd.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	schemastr, err := configd.GetModuleSchemas()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bs := bytes.NewBufferString(schemastr)
	schemas := &Schemas{Schema: make([]*Schema, 0)}

	dec := xml.NewDecoder(bs)
	dec.Decode(&schemas)

	// Get features and add them into the retrieved schema
	features, err := configd.GetFeatures()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	schemas.setFeatures(features)

	deviations, err := configd.GetDeviations()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	schemas.setDeviations(deviations)

	for _, sch := range schemas.Schema {
		fmt.Println(sch)
	}
}
