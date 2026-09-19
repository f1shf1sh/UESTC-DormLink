package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunWatchWithConfigPath(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/cgi-bin/rad_user_info" {
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
		fmt.Fprintf(w, "%s({\"error\":\"ok\"})", r.URL.Query().Get("callback"))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "router.json")
	data, err := json.Marshal(map[string]string{
		"username": "dummy", "password": "fake-password", "portal_ip": server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	// Cancel when the first status has been reported, without a real polling delay.
	writer := cancelWriter{Writer: &output, cancel: cancel}
	if err := run(ctx, []string{"-watch", "-config", path, "-interval", "1h"}, writer, &output); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want one status request", requests.Load())
	}
	if strings.Contains(output.String(), "fake-password") {
		t.Fatal("output contains password")
	}
}

type cancelWriter struct {
	Writer *bytes.Buffer
	cancel context.CancelFunc
}

func (w cancelWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	w.cancel()
	return n, err
}

func TestRunRejectsInvalidFlags(t *testing.T) {
	for _, args := range [][]string{
		{"-watch", "-interval", "0"},
		{"-watch", "-interval", "-1s"},
		{"-interval", "invalid"},
		{"unexpected"},
	} {
		var output bytes.Buffer
		if err := run(context.Background(), args, &output, &output); err == nil {
			t.Fatalf("run(%v) succeeded", args)
		}
	}
}

func TestRunHelpDoesNotLoadConfig(t *testing.T) {
	var output bytes.Buffer
	if err := run(context.Background(), []string{"-help"}, &output, &output); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("help error = %v", err)
	}
	for _, option := range []string{"-watch", "-config", "-interval"} {
		if !strings.Contains(output.String(), option) {
			t.Errorf("help is missing %s", option)
		}
	}
}

func TestRunCanceledWatchStops(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	cancel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"username":"dummy","password":"fake","portal_ip":"http://127.0.0.1:1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run(ctx, []string{"-watch", "-config", path}, &output, &output); err != nil {
		t.Fatal(err)
	}
}
