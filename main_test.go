// Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestHandler(t *testing.T) (http.Handler, *logStore, *clientRegistry) {
	t.Helper()
	store, err := newLogStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clients := newClientRegistry(2)
	updates := newUpdateChecker()
	info := serverInfo{Name: appName, Version: appVersion, Developer: developer, Website: "https://www.ajayraikvar.com", Repository: repoURL, Hostname: "test-host", Port: 8080, Addresses: []string{"192.168.1.10"}, MaxClients: 2}
	return newHandler(info, store, updates, clients), store, clients
}

func TestHomeAndEmbeddedDashboardAssets(t *testing.T) {
	h, _, _ := newTestHandler(t)
	for _, path := range []string{"/", "/app.js", "/style.css", "/admin.js", "/admin.css"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		r.RemoteAddr = "127.0.0.1:1000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, w.Code)
		}
	}
}

func TestInfoAndUpdateEndpoints(t *testing.T) {
	h, _, _ := newTestHandler(t)
	r := httptest.NewRequest(http.MethodGet, "/api/info", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var info serverInfo
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.Version != appVersion || info.MaxClients != 2 || info.Name != "Speed Test" {
		t.Fatalf("unexpected info: %+v", info)
	}
	r = httptest.NewRequest(http.MethodGet, "/api/update", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var update updateSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &update); err != nil {
		t.Fatal(err)
	}
	if update.CurrentVersion != appVersion {
		t.Fatalf("current version = %s", update.CurrentVersion)
	}
}

func heartbeat(t *testing.T, h http.Handler, id, ip string) {
	t.Helper()
	body := `{"clientId":"` + id + `","clientName":"Test PC","platform":"Windows","browser":"Edge","connectionType":"ethernet"}`
	r := httptest.NewRequest(http.MethodPost, "/api/client/heartbeat", strings.NewReader(body))
	r.RemoteAddr = ip + ":50000"
	r.Header.Set("User-Agent", "SpeedTest-Test-Agent")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestClientRegistryLimitAndExpiryShape(t *testing.T) {
	c := newClientRegistry(1)
	if _, _, err := c.Heartbeat(clientHeartbeat{ClientID: "client-a", ClientName: "Office PC"}, "192.168.1.20", "Edge"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.Heartbeat(clientHeartbeat{ClientID: "client-b"}, "192.168.1.21", "Chrome"); err == nil {
		t.Fatal("expected client capacity error")
	}
	if len(c.List()) != 1 {
		t.Fatalf("active sessions = %d", len(c.List()))
	}
}

func TestMultiStreamUploadRequiresRegisteredClient(t *testing.T) {
	h, _, _ := newTestHandler(t)
	r := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader("small-payload"))
	r.RemoteAddr = "192.168.1.20:51000"
	r.Header.Set("X-Speed-Test-Client", "client-a")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("unregistered upload status=%d", w.Code)
	}
	heartbeat(t, h, "client-a", "192.168.1.20")
	r = httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader("small-payload"))
	r.RemoteAddr = "192.168.1.20:51000"
	r.Header.Set("X-Speed-Test-Client", "client-a")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("registered upload status=%d body=%s", w.Code, w.Body.String())
	}
	var response map[string]int64
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["receivedBytes"] != int64(len("small-payload")) {
		t.Fatalf("received bytes=%d", response["receivedBytes"])
	}
}

func TestResultSavedWithClientDiagnostics(t *testing.T) {
	h, store, _ := newTestHandler(t)
	heartbeat(t, h, "client-a", "192.168.1.20")
	payload := `{"clientId":"client-a","clientName":"Office PC","platform":"Windows 11","browser":"Edge","connectionType":"ethernet","mode":"bidirectional","streams":4,"durationSeconds":10,"downloadMbps":940.2,"uploadMbps":880.5,"downloadMinMbps":700,"downloadPeakMbps":980,"uploadMinMbps":650,"uploadPeakMbps":920,"latencyMs":1.2,"latencyMinMs":0.8,"latencyMaxMs":2.1,"jitterMs":0.2,"downloadBytes":1175000000,"uploadBytes":1100000000}`
	r := httptest.NewRequest(http.MethodPost, "/api/results", strings.NewReader(payload))
	r.RemoteAddr = "192.168.1.20:54321"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("result status=%d body=%s", w.Code, w.Body.String())
	}
	rows, err := store.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	x := rows[0]
	if x.ClientIP != "192.168.1.20" || x.ClientName != "Test PC" || x.DownloadMbps != 940.2 || x.Mode != "bidirectional" || x.Streams != 4 || x.DownloadPeakMbps != 980 {
		t.Fatalf("unexpected row: %+v", x)
	}
}

func TestAdminIsLoopbackOrOwnHostOnly(t *testing.T) {
	h, store, _ := newTestHandler(t)
	store.Append(testLog{ID: "id-1", ClientName: "PC", ClientIP: "192.168.1.8", DownloadMbps: 100, Status: "completed"})
	remote := httptest.NewRequest(http.MethodGet, "/api/admin/logs?format=csv", nil)
	remote.RemoteAddr = "192.168.1.42:40000"
	rw := httptest.NewRecorder()
	h.ServeHTTP(rw, remote)
	if rw.Code != http.StatusForbidden {
		t.Fatalf("remote admin status=%d", rw.Code)
	}
	local := httptest.NewRequest(http.MethodGet, "/api/admin/logs?format=csv", nil)
	local.RemoteAddr = "127.0.0.1:40000"
	lw := httptest.NewRecorder()
	h.ServeHTTP(lw, local)
	if lw.Code != http.StatusOK || !strings.Contains(lw.Body.String(), "id-1") {
		t.Fatalf("local export status=%d body=%s", lw.Code, lw.Body.String())
	}
}

func TestPersistentLogsSurviveStoreReopen(t *testing.T) {
	dir := t.TempDir()
	s, err := newLogStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Append(testLog{ID: "persist-1", Status: "completed"}); err != nil {
		t.Fatal(err)
	}
	s2, err := newLogStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s2.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "persist-1" {
		t.Fatalf("persisted rows=%+v", rows)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "tests.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"2.0.0", "1.9.9", 1}, {"v2.0.0", "2.0.0", 0}, {"1.0.0", "2.0.0", -1}}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%s,%s)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCorrespondingSourceAndLicenseAreServedFromPackage(t *testing.T) {
	h, _, _ := newTestHandler(t)
	tmp := t.TempDir()
	exe := filepath.Join(tmp, "speed-test.exe")
	oldExecutablePath := executablePath
	executablePath = func() (string, error) { return exe, nil }
	t.Cleanup(func() { executablePath = oldExecutablePath })

	sourceName := "Speed-Test-v" + appVersion + "-Source.zip"
	if err := os.WriteFile(filepath.Join(tmp, sourceName), []byte("test-source-zip"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "LICENSE"), []byte("GNU AFFERO GENERAL PUBLIC LICENSE"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		path        string
		wantStatus  int
		wantContent string
		wantType    string
	}{
		{path: "/source.zip", wantStatus: http.StatusOK, wantContent: "test-source-zip", wantType: "application/zip"},
		{path: "/LICENSE", wantStatus: http.StatusOK, wantContent: "GNU AFFERO GENERAL PUBLIC LICENSE", wantType: "text/plain"},
	} {
		r := httptest.NewRequest(http.MethodGet, tc.path, nil)
		r.RemoteAddr = "127.0.0.1:1000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.wantStatus || !strings.Contains(w.Body.String(), tc.wantContent) {
			t.Fatalf("%s returned status %d and body %q", tc.path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Header().Get("Content-Type"), tc.wantType) {
			t.Fatalf("%s content type = %q", tc.path, w.Header().Get("Content-Type"))
		}
	}
}

func TestSourceDownloadRejectsNonGet(t *testing.T) {
	h, _, _ := newTestHandler(t)
	r := httptest.NewRequest(http.MethodPost, "/source.zip", strings.NewReader("ignored"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /source.zip status = %d", w.Code)
	}
}
