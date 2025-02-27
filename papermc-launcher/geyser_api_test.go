package main

import (
	"os"
	"testing"
)

func TestUpdateGeyser(t *testing.T) {
	dir := t.TempDir()
	versionsFile := dir + "/versions.json"
	err := os.MkdirAll(dir+"/plugins/update", 0777)
	if err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	info := VersionsInfo{
		PaperVer: VersionInfo{
			Version: "1.21.4",
			Build:   121,
		},
		Plugins: map[string]VersionInfo{
			"geyser": VersionInfo{
				Version: "2.6.1",
				Build:   701,
			},
		},
	}
	err = DumpVersionsInfo(info, versionsFile)
	if err != nil {
		t.Fatalf("Failed to write versions info: %v", err)
	}
	err = LoadGeyserPlugin(dir, "geyser", versionsFile)
	if err != nil {
		t.Errorf("Failed to load geyser: %v", err)
	}
}
func TestUpdateFloodgate(t *testing.T) {
	dir := t.TempDir()
	versionsFile := dir + "/versions.json"
	err := os.MkdirAll(dir+"/plugins/update", 0777)
	if err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	info := VersionsInfo{
		PaperVer: VersionInfo{
			Version: "1.21.4",
			Build:   121,
		},
		Plugins: map[string]VersionInfo{
			"floodgate": VersionInfo{
				Version: "2.2.4",
				Build:   42,
			},
		},
	}
	err = DumpVersionsInfo(info, versionsFile)
	if err != nil {
		t.Fatalf("Failed to write versions info: %v", err)
	}
	err = LoadGeyserPlugin(dir, "floodgate", versionsFile)
	if err != nil {
		t.Errorf("Failed to load floodgate: %v", err)
	}
}
