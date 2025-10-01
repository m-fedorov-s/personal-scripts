package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const HANGAR_API_LATEST_RELEASE = "https://hangar.papermc.io/api/v1/projects/%v/latestrelease"
const HANGAR_API_VERSION_INFO = "https://hangar.papermc.io/api/v1/projects/%v/versions/%v"

type HangarDownloadInfo struct {
	DownloadUrl string `json:"downloadUrl"`
	ExternalUrl string `json:"externalUrl"`
	FileInfo    struct {
		Name       string `json:"name"`
		Sha256Hash string `json:"sha256Hash"`
		SizeBytes  uint   `json:"sizeBytes"`
	} `json:fileInfo"`
}

type HangarDependencyInfo struct {
	ExternalUrl string `json:"externalUrl"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	ProjectId   int    `json:"projectId"`
	Required    bool   `json:"required"`
}

type HangarVersionInfo struct {
	Author  string `json:"author"`
	Channel struct {
		Color       string    `json:"color"`
		CreatedAt   time.Time `json:"createdAt"`
		Description string    `json:"description"`
		Flags       []string  `json:"description"`
		Name        string    `json:"name"`
	} `json:"channel"`
	CreatedAt                     time.Time                         `json:"createdAt"`
	Description                   string                            `json:"description"`
	Downloads                     map[string]HangarDownloadInfo     `json:"downloads"`
	Id                            int                               `json:"id"`
	MemberNames                   []string                          `json:"memberNames"`
	Name                          string                            `json:"name"`
	PinnedStatus                  string                            `json:"pinnedStatus"`
	PlatformDependencies          map[string][]string               `json:"platformDependencies"`
	PlatformDependenciesFormatted map[string][]string               `json:"platformDependenciesFormatted"`
	PluginDependencies            map[string][]HangarDependencyInfo `json:"pluginDependencies"`
	ProjectId                     int                               `json:"projectId"`
	ReviewState                   string                            `json:"reviewState"`
	Stats                         struct {
		PlatformDownloads map[string]int `json:"platformDownloads"`
		TotalDownloads    int            `json:"totalDownloads"`
	} `json:"stats"`
	Visibility string `json:visibility"`
}

func GetLatestVersionFromHangarImpl(url string, project string) (string, error) {
	resp, err := http.Get(fmt.Sprintf(url, project))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Bad status code: %v", resp.StatusCode)
	}
	value, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(value[:]), nil
}

func GetLatestVersionFromHangar(project string) (string, error) {
	return GetLatestVersionFromHangarImpl(HANGAR_API_LATEST_RELEASE, project)
}

func GetVersionInfoHangarImpl(url, project, version string) (HangarVersionInfo, error) {
	resp, err := http.Get(fmt.Sprintf(url, project, version))
	if err != nil {
		return HangarVersionInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return HangarVersionInfo{}, fmt.Errorf("Bad status code: %v", resp.StatusCode)
	}
	var info HangarVersionInfo
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&info)
	if err != nil {
		return HangarVersionInfo{}, err
	}
	return info, nil
}

func GetVersionInfoHangar(project, version string) (HangarVersionInfo, error) {
	return GetVersionInfoHangarImpl(HANGAR_API_VERSION_INFO, project, version)
}

func LoadHangarPlugin(dir, projectName string, oldMeta *VersionInfo) (VersionInfo, error) {
	fmt.Printf("Updating %v...\n", projectName)
	loadDir := dir + "/plugins"
	if oldMeta != nil {
		loadDir += "/update"
	}
	latestVersion, err := GetLatestVersionFromHangar(projectName)
	if err != nil {
		return VersionInfo{}, err
	}
	if latestVersion == oldMeta.Version {
		fmt.Printf("Already newest version of %v\n", projectName)
		return *oldMeta, nil
	}
	versionMeta, err := GetVersionInfoHangar(projectName, latestVersion)
	if err != nil {
		return VersionInfo{}, err
	}
	downloadMeta, ok := versionMeta.Downloads["PAPER"]
	if !ok {
		return VersionInfo{}, fmt.Errorf("Failed to find download for version %v", latestVersion)
	}
	if downloadMeta.ExternalUrl != "" || downloadMeta.DownloadUrl == "" {
		return VersionInfo{}, fmt.Errorf("Detected hangar external download!")
	}
	fmt.Printf("Downloading %v version %v\n", projectName, latestVersion)
	checksum := Checksum{
		Type:  SHA256,
		Value: downloadMeta.FileInfo.Sha256Hash,
	}
	url := downloadMeta.DownloadUrl
	filename := downloadMeta.FileInfo.Name
	err = LoadFileIfDoesNotExist(url, loadDir, filename, checksum)
	if err != nil && !os.IsExist(err) {
		return VersionInfo{}, err
	}
	return VersionInfo{
		Version: latestVersion,
	}, err
}
