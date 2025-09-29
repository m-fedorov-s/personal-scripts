package main

import "fmt"

func UpdateServer(workDir string) error {
	metaLocation := fmt.Sprintf("./%v/%v", workDir, METADATA_FILE)
	meta, err := LoadVersionsInfo(metaLocation)
	if err != nil {
		return err
	}
	if meta.Loader.Type != "paper" {
		return fmt.Errorf("Launcher %v not supported.", meta.Loader.Type)
	}
	oldMeta := meta.Loader.VersionMeta
	newMeta, err := LoadPaper(workDir, oldMeta)
	if err != nil {
		return err
	}
	meta.Loader.VersionMeta = newMeta
	for i, _ := range meta.Plugins {
		pluginMeta := meta.Plugins[i]
		var err error
		var newMeta VersionInfo
		switch pluginMeta.Provider {
		case "geyser":
			newMeta, err = LoadGeyserPlugin(workDir, pluginMeta.Name, &pluginMeta.VersionMeta)
		case "hangar":
			newMeta, err = LoadHangarPlugin(workDir, pluginMeta.Name, &pluginMeta.VersionMeta)
		case "modrinth":
			newMeta, err = LoadModrinthPlugin(workDir, pluginMeta.Name, &pluginMeta.VersionMeta)
		default:
			fmt.Printf("Unknown plugins provider: %v\n", pluginMeta.Provider)
		}
		if err != nil {
			fmt.Printf("[WARN] Failed to update plugin %v: %v", pluginMeta.Name, err)
			continue
		}
		meta.Plugins[i].VersionMeta = newMeta
	}
	DumpVersionsInfo(meta, metaLocation)
	return nil
}
