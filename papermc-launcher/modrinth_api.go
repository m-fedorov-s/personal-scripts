package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"
)

const MODRINTH_VERSION_API = "https://api.modrinth.com/v2/project/%v/version"

type ModrinthVersionInfo struct {
	GameVersions    []string  `json:"game_versions"`
	Loaders         []string  `json:"loaders"`
	Id              string    `json:"id"`
	ProjectId       string    `json:"project_id"`
	AuthorId        string    `json:"author_id"`
	Fatured         bool      `json:"featured"`
	Name            string    `json:"name"`
	VersionNumber   string    `json:"version_number"`
	Changelog       string    `json:"changelog"`
	ChangelogUrl    string    `json:"changelog_url"`
	DatePublished   time.Time `json:"date_published,format:iso8601"`
	Downloads       uint      `json:"downloads"`
	VersionType     string    `json:"version_type"`
	Status          string    `json:"status"`
	RequestedStatus string    `json:"requested_status"`
	Files           []struct {
		Hashes struct {
			Sha512 string `json:"sha512"`
			Sha1   string `json:"sha1"`
		} `json:hashes`
		Url      string `json:"url"`
		Filename string `json:"filename"`
		Primary  bool   `json:"primary"`
		Size     uint   `json:"size"`
		FileType string `json:"file_type"`
	} `json:files`
	Dependencies []struct {
		ProjectId      string `json:"project_id"`
		VersionId      string `json:"version_id"`
		FileName       string `json:"file_name"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
}

func GetLatestVersionFromModrinthImpl(url string, project string, loader string) (ModrinthVersionInfo, error) {
	var infos []ModrinthVersionInfo
	resp, err := http.Get(fmt.Sprintf(url, project))
	if err != nil {
		return ModrinthVersionInfo{}, err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&infos)
	if err != nil {
		return ModrinthVersionInfo{}, err
	}
	filtered := []ModrinthVersionInfo{}
	for _, info := range infos {
		if slices.Contains(info.Loaders, loader) {
			filtered = append(filtered, info)
		}
	}
	if len(filtered) == 0 {
		return ModrinthVersionInfo{}, fmt.Errorf("No versions found")
	}
	result := slices.MaxFunc(filtered, func(l, r ModrinthVersionInfo) int {
		return l.DatePublished.Compare(r.DatePublished)
	})
	return result, nil
}

func GetLatestVersionFromModrinth(project string, loader string) (ModrinthVersionInfo, error) {
	return GetLatestVersionFromModrinthImpl(MODRINTH_VERSION_API, project, loader)
}

func LoadModrinthPlugin(dir, projectName string, oldMeta *VersionInfo) (VersionInfo, error) {
	fmt.Printf("Updating %v...\n", projectName)
	loadDir := dir + "/plugins"
	if oldMeta != nil {
		loadDir += "/update"
	}
	latestVersion, err := GetLatestVersionFromModrinth(projectName, "paper")
	if err != nil {
		return VersionInfo{}, err
	}
	if latestVersion.VersionNumber == oldMeta.Version {
		fmt.Printf("Already newest version of %v\n", projectName)
		return *oldMeta, nil
	}
	fmt.Printf("Downloading %v version %v\n", projectName, latestVersion.VersionNumber)
	checksum := Checksum{
		Type:  SHA512,
		Value: latestVersion.Files[0].Hashes.Sha512,
	}
	url := latestVersion.Files[0].Url
	filename := latestVersion.Files[0].Filename
	err = LoadFileIfDoesNotExist(url, loadDir, filename, checksum)
	if err != nil && !os.IsExist(err) {
		return VersionInfo{}, err
	}
	return VersionInfo{
		Version: latestVersion.VersionNumber,
	}, nil
}
