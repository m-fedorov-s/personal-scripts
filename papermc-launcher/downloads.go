package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
)

const PAPER_API_VERSION_URL = "https://api.papermc.io/v2/projects/paper"
const PAPER_API_BUILDS_URL_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds"
const PAPER_API_JAR_DOWNLOAD_TEMPLATE = "https://api.papermc.io/v2/projects/paper/versions/%v/builds/%v/downloads/%v"

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
		return fmt.Errorf("Checksum does not match")
	}
	return nil
}
