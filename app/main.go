// Old Maid - A Victorian Parlor Game (c) 2026 Ironite.
// Serves the game locally and opens it in its own app window.
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

var version = "0.0.0-dev" // set with -ldflags "-X main.version=..."

const productName = "Old Maid"

//go:embed web
var webFS embed.FS

var lastPing atomic.Int64

func dataDir() string { // %APPDATA%\Ironite\Old Maid  (saved record & settings)
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Ironite", productName)
}
func cacheDir() string { // %LOCALAPPDATA%\Ironite\Old Maid  (window profile)
	d, err := os.UserCacheDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "Ironite", productName)
}

func routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		path := filepath.Join(dataDir(), "save.json")
		switch r.Method {
		case http.MethodGet:
			b, err := os.ReadFile(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(b)
		case http.MethodPost:
			b, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
			if err != nil || !json.Valid(b) {
				http.Error(w, "bad save", http.StatusBadRequest)
				return
			}
			_ = os.MkdirAll(dataDir(), 0o755)
			tmp := path + ".tmp"
			if err := os.WriteFile(tmp, b, 0o644); err == nil {
				_ = os.Rename(tmp, path)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "/version") {
		fmt.Println(version)
		return
	}
	logf, _ := os.OpenFile(filepath.Join(os.TempDir(), "IroniteOldMaid.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if logf != nil {
		log.SetOutput(logf)
	}
	log.Printf("%s %s starting", productName, version)
	if !singleInstance() {
		return
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("Old Maid could not start: " + err.Error())
	}
	go func() { _ = http.Serve(ln, routes()) }()
	url := fmt.Sprintf("http://%s/", ln.Addr().String())
	if os.Getenv("IRONITE_DEV") != "" {
		fmt.Println(url)
		select {}
	}
	started := time.Now()
	if wait, ok := openWindow(url); ok {
		wait()
		if time.Since(started) > 5*time.Second {
			return
		}
	} else {
		openDefaultBrowser(url)
	}
	// The browser handed the page to another window: stay up while it's open.
	for {
		time.Sleep(5 * time.Second)
		lp := lastPing.Load()
		if (lp != 0 && time.Since(time.Unix(lp, 0)) > 2*time.Minute) || (lp == 0 && time.Since(started) > 90*time.Second) {
			return
		}
	}
}
