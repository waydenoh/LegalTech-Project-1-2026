package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/yourusername/sg-playbook-reviewer/internal/exporter"
	"github.com/yourusername/sg-playbook-reviewer/internal/llm"
	"github.com/yourusername/sg-playbook-reviewer/internal/playbook"
	"github.com/yourusername/sg-playbook-reviewer/internal/reviewer"
	"github.com/yourusername/sg-playbook-reviewer/web"
)

type reviewRequest struct {
	Playbook         *playbook.Playbook `json:"playbook"`
	ContractText     string             `json:"contract_text"`
	GenerateRedlines bool               `json:"generate_redlines"`
	Model            string             `json:"model"`
}

func main() {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	defaultModel := os.Getenv("OLLAMA_MODEL")
	if defaultModel == "" {
		defaultModel = "qwen3.5:4b"
	}

	defaultPB, err := playbook.DefaultPlaybook()
	if err != nil {
		log.Fatalf("failed to load default playbook: %v", err)
	}

	var lastPing atomic.Int64
	lastPing.Store(time.Now().Unix())
	shutdownCh := make(chan struct{}, 1)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/ping", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		select {
		case shutdownCh <- struct{}{}:
		default:
		}
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		client := llm.NewOllamaClient(ollamaURL, defaultModel)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		models, err := client.ListModels(ctx)
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"ollama_url": ollamaURL,
			"models":     models,
			"ollama_err": errMsg,
		})
	})

	mux.HandleFunc("GET /api/playbook", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, defaultPB)
	})

	mux.HandleFunc("POST /api/review", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		var req reviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request body"})
			return
		}
		if req.ContractText == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "contract_text is required"})
			return
		}

		pb := defaultPB
		if req.Playbook != nil && len(req.Playbook.Rules) > 0 {
			pb = req.Playbook
		}
		modelName := defaultModel
		if req.Model != "" {
			modelName = req.Model
		}

		client := llm.NewOllamaClient(ollamaURL, modelName)
		eng := reviewer.NewEngine(client, 2)

		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
		defer cancel()

		report, err := eng.ReviewContract(ctx, *pb, req.ContractText, req.GenerateRedlines)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"report":   report,
			"markdown": exporter.ToMarkdown(*pb, *report),
		})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		lastPing.Store(time.Now().Unix())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(web.IndexHTML)
	})

	// Bind to a free ephemeral port so multiple launches never collide
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to bind local port: %v", err)
	}
	appURL := fmt.Sprintf("http://%s", listener.Addr().String())

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(listener)
	}()

	// Launch as a standalone Windows desktop window (Edge WebView app mode — no Chrome, no browser tabs, no terminal window)
	cmd := launchDesktopWindow(appURL)
	if cmd != nil {
		go func() {
			_ = cmd.Wait()
			time.Sleep(800 * time.Millisecond)
			if time.Now().Unix()-lastPing.Load() > 2 {
				select {
				case shutdownCh <- struct{}{}:
				default:
				}
			}
		}()
	}

	<-shutdownCh
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func launchDesktopWindow(url string) *exec.Cmd {
	profileDir := filepath.Join(os.TempDir(), "sg-playbook-reviewer-webview")
	_ = os.MkdirAll(profileDir, 0755)

	if runtime.GOOS == "windows" {
		edgeCandidates := []string{
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
		for _, edge := range edgeCandidates {
			if _, err := os.Stat(edge); err == nil {
				cmd := exec.Command(edge,
					"--app="+url,
					"--window-size=1360,860",
					"--user-data-dir="+profileDir,
					"--no-first-run",
					"--no-default-browser-check",
				)
				if err := cmd.Start(); err == nil {
					return cmd
				}
			}
		}
		// Fallback to default Windows protocol handler if Edge path differs
		cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		_ = cmd.Start()
		return nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
