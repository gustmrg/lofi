package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckLatestDetectsNewerRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/gustmrg/lofi/releases/latest" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[]}`))
	}))
	defer srv.Close()

	res, err := (Client{APIBase: srv.URL}).CheckLatest(context.Background(), "0.1.0")
	if err != nil {
		t.Fatalf("CheckLatest(): %v", err)
	}
	if !res.Newer || res.Latest != "0.2.0" || res.Current != "0.1.0" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestNoticeCheckerUsesFreshCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "update-check.json")
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	if err := writeCache(cachePath, updateCache{
		CheckedAt: now.Add(-time.Hour),
		Latest:    "0.2.0",
		Notice:    "cached notice",
	}); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	called := false
	checker := NoticeChecker{
		Client: Client{HTTPClient: &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, fmt.Errorf("should not call network")
		})}},
		Current:   "0.1.0",
		CachePath: cachePath,
		Now:       func() time.Time { return now },
	}
	notice, err := checker.Notice(context.Background())
	if err != nil {
		t.Fatalf("Notice(): %v", err)
	}
	if called {
		t.Fatal("fresh cache should avoid network")
	}
	if notice != "cached notice" {
		t.Fatalf("notice = %q", notice)
	}
}

func TestUpdaterRunReplacesExecutable(t *testing.T) {
	archive := makeArchive(t, "new binary")
	sum := sha256.Sum256(archive)
	checksums := hex.EncodeToString(sum[:]) + "  lofi_v0.2.0_darwin_arm64.tar.gz\n"

	baseURL := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/gustmrg/lofi/releases/latest":
			fmt.Fprintf(w, `{"tag_name":"v0.2.0","assets":[{"name":"lofi_v0.2.0_darwin_arm64.tar.gz","browser_download_url":"%s/archive"},{"name":"checksums.txt","browser_download_url":"%s/checksums"}]}`, baseURL, baseURL)
		case "/archive":
			_, _ = w.Write(archive)
		case "/checksums":
			_, _ = w.Write([]byte(checksums))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	baseURL = srv.URL

	exe := filepath.Join(t.TempDir(), "lofi")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}

	res, err := (Updater{
		Client:         Client{APIBase: srv.URL},
		Current:        "0.1.0",
		GOOS:           "darwin",
		GOARCH:         "arm64",
		ExecutablePath: exe,
	}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run(): %v", err)
	}
	if !res.Newer || res.Latest != "0.2.0" {
		t.Fatalf("unexpected result: %+v", res)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "new binary" {
		t.Fatalf("exe = %q", got)
	}
	backup, err := os.ReadFile(exe + ".bak")
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != "old binary" {
		t.Fatalf("backup = %q", backup)
	}
}

func TestUpdaterRunRejectsChecksumMismatch(t *testing.T) {
	archive := makeArchive(t, "new binary")
	baseURL := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/gustmrg/lofi/releases/latest":
			fmt.Fprintf(w, `{"tag_name":"v0.2.0","assets":[{"name":"lofi_v0.2.0_linux_amd64.tar.gz","browser_download_url":"%s/archive"},{"name":"checksums.txt","browser_download_url":"%s/checksums"}]}`, baseURL, baseURL)
		case "/archive":
			_, _ = w.Write(archive)
		case "/checksums":
			_, _ = w.Write([]byte(strings.Repeat("0", 64) + "  lofi_v0.2.0_linux_amd64.tar.gz\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	baseURL = srv.URL

	exe := filepath.Join(t.TempDir(), "lofi")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	_, err := (Updater{
		Client:         Client{APIBase: srv.URL},
		Current:        "0.1.0",
		GOOS:           "linux",
		GOARCH:         "amd64",
		ExecutablePath: exe,
	}).Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Run() error = %v", err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read exe: %v", err)
	}
	if string(got) != "old binary" {
		t.Fatalf("exe should be unchanged, got %q", got)
	}
}

func makeArchive(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte(content)
	if err := tw.WriteHeader(&tar.Header{
		Name: "lofi",
		Mode: 0o755,
		Size: int64(len(body)),
	}); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("write body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	return buf.Bytes()
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
