package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const METADATA_FILE = "pl_metadata.json"

type VersionInfo struct {
	Version string `json:"version"`
	Build   int    `json:"build,omitempty"`
}

type PluginMeta struct {
	Name        string      `json:"name"`
	Provider    string      `json:"provider"`
	VersionMeta VersionInfo `json:"version_meta"`
}

type VersionsInfo struct {
	Loader struct {
		Type        string      `json:"name"`
		VersionMeta VersionInfo `json:"version_meta"`
	} `json:"loader"`
	Plugins []PluginMeta `json:"plugins,omitempty"`
}

func LoadVersionsInfo(versionsFile string) (VersionsInfo, error) {
	file, err := os.Open(versionsFile)
	if err != nil {
		return VersionsInfo{}, fmt.Errorf("error opening versions_info file: %w", err)
	}
	defer file.Close()

	var info VersionsInfo
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&info); err != nil {
		return VersionsInfo{}, fmt.Errorf("error decoding versions_info: %w", err)
	}

	return info, nil
}

func DumpVersionsInfo(info VersionsInfo, versionsFile string) error {
	f, err := os.OpenFile(versionsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(info)
	return err
}
