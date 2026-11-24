// Command anisette-server is a small HTTP wrapper around ipatool's local
// Anisette engine (internal/storeauth of majd/ipatool v2.8.0+, MIT).
//
// It serves Apple device-verification (Anisette/ADI) headers over HTTP so
// that an iOS app on the same network — NVDAppleIDDownload — can attach them
// to its App Store sign-in requests. This is the same architecture AltStore
// and SideStore use with Dadoum's anisette-server, except the engine here is
// ipatool's modern ADI implementation, which is the only one known to speak
// Apple's post-November-2026 otp/1.5 provisioning flow. (The older Sierra
// generation explicitly cannot consume that flow; see
// internal/storeauth/anisette.go in this repository.)
//
// Security model — read this before running it:
//
//   - The server NEVER receives, logs, or handles an Apple ID or password.
//     It only proves "this machine is a device" and returns the resulting
//     headers. Sign-in itself still happens inside the app, with the
//     user's own credentials.
//   - By default it listens only on 127.0.0.1:6969. Use --listen to bind
//     another address (for example the PC's LAN IP so a phone can reach
//     it), and only on a network you trust.
//   - The provisioned machine identity (adi.json) lives in the data
//     directory and is private to this machine. Do NOT share it: anyone
//     holding it can impersonate this machine's device identity to Apple.
//   - First run downloads ~11 GB of Apple macOS restore data (IPSW
//     27.0.1) from updates.cdn-apple.com and extracts 7 Apple binaries
//     from it. Those binaries are Apple-copyrighted code: they are NOT
//     bundled in this program and are NOT redistributed by it — they are
//     fetched from Apple's own CDN at runtime, exactly like ipatool does,
//     and cached locally. Allow plenty of free disk space and time for
//     the first provisioning.
//
// Endpoints:
//
//	GET /          same as /anisette
//	GET /anisette  fresh Anisette headers as JSON
//	GET /health    {"status":"ok","provisioned":...} — never provisions
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/majd/ipatool/v2/internal/storeauth"
)

const (
	// DefaultListen matches Dadoum's anisette-server convention (port 6969).
	DefaultListen = "127.0.0.1:6969"
	// StateFileName holds the provisioned machine identity (Identifier +
	// Provisioning blob, JSON). It is the server-side equivalent of the
	// adi.pb state ipatool keeps in the OS keychain.
	StateFileName = "adi.json"
	// defaultGenerateTimeout bounds one header generation (including the
	// first-run ~11 GB IPSW download). Zero means no timeout.
	defaultGenerateTimeout = 60 * time.Minute
)

var version = "dev"

type server struct {
	mu           sync.Mutex
	statePath    string
	state        []byte // persisted Anisette state; empty until first provision
	provisioned  bool
	provisioning bool
	client       *http.Client
	genTimeout   time.Duration
}

func main() {
	listen := flag.String("listen", DefaultListen, "address to listen on, e.g. 127.0.0.1:6969 (use 0.0.0.0:6969 only on a trusted LAN)")
	dataDir := flag.String("data-dir", "", "directory for adi.json (default: %APPDATA%/NVD-Anisette-Server on Windows, ~/.config/NVD-Anisette-Server elsewhere)")
	provisionOnStart := flag.Bool("provision-on-start", false, "provision the machine identity at startup instead of on the first /anisette request")
	genTimeout := flag.Duration("generate-timeout", defaultGenerateTimeout, "max time for one header generation (first run downloads ~11 GB from Apple); 0 = no timeout")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("NVD-Anisette-Server %s (ipatool Anisette engine)\n", version)
		return
	}

	dir := *dataDir
	if dir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			log.Fatalf("cannot find config directory (use --data-dir): %v", err)
		}
		dir = filepath.Join(configDir, "NVD-Anisette-Server")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Fatalf("cannot create data directory %s: %v", dir, err)
	}
	statePath := filepath.Join(dir, StateFileName)

	s := &server{
		statePath:  statePath,
		client:     &http.Client{Timeout: 120 * time.Second},
		genTimeout: *genTimeout,
	}
	if data, err := os.ReadFile(statePath); err == nil {
		s.state = data
		s.provisioned = true // state exists; OTP generation still re-provisions each call
		log.Printf("loaded existing machine identity from %s", statePath)
	} else if !os.IsNotExist(err) {
		log.Fatalf("cannot read state file %s: %v", statePath, err)
	}

	if !strings.HasPrefix(*listen, "127.0.0.1") && !strings.HasPrefix(*listen, "localhost") {
		log.Printf("WARNING: listening on %s — anyone on that network can request Anisette headers from this machine. Only do this on a trusted LAN.", *listen)
	}

	if *provisionOnStart {
		log.Printf("provisioning machine identity now (first run downloads ~11 GB from Apple, please wait)...")
		if _, err := s.generate(context.Background()); err != nil {
			log.Fatalf("initial provisioning failed: %v", err)
		}
		log.Printf("machine identity provisioned and saved to %s", statePath)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleAnisette)
	mux.HandleFunc("/anisette", s.handleAnisette)
	mux.HandleFunc("/health", s.handleHealth)

	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 30 * time.Second,
	}

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("cannot listen on %s: %v", *listen, err)
	}

	log.Printf("NVD-Anisette-Server %s listening on http://%s", version, *listen)
	log.Printf("Endpoints: GET /health, GET /anisette (and GET /)")
	log.Printf("Data directory: %s (machine identity: %s — DO NOT SHARE)", dir, statePath)
	log.Printf("This server never receives or logs any Apple ID or password.")
	if !s.provisioned {
		log.Printf("First /anisette request will provision this machine: ~11 GB download from Apple, may take a long time. Check progress with GET /health (provisioning=true).")
	}

	go func() {
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

// generate returns a fresh set of Anisette headers. Every call runs the full
// machine provisioning + OTP flow through ipatool's engine, so headers are
// never reused across requests. Calls are serialised: the underlying ADI
// runtime and the state file are not safe for concurrent use.
func (s *server) generate(ctx context.Context) (http.Header, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.provisioning = true
	defer func() { s.provisioning = false }()

	if s.genTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.genTimeout)
		defer cancel()
	}

	headers, updated, err := storeauth.AnisetteHeaders(ctx, s.client, s.state)
	if err != nil {
		return nil, err
	}

	if err := writeFileAtomic(s.statePath, updated, 0o600); err != nil {
		return nil, fmt.Errorf("save machine identity: %w", err)
	}
	s.state = updated
	s.provisioned = true

	return headers, nil
}

func (s *server) handleAnisette(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/" && r.URL.Path != "/anisette" {
		http.NotFound(w, r)
		return
	}

	headers, err := s.generate(r.Context())
	if err != nil {
		log.Printf("anisette generation failed: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("anisette generation failed: %v", err),
		})
		return
	}

	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) > 0 {
			out[key] = values[0]
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	provisioned, provisioning := s.provisioned, s.provisioning
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"version":      version,
		"provisioned":  provisioned,
		"provisioning": provisioning,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// loggingMiddleware logs one short line per request: method, path, status,
// duration. It never logs request bodies or headers — this server must never
// see an Apple ID or password, and its logs must prove that.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, recorder.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
