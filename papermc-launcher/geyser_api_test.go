package main

import (
	"os"
	"testing"
)

func TestUpdateGeyser(t *testing.T) {
	dir := t.TempDir()
	err := os.MkdirAll(dir+"/plugins/update", 0777)
	if err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	oldMeta := VersionInfo{
		Version: "2.6.1",
		Build:   701,
	}
	newMeta, err := LoadGeyserPlugin(dir, "geyser", &oldMeta)
	if err != nil {
		t.Errorf("Failed to load geyser: %v", err)
	}
	if newMeta.Build < 702 {
		t.Errorf("Wring build number in new meta: %v", newMeta.Build)
	}
}
func TestUpdateFloodgate(t *testing.T) {
	dir := t.TempDir()
	err := os.MkdirAll(dir+"/plugins/update", 0777)
	if err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	oldMeta := &VersionInfo{
		Version: "2.2.4",
		Build:   42,
	}
	_, err = LoadGeyserPlugin(dir, "floodgate", oldMeta)
	if err != nil {
		t.Errorf("Failed to load floodgate: %v", err)
	}
}
