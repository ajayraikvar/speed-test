// Copyright (c) 2026 Ajay Raikvar. All rights reserved for the original project branding and marks.
// SPDX-License-Identifier: AGPL-3.0-or-later
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

var appVersion = "1.0.0"

const (
	appName       = "Speed Test"
	developer     = "www.ajayraikvar.com"
	repoAPI       = "https://api.github.com/repos/ajayraikvar/speed-test/releases/latest"
	repoURL       = "https://github.com/ajayraikvar/speed-test"
	maxUploadSize = int64(512 << 20) // 512 MiB per request safety limit
	maxResultBody = int64(64 << 10)
	maxLogRows    = 5000
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
}

type resultInput struct {
	DurationSeconds float64 `json:"durationSeconds"`
	DownloadMbps    float64 `json:"downloadMbps"`
	UploadMbps      float64 `json:"uploadMbps"`
	LatencyMs       float64 `json:"latencyMs"`
	JitterMs        float64 `json:"jitterMs"`
	DownloadBytes   int64   `json:"downloadBytes"`
	UploadBytes     int64   `json:"uploadBytes"`
	ClientVersion   string  `json:"clientVersion"`
}

type testLog struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	ClientIP        string    `json:"clientIp"`
	UserAgent       string    `json:"userAgent"`
	DurationSeconds float64   `json:"durationSeconds"`
	DownloadMbps    float64   `json:"downloadMbps"`
	UploadMbps      float64   `json:"uploadMbps"`
	LatencyMs       float64   `json:"latencyMs"`
	JitterMs        float64   `json:"jitterMs"`
	DownloadBytes   int64     `json:"downloadBytes"`
	UploadBytes     int64     `json:"uploadBytes"`
	ClientVersion   string    `json:"clientVersion"`
	Status          string    `json:"status"`
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
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
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
	return &updateChecker{
		state:  updateSnapshot{Status: "checking", CurrentVersion: appVersion},
		client: &http.Client{Timeout: 4 * time.Second},
	}
}

func (u *updateChecker) Snapshot() updateSnapshot {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.state
}

func (u *updateChecker) set(state updateSnapshot) {
	u.mu.Lock()
	u.state = state
	u.mu.Unlock()
}

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
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil || release.TagName == "" {
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
		fmt.Printf("\nUPDATE AVAILABLE: Speed Test %s is available (this host is running %s). Restart with Start-Speed-Test.bat to update.\nRelease: %s\n\n", next.LatestVersion, appVersion, next.ReleaseURL)
	}
}

func (u *updateChecker) run(ctx context.Context) {
	check := func() {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		u.refresh(checkCtx)
	}
	check()
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
		parts := strings.Split(v, ".")
		out := make([]int, 3)
		for i := 0; i < len(parts) && i < 3; i++ {
			out[i], _ = strconv.Atoi(parts[i])
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
		if d, err := os.UserConfigDir(); err == nil {
			base = d
		}
	}
	if base == "" {
		base = "."
	}
	return filepath.Join(base, "AjayRaikvar", "SpeedTest")
}

func main() {
	portFlag := flag.Int("port", 80, "TCP port for the web server (1-65535)")
	dataDirFlag := flag.String("data-dir", defaultDataDir(), "Directory for persistent host logs and settings")
	flag.Parse()
	if *portFlag < 1 || *portFlag > 65535 {
		log.Fatalf("Invalid port %d. Choose a port from 1 to 65535.", *portFlag)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(*portFlag)))
	if err != nil {
		log.Fatalf("Could not start Speed Test on port %d: %v\nThe port may be in use. Try 8080 or another free port.", *portFlag, err)
	}
	store, err := newLogStore(*dataDirFlag)
	if err != nil {
		_ = listener.Close()
		log.Fatalf("Could not initialize persistent test logs: %v", err)
	}
	ips := detectLANIPs()
	hostname, _ := os.Hostname()
	info := serverInfo{
		Name: appName, Version: appVersion, Developer: developer,
		Website: "https://www.ajayraikvar.com", Repository: repoURL,
		Hostname: hostname, Port: *portFlag, Addresses: ips,
	}
	updates := newUpdateChecker()
	server := &http.Server{
		Handler:           newHandler(info, store, updates),
		ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 90 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go updates.run(ctx)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Println("\n============================================================")
	fmt.Printf("  %s v%s\n  Original developer: Ajay Raikvar\n  Developed by %s\n", appName, appVersion, developer)
	fmt.Println("============================================================")
	fmt.Printf("Host name : %s\nHost port : %d\n", hostname, *portFlag)
	fmt.Println("\nOpen one of these addresses from another PC on the same network:")
	if len(ips) == 0 {
		fmt.Println("  No LAN IPv4 address detected. Check network connectivity.")
	}
	for _, ip := range ips {
		fmt.Printf("  http://%s\n", net.JoinHostPort(ip, strconv.Itoa(*portFlag)))
	}
	fmt.Printf("\nHost dashboard : http://127.0.0.1:%d/admin\n", *portFlag)
	fmt.Printf("Persistent logs: %s\n", filepath.Join(*dataDirFlag, "logs", "tests.jsonl"))
	fmt.Println("\nKeep this window open. Press Ctrl+C to stop the host.")
	fmt.Println("============================================================")
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Speed Test server stopped unexpectedly: %v", err)
	}
	fmt.Println("Speed Test host stopped.")
}

func newHandler(info serverInfo, store *logStore, updates *updateChecker) http.Handler {
	mux := http.NewServeMux()
	staticFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackRequest(r) {
			http.Error(w, "Host dashboard is available only on the host PC.", http.StatusForbidden)
			return
		}
		http.Redirect(w, r, "/admin.html", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/admin.html", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		if !isLoopbackRequest(r) {
			http.Error(w, "Host dashboard is available only on the host PC.", http.StatusForbidden)
			return
		}
		serveEmbeddedFile(w, staticFS, "admin.html", "text/html; charset=utf-8")
	})
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		writeJSON(w, http.StatusOK, info)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/update", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		writeJSON(w, http.StatusOK, updates.Snapshot())
	})
	mux.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, no-transform")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming is not supported", http.StatusInternalServerError)
			return
		}
		block := make([]byte, 1<<20)
		for i := range block {
			block[i] = byte((i*31 + 17) & 0xff)
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
			flusher.Flush()
		}
	})
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
		received, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "Upload exceeds the 512 MiB safety limit", http.StatusRequestEntityTooLarge)
				return
			}
			if r.Context().Err() != nil {
				return
			}
			http.Error(w, "Could not read upload stream", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"receivedBytes": received})
	})
	mux.HandleFunc("/api/results", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxResultBody)
		var input resultInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid result payload", http.StatusBadRequest)
			return
		}
		if !validResult(input) {
			http.Error(w, "Result values are outside permitted limits", http.StatusBadRequest)
			return
		}
		id, err := newID()
		if err != nil {
			http.Error(w, "Could not create test ID", http.StatusInternalServerError)
			return
		}
		row := testLog{
			ID: id, Timestamp: time.Now().UTC(), ClientIP: remoteIP(r.RemoteAddr),
			UserAgent: truncate(r.UserAgent(), 256), DurationSeconds: input.DurationSeconds,
			DownloadMbps: input.DownloadMbps, UploadMbps: input.UploadMbps,
			LatencyMs: input.LatencyMs, JitterMs: input.JitterMs,
			DownloadBytes: input.DownloadBytes, UploadBytes: input.UploadBytes,
			ClientVersion: truncate(input.ClientVersion, 32), Status: "completed",
		}
		if err := store.Append(row); err != nil {
			http.Error(w, "Could not save result on host", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"testId": id, "saved": "true"})
	})
	mux.HandleFunc("/api/admin/logs", func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		if !isLoopbackRequest(r) {
			http.Error(w, "Host logs are available only on the host PC.", http.StatusForbidden)
			return
		}
		rows, err := store.ReadAll()
		if err != nil {
			http.Error(w, "Could not read host logs", http.StatusInternalServerError)
			return
		}
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=Speed-Test-Host-Logs.json")
			_ = json.NewEncoder(w).Encode(map[string]any{"count": len(rows), "items": rows})
			return
		}
		if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", "attachment; filename=Speed-Test-Host-Logs.csv")
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{"test_id", "timestamp_utc", "client_ip", "user_agent", "duration_seconds", "download_mbps", "upload_mbps", "latency_ms", "jitter_ms", "download_bytes", "upload_bytes", "client_version", "status"})
			for _, row := range rows {
				_ = cw.Write([]string{row.ID, row.Timestamp.Format(time.RFC3339), row.ClientIP, row.UserAgent,
					fmt.Sprintf("%.2f", row.DurationSeconds), fmt.Sprintf("%.3f", row.DownloadMbps), fmt.Sprintf("%.3f", row.UploadMbps),
					fmt.Sprintf("%.3f", row.LatencyMs), fmt.Sprintf("%.3f", row.JitterMs), strconv.FormatInt(row.DownloadBytes, 10),
					strconv.FormatInt(row.UploadBytes, 10), row.ClientVersion, row.Status})
			}
			cw.Flush()
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"count": len(rows), "items": rows})
	})
	mux.Handle("/", fileServer)
	return securityHeaders(mux)
}

func serveEmbeddedFile(w http.ResponseWriter, source fs.FS, name, contentType string) {
	b, err := fs.ReadFile(source, name)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	return false
}

func validResult(v resultInput) bool {
	finite := func(n float64) bool { return n >= 0 && n < 1e9 }
	return v.DurationSeconds >= 1 && v.DurationSeconds <= 600 && finite(v.DownloadMbps) && finite(v.UploadMbps) &&
		finite(v.LatencyMs) && finite(v.JitterMs) && v.DownloadBytes >= 0 && v.UploadBytes >= 0 &&
		v.DownloadBytes <= maxUploadSize*100 && v.UploadBytes <= maxUploadSize*100
}

func newID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func remoteIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func isLoopbackRequest(r *http.Request) bool {
	ip := net.ParseIP(remoteIP(r.RemoteAddr))
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
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
	found := make(map[string]bool)
	if conn, err := net.DialTimeout("udp", "192.0.2.1:80", 400*time.Millisecond); err == nil {
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP.To4() != nil && !addr.IP.IsLoopback() {
			found[addr.IP.String()] = true
		}
		_ = conn.Close()
	}
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
					found[ip4.String()] = true
				}
			}
		}
	}
	ips := make([]string, 0, len(found))
	for ip := range found {
		ips = append(ips, ip)
	}
	sort.Slice(ips, func(i, j int) bool {
		pi, pj := net.ParseIP(ips[i]).IsPrivate(), net.ParseIP(ips[j]).IsPrivate()
		if pi != pj {
			return pi
		}
		return ips[i] < ips[j]
	})
	return ips
}
