// Copyright (C) 2026 Ajay Raikvar. SPDX-License-Identifier: AGPL-3.0-or-later. See LICENSE and NOTICE.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed web/*
var webFiles embed.FS

var appVersion = "2.0.0"

const (
	appName       = "Speed Test"
	developer     = "www.ajayraikvar.com"
	repoAPI       = "https://api.github.com/repos/ajayraikvar/speed-test/releases/latest"
	repoURL       = "https://github.com/ajayraikvar/speed-test"
	maxUploadSize = int64(512 << 20)
	maxResultBody = int64(64 << 10)
	maxClientBody = int64(8 << 10)
	maxLogRows    = 10000
	sessionTTL    = 35 * time.Second
)

type serverInfo struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Developer  string   `json:"developer"`
	Website    string   `json:"website"`
	Repository string   `json:"repository"`
	Hostname   string   `json:"hostname"`
	Port       int      `json:"port"`
	Addresses  []string `json:"addresses"`
	Interfaces []string `json:"interfaces"`
	MaxClients int      `json:"maxClients"`
}

type resultInput struct {
	ClientID         string  `json:"clientId"`
	ClientName       string  `json:"clientName"`
	Platform         string  `json:"platform"`
	Browser          string  `json:"browser"`
	ConnectionType   string  `json:"connectionType"`
	Mode             string  `json:"mode"`
	Streams          int     `json:"streams"`
	DurationSeconds  float64 `json:"durationSeconds"`
	DownloadMbps     float64 `json:"downloadMbps"`
	UploadMbps       float64 `json:"uploadMbps"`
	DownloadMinMbps  float64 `json:"downloadMinMbps"`
	DownloadPeakMbps float64 `json:"downloadPeakMbps"`
	UploadMinMbps    float64 `json:"uploadMinMbps"`
	UploadPeakMbps   float64 `json:"uploadPeakMbps"`
	LatencyMs        float64 `json:"latencyMs"`
	LatencyMinMs     float64 `json:"latencyMinMs"`
	LatencyMaxMs     float64 `json:"latencyMaxMs"`
	JitterMs         float64 `json:"jitterMs"`
	DownloadBytes    int64   `json:"downloadBytes"`
	UploadBytes      int64   `json:"uploadBytes"`
}

type testLog struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	ClientID         string    `json:"clientId"`
	ClientName       string    `json:"clientName"`
	ClientIP         string    `json:"clientIp"`
	UserAgent        string    `json:"userAgent"`
	Platform         string    `json:"platform"`
	Browser          string    `json:"browser"`
	ConnectionType   string    `json:"connectionType"`
	Mode             string    `json:"mode"`
	Streams          int       `json:"streams"`
	DurationSeconds  float64   `json:"durationSeconds"`
	DownloadMbps     float64   `json:"downloadMbps"`
	UploadMbps       float64   `json:"uploadMbps"`
	DownloadMinMbps  float64   `json:"downloadMinMbps"`
	DownloadPeakMbps float64   `json:"downloadPeakMbps"`
	UploadMinMbps    float64   `json:"uploadMinMbps"`
	UploadPeakMbps   float64   `json:"uploadPeakMbps"`
	LatencyMs        float64   `json:"latencyMs"`
	LatencyMinMs     float64   `json:"latencyMinMs"`
	LatencyMaxMs     float64   `json:"latencyMaxMs"`
	JitterMs         float64   `json:"jitterMs"`
	DownloadBytes    int64     `json:"downloadBytes"`
	UploadBytes      int64     `json:"uploadBytes"`
	Status           string    `json:"status"`
}

type logStore struct {
	path string
	mu   sync.Mutex
}

func newLogStore(dataDir string) (*logStore, error) {
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	return &logStore{path: filepath.Join(logDir, "tests.jsonl")}, nil
}
func (s *logStore) Append(item testLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(item); err != nil {
		return err
	}
	return f.Sync()
}
func (s *logStore) ReadAll() ([]testLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []testLog{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows := make([]testLog, 0, 128)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var item testLog
		if json.Unmarshal(scanner.Bytes(), &item) == nil {
			rows = append(rows, item)
			if len(rows) > maxLogRows {
				rows = rows[len(rows)-maxLogRows:]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

type clientHeartbeat struct {
	ClientID       string `json:"clientId"`
	ClientName     string `json:"clientName"`
	Platform       string `json:"platform"`
	Browser        string `json:"browser"`
	ConnectionType string `json:"connectionType"`
}
type clientSession struct {
	ClientID       string    `json:"clientId"`
	ClientName     string    `json:"clientName"`
	ClientIP       string    `json:"clientIp"`
	Platform       string    `json:"platform"`
	Browser        string    `json:"browser"`
	ConnectionType string    `json:"connectionType"`
	UserAgent      string    `json:"userAgent"`
	FirstSeen      time.Time `json:"firstSeen"`
	LastSeen       time.Time `json:"lastSeen"`
}
type clientRegistry struct {
	mu    sync.Mutex
	items map[string]clientSession
	max   int
}

func newClientRegistry(max int) *clientRegistry {
	if max < 1 {
		max = 16
	}
	if max > 128 {
		max = 128
	}
	return &clientRegistry{items: make(map[string]clientSession), max: max}
}
func (c *clientRegistry) pruneLocked(now time.Time) {
	for id, item := range c.items {
		if now.Sub(item.LastSeen) > sessionTTL {
			delete(c.items, id)
		}
	}
}
func (c *clientRegistry) Heartbeat(input clientHeartbeat, ip, ua string) (clientSession, int, error) {
	if strings.TrimSpace(input.ClientID) == "" || len(input.ClientID) > 80 {
		return clientSession{}, 0, errors.New("invalid client ID")
	}
	now := time.Now().UTC()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(now)
	item, exists := c.items[input.ClientID]
	if !exists && len(c.items) >= c.max {
		return clientSession{}, len(c.items), errors.New("host client limit reached")
	}
	if !exists {
		item.FirstSeen = now
	}
	item.ClientID = input.ClientID
	item.ClientName = cleanText(input.ClientName, 40)
	if item.ClientName == "" {
		item.ClientName = "Browser client"
	}
	item.ClientIP = ip
	item.Platform = cleanText(input.Platform, 80)
	item.Browser = cleanText(input.Browser, 80)
	item.ConnectionType = cleanText(input.ConnectionType, 40)
	item.UserAgent = truncate(ua, 256)
	item.LastSeen = now
	c.items[input.ClientID] = item
	return item, len(c.items), nil
}
func (c *clientRegistry) Disconnect(id, ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, ok := c.items[id]; ok && item.ClientIP == ip {
		delete(c.items, id)
	}
}
func (c *clientRegistry) Get(id string) (clientSession, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(time.Now().UTC())
	x, ok := c.items[id]
	return x, ok
}
func (c *clientRegistry) List() []clientSession {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(time.Now().UTC())
	out := make([]clientSession, 0, len(c.items))
	for _, x := range c.items {
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastSeen.After(out[j].LastSeen) })
	return out
}

type updateSnapshot struct {
	Status          string    `json:"status"`
	CurrentVersion  string    `json:"currentVersion"`
	LatestVersion   string    `json:"latestVersion,omitempty"`
	UpdateAvailable bool      `json:"updateAvailable"`
	ReleaseURL      string    `json:"releaseUrl,omitempty"`
	CheckedAt       time.Time `json:"checkedAt,omitempty"`
}
type updateChecker struct {
	mu     sync.RWMutex
	state  updateSnapshot
	client *http.Client
}

func newUpdateChecker() *updateChecker {
	return &updateChecker{state: updateSnapshot{Status: "checking", CurrentVersion: appVersion}, client: &http.Client{Timeout: 5 * time.Second}}
}
func (u *updateChecker) Snapshot() updateSnapshot { u.mu.RLock(); defer u.mu.RUnlock(); return u.state }
func (u *updateChecker) set(s updateSnapshot)     { u.mu.Lock(); u.state = s; u.mu.Unlock() }
func (u *updateChecker) refresh(ctx context.Context) {
	next := updateSnapshot{Status: "offline", CurrentVersion: appVersion, CheckedAt: time.Now().UTC()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repoAPI, nil)
	if err != nil {
		u.set(next)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AjayRaikvar-Speed-Test-Updater")
	resp, err := u.client.Do(req)
	if err != nil {
		u.set(next)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		next.Status = "not-published"
		u.set(next)
		return
	}
	if resp.StatusCode != http.StatusOK {
		u.set(next)
		return
	}
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release) != nil || release.TagName == "" {
		u.set(next)
		return
	}
	next.LatestVersion = strings.TrimPrefix(release.TagName, "v")
	next.ReleaseURL = release.HTMLURL
	next.UpdateAvailable = compareVersions(next.LatestVersion, appVersion) > 0
	if next.UpdateAvailable {
		next.Status = "update-available"
	} else {
		next.Status = "up-to-date"
	}
	u.set(next)
	if next.UpdateAvailable {
		fmt.Printf("\nUPDATE AVAILABLE: Speed Test %s is available. Current host: %s. Restart Start-Speed-Test.bat to update.\nRelease: %s\n\n", next.LatestVersion, appVersion, next.ReleaseURL)
	}
}
func (u *updateChecker) run(ctx context.Context) {
	check := func() { c, cancel := context.WithTimeout(ctx, 6*time.Second); defer cancel(); u.refresh(c) }
	check()
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			check()
		}
	}
}
func compareVersions(a, b string) int {
	parse := func(v string) []int {
		v = strings.TrimPrefix(strings.TrimSpace(v), "v")
		if i := strings.IndexAny(v, "-+"); i >= 0 {
			v = v[:i]
		}
		p := strings.Split(v, ".")
		out := make([]int, 3)
		for i := 0; i < len(p) && i < 3; i++ {
			out[i], _ = strconv.Atoi(p[i])
		}
		return out
	}
	x, y := parse(a), parse(b)
	for i := range x {
		if x[i] > y[i] {
			return 1
		}
		if x[i] < y[i] {
			return -1
		}
	}
	return 0
}

func defaultDataDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		if d, e := os.UserConfigDir(); e == nil {
			base = d
		}
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "AjayRaikvar", "SpeedTest")
}
func main() {
	port := flag.Int("port", 80, "TCP port for the web server (1-65535)")
	dataDir := flag.String("data-dir", defaultDataDir(), "Persistent data directory")
	maxClients := flag.Int("max-clients", 16, "Maximum connected browser clients (1-128)")
	flag.Parse()
	if *port < 1 || *port > 65535 {
		log.Fatalf("Invalid port %d. Choose 1-65535.", *port)
	}
	if *maxClients < 1 || *maxClients > 128 {
		log.Fatalf("Invalid max clients %d. Choose 1-128.", *maxClients)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(*port)))
	if err != nil {
		log.Fatalf("Could not start Speed Test on port %d: %v. Try 8080 or another free port.", *port, err)
	}
	store, err := newLogStore(*dataDir)
	if err != nil {
		_ = listener.Close()
		log.Fatalf("Could not initialize host logs: %v", err)
	}
	ips := detectLANIPs()
	hostname, _ := os.Hostname()
	info := serverInfo{Name: appName, Version: appVersion, Developer: developer, Website: "https://www.ajayraikvar.com", Repository: repoURL, Hostname: hostname, Port: *port, Addresses: ips, Interfaces: detectInterfaces(), MaxClients: *maxClients}
	clients := newClientRegistry(*maxClients)
	updates := newUpdateChecker()
	handler := newHandler(info, store, updates, clients)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go updates.run(ctx)
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = server.Shutdown(c)
	}()
	fmt.Println("\n============================================================")
	fmt.Printf("  %s v%s\n  Original developer: Ajay Raikvar\n  Developed by %s\n", appName, appVersion, developer)
	fmt.Println("============================================================")
	fmt.Printf("Host name : %s\nHost port : %d\nMax clients: %d\n", hostname, *port, *maxClients)
	fmt.Println("\nOpen one of these addresses from another PC on the same network:")
	if len(ips) == 0 {
		fmt.Println("  No LAN IPv4 address detected. Check network connectivity.")
	}
	for _, ip := range ips {
		fmt.Printf("  http://%s\n", net.JoinHostPort(ip, strconv.Itoa(*port)))
	}
	fmt.Printf("\nHost dashboard : http://127.0.0.1:%d/admin\nPersistent logs: %s\n", *port, filepath.Join(*dataDir, "logs", "tests.jsonl"))
	fmt.Println("Keep this window open. Press Ctrl+C to stop the host.")
	fmt.Println("============================================================")
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Speed Test stopped: %v", err)
	}
	fmt.Println("Speed Test host stopped.")
}

type appHandler struct {
	info    serverInfo
	store   *logStore
	updates *updateChecker
	clients *clientRegistry
	static  fs.FS
}

func newHandler(info serverInfo, store *logStore, updates *updateChecker, clients *clientRegistry) http.Handler {
	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	a := &appHandler{info: info, store: store, updates: updates, clients: clients, static: static}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", a.infoHandler)
	mux.HandleFunc("/api/ping", a.pingHandler)
	mux.HandleFunc("/api/update", a.updateHandler)
	mux.HandleFunc("/api/download", a.downloadHandler)
	mux.HandleFunc("/api/upload", a.uploadHandler)
	mux.HandleFunc("/api/client/heartbeat", a.heartbeatHandler)
	mux.HandleFunc("/api/client/disconnect", a.disconnectHandler)
	mux.HandleFunc("/api/results", a.resultsHandler)
	mux.HandleFunc("/admin", a.adminHandler)
	mux.HandleFunc("/admin.html", a.adminHandler)
	mux.HandleFunc("/api/admin/logs", a.adminLogsHandler)
	mux.HandleFunc("/api/admin/clients", a.adminClientsHandler)
	mux.HandleFunc("/api/admin/stats", a.adminStatsHandler)
	mux.HandleFunc("/source.zip", a.sourceArchiveHandler)
	mux.HandleFunc("/LICENSE", a.packageNoticeHandler("LICENSE", "text/plain; charset=utf-8"))
	mux.HandleFunc("/NOTICE", a.packageNoticeHandler("NOTICE", "text/plain; charset=utf-8"))
	mux.Handle("/", http.FileServer(http.FS(static)))
	return securityHeaders(mux)
}

var executablePath = os.Executable

func packageFilePath(name string) (string, error) {
	executable, err := executablePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(executable), name), nil
}

func (a *appHandler) sourceArchiveHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	name := fmt.Sprintf("Speed-Test-v%s-Source.zip", appVersion)
	path, err := packageFilePath(name)
	if err != nil {
		http.Error(w, "Could not locate the source package", http.StatusNotFound)
		return
	}
	if _, err := os.Stat(path); err != nil {
		http.Error(w, "The corresponding source archive is not beside this executable. Download the source package from the official project repository.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func (a *appHandler) packageNoticeHandler(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		path, err := packageFilePath(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if _, err := os.Stat(path); err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, path)
	}
}

func (a *appHandler) infoHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	x := a.info
	x.MaxClients = a.clients.max
	writeJSON(w, 200, x)
}
func (a *appHandler) pingHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *appHandler) updateHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, 200, a.updates.Snapshot())
}
func (a *appHandler) downloadHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if !a.validClientRequest(r) {
		http.Error(w, "Register as a client before testing", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Encoding", "identity")
	block := make([]byte, 256<<10)
	for i := range block {
		block[i] = byte((i*37 + 29) & 255)
	}
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", 500)
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		if _, err := w.Write(block); err != nil {
			return
		}
		f.Flush()
	}
}
func (a *appHandler) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if !a.validClientRequest(r) {
		http.Error(w, "Register as a client before testing", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	n, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "Upload exceeds safety limit", http.StatusRequestEntityTooLarge)
			return
		}
		if r.Context().Err() != nil {
			return
		}
		http.Error(w, "Could not read upload stream", http.StatusBadRequest)
		return
	}
	writeJSON(w, 200, map[string]int64{"receivedBytes": n})
}
func (a *appHandler) validClientRequest(r *http.Request) bool {
	id := r.Header.Get("X-Speed-Test-Client")
	if id == "" {
		id = r.URL.Query().Get("client")
	}
	item, ok := a.clients.Get(id)
	return ok && item.ClientIP == remoteIP(r.RemoteAddr)
}
func (a *appHandler) heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxClientBody)
	var input clientHeartbeat
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid client heartbeat", 400)
		return
	}
	item, count, err := a.clients.Heartbeat(input, remoteIP(r.RemoteAddr), r.UserAgent())
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "activeClients": count, "maxClients": a.info.MaxClients})
		return
	}
	writeJSON(w, 200, map[string]any{"client": item, "activeClients": count, "maxClients": a.info.MaxClients, "serverTime": time.Now().UTC()})
}
func (a *appHandler) disconnectHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxClientBody)
	var input struct {
		ClientID string `json:"clientId"`
	}
	if json.NewDecoder(r.Body).Decode(&input) == nil {
		a.clients.Disconnect(input.ClientID, remoteIP(r.RemoteAddr))
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *appHandler) resultsHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxResultBody)
	var v resultInput
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, "Invalid result payload", 400)
		return
	}
	if !validResult(v) {
		http.Error(w, "Result values are outside permitted limits", 400)
		return
	}
	session, ok := a.clients.Get(v.ClientID)
	if !ok || session.ClientIP != remoteIP(r.RemoteAddr) {
		http.Error(w, "Client session expired. Reconnect and retry.", http.StatusConflict)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "Could not create test ID", 500)
		return
	}
	mode := v.Mode
	if mode != "sequential" && mode != "bidirectional" {
		mode = "sequential"
	}
	if v.Streams < 1 {
		v.Streams = 1
	}
	if v.Streams > 8 {
		v.Streams = 8
	}
	row := testLog{ID: id, Timestamp: time.Now().UTC(), ClientID: truncate(v.ClientID, 80), ClientName: session.ClientName, ClientIP: session.ClientIP, UserAgent: session.UserAgent, Platform: session.Platform, Browser: session.Browser, ConnectionType: session.ConnectionType, Mode: mode, Streams: v.Streams, DurationSeconds: v.DurationSeconds, DownloadMbps: v.DownloadMbps, UploadMbps: v.UploadMbps, DownloadMinMbps: v.DownloadMinMbps, DownloadPeakMbps: v.DownloadPeakMbps, UploadMinMbps: v.UploadMinMbps, UploadPeakMbps: v.UploadPeakMbps, LatencyMs: v.LatencyMs, LatencyMinMs: v.LatencyMinMs, LatencyMaxMs: v.LatencyMaxMs, JitterMs: v.JitterMs, DownloadBytes: v.DownloadBytes, UploadBytes: v.UploadBytes, Status: "completed"}
	if err := a.store.Append(row); err != nil {
		http.Error(w, "Could not save host log", 500)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"testId": id, "saved": true})
}
func (a *appHandler) adminHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if !isHostLocal(r, a.info.Addresses) {
		http.Error(w, "Host dashboard is available only on the host PC", http.StatusForbidden)
		return
	}
	b, err := fs.ReadFile(a.static, "admin.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}
func (a *appHandler) adminLogsHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if !isHostLocal(r, a.info.Addresses) {
		http.Error(w, "Host logs are available only on the host PC", 403)
		return
	}
	rows, err := a.store.ReadAll()
	if err != nil {
		http.Error(w, "Could not read logs", 500)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=Speed-Test-v2-Host-Logs.csv")
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"test_id", "timestamp_utc", "client_name", "client_ip", "browser", "platform", "connection_type", "mode", "streams", "duration_seconds", "download_mbps", "upload_mbps", "download_min_mbps", "download_peak_mbps", "upload_min_mbps", "upload_peak_mbps", "latency_ms", "latency_min_ms", "latency_max_ms", "jitter_ms", "download_bytes", "upload_bytes", "status"})
		for _, x := range rows {
			_ = cw.Write([]string{x.ID, x.Timestamp.Format(time.RFC3339), x.ClientName, x.ClientIP, x.Browser, x.Platform, x.ConnectionType, x.Mode, strconv.Itoa(x.Streams), f64(x.DurationSeconds), f64(x.DownloadMbps), f64(x.UploadMbps), f64(x.DownloadMinMbps), f64(x.DownloadPeakMbps), f64(x.UploadMinMbps), f64(x.UploadPeakMbps), f64(x.LatencyMs), f64(x.LatencyMinMs), f64(x.LatencyMaxMs), f64(x.JitterMs), strconv.FormatInt(x.DownloadBytes, 10), strconv.FormatInt(x.UploadBytes, 10), x.Status})
		}
		cw.Flush()
		return
	}
	writeJSON(w, 200, map[string]any{"count": len(rows), "items": rows})
}
func (a *appHandler) adminClientsHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if !isHostLocal(r, a.info.Addresses) {
		http.Error(w, "Host dashboard is local only", 403)
		return
	}
	items := a.clients.List()
	writeJSON(w, 200, map[string]any{"count": len(items), "maxClients": a.info.MaxClients, "items": items})
}
func (a *appHandler) adminStatsHandler(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if !isHostLocal(r, a.info.Addresses) {
		http.Error(w, "Host dashboard is local only", 403)
		return
	}
	rows, _ := a.store.ReadAll()
	var d, u, l float64
	for _, x := range rows {
		d += x.DownloadMbps
		u += x.UploadMbps
		l += x.LatencyMs
	}
	n := float64(len(rows))
	if n > 0 {
		d /= n
		u /= n
		l /= n
	}
	writeJSON(w, 200, map[string]any{"totalTests": len(rows), "activeClients": len(a.clients.List()), "maxClients": a.info.MaxClients, "averageDownloadMbps": d, "averageUploadMbps": u, "averageLatencyMs": l})
}

func validResult(v resultInput) bool {
	finite := func(n float64) bool { return n >= 0 && n < 1e9 }
	return v.DurationSeconds >= 1 && v.DurationSeconds <= 600 && finite(v.DownloadMbps) && finite(v.UploadMbps) && finite(v.DownloadMinMbps) && finite(v.DownloadPeakMbps) && finite(v.UploadMinMbps) && finite(v.UploadPeakMbps) && finite(v.LatencyMs) && finite(v.LatencyMinMs) && finite(v.LatencyMaxMs) && finite(v.JitterMs) && v.DownloadBytes >= 0 && v.UploadBytes >= 0 && v.DownloadBytes <= maxUploadSize*100 && v.UploadBytes <= maxUploadSize*100
}
func newID() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
func cleanText(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " "))
	return truncate(s, n)
}
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
func f64(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
func remoteIP(addr string) string {
	h, _, e := net.SplitHostPort(addr)
	if e != nil {
		return addr
	}
	return h
}
func isHostLocal(r *http.Request, ips []string) bool {
	ip := net.ParseIP(remoteIP(r.RemoteAddr))
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	for _, s := range ips {
		if ip.Equal(net.ParseIP(s)) {
			return true
		}
	}
	return false
}
func requireMethod(w http.ResponseWriter, r *http.Request, m string) bool {
	if r.Method == m {
		return true
	}
	w.Header().Set("Allow", m)
	http.Error(w, "Method not allowed", 405)
	return false
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
func detectLANIPs() []string {
	found := map[string]bool{}
	if c, e := net.DialTimeout("udp", "192.0.2.1:80", 400*time.Millisecond); e == nil {
		if a, ok := c.LocalAddr().(*net.UDPAddr); ok && a.IP.To4() != nil && !a.IP.IsLoopback() {
			found[a.IP.String()] = true
		}
		_ = c.Close()
	}
	ifs, e := net.Interfaces()
	if e == nil {
		for _, it := range ifs {
			if it.Flags&net.FlagUp == 0 || it.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, _ := it.Addrs()
			for _, a := range addrs {
				var ip net.IP
				switch x := a.(type) {
				case *net.IPNet:
					ip = x.IP
				case *net.IPAddr:
					ip = x.IP
				}
				if p := ip.To4(); p != nil && !p.IsLoopback() && !p.IsLinkLocalUnicast() {
					found[p.String()] = true
				}
			}
		}
	}
	out := make([]string, 0, len(found))
	for ip := range found {
		out = append(out, ip)
	}
	sort.Slice(out, func(i, j int) bool {
		pi, pj := net.ParseIP(out[i]).IsPrivate(), net.ParseIP(out[j]).IsPrivate()
		if pi != pj {
			return pi
		}
		return out[i] < out[j]
	})
	return out
}
func detectInterfaces() []string {
	ifs, e := net.Interfaces()
	if e != nil {
		return []string{}
	}
	out := []string{}
	for _, it := range ifs {
		if it.Flags&net.FlagUp == 0 || it.Flags&net.FlagLoopback != 0 {
			continue
		}
		kind := "network"
		if strings.Contains(strings.ToLower(it.Name), "wi-fi") || strings.Contains(strings.ToLower(it.Name), "wlan") {
			kind = "wireless"
		} else if strings.Contains(strings.ToLower(it.Name), "eth") || strings.Contains(strings.ToLower(it.Name), "en") {
			kind = "wired/other"
		}
		out = append(out, it.Name+" ("+kind+")")
	}
	sort.Strings(out)
	return out
}
