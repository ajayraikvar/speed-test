// Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := newLogStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	updates := newUpdateChecker()
	return httptest.NewServer(newHandler(serverInfo{
		Name: appName, Version: appVersion, Developer: developer,
		Website: "https://www.ajayraikvar.com", Repository: repoURL,
		Hostname: "test-host", Port: 8080, Addresses: []string{"192.168.1.10"},
	}, store, updates))
}

func TestHomePageAndBranding(t *testing.T) {
	s := testServer(t)
	defer s.Close()
	resp, err := http.Get(s.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Speed Test") || !strings.Contains(string(body), "Ajay Raikvar") {
		t.Fatal("home page is missing product or creator branding")
	}
	if strings.Contains(string(body), "LAN Speed Test") {
		t.Fatal("old product branding remains in the home page")
	}
}

func TestEmbeddedAssetsAndInfo(t *testing.T) {
	s := testServer(t)
	defer s.Close()
	for _, path := range []string{"/app.js", "/style.css", "/admin.js", "/admin.css"} {
		resp, err := http.Get(s.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d", path, resp.StatusCode)
		}
	}
	resp, err := http.Get(s.URL + "/api/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var info serverInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Port != 8080 || info.Developer != developer || info.Name != "Speed Test" || len(info.Addresses) != 1 {
		t.Fatalf("unexpected info response: %+v", info)
	}
}

func TestPingAndUpdateEndpoints(t *testing.T) {
	s := testServer(t)
	defer s.Close()
	resp, err := http.Get(s.URL + "/api/ping")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("ping status = %d", resp.StatusCode)
	}
	resp, err = http.Get(s.URL + "/api/update")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var state updateSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.CurrentVersion != appVersion {
		t.Fatalf("current version = %q", state.CurrentVersion)
	}
}

func TestUploadCountsBytes(t *testing.T) {
	s := testServer(t)
	defer s.Close()
	resp, err := http.Post(s.URL+"/api/upload", "application/octet-stream", strings.NewReader("test-data"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result struct {
		Received int64 `json:"receivedBytes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || result.Received != int64(len("test-data")) {
		t.Fatalf("status=%d received=%d", resp.StatusCode, result.Received)
	}
}

func TestResultSavedToPersistentHostLog(t *testing.T) {
	dir := t.TempDir()
	store, err := newLogStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	updates := newUpdateChecker()
	h := newHandler(serverInfo{Name: appName, Version: appVersion}, store, updates)
	payload := `{"durationSeconds":10,"downloadMbps":940.2,"uploadMbps":880.5,"latencyMs":1.2,"jitterMs":0.2,"downloadBytes":1175000000,"uploadBytes":1100000000,"clientVersion":"1.0.0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/results", strings.NewReader(payload))
	req.RemoteAddr = "192.168.1.20:54321"
	req.Header.Set("User-Agent", "Test Browser")
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rw.Code, rw.Body.String())
	}
	logPath := filepath.Join(dir, "logs", "tests.jsonl")
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var row testLog
	if err := json.Unmarshal(bytesTrimSpace(contents), &row); err != nil {
		t.Fatal(err)
	}
	if row.ClientIP != "192.168.1.20" || row.DownloadMbps != 940.2 || row.Status != "completed" {
		t.Fatalf("unexpected log row: %+v", row)
	}
}

func bytesTrimSpace(b []byte) []byte { return []byte(strings.TrimSpace(string(b))) }

func TestAdminDashboardIsLoopbackOnly(t *testing.T) {
	s := testServer(t)
	defer s.Close()
	store, err := newLogStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hLocal := newHandler(serverInfo{}, store, newUpdateChecker())
	localReq := httptest.NewRequest(http.MethodGet, "/admin", nil)
	localReq.RemoteAddr = "127.0.0.1:45678"
	localRW := httptest.NewRecorder()
	hLocal.ServeHTTP(localRW, localReq)
	if localRW.Code != http.StatusTemporaryRedirect {
		t.Fatalf("localhost admin status = %d", localRW.Code)
	}
	store2, err2 := newLogStore(t.TempDir())
	if err2 != nil {
		t.Fatal(err2)
	}
	h := newHandler(serverInfo{}, store2, newUpdateChecker())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs", nil)
	req.RemoteAddr = "192.168.1.42:50000"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusForbidden {
		t.Fatalf("remote logs status = %d", rw.Code)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"1.1.0", "1.0.9", 1}, {"v1.0.0", "1.0.0", 0}, {"1.0.0", "1.0.1", -1}}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestUpdateCheckerOfflineDoesNotBlockStartup(t *testing.T) {
	u := newUpdateChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	u.refresh(ctx)
	if u.Snapshot().CurrentVersion != appVersion {
		t.Fatal("offline state lost current version")
	}
}

func TestAdminCSVExportIsLoopbackOnly(t *testing.T) {
	store, err := newLogStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(testLog{ID: "test-123", Timestamp: time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC), ClientIP: "192.168.1.8", DownloadMbps: 100.5, Status: "completed"}); err != nil {
		t.Fatal(err)
	}
	h := newHandler(serverInfo{}, store, newUpdateChecker())
	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs?format=csv", nil)
	req.RemoteAddr = "127.0.0.1:60000"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK || !strings.Contains(rw.Body.String(), "test-123") || !strings.Contains(rw.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("unexpected CSV response: code=%d body=%s", rw.Code, rw.Body.String())
	}
}
