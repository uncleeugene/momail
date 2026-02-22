package monitor

import (
	"fmt"
	"sync"
	"time"
)

// SessionInfo represents the state of a single active BinkP session.
type SessionInfo struct {
	ID          string    `json:"id"`
	Remote      string    `json:"remote"`
	Direction   string    `json:"direction"` // "IN" or "OUT"
	State       string    `json:"state"`     // "Handshake", "Receiving", "Sending"
	CurrentFile string    `json:"current_file"`
	FilePos     int64     `json:"file_pos"`
	FileSize    int64     `json:"file_size"`
	StartedAt   time.Time `json:"started_at"`
	EndTime     time.Time `json:"end_time,omitempty"`
}

// QueueEntry represents a node in the outbound queue.
type QueueEntry struct {
	Address      string `json:"address"`
	Flavor       string `json:"flavor"`        // "Crash", "Normal", "Direct", "Hold"
	Files        int    `json:"files"`         // Number of files in the flow file
	NetmailSize  int64  `json:"netmail_size"`  // Total size of netmail bundles
	EchomailSize int64  `json:"echomail_size"` // Total size of echomail bundles
	IsSuspended  bool   `json:"is_suspended"`  // Whether the node is suspended
	IsBusy       bool   `json:"is_busy"`       // Whether the node is busy
}

// AppStatus represents the global state of the mailer.
type AppStatus struct {
	Address        string                 `json:"address"`
	Version        string                 `json:"version"`
	Uptime         string                 `json:"uptime"`
	Sessions       map[string]SessionInfo `json:"sessions"`
	RecentSessions []SessionInfo          `json:"recent_sessions"`
	NextScanIn     int                    `json:"next_scan_in"`
	Muted          bool                   `json:"muted"`
	OutboundQueue  []QueueEntry           `json:"outbound_queue"`
}

// OutboundQueue manages the list of nodes waiting to be polled.
type OutboundQueue struct {
	items []QueueEntry
	mu    sync.RWMutex
}

func (q *OutboundQueue) Set(items []QueueEntry) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = items
}

func (q *OutboundQueue) Get() []QueueEntry {
	q.mu.RLock()
	defer q.mu.RUnlock()
	items := make([]QueueEntry, len(q.items))
	copy(items, q.items)
	return items
}

var (
	mu          sync.RWMutex
	sessions    = make(map[string]SessionInfo)
	startTime   = time.Now()
	nextScan    time.Time
	nodeAddress string
	isMuted     bool
	history     []SessionInfo
	Queue       = &OutboundQueue{}
)

// SetNodeAddress sets the node address for monitoring.
func SetNodeAddress(addr string) {
	mu.Lock()
	defer mu.Unlock()
	nodeAddress = addr
}

// SetMuted updates the muted state of the mailer.
func SetMuted(muted bool) {
	mu.Lock()
	defer mu.Unlock()
	isMuted = muted
}

// IsMuted returns the current muted state.
func IsMuted() bool {
	mu.RLock()
	defer mu.RUnlock()
	return isMuted
}

// SetNextScan updates the time of the next scheduled scan.
func SetNextScan(t time.Time) {
	mu.Lock()
	defer mu.Unlock()
	nextScan = t
}

// RegisterSession adds or updates a session in the monitor.
func RegisterSession(id string, info SessionInfo) {
	mu.Lock()
	defer mu.Unlock()
	sessions[id] = info
}

// UnregisterSession removes a session from the monitor.
func UnregisterSession(id string) {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := sessions[id]; ok {
		delete(sessions, id)
		history = append([]SessionInfo{s}, history...)
		if len(history) > 10 {
			history = history[:10]
		}
	}
}

// HasActiveSessions checks if there are any active sessions.
func HasActiveSessions() bool {
	mu.RLock()
	defer mu.RUnlock()
	return len(sessions) > 0
}

// IsNodeActive checks if there is an active session for the given address.
func IsNodeActive(addr string) bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, s := range sessions {
		if s.Remote == addr {
			return true
		}
	}
	return false
}

// UpdateProgress is a helper to update just the file progress of a session.
func UpdateProgress(id string, state, file string, pos, size int64) {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := sessions[id]; ok {
		s.State = state
		s.CurrentFile = file
		s.FilePos = pos
		s.FileSize = size
		sessions[id] = s
	}
}

// GetStatus returns a snapshot of the current application state.
func GetStatus() AppStatus {
	mu.RLock()
	defer mu.RUnlock()

	// Copy map to avoid race conditions during JSON marshaling
	sessionsCopy := make(map[string]SessionInfo, len(sessions))
	for k, v := range sessions {
		sessionsCopy[k] = v
	}

	historyCopy := make([]SessionInfo, len(history))
	copy(historyCopy, history)

	// Copy queue to avoid race conditions
	queueCopy := Queue.Get()

	var nextScanIn int
	if !nextScan.IsZero() {
		if d := time.Until(nextScan); d > 0 {
			nextScanIn = int(d.Seconds())
		}
	}

	duration := time.Since(startTime).Round(time.Second)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	uptimeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	return AppStatus{
		Address:        nodeAddress,
		Version:        "0.1",
		Uptime:         uptimeStr,
		Sessions:       sessionsCopy,
		RecentSessions: historyCopy,
		NextScanIn:     nextScanIn,
		Muted:          isMuted,
		OutboundQueue:  queueCopy,
	}
}
