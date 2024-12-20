package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const PAPER_API_VERSION_URL = "https://api.papermc.io/v2/projects/paper"
const PAPER_API_BUILDS_URL_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds"
const PAPER_API_JAR_DOWNLOAD_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds/%v/downloads/%v"

const VERSIONS_FILE = "version.json"

type VersionInfo struct {
	Version string `json:"version"`
	Build   int    `json:"build"`
}

type VersionsInfo struct {
	PaperVer VersionInfo            `json:"paper"`
	Plugins  map[string]VersionInfo `json:"plugins,omitempty"`
}

// LoadConfig loads the configuration from a JSON file
func LoadVersionsInfo() (VersionsInfo, error) {
	file, err := os.Open(VERSIONS_FILE)
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

func DumpVersionsInfo(info VersionsInfo) error {
	f, err := os.OpenFile(VERSIONS_FILE, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(info)
	return err
}

func LoadFileIfDoesNotExist(url, dir, filename, checksum string) error {
	f, err := os.OpenFile(dir+"/"+filename, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	downloadRes, err := http.Get(url)
	if err != nil {
		panic(err)
		return err
	}
	defer downloadRes.Body.Close()
	_, err = io.Copy(f, downloadRes.Body)
	if err != nil {
		return err
	}
	if checksum == "" {
		return nil
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	if checksum != fmt.Sprintf("%x", h.Sum(nil)) {
		return fmt.Errorf("Sha256 does not match")
	}
	return nil
}

func LoadPaper(dir string) {
	info, err := LoadVersionsInfo()
	if err != nil {
		fmt.Printf("[WARN] Failed to read versions info from %v\n", VERSIONS_FILE)
	}
	resp, err := http.Get(PAPER_API_VERSION_URL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var parsed map[string]interface{}
	json.Unmarshal(body, &parsed)
	version := parsed["versions"].([]interface{})[len(parsed["versions"].([]interface{}))-1].(string)
	if version != info.PaperVer.Version {
		fmt.Printf("A new version of paper found: %v (current is %v). Would you like to update? [y/N]\n", version, info.PaperVer.Version)
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "yes" && strings.ToLower(answer) != "y" {
			version = info.PaperVer.Version
		}
	}
	fmt.Println("Chosen version: " + version)
	buildsResp, err := http.Get(fmt.Sprintf(PAPER_API_BUILDS_URL_TEMPLATE, version))
	if err != nil {
		panic(err)
	}
	defer buildsResp.Body.Close()
	body, err = io.ReadAll(buildsResp.Body)
	if err != nil {
		panic(err)
	}
	json.Unmarshal(body, &parsed)
	build := parsed["builds"].([]interface{})[len(parsed["builds"].([]interface{}))-1].(map[string]interface{})
	buildNumber := int(build["build"].(float64))
	if info.PaperVer.Build > 0 && info.PaperVer.Build == buildNumber {
		fmt.Println("Already latest paper build")
		return
	}
	filename := build["downloads"].(map[string]interface{})["application"].(map[string]interface{})["name"].(string)
	checksum := build["downloads"].(map[string]interface{})["application"].(map[string]interface{})["sha256"].(string)
	url := fmt.Sprintf(PAPER_API_JAR_DOWNLOAD_TEMPLATE, version, buildNumber, filename)
	err = LoadFileIfDoesNotExist(url, dir, filename, checksum)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	err = os.Remove(dir + "/paper.jar")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(err)
	}
	err = os.Symlink(filename, dir+"/paper.jar")
	if err != nil {
		panic(err)
	}
	info.PaperVer.Version = version
	info.PaperVer.Build = buildNumber
	err = DumpVersionsInfo(info)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sucessfuly downloaded %v\n", filename)
}
