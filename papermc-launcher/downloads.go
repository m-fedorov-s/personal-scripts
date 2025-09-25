package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strings"
)

const PAPER_API_VERSION_URL = "https://api.papermc.io/v2/projects/paper"
const PAPER_API_BUILDS_URL_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds"
const PAPER_API_JAR_DOWNLOAD_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds/%v/downloads/%v"

const VERSIONS_FILE = "versions.json"

type ChecksumType int

const (
	SHA1 ChecksumType = iota
	SHA256
	SHA512
)

type Checksum struct {
	Type  ChecksumType
	Value string
}

type VersionInfo struct {
	Version string `json:"version"`
	Build   int    `json:"build"`
}

type VersionsInfo struct {
	PaperVer VersionInfo            `json:"paper"`
	Plugins  map[string]VersionInfo `json:"plugins,omitempty"`
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
	f, err := os.OpenFile(versionsFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(info)
	return err
}

func LoadFileIfDoesNotExist(url, dir, filename string, checksum Checksum) error {
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
	if checksum.Value == "" || (checksum.Type != SHA1 && checksum.Type != SHA256 && checksum.Type != SHA512) {
		return nil
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	var h hash.Hash
	switch checksum.Type {
	case SHA1:
		h = sha1.New()
	case SHA256:
		h = sha256.New()
	case SHA512:
		h = sha512.New()
	}
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	if checksum.Value != fmt.Sprintf("%x", h.Sum(nil)) {
		return fmt.Errorf("Sha256 does not match")
	}
	return nil
}

func LoadPaper(dir string) {
	info, err := LoadVersionsInfo(VERSIONS_FILE)
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
	checksum := Checksum{
		Type:  SHA256,
		Value: build["downloads"].(map[string]interface{})["application"].(map[string]interface{})["sha256"].(string),
	}
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
	err = DumpVersionsInfo(info, VERSIONS_FILE)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sucessfuly downloaded %v\n", filename)
}
