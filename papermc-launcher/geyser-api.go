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

func LoadGeyser(dir string) error {
	fmt.Println("Loading geyser")
	info, err := LoadVersionsInfo()
	if err != nil {
		fmt.Printf("[WARN] Failed to read versions info from %v\n", VERSIONS_FILE)
	}
	loadDir := dir + "/plugins"
	ver, ok := info.Plugins["geyser"]
	if ok {
		loadDir += "/update"
	}
	latestVer, err := GetLatestVersion("geyser")
	if err != nil {
		return err
	}
	latestBuild, err := GetLatestBuild("geyser", latestVer)
	if err != nil {
		return err
	}
	if ver.Build > 0 && ver.Build == latestBuild.Build {
		fmt.Println("Geyser already latest build")
		return nil
	}
	checksum := latestBuild.Downloads["spigot"].Sha256
	url := fmt.Sprintf(GEYSER_API_DOWNLOAD_URL, "geyser", latestVer, latestBuild.Build, "spigot")
	fmt.Printf("Hitting url %v\n", url)
	err = LoadFileIfDoesNotExist(url, loadDir, "Geyser-Spigot.jar", checksum)
	if err != nil && !os.IsExist(err) {
		return err
	}
	if info.Plugins == nil {
		info.Plugins = make(map[string]VersionInfo)
	}
	info.Plugins["geyser"] = VersionInfo{
		Version: latestVer,
		Build:   latestBuild.Build,
	}
	err = DumpVersionsInfo(info)

	return err
}
