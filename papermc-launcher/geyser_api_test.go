package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseGeyserResponceVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("test_data/geyser_answer_project_info.json")
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
	url := fmt.Sprintf("%v/%%v", ts.URL)
	version, err := GetLatestGeyserVersionInfoImpl(url, "geyser")
	if err != nil {
		t.Fatal(err)
	}
	expected := "2.9.0"
	if version != expected {
		t.Errorf("Got wrong version! Expected %v, got %v", expected, version)
	}
}

func TestParseGeyserResponceBuild(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("test_data/geyser_answer_version_info.json")
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
	url := fmt.Sprintf("%v/%%v/%%v", ts.URL)
	info, err := GetLatestGeyserBuildImpl(url, "geyser", "2.9.0")
	if err != nil {
		t.Fatal(err)
	}
	expected := 984
	if info.Build != expected {
		t.Errorf("Got wrong version! Expected %v, got %v", expected, info.Build)
	}
}
