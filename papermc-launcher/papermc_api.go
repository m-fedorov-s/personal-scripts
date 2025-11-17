package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

const PAPER_API_VERSIONS = "https://fill.papermc.io/v3/projects/%v/versions"
const PAPER_API_BUILDS = "https://fill.papermc.io/v3/projects/%v/versions/%v/builds"

type Date time.Time

func (t *Date) UnmarshalJSON(b []byte) (err error) {
	parsed, err := time.Parse(`"2006-01-02"`, string(b))
	if err == nil {
		*t = Date(parsed)
	}
	return
}

type PaperVersionInfo struct {
	Builds  []int `json:"builds"`
	Version struct {
		Id   version.Version `json:"id"`
		Java struct {
			Flags struct {
				Recommended []string `json:"recommended"`
			} `json:"flags"`
			Version struct {
				Minimum int `json:"minimum"`
			} `json:"version"`
		} `json:"java"`
		Support struct {
			End    Date   `json:"end"`
			Status string `json:"status"`
		} `json:"support"`
	} `json:"version"`
}

type PaperVersionsList struct {
	Versions []PaperVersionInfo `json:"versions"`
}

type PaperBuildInfo struct {
	Id      int       `json:"id"`
	Time    time.Time `json:"time"`
	Channel string    `json:"channel"`
	Commits []struct {
		Sha     string    `json:"sha"`
		Time    time.Time `json:"time"`
		Message string    `json:"message"`
	} `json:"commits"`
	Downloads map[string]struct {
		Name      string `json:"name"`
		Checksums struct {
			Sha256 string `json:"sha256"`
		} `json:"checksums"`
		Size uint32 `json:"size"`
		Url  string `json:"url"`
	} `json:"downloads"`
}

func GetLatestVersionInfoPaperImpl(url, project string) (PaperVersionInfo, error) {
	resp, err := http.Get(fmt.Sprintf(url, project))
	if err != nil {
		return PaperVersionInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PaperVersionInfo{}, fmt.Errorf("Bad status code: %v", resp.StatusCode)
	}
	var infos PaperVersionsList
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&infos)
	if err != nil {
		return PaperVersionInfo{}, err
	}
	result := slices.MaxFunc(infos.Versions, func(l, r PaperVersionInfo) int {
		return l.Version.Id.Compare(&r.Version.Id)
	})
	return result, nil
}

func GetLatestVersionInfoPaper(project string) (PaperVersionInfo, error) {
	return GetLatestVersionInfoPaperImpl(PAPER_API_VERSIONS, project)
}

func GetLatestBuildInfoPaperImpl(url, project, v string) (PaperBuildInfo, error) {
	resp, err := http.Get(fmt.Sprintf(url, project, v))
	if err != nil {
		return PaperBuildInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PaperBuildInfo{}, fmt.Errorf("Bad status code: %v", resp.StatusCode)
	}
	var infos []PaperBuildInfo
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&infos)
	if err != nil {
		return PaperBuildInfo{}, err
	}
	result := slices.MaxFunc(infos, func(l, r PaperBuildInfo) int {
		return cmp.Compare(l.Id, r.Id)
	})
	return result, nil
}

func GetLatestBuildInfoPaper(project, version string) (PaperBuildInfo, error) {
	return GetLatestBuildInfoPaperImpl(PAPER_API_BUILDS, project, version)
}

func LoadPaper(dir string, oldMeta VersionInfo) (VersionInfo, error) {
	latestVersion, err := GetLatestVersionInfoPaper("paper")
	if err != nil {
		return VersionInfo{}, err
	}
	chosenVersion := latestVersion.Version.Id.String()
	if latestVersion.Version.Id.String() != oldMeta.Version {
		fmt.Printf("A new version of paper found: %v (current is %v). Would you like to update? [y/N]\n", latestVersion.Version.Id.String(), oldMeta.Version)
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "yes" && strings.ToLower(answer) != "y" {
			chosenVersion = oldMeta.Version
		}
	}
	fmt.Println("Chosen version: " + chosenVersion)
	latestBuild, err := GetLatestBuildInfoPaper("paper", chosenVersion)
	if err != nil {
		return VersionInfo{}, err
	}
	if chosenVersion == oldMeta.Version && latestBuild.Id == oldMeta.Build {
		fmt.Println("Already latest paper build")
		return oldMeta, nil
	}
	loaded := ""
	for server, info := range latestBuild.Downloads {
		fmt.Printf("Loading from %v...\n", server)
		checksum := Checksum{
			Type:  SHA256,
			Value: info.Checksums.Sha256,
		}
		err := LoadFileIfDoesNotExist(info.Url, dir, info.Name, checksum)
		if err != nil && !os.IsExist(err) {
			fmt.Printf("Failed: %v\n", err)
			continue
		}
		loaded = info.Name
	}
	if loaded == "" {
		return VersionInfo{}, fmt.Errorf("Failed to load file from any server")
	}
	err = os.Remove(dir + "/paper.jar")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return VersionInfo{}, err
	}
	err = os.Symlink(loaded, dir+"/paper.jar")
	if err != nil {
		return VersionInfo{}, err
	}
	fmt.Printf("Sucessfuly downloaded %v\n", loaded)
	return VersionInfo{
		Version: chosenVersion,
		Build:   latestBuild.Id,
	}, nil
}
