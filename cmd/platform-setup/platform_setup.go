// Copyright (c) 2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/danos/config/platform"
)

const (
	PlatformIDFile = "/var/lib/vyatta-platform/platform-id.conf"
)

var platformid string
var definitionsDir string

type PlatformConfig struct {
	PlatformID string `json:"platform-id"`
}

func init() {
	flag.StringVar(&definitionsDir, "definitions", platform.DefaultBaseDir, "Platform identifier")
	flag.StringVar(&platformid, "platformid", "", "Platform identifier")
}

func loadInitialBootPlatformID() (*PlatformConfig, error) {
	pID := &PlatformConfig{}

	f, err := os.Open(PlatformIDFile)

	if err != nil {
		// Assume initial boot
		if os.IsNotExist(err) {
			return pID, nil
		}
		return pID, err
	}

	defer f.Close()

	reader := json.NewDecoder(f)
	reader.Decode(&pID)

	return pID, nil
}

func saveInitialBootPlatformID(id *PlatformConfig) error {
	dir := filepath.Dir(PlatformIDFile)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {

		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(PlatformIDFile)

	if err != nil {
		return err
	}
	defer f.Close()

	writer := json.NewEncoder(f)
	err = writer.Encode(id)
	return err
}

func main() {
	flag.Parse()

	id, err := loadInitialBootPlatformID()

	if err != nil {
		// report errors, but keep going.
		fmt.Printf("Error loading Platform Config: %s\n", err.Error())
	}

	if id.PlatformID != platformid {
		if id.PlatformID != "" {
			fmt.Printf("Platform has changed since initial boot, was: %s now: %s\n",
				id.PlatformID, platformid)
		} else {
			// Initial boot
			id.PlatformID = platformid
			err := saveInitialBootPlatformID(id)
			if err != nil {
				fmt.Printf("PlatformId save error: %s\n", err.Error())
			}
		}
	}

	// Log the Platform ID
	fmt.Printf("PlatformID set to %s\n", platformid)

	_, err = platform.NewPlatform().LoadDefinitions().CreatePlatform(platformid)

	if err != nil {
		fmt.Printf("Error creating platform: %s\n", err.Error())
	}
}
