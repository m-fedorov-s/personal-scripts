package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHangarParsing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("./test_data/hangar_answer.json")
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
	info, err := GetVersionInfoHangarImpl(fmt.Sprintf("%v/%%v/%%v", ts.URL), "test_project", "5.4.2")
	if err != nil {
		t.Error(err)
	}
	expected := "5.4.2"
	if info.Name != expected {
		t.Errorf("Expected version %v, found %v", expected, info.Name)
	}
}
