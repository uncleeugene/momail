package api

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"uncleeugene.kz/momail/binkp"
	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/ftn"
	"uncleeugene.kz/momail/logstream"
	"uncleeugene.kz/momail/logutil"
	"uncleeugene.kz/momail/monitor"
	"uncleeugene.kz/momail/scheduler"
)

//go:embed dashboard.html
var dashboardHTML []byte

// Start initializes the HTTP API server.
func Start(ctx context.Context, cfg *config.Config, reloadFunc func()) {
	if cfg.APIPort == 0 {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", handleStatus)
	mux.Handle("/api/control", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleControl(w, r, cfg, reloadFunc)
	}))
	mux.HandleFunc("/api/queue", handleQueue)
	mux.HandleFunc("/api/queue/", func(w http.ResponseWriter, r *http.Request) {
		handleQueueItem(w, r, cfg)
	})
	mux.HandleFunc("/api/nodelist/", func(w http.ResponseWriter, r *http.Request) {
		handleNodelist(w, r, cfg)
	})
	mux.HandleFunc("/api/logstream", logstream.ServeWs)
	mux.HandleFunc("/dashboard", handleDashboard)

	// Wrap mux with authentication middleware
	handler := authMiddleware(cfg, mux)

	addr := fmt.Sprintf(":%d", cfg.APIPort)
	server := &http.Server{Addr: addr, Handler: handler}

	go func() {
		log.Println(logutil.Info("API server listening on %s", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println(logutil.Error("API server error: %v", err))
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()
}

func authMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.APIToken != "" {
			authHeader := r.Header.Get("Authorization")
			// Allow token in query parameter for WebSockets or simple browser access
			if authHeader == "" {
				token := r.URL.Query().Get("token")
				if token != "" {
					authHeader = "Bearer " + token
				}
			}

			if authHeader != "Bearer "+cfg.APIToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(dashboardHTML)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := monitor.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func handleQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	queue := monitor.Queue.Get()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queue)
}

func handleQueueItem(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	addrStr := strings.TrimPrefix(r.URL.Path, "/api/queue/")
	if addrStr == "" {
		http.Error(w, "Address required", http.StatusBadRequest)
		return
	}

	target, err := ftn.ParseFidoAddress(addrStr, cfg.ParsedAddress.Zone)
	if err != nil {
		http.Error(w, "Invalid address", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodDelete {
		flows := binkp.FindFlowFiles(cfg, target)
		if len(flows) == 0 {
			http.Error(w, "Poll not found", http.StatusNotFound)
			return
		}

		if r.URL.Query().Get("delete_files") == "true" {
			for _, flow := range flows {
				f, err := os.Open(flow.Path)
				if err == nil {
					scanner := bufio.NewScanner(f)
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if line == "" || strings.HasPrefix(line, "#") {
							continue
						}
						path := line
						if strings.HasPrefix(line, "^") {
							path = line[1:]
						}
						os.Remove(path)
						log.Println(logutil.Info("API: Deleted file %s", path))
					}
					f.Close()
				}
			}
		}

		for _, flow := range flows {
			os.Remove(flow.Path)
		}
		log.Println(logutil.Info("API: Deleted poll for %s", target))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "message": "Poll deleted"}`))
		return
	}

	if r.Method == http.MethodPatch {
		var update struct {
			Flavor    *string `json:"flavor"`
			Suspended *bool   `json:"suspended"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "Invalid body", http.StatusBadRequest)
			return
		}

		if update.Suspended != nil {
			hldPath, err := scheduler.GetHoldFilePath(cfg, target)
			if err == nil {
				if *update.Suspended {
					if err := os.WriteFile(hldPath, []byte("suspended via api"), 0644); err != nil {
						log.Println(logutil.Error("API: Failed to suspend %s: %v", target, err))
						http.Error(w, "Failed to suspend node", http.StatusInternalServerError)
						return
					}
					log.Println(logutil.Info("API: Suspended node %s", target))
				} else {
					os.Remove(hldPath)
					log.Println(logutil.Info("API: Unsuspended node %s", target))
				}
			}
		}

		if update.Flavor != nil {
			flows := binkp.FindFlowFiles(cfg, target)
			if len(flows) > 0 {
				newPath, err := scheduler.GetPollFilePath(cfg, target, *update.Flavor)
				if err == nil {
					newExt := filepath.Ext(newPath)
					for _, flow := range flows {
						oldExt := filepath.Ext(flow.Path)
						if oldExt == newExt {
							continue
						}
						base := strings.TrimSuffix(flow.Path, oldExt)
						dest := base + newExt
						if err := os.Rename(flow.Path, dest); err != nil {
							log.Println(logutil.Error("API: Failed to change flavor for %s: %v", target, err))
						} else {
							log.Println(logutil.Info("API: Changed flavor for %s to %s", target, *update.Flavor))
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "message": "Node updated"}`))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleNodelist(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	addrStr := strings.TrimPrefix(r.URL.Path, "/api/nodelist/")
	if addrStr == "" {
		http.Error(w, "Address required", http.StatusBadRequest)
		return
	}

	node, err := scheduler.GetNode(addrStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Node not found: %v", err), http.StatusNotFound)
		return
	}

	response := struct {
		Address  string   `json:"address"`
		Sysop    string   `json:"sysop"`
		Location string   `json:"location"`
		Flags    []string `json:"flags"`
		Phone    string   `json:"phone"`
		DNS      string   `json:"dns"`
		DNSRoot  string   `json:"dns_root"`
	}{
		Address:  node.Address.String(),
		Sysop:    node.Sysop,
		Location: node.Location,
		Flags:    node.Flags,
		Phone:    node.Phone,
	}

	if !cfg.DisableDNS {
		response.DNSRoot = cfg.DNSRoot
		if host, port, ok := scheduler.LookupBinkpNet(&node.Address, cfg.DNSRoot); ok {
			response.DNS = fmt.Sprintf("%s:%d", host, port)
		}
	} else {
		response.DNSRoot = "Disabled"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleControl(w http.ResponseWriter, r *http.Request, cfg *config.Config, reloadFunc func()) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Action    string `json:"action"`
		Address   string `json:"address"`
		Normal    bool   `json:"normal"`
		Flavor    string `json:"flavor"`
		Command   string `json:"command"`
		Suspended bool   `json:"suspended"`
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"status": "error", "message": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	switch reqBody.Action {
	case "force_scan":
		scheduler.TriggerScan("API request")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "Outbound scan triggered"}`))
	case "reload_config":
		if reloadFunc != nil {
			reloadFunc()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok", "message": "Config reload triggered"}`))
		} else {
			http.Error(w, `{"status": "error", "message": "Reload function not available"}`, http.StatusInternalServerError)
		}
	case "mute":
		monitor.SetMuted(true)
		log.Println(logutil.Warn("System muted via API"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "System muted"}`))
	case "unmute":
		monitor.SetMuted(false)
		log.Println(logutil.Info("System unmuted via API"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "message": "System unmuted"}`))
	case "queue_poll":
		if reqBody.Address == "" {
			http.Error(w, `{"status": "error", "message": "address is required for queue_poll"}`, http.StatusBadRequest)
			return
		}
		target, err := ftn.ParseFidoAddress(reqBody.Address, cfg.ParsedAddress.Zone)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"status": "error", "message": "invalid address: %v"}`, err), http.StatusBadRequest)
			return
		}
		flavor := reqBody.Flavor
		if flavor == "" {
			if reqBody.Normal {
				flavor = "normal"
			} else {
				flavor = "crash"
			}
		}

		if reqBody.Suspended {
			hldPath, err := scheduler.GetHoldFilePath(cfg, target)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"status": "error", "message": "could not get hold file path: %v"}`, err), http.StatusInternalServerError)
				return
			}
			if err := scheduler.CreateHoldFile(hldPath); err != nil {
				http.Error(w, fmt.Sprintf(`{"status": "error", "message": "failed to create hold file: %v"}`, err), http.StatusInternalServerError)
				return
			}
		}

		pollPath, err := scheduler.GetPollFilePath(cfg, target, flavor)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"status": "error", "message": "could not get poll file path: %v"}`, err), http.StatusInternalServerError)
			return
		}
		f, err := os.Create(pollPath)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"status": "error", "message": "failed to create poll file: %v"}`, err), http.StatusInternalServerError)
			return
		}
		f.Close()
		log.Println(logutil.Warn("Poll (%s) for %s queued via API", flavor, target))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ok", "message": "Poll queued for %s"}`, target)
	case "run_task":
		if reqBody.Command == "" {
			http.Error(w, `{"status": "error", "message": "command is required for run_task"}`, http.StatusBadRequest)
			return
		}
		for _, task := range cfg.Tasks {
			if task.Command == reqBody.Command {
				log.Println(logutil.Warn("Executing task '%s' via API", task.Command))
				cmd := exec.Command("sh", "-c", task.Command)
				if output, err := cmd.CombinedOutput(); err != nil {
					log.Println(logutil.Error("API-triggered task execution failed: %v\nOutput: %s", err, string(output)))
					http.Error(w, fmt.Sprintf(`{"status": "error", "message": "task execution failed: %v"}`, err), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"status": "ok", "message": "Task '%s' executed"}`, task.Command)
				return
			}
		}
		http.Error(w, fmt.Sprintf(`{"status": "error", "message": "task with command '%s' not found in config"}`, reqBody.Command), http.StatusNotFound)
	default:
		http.Error(w, fmt.Sprintf(`{"status": "error", "message": "unknown action: %s"}`, reqBody.Action), http.StatusBadRequest)
	}
}
