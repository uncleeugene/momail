package scheduler

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"uncleeugene.kz/momail/binkp"
	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/ftn"
	"uncleeugene.kz/momail/logutil"
	"uncleeugene.kz/momail/monitor"
	"uncleeugene.kz/momail/nodelist"
)

var triggerScan = make(chan string, 1)
var lastLogState = make(map[string]string)
var globalNodelist *nodelist.List
var failureCounts = make(map[string]int)
var lastAttemptTime = make(map[string]time.Time)

// TriggerScan sends a signal to start an immediate outbound scan.
func TriggerScan(reason string) {
	select {
	case triggerScan <- reason:
	default: // Don't block if a scan is already queued
	}
}

// Start initializes the scheduler to poll for outgoing mail.
func Start(cfg *config.Config, initialScan bool) (shutdown func()) {
	// Ensure outbound directory exists so we can watch it
	if err := os.MkdirAll(cfg.Outbound, 0755); err != nil {
		log.Println(logutil.Error("Failed to create outbound directory: %v", err))
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(logutil.Error("Failed to create outbound watcher: %v", err))
		return func() {}
	}

	// Initialize global nodelist
	globalNodelist = &nodelist.List{Nodes: make(map[string]nodelist.Node)}

	loadList := func(path string) {
		log.Println(logutil.Info("Loading nodelist from %s...", path))
		if nl, err := nodelist.Load(path); err != nil {
			log.Println(logutil.Error("Failed to load nodelist: %v", err))
		} else {
			globalNodelist.Merge(nl)
			log.Println(logutil.Success("Loaded %d nodes from %s.", len(nl.Nodes), path))
		}
	}

	if cfg.NodelistDir != "" {
		bases := cfg.Nodelists
		// Fallback for backward compatibility: if no list specified, find *any* latest file
		if len(bases) == 0 {
			bases = []string{""}
		}

		for _, base := range bases {
			if latest, err := nodelist.FindLatest(cfg.NodelistDir, base); err == nil {
				log.Println(logutil.Info("Found latest nodelist for '%s': %s", base, latest))
				loadList(latest)
			} else if base != "" {
				log.Println(logutil.Warn("No nodelist found for base '%s' in %s", base, cfg.NodelistDir))
			}
		}
	}

	// Add outbound directories recursively
	err = filepath.Walk(cfg.Outbound, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				log.Println(logutil.Error("Failed to watch %s: %v", path, err))
			}
		}
		return nil
	})
	if err != nil {
		log.Println(logutil.Error("Failed to walk outbound directory: %v", err))
	}

	quit := make(chan struct{})

	go func() {
		defer watcher.Close()
		log.Println(logutil.Debug("Scheduler started. Watching %s for changes...", cfg.Outbound))

		minDelay := time.Duration(cfg.RescanPeriod) * time.Second
		ticker := time.NewTicker(minDelay)
		defer ticker.Stop()

		// Track last scan time to enforce minDelay
		lastScan := time.Now().Add(-minDelay) // Allow immediate first scan

		// Initial scan
		if initialScan {
			ScanAndPoll(cfg)
			lastScan = time.Now()
			ticker.Reset(minDelay)
		}
		monitor.SetNextScan(time.Now().Add(minDelay))

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// If directory created, add to watcher
				if event.Op&fsnotify.Create == fsnotify.Create {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						watcher.Add(event.Name)
					}
				}

				//// Rescan immediately on outbound change
				if !monitor.HasActiveSessions() {
					scanOutbound(cfg)
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println(logutil.Error("Watcher error: %v", err))

			case <-ticker.C:
				if !monitor.HasActiveSessions() {
					ScanAndPoll(cfg)
					lastScan = time.Now()
					monitor.SetNextScan(lastScan.Add(minDelay))
				}

			case reason := <-triggerScan:
				log.Println(logutil.Warn("Outbound scan triggered due to %s", reason))
				if !monitor.HasActiveSessions() {
					ScanAndPoll(cfg)
					lastScan = time.Now()
					ticker.Reset(minDelay)
					monitor.SetNextScan(lastScan.Add(minDelay))
				} else {
					log.Println(logutil.Debug("Scan skipped: session active"))
				}

			case <-quit:
				return
			}
		}
	}()

	return func() {
		close(quit)
	}
}

// Scan checks the outbound directory and updates the monitor status.

func ScanAndPoll(cfg *config.Config) {
	if monitor.IsMuted() {
		log.Println(logutil.Warn("System is muted. Skipping outbound scan."))
		return
	}

	queue := scanOutbound(cfg)
	seen := make(map[string]bool)

	// 3. Poll nodes in the queue
	for _, entry := range queue {
		seen[entry.Address] = true

		// Check retry delay to prevent rapid looping on failures
		if !lastAttemptTime[entry.Address].IsZero() && time.Since(lastAttemptTime[entry.Address]) < time.Duration(cfg.RescanPeriod)*time.Second {
			continue
		}

		// Check if node is already active (e.g. incoming session)
		if monitor.IsNodeActive(entry.Address) {
			log.Println(logutil.Debug("Node %s is already active. Skipping.", entry.Address))
			continue
		}

		// Check busy status
		target, err := ftn.ParseFidoAddress(entry.Address, 0)
		var bsyPath string
		var hldPath string
		if err == nil {
			bsyPath, _ = GetBusyFilePath(cfg, target)
			hldPath, _ = GetHoldFilePath(cfg, target)

			// Check if node is suspended (.hld flag)
			if hldPath != "" {
				if info, err := os.Stat(hldPath); err == nil {
					if time.Since(info.ModTime()) < time.Duration(cfg.HoldTime)*time.Minute {
						if lastLogState[entry.Address] != "Suspended" {
							log.Println(logutil.Warn("Node %s is suspended until %s. Skipping.", entry.Address, info.ModTime().Add(time.Duration(cfg.HoldTime)*time.Minute).Format("15:04:05")))
							lastLogState[entry.Address] = "Suspended"
						}
						continue
					}
					// Expired, remove it
					log.Println(logutil.Info("Suspension expired for %s. Removing flag.", entry.Address))
					os.Remove(hldPath)
					failureCounts[entry.Address] = 0
				}
			}

			if bsyPath != "" {
				if info, err := os.Stat(bsyPath); err == nil {
					// Check if stale
					if cfg.StaleBusyTimeout > 0 && time.Since(info.ModTime()) > time.Duration(cfg.StaleBusyTimeout)*time.Hour {
						log.Println(logutil.Warn("Removing stale busy flag for %s (age: %s)", entry.Address, time.Since(info.ModTime()).Round(time.Minute)))
						if err := os.Remove(bsyPath); err != nil {
							log.Println(logutil.Error("Failed to remove stale busy flag: %v", err))
							continue
						}
					} else {
						log.Println(logutil.Debug("Node %s is busy (locked). Skipping.", entry.Address))
						continue
					}
				}
			}
		}

		// Determine if we should log lookup details to avoid spam on Hold nodes
		shouldLog := true
		if entry.Flavor == "Hold" && lastLogState[entry.Address] == entry.Flavor {
			shouldLog = false
		}
		link := findLink(cfg, entry.Address, shouldLog)
		if link != nil {
			if entry.Flavor != "Hold" {
				if bsyPath != "" {
					createBusyFile(bsyPath)
				}

				log.Println(logutil.Info("Found mail for %s. Initiating connection...", link.Address))
				// Update state so we know the last seen state was not Hold
				lastLogState[entry.Address] = entry.Flavor
				lastAttemptTime[entry.Address] = time.Now()
				if err := binkp.Dial(cfg, link); err != nil {
					log.Println(logutil.Error("Failed to call %s: %v", link.Address, err))

					// Handle failure counting
					failureCounts[entry.Address]++
					if failureCounts[entry.Address] >= cfg.MaxDialAttempts {
						log.Println(logutil.Warn("Max dial attempts (%d) reached for %s. Suspending for %d minutes.", cfg.MaxDialAttempts, entry.Address, cfg.HoldTime))
						CreateHoldFile(hldPath)
						failureCounts[entry.Address] = 0
					}
				} else {
					failureCounts[entry.Address] = 0
				}
				lastAttemptTime[entry.Address] = time.Now()
				lastAttemptTime[entry.Address] = time.Now()

				if bsyPath != "" {
					os.Remove(bsyPath)
				}
			} else {
				// Only report Hold status if it has changed (or is new)
				if lastLogState[entry.Address] != entry.Flavor {
					log.Println(logutil.Warn("Mail for %s is on hold. Skipping...", link.Address))
					lastLogState[entry.Address] = entry.Flavor
				}
			}
		}
	}

	// Clean up state for nodes that no longer have mail in the queue
	for addr := range lastLogState {
		if !seen[addr] {
			delete(lastLogState, addr)
		}
	}
	for addr := range lastAttemptTime {
		if !seen[addr] {
			delete(lastAttemptTime, addr)
		}
	}
}

func scanOutbound(cfg *config.Config) []monitor.QueueEntry {
	var queue []monitor.QueueEntry

	// Helper struct to aggregate data per address
	type queueData struct {
		flavor       string
		priority     int
		netmailSize  int64
		echomailSize int64
		fileCount    int
	}
	aggregatedData := make(map[string]*queueData)

	// Helper to update flows map
	updateFlow := func(addr, path, ext string) {
		data, exists := aggregatedData[addr]
		if !exists {
			data = &queueData{priority: 99} // a high number for initial comparison
			aggregatedData[addr] = data
		}

		prio := getPriorityFromExt(ext)
		if prio < data.priority {
			data.priority = prio
			data.flavor = getStateFromExt(ext)
		}

		if strings.HasSuffix(ext, "ut") { // Netmail (.cut, .out, etc)
			if info, err := os.Stat(path); err == nil {
				data.netmailSize += info.Size()
				data.fileCount++
			}
		} else { // Echomail (.clo, .lo, etc)
			count, size, _ := calculateTotalSizeInFlow(path)
			data.echomailSize += size
			data.fileCount += count
		}
	}
	// Read root outbound directory
	entries, err := os.ReadDir(cfg.Outbound)
	if err != nil {
		log.Println(logutil.Error("Failed to read outbound directory: %v", err))
		return queue
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		lowerName := strings.ToLower(name)
		var zone uint16

		if lowerName == "outbound" {
			zone = cfg.DefaultZone
		} else if strings.HasPrefix(lowerName, "outbound.") && len(lowerName) == 12 {
			// outbound.ZZZ
			if z, err := strconv.ParseUint(lowerName[9:], 16, 16); err == nil {
				zone = uint16(z)
			} else {
				continue
			}
		} else {
			continue
		}

		// Scan the zone directory
		zoneDir := filepath.Join(cfg.Outbound, name)
		zoneEntries, err := os.ReadDir(zoneDir)
		if err != nil {
			continue
		}

		for _, zEntry := range zoneEntries {
			zName := zEntry.Name()
			lowerZName := strings.ToLower(zName)

			if zEntry.IsDir() {
				// Check for point directory: NNNNFFFF.pnt
				if len(lowerZName) == 12 && strings.HasSuffix(lowerZName, ".pnt") {
					if netID, err := strconv.ParseUint(lowerZName[:4], 16, 16); err == nil {
						if nodeID, err := strconv.ParseUint(lowerZName[4:8], 16, 16); err == nil {
							// Scan point directory
							pntDir := filepath.Join(zoneDir, zName)
							pntEntries, err := os.ReadDir(pntDir)
							if err == nil {
								for _, pEntry := range pntEntries {
									if !pEntry.IsDir() && len(pEntry.Name()) == 12 {
										pName := pEntry.Name()
										pExt := strings.ToLower(pName[8:])
										if isValidFlowExt(pExt) {
											if pointID, err := strconv.ParseUint(pName[4:8], 16, 16); err == nil {
												addr := fmt.Sprintf("%d:%d/%d.%d", zone, netID, nodeID, pointID)
												updateFlow(addr, filepath.Join(pntDir, pName), pExt)
											}
										}
									}
								}
							}
						}
					}
				}
			} else {
				// File: NNNNFFFF.ext
				if len(lowerZName) == 12 {
					ext := lowerZName[8:]
					if isValidFlowExt(ext) {
						if netID, err := strconv.ParseUint(lowerZName[:4], 16, 16); err == nil {
							if nodeID, err := strconv.ParseUint(lowerZName[4:8], 16, 16); err == nil {
								addr := fmt.Sprintf("%d:%d/%d", zone, netID, nodeID)
								updateFlow(addr, filepath.Join(zoneDir, zName), ext)
							}
						}
					}
				}
			}
		}
	}

	// Convert map to queue
	for addr, data := range aggregatedData {
		var isBusy, isSuspended bool
		if target, err := ftn.ParseFidoAddress(addr, 0); err == nil {
			if bsyPath, err := GetBusyFilePath(cfg, target); err == nil {
				if _, err := os.Stat(bsyPath); err == nil {
					isBusy = true
				}
			}
			if hldPath, err := GetHoldFilePath(cfg, target); err == nil {
				if _, err := os.Stat(hldPath); err == nil {
					isSuspended = true
				}
			}
		}

		queue = append(queue, monitor.QueueEntry{
			Address:      addr,
			Flavor:       data.flavor,
			Files:        data.fileCount,
			NetmailSize:  data.netmailSize,
			EchomailSize: data.echomailSize,
			IsBusy:       isBusy,
			IsSuspended:  isSuspended,
		})
	}
	monitor.Queue.Set(queue)
	return queue
}

func findLink(cfg *config.Config, addr string, logLookup bool) *config.Link {
	// Parse the address from the queue to compare structurally
	target, err := ftn.ParseFidoAddress(addr, 0)
	var configLink *config.Link

	if err != nil {
		// Fallback to string comparison if parsing fails
		for i := range cfg.Links {
			if cfg.Links[i].Address == addr {
				if cfg.Links[i].Host != "" {
					if logLookup {
						log.Println(logutil.Info("Resolved %s via config (string match) to %s", addr, cfg.Links[i].Host))
					}
					return &cfg.Links[i]
				}
				return nil
			}
		}
		return nil
	}

	for i := range cfg.Links {
		l := &cfg.Links[i]
		if l.ParsedAddress.Zone == target.Zone &&
			l.ParsedAddress.Net == target.Net &&
			l.ParsedAddress.Node == target.Node &&
			l.ParsedAddress.Point == target.Point {
			if l.Host != "" {
				if logLookup {
					log.Println(logutil.Info("Resolved %s via config to %s", target, l.Host))
				}
				return l
			}
			// Found in config, but no Host. Save it and try to resolve Host via other means.
			configLink = l
			break
		}
	}

	// Define lookup strategies
	lookupNodelist := func() *config.Link {
		if globalNodelist != nil {
			// Nodelist map keys are string representations of the address
			if node, ok := globalNodelist.Nodes[target.String()]; ok {
				host, port, isBinkp := node.GetBinkpAddress()
				if isBinkp {
					if logLookup {
						log.Println(logutil.Info("Resolved %s via nodelist to %s:%d", target, host, port))
					}

					if configLink != nil {
						// Merge config (password) with nodelist (host)
						merged := *configLink
						merged.Host = fmt.Sprintf("%s:%d", host, port)
						return &merged
					}

					// Construct a dynamic link configuration
					return &config.Link{
						Address:       node.Address.String(),
						Host:          fmt.Sprintf("%s:%d", host, port),
						ParsedAddress: &node.Address,
						// Password is left empty (public session)
					}
				}
			}
		}
		return nil
	}

	lookupDNS := func() *config.Link {
		if !cfg.DisableDNS {
			if host, port, ok := LookupBinkpNet(target, cfg.DNSRoot); ok {
				if logLookup {
					log.Println(logutil.Info("Resolved %s via %s to %s:%d", target, cfg.DNSRoot, host, port))
				}

				if configLink != nil {
					// Merge config (password) with DNS (host)
					merged := *configLink
					merged.Host = fmt.Sprintf("%s:%d", host, port)
					return &merged
				}

				return &config.Link{
					Address:       target.String(),
					Host:          fmt.Sprintf("%s:%d", host, port),
					ParsedAddress: target,
				}
			}
		}
		return nil
	}

	// Determine order
	preferDNS := false
	method := cfg.LookupMethod
	if configLink != nil && configLink.LookupMethod != "" {
		method = configLink.LookupMethod
	}
	lowerMethod := strings.ToLower(method)
	if lowerMethod == "dns" || lowerMethod == "binkp" || lowerMethod == "binkp.net" {
		preferDNS = true
	}

	if preferDNS {
		if l := lookupDNS(); l != nil {
			return l
		}
		if l := lookupNodelist(); l != nil {
			return l
		}
	} else {
		if l := lookupNodelist(); l != nil {
			return l
		}
		if l := lookupDNS(); l != nil {
			return l
		}
	}

	return nil
}

func isValidFlowExt(ext string) bool {
	switch ext {
	case ".clo", ".dlo", ".hlo", ".lo", ".flo",
		".cut", ".dut", ".hut", ".out", ".fut":
		return true
	}
	return false
}

func getStateFromExt(ext string) string {
	switch ext {
	case ".clo", ".cut":
		return "Crash"
	case ".dlo", ".dut":
		return "Direct"
	case ".hlo", ".hut":
		return "Hold"
	case ".lo", ".flo", ".out", ".fut":
		return "Normal"
	}
	return "Unknown"
}

func getPriorityFromExt(ext string) int {
	switch ext {
	case ".clo", ".cut":
		return 0
	case ".dlo", ".dut":
		return 1
	case ".hlo", ".hut":
		return 2
	case ".lo", ".out":
		return 3
	case ".flo", ".fut":
		return 4
	}
	return 99
}

// Helper function to count files in a flow file.
func calculateTotalSizeInFlow(path string) (int, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	count := 0
	var totalSize int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
			filePath := line
			if strings.HasPrefix(line, "^") {
				filePath = line[1:]
			}
			if info, err := os.Stat(filePath); err == nil {
				totalSize += info.Size()
			}
		}
	}
	return count, totalSize, scanner.Err()
}

// LookupBinkpNet attempts to resolve a FidoNet address using the binkp.net DNS zone.
func LookupBinkpNet(addr *ftn.FidoAddress, root string) (string, int, bool) {
	// Format: p<Point>.f<Node>.n<Net>.z<Zone>.<root>
	var domain string
	if addr.Point > 0 {
		domain = fmt.Sprintf("p%d.f%d.n%d.z%d.%s", addr.Point, addr.Node, addr.Net, addr.Zone, root)
	} else {
		domain = fmt.Sprintf("f%d.n%d.z%d.%s", addr.Node, addr.Net, addr.Zone, root)
	}

	if ips, err := net.LookupHost(domain); err == nil && len(ips) > 0 {
		return ips[0], 24554, true
	}
	return "", 0, false
}

// GetNode returns the nodelist entry for a given address.
func GetNode(addr string) (*nodelist.Node, error) {
	if globalNodelist == nil {
		return nil, fmt.Errorf("nodelist not loaded")
	}
	target, err := ftn.ParseFidoAddress(addr, 0)
	if err != nil {
		return nil, err
	}
	if node, ok := globalNodelist.Nodes[target.String()]; ok {
		return &node, nil
	}
	return nil, fmt.Errorf("node not found")
}

// CreateHoldFile creates a .hld file to suspend a node.
func CreateHoldFile(path string) error {
	return os.WriteFile(path, []byte("suspended"), 0644)
}

// createBusyFile creates a .bsy file containing the current PID.
func createBusyFile(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// GetBusyFilePath calculates the path for a busy file.
func GetBusyFilePath(cfg *config.Config, remote *ftn.FidoAddress) (string, error) {
	baseDir, nameBase, err := getBsoBase(cfg, remote)
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, nameBase+".bsy"), nil
}

// GetHoldFilePath calculates the path for a hold (.hld) file.
func GetHoldFilePath(cfg *config.Config, remote *ftn.FidoAddress) (string, error) {
	baseDir, nameBase, err := getBsoBase(cfg, remote)
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, nameBase+".hld"), nil
}

// GetPollFilePath calculates the correct Binkley-style path for a poll file.
func GetPollFilePath(cfg *config.Config, remote *ftn.FidoAddress, flavor string) (string, error) {
	baseDir, nameBase, err := getBsoBase(cfg, remote)
	if err != nil {
		return "", err
	}

	ext := ".clo"
	switch strings.ToLower(flavor) {
	case "normal", "flo":
		ext = ".flo"
	case "direct", "dlo":
		ext = ".dlo"
	case "hold", "hlo":
		ext = ".hlo"
	case "crash", "clo":
		ext = ".clo"
	}

	return filepath.Join(baseDir, nameBase+ext), nil
}

func getBsoBase(cfg *config.Config, remote *ftn.FidoAddress) (string, string, error) {
	var baseDir string
	if remote.Zone == cfg.DefaultZone {
		baseDir = filepath.Join(cfg.Outbound, "outbound")
	} else {
		baseDir = filepath.Join(cfg.Outbound, fmt.Sprintf("outbound.%03x", remote.Zone))
	}

	var nameBase string
	if remote.Point != 0 {
		dir := filepath.Join(baseDir, fmt.Sprintf("%04x%04x.pnt", remote.Net, remote.Node))
		baseDir = dir
		nameBase = fmt.Sprintf("0000%04x", remote.Point)
	} else {
		nameBase = fmt.Sprintf("%04x%04x", remote.Net, remote.Node)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", "", err
	}
	return baseDir, nameBase, nil
}
