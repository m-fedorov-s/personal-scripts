package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPapermcParsingVersionsList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("test_data/papermc_answer_versions.json")
		if err != nil {
			t.Fatalf("Failed to open test file: %v", err)
		}
		defer file.Close()
		n, err := io.Copy(w, file)
		if err != nil {
			t.Errorf("Failed to copy all data to the mock response (copied %v bytes): %v", n, err)
		}
	}))
	defer ts.Close()
	version, err := GetLatestVersionInfoPaperImpl(fmt.Sprintf("%v/%%v", ts.URL), "paper")
	if err != nil {
		t.Error(err)
	}
	expected := "1.21.10"
	if version.Version.Id.String() != expected {
		t.Errorf("Expected version %v, found %v", expected, version.Version.Id.String())
	}
}

func TestPapermcParsingBuildInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("test_data/papermc_answer_builds.json")
		if err != nil {
			t.Fatalf("Failed to open test file: %v", err)
		}
		defer file.Close()
		n, err := io.Copy(w, file)
		if err != nil {
			t.Errorf("Failed to copy all data to the mock response (copied %v bytes): %v", n, err)
		}
	}))
	defer ts.Close()
	build, err := GetLatestBuildInfoPaperImpl(fmt.Sprintf("%v/%%v/%%v", ts.URL), "paper", "1.21.10")
	if err != nil {
		t.Error(err)
	}
	expected := 113
	if build.Id != expected {
		t.Errorf("Expected build %v, found %v", expected, build.Id)
	}
}
