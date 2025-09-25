package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestModringthParsing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("modrinth_answer.json")
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
	version, err := GetLatestVersionFromModrinthImpl(fmt.Sprintf("%v/%%v", ts.URL), "test_project", "paper")
	if err != nil {
		t.Error(err)
	}
	expected := "2.8.3-b936"
	if version.VersionNumber != expected {
		t.Errorf("Expected version %v, found %v", expected, version.VersionNumber)
	}
}
