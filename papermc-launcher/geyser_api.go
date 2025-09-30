package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const GEYSER_API_PROJECT_INFO = "https://download.geysermc.org/v2/projects/%v"
const GEYSER_API_VERSION_INFO = "https://download.geysermc.org/v2/projects/%v/versions/%v/builds"
const GEYSER_API_DOWNLOAD_URL = "https://download.geysermc.org/v2/projects/%v/versions/%v/builds/%v/downloads/%v"

// https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot

type ProjectInfo struct {
	ProjectID   string   `json:"project_id"`
	ProjectName string   `json:"project_name"`
	Versions    []string `json:"versions"`
}

type DownloadInfo struct {
	Name   string `json:"name"`
	Sha256 string `json:"sha256"`
}

type BuildInfo struct {
	Build    int    `json:"build"`
	Time     string `json:"time"`
	Channel  string `json:"channel"`
	Promoted bool   `json:"promoted"`
	Changes  []struct {
		Commit  string `json:"commit"`
		Summary string `json:"summary"`
		Message string `json:"message"`
	} `json:"changes"`
	Downloads map[string]DownloadInfo `json:"downloads"`
}

type GeyserVersionInfo struct {
	ProjectID   string      `json:"project_id"`
	ProjectName string      `json:"project_name"`
	Version     string      `json:"version"`
	Builds      []BuildInfo `json:"builds"`
}

func GetLatestVersion(id string) (string, error) {
	var info ProjectInfo
	resp, err := http.Get(fmt.Sprintf(GEYSER_API_PROJECT_INFO, id))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&info)
	if err != nil {
		return "", err
	}
	if len(info.Versions) == 0 {
		return "", fmt.Errorf("No versions found")
	}
	return info.Versions[len(info.Versions)-1], nil
}

func GetLatestBuild(id, ver string) (BuildInfo, error) {
	var info GeyserVersionInfo
	resp, err := http.Get(fmt.Sprintf(GEYSER_API_VERSION_INFO, id, ver))
	if err != nil {
		return BuildInfo{}, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&info)
	if err != nil {
		return BuildInfo{}, err
	}
	if len(info.Builds) == 0 {
		return BuildInfo{}, fmt.Errorf("No builds found")
	}
	return info.Builds[len(info.Builds)-1], nil
}

func LoadGeyserPlugin(dir, pluginName string, oldMeta *VersionInfo) (VersionInfo, error) {
	fmt.Printf("Updating %v...\n", pluginName)
	loadDir := dir + "/plugins"
	if oldMeta != nil {
		loadDir += "/update"
	}
	latestVer, err := GetLatestVersion(pluginName)
	if err != nil {
		return VersionInfo{}, err
	}
	latestBuild, err := GetLatestBuild(pluginName, latestVer)
	if err != nil {
		return VersionInfo{}, err
	}
	if oldMeta != nil && oldMeta.Build == latestBuild.Build {
		fmt.Printf("Already latest build of %v\n", pluginName)
		return *oldMeta, nil
	}
	platform := "spigot"
	fmt.Printf("Downloading %v version %v build #%v for %v\n", pluginName, latestVer, latestBuild.Build, platform)
	checksum := Checksum{
		Type:  SHA256,
		Value: latestBuild.Downloads["spigot"].Sha256,
	}
	url := fmt.Sprintf(GEYSER_API_DOWNLOAD_URL, pluginName, latestVer, latestBuild.Build, platform)
	filename := fmt.Sprintf("%v-spigot.jar", pluginName)
	err = LoadFileIfDoesNotExist(url, loadDir, filename, checksum)
	if err != nil && !os.IsExist(err) {
		return VersionInfo{}, err
	}
	return VersionInfo{
		Version: latestVer,
		Build:   latestBuild.Build,
	}, nil
}
