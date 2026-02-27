package binkp

import (
	"bufio"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/ftn"
	"uncleeugene.kz/momail/logutil"
	"uncleeugene.kz/momail/monitor"
	"uncleeugene.kz/momail/nodelist"
)

// BinkP Commands (v1.0+)
const (
	M_NUL  = 0
	M_ADR  = 1
	M_PWD  = 2
	M_FILE = 3
	M_OK   = 4
	M_EOB  = 5
	M_GOT  = 6
	M_ERR  = 7
	M_BSY  = 8
	M_GET  = 9
	M_SKIP = 10
)

// errSessionFinished is a sentinel error to signal a clean session shutdown.
var errSessionFinished = errors.New("session finished normally")

// ErrRemoteBusy indicates the remote system sent M_BSY.
var ErrRemoteBusy = errors.New("remote system is busy")

// OnSessionEnd is called when a session finishes.
// success is true if the session completed without error.
var OnSessionEnd func(success bool)

// RequestedFile represents a file queued for sending via FREQ.
type RequestedFile struct {
	Path        string
	SendAs      string // Optional: name to send as. If empty, uses filename.
	DeleteAfter bool   // If true, delete file after sending.
	Offset      int64  // Starting offset for partial transfers.
}

// Session handles the state of a BinkP connection.
type Session struct {
	conn   net.Conn
	config *config.Config
	reader *bufio.Reader
	id     string
	dir    string // "incoming" or "outgoing"

	// Receive state
	currentFile   *os.File
	recvName      string
	recvTimestamp string
	recvSize      int64
	recvBytes     int64

	// Remote addresses claimed in M_ADR
	remoteAddrs []*ftn.FidoAddress

	// activeLink is the authenticated link configuration (if any)
	activeLink *config.Link

	// Authentication state
	ourChallenge    string
	remoteChallenge string
	pwdSent         bool

	// remoteLink is the target link configuration for outgoing connections
	remoteLink *config.Link

	// pendingCommands holds triggers to be executed after session
	pendingCommands []string

	// Concurrency control for full-duplex
	wg          sync.WaitGroup
	shutdown    chan struct{}
	eobMutex    sync.Mutex
	weSentEOB   bool
	theySentEOB bool

	// requestedFiles holds paths to files requested by the remote system
	requestedFiles []RequestedFile

	// Statistics
	startTime     time.Time
	filesReceived int
	bytesReceived int64
	filesSent     int
	bytesSent     int64
}

// NewSession creates a new BinkP session.
func NewSession(conn net.Conn, cfg *config.Config, remoteLink *config.Link, direction string) *Session {
	return &Session{
		conn:       conn,
		config:     cfg,
		dir:        direction,
		reader:     bufio.NewReader(conn),
		remoteLink: remoteLink,
		id:         conn.RemoteAddr().String(),
		startTime:  time.Now(),
		shutdown:   make(chan struct{}),
	}
}

func (s *Session) performHandshake() error {
	log.Println(logutil.Debug("Starting handshake..."))
	s.updateMonitor("Handshake", "", 0, 0)

	// Send our initial info
	if err := s.sendHandshake(); err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	var remoteSentADR, authDone bool

	// Handshake loop: continues until we have the remote's address and authentication is resolved.
	for !remoteSentADR || !authDone {
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		isCmd, payload, err := s.readFrame()
		if err != nil {
			return fmt.Errorf("handshake read failed: %w", err)
		}
		if !isCmd {
			return fmt.Errorf("protocol error: received data frame during handshake")
		}
		if len(payload) == 0 {
			continue
		}

		cmd := payload[0]
		data := string(payload[1:])

		switch cmd {
		case M_NUL:
			log.Println(logutil.Muted("[Remote] INFO: %s", data))
			var challenge string
			if strings.HasPrefix(data, "CRAM-MD5-") {
				challenge = data[9:]
			} else if strings.HasPrefix(data, "OPT ") {
				for _, part := range strings.Fields(data) {
					if strings.HasPrefix(part, "CRAM-MD5-") {
						challenge = part[9:]
						break
					}
				}
			}
			if challenge != "" {
				s.remoteChallenge = challenge
				// If we are outgoing and haven't sent PWD, send CRAM response now
				if s.remoteLink != nil && !s.pwdSent {
					s.sendCramResponse(s.remoteLink.Password)
				}
			}
		case M_ADR:
			log.Println(logutil.Remote("[Remote] ADR: %s", data))
			parts := strings.Fields(data)
			for _, p := range parts {
				if idx := strings.Index(p, "@"); idx != -1 {
					p = p[:idx]
				}
				if addr, err := ftn.ParseFidoAddress(p, s.config.ParsedAddress.Zone); err == nil {
					s.remoteAddrs = append(s.remoteAddrs, addr)
				}
			}
			remoteSentADR = true
			s.updateMonitor("Handshake", "", 0, 0)

			// Now that we have their address, we can determine auth state.
			if s.dir == "outgoing" {
				if s.remoteLink != nil && !s.pwdSent {
					if s.remoteChallenge == "" && s.remoteLink.Password != "" {
						// Fallback to plain password if CRAM is not offered by remote
						s.writeCommand(M_PWD, s.remoteLink.Password)
						s.pwdSent = true
					} else if s.remoteLink.Password == "" {
						// Outgoing to a link with no password. Auth is complete.
						log.Println(logutil.Info("Outgoing session to password-less link, auth complete."))
						s.activeLink = s.remoteLink
						authDone = true
					}
				}
			} else { // This is an incoming session
				link := s.findLink()
				if link == nil {
					log.Println(logutil.Warn("No link found for remote address. Session is unauthenticated."))
					authDone = true
				} else if link.Password == "" {
					log.Println(logutil.Info("Link found with no password required. Session is authenticated."))
					s.activeLink = link
					authDone = true
				} else if s.remoteChallenge != "" && !s.pwdSent {
					// Link requires a password, and remote sent a challenge. Respond for mutual auth.
					if link.Password != "" {
						s.sendCramResponse(link.Password)
					}
				}
			}
		case M_PWD:
			log.Println(logutil.Remote("[Remote] PWD: *****"))
			link := s.findLink()
			if link == nil {
				log.Println(logutil.Warn("Unknown node (no address match), treating as unprotected session"))
				s.writeCommand(M_OK, "Unprotected session")
				authDone = true
				continue
			}

			if link.Password == "" {
				s.activeLink = link
				s.writeCommand(M_OK, "Password not required")
				log.Println(logutil.Success("-> No password required for this link, accepted"))
				authDone = true
				continue
			}

			valid := false
			if strings.HasPrefix(data, "CRAM-MD5-") {
				digest := data[9:]
				mac := hmac.New(md5.New, []byte(link.Password))
				challengeBytes, err := hex.DecodeString(s.ourChallenge)
				if err != nil {
					challengeBytes = []byte(s.ourChallenge)
				}
				mac.Write(challengeBytes)
				expected := hex.EncodeToString(mac.Sum(nil))

				if strings.EqualFold(digest, expected) {
					valid = true
				} else {
					mac2 := hmac.New(md5.New, []byte(link.Password))
					mac2.Write([]byte(s.ourChallenge))
					expected2 := hex.EncodeToString(mac2.Sum(nil))
					if strings.EqualFold(digest, expected2) {
						valid = true
					} else {
						log.Println(logutil.Debug("CRAM-MD5 mismatch"))
					}
				}
			} else if data == link.Password {
				valid = true
			}

			if valid {
				s.activeLink = link
				s.writeCommand(M_OK, "Password accepted")
				log.Println(logutil.Success("-> Password accepted"))
				authDone = true
			} else {
				s.writeCommand(M_ERR, "Bad password")
				return fmt.Errorf("authentication failed: bad password")
			}
		case M_OK:
			log.Println(logutil.Muted("[Remote] OK: %s", data))
			if s.remoteLink != nil && s.pwdSent && s.activeLink == nil {
				s.activeLink = s.remoteLink
				log.Println(logutil.Success("-> Remote accepted password, session considered protected"))
				authDone = true
			}
		case M_ERR:
			return fmt.Errorf("handshake failed, remote error: %s", data)
		case M_BSY:
			return ErrRemoteBusy
		default:
			log.Println(logutil.Warn("Received unexpected command %d during handshake", cmd))
		}
	}

	log.Println(logutil.Success("Handshake complete."))
	return nil
}

// Run starts the BinkP protocol loop.
func (s *Session) Run() (err error) {
	if monitor.IsMuted() {
		return fmt.Errorf("system is muted")
	}

	defer func() {
		// Ensure the connection is always closed when the session ends.
		s.conn.Close()

		if s.currentFile != nil {
			s.currentFile.Close()
		}
		s.executePendingTriggers()
		s.logSummary(err)

		finalState := "Finished"
		if err != nil && err != errSessionFinished {
			finalState = "Error"
		}
		s.updateMonitor(finalState, "", 0, 0)
		monitor.UnregisterSession(s.id)
		if OnSessionEnd != nil {
			OnSessionEnd(err == nil || err == errSessionFinished)
		}
	}()

	// 1. Synchronous Handshake & Authentication
	if err = s.performHandshake(); err != nil {
		return err // The defer will handle logging this error
	}

	// Handshake is complete and session is authenticated (or determined to be insecure).
	// `s.activeLink` is now definitively set if applicable.

	// 2. Start concurrent file transfers
	s.wg.Add(1)
	go s.fileSender()

	// 3. This goroutine becomes the file reader
	readerErr := s.fileReader()

	// 4. Signal sender to stop and wait for it to finish
	close(s.shutdown)
	s.wg.Wait()

	// 5. Determine final error state
	if readerErr != nil && readerErr != errSessionFinished {
		err = readerErr
	}

	return err // The named return `err` will be used by the defer
}

func (s *Session) fileSender() {
	defer s.wg.Done()

	s.updateMonitor("Sending", "", 0, 0)
	if err := s.sendFiles(); err != nil {
		// A sending error should terminate the session.
		log.Println(logutil.Error("Error during send: %v. Closing session.", err))
		s.conn.Close() // Force the reader to exit
	} else {
		s.eobMutex.Lock()
		s.weSentEOB = true
		theyAreDone := s.theySentEOB
		s.eobMutex.Unlock()
		if theyAreDone {
			log.Println(logutil.Debug("Both sides sent EOB, closing connection."))
			s.conn.Close()
		}
	}
}

func (s *Session) fileReader() error {
	for {
		s.conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		isCmd, payload, err := s.readFrame()
		if err != nil {
			if err == io.EOF {
				return nil // Connection closed cleanly by remote
			}

			// When the fileSender determines the session is complete, it closes the
			// connection to unblock this fileReader. We check if this was an expected
			// closure and, if so, treat it as a clean exit, not an error.
			s.eobMutex.Lock()
			isFinished := s.weSentEOB && s.theySentEOB
			s.eobMutex.Unlock()
			if isFinished && errors.Is(err, net.ErrClosed) {
				return nil // This was a clean shutdown signal from the sender.
			}
			return err
		}

		if isCmd {
			if err := s.handleCommand(payload); err != nil {
				// errSessionFinished is a signal for a clean exit.
				if err == errSessionFinished {
					return err
				}
				return err
			}
		} else {
			if err := s.handleData(payload); err != nil {
				return err
			}
		}
	}
}

func (s *Session) sendHandshake() error {

	// First thingo send is MD5 challenge for CRAM-MD5 authentication if the remote supports it.
	// This is a common extension and allows us to do mutual authentication without sending
	// passwords in plain text.
	s.ourChallenge = generateChallenge()
	s.writeCommand(M_NUL, "OPT CRAM-MD5-"+s.ourChallenge)

	// M_NUL is used for system info. Format: "KEY Value"
	// Standard BinkP info keys
	if s.config.SystemName != "" {
		s.writeCommand(M_NUL, "SYS "+s.config.SystemName)
	}
	if s.config.Sysop != "" {
		s.writeCommand(M_NUL, "ZYZ "+s.config.Sysop)
	}
	if s.config.Location != "" {
		s.writeCommand(M_NUL, "LOC "+s.config.Location)
	}
	// NDL: NodeList flags/capabilities
	s.writeCommand(M_NUL, "NDL "+s.config.NodelistFlags)
	// VER: Software version
	s.writeCommand(M_NUL, "VER momail v0.1. binkp/1.0")

	// CRAM-MD5: Send our challenge

	// M_ADR sends our address list
	if err := s.writeCommand(M_ADR, s.config.ParsedAddress.String()); err != nil {
		return err
	}

	return nil
}

func (s *Session) writeCommand(cmd byte, data string) error {
	// Frame: [Header (2 bytes)] [Cmd (1 byte)] [Data...]
	// Header = Length | 0x8000 (Command Bit)
	// Length includes the Cmd byte.
	payloadLen := 1 + len(data)
	buf := make([]byte, 2+payloadLen)

	binary.BigEndian.PutUint16(buf[0:2], uint16(payloadLen)|0x8000)
	buf[2] = cmd
	copy(buf[3:], data)

	_, err := s.conn.Write(buf)
	return err
}

func (s *Session) readFrame() (bool, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(s.reader, header); err != nil {
		return false, nil, err
	}

	val := binary.BigEndian.Uint16(header)
	isCmd := (val & 0x8000) != 0
	length := int(val & 0x7FFF)

	payload := make([]byte, length)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return false, nil, err
	}

	return isCmd, payload, nil
}

func (s *Session) handleCommand(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	cmd := payload[0]
	data := string(payload[1:])

	switch cmd {
	case M_FILE:
		return s.handleFile(data)
	case M_EOB:
		return s.handleEOB()
	case M_ERR:
		log.Println(logutil.Error("[Remote] ERR: %s", data))
		return fmt.Errorf("remote error: %s", data)
	case M_BSY:
		log.Println(logutil.Warn("[Remote] BSY: %s", data))
		return ErrRemoteBusy
	case M_GET:
		return s.handleGet(data)
	default:
		log.Println(logutil.Warn("[Remote] Unexpected CMD %d: %s", cmd, data))
	}
	return nil
}

func (s *Session) sendCramResponse(password string) error {
	if password == "" {
		return nil
	}
	mac := hmac.New(md5.New, []byte(password))
	challengeBytes, err := hex.DecodeString(s.remoteChallenge)
	if err != nil {
		challengeBytes = []byte(s.remoteChallenge)
	}
	mac.Write(challengeBytes)
	digest := hex.EncodeToString(mac.Sum(nil))
	s.pwdSent = true
	return s.writeCommand(M_PWD, "CRAM-MD5-"+digest)
}

func generateChallenge() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Session) handleGet(args string) error {
	if s.config.FreqDir == "" {
		log.Println(logutil.Warn("Remote requested file '%s' but filebox is disabled", args))
		return nil
	}

	// M_GET arguments: name [size] [time] [offset].
	parts := strings.Fields(args)
	if len(parts) == 0 {
		return nil
	}
	name := parts[0]
	var offset int64
	if len(parts) > 3 {
		offset, _ = strconv.ParseInt(parts[3], 10, 64)
	}

	if strings.EqualFold(name, "FILES") {
		return s.handleMagicFiles()
	}

	if strings.EqualFold(name, "NODELIST") {
		return s.handleMagicNodelist()
	}

	// Security: Prevent directory traversal
	cleanPath := filepath.Clean(filepath.Join(s.config.FreqDir, name))
	absFileBox, _ := filepath.Abs(s.config.FreqDir)
	absPath, _ := filepath.Abs(cleanPath)

	if !strings.HasPrefix(absPath, absFileBox) {
		log.Println(logutil.Error("Security: Remote tried to access invalid path: %s", name))
		return nil
	}

	if _, err := os.Stat(absPath); err != nil {
		log.Println(logutil.Warn("Remote requested missing file: %s", name))
		return nil
	}

	log.Println(logutil.Info("Queuing requested file: %s (offset: %d)", name, offset))
	s.requestedFiles = append(s.requestedFiles, RequestedFile{Path: absPath, Offset: offset})
	return nil
}

func (s *Session) handleMagicFiles() error {
	if err := os.MkdirAll(s.config.TempInbound, 0755); err != nil {
		log.Println(logutil.Error("Failed to create temp dir for magic file: %v", err))
		return nil
	}

	f, err := os.CreateTemp(s.config.TempInbound, "FILES-*.LST")
	if err != nil {
		log.Println(logutil.Error("Failed to create magic file listing: %v", err))
		return nil
	}
	defer f.Close()

	fmt.Fprintf(f, "File listing for %s\r\n", s.config.SystemName)
	fmt.Fprintf(f, "-------------------------------------------------------------------------------\r\n")

	entries, err := os.ReadDir(s.config.FreqDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				info, _ := entry.Info()
				fmt.Fprintf(f, "%-30s %10d  %s\r\n", entry.Name(), info.Size(), info.ModTime().Format("2006-01-02 15:04"))
			}
		}
	} else {
		fmt.Fprintf(f, "Error reading filebox: %v\r\n", err)
	}
	fmt.Fprintf(f, "-------------------------------------------------------------------------------\r\n")

	log.Println(logutil.Info("Queuing magic file: FILES"))
	s.requestedFiles = append(s.requestedFiles, RequestedFile{Path: f.Name(), SendAs: "FILES.LST", DeleteAfter: true})
	return nil
}

func (s *Session) handleMagicNodelist() error {
	if s.config.NodelistDir == "" {
		log.Println(logutil.Warn("Remote requested NODELIST but nodelist_dir is not configured"))
		return nil
	}

	// Try to find "nodelist" first as it is the standard name
	path, err := nodelist.FindLatest(s.config.NodelistDir, "nodelist")

	// If not found, and we have configured nodelists, try the first one
	if err != nil && len(s.config.Nodelists) > 0 {
		if !strings.EqualFold(s.config.Nodelists[0], "nodelist") {
			path, err = nodelist.FindLatest(s.config.NodelistDir, s.config.Nodelists[0])
		}
	}

	if err == nil && path != "" {
		log.Println(logutil.Info("Queuing magic file: NODELIST -> %s", filepath.Base(path)))
		s.requestedFiles = append(s.requestedFiles, RequestedFile{Path: path})
	} else {
		log.Println(logutil.Warn("Remote requested NODELIST but no suitable file found in %s", s.config.NodelistDir))
	}
	return nil
}

// findLink attempts to find a configured link that matches one of the
// remote system's addresses.
func (s *Session) findLink() *config.Link {
	for _, remoteAddr := range s.remoteAddrs {
		for i := range s.config.Links {
			link := &s.config.Links[i]
			// Compare Zone, Net, Node. Ignore point for link matching.
			if link.ParsedAddress.Zone == remoteAddr.Zone &&
				link.ParsedAddress.Net == remoteAddr.Net &&
				link.ParsedAddress.Node == remoteAddr.Node {
				return link
			}
		}
	}

	// If we initiated the connection, fallback to the target link configuration.
	// This handles cases where M_PWD arrives before M_ADR.
	if s.remoteLink != nil {
		return s.remoteLink
	}
	return nil
}

func (s *Session) handleFile(args string) error {
	if s.currentFile != nil {
		if err := s.finishCurrentFile(); err != nil {
			return err
		}
	}

	// M_FILE format: "filename size timestamp offset"
	parts := strings.Fields(args)
	if len(parts) < 2 {
		return fmt.Errorf("malformed M_FILE: %s", args)
	}

	name := parts[0]
	size, _ := strconv.ParseInt(parts[1], 10, 64)
	timestamp := "0"
	if len(parts) > 2 {
		timestamp = parts[2]
	}
	var offset int64
	if len(parts) > 3 {
		offset, _ = strconv.ParseInt(parts[3], 10, 64)
	}

	// Security: Ensure we only write to the temp directory
	baseName := filepath.Base(name)
	if err := os.MkdirAll(s.config.TempInbound, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	tmpPath := filepath.Join(s.config.TempInbound, baseName)

	var f *os.File
	var err error

	if offset > 0 {
		// Attempt to resume
		info, statErr := os.Stat(tmpPath)
		if statErr == nil && info.Size() == offset {
			// Valid resume
			log.Println(logutil.Info("Resuming download for %s at offset %d", baseName, offset))
			f, err = os.OpenFile(tmpPath, os.O_WRONLY, 0644)
			if err == nil {
				_, err = f.Seek(offset, io.SeekStart)
			}
		} else {
			// Invalid resume conditions
			if statErr != nil {
				log.Println(logutil.Warn("Cannot resume %s: temp file not found. Starting from beginning.", baseName))
			} else {
				log.Println(logutil.Warn("Cannot resume %s: offset mismatch (expected %d, got %d). Starting from beginning.", baseName, offset, info.Size()))
			}
			// Fallback to creating a new file
			offset = 0
			f, err = os.Create(tmpPath)
		}
	} else {
		// Start from beginning
		f, err = os.Create(tmpPath)
	}

	if err != nil {
		s.writeCommand(M_SKIP, name) // Tell remote to skip this file
		return fmt.Errorf("failed to open/create file %s: %w", tmpPath, err)
	}

	s.currentFile = f
	s.recvName = baseName
	s.recvTimestamp = timestamp
	s.recvSize = size
	s.recvBytes = offset // Start counting from the offset

	s.updateMonitor("Receiving", baseName, 0, size)
	log.Println(logutil.Warn("Receiving %s (%d bytes)...", baseName, size))
	return nil
}

func (s *Session) handleData(data []byte) error {
	if s.currentFile == nil {
		return nil // Ignore data if no file is open
	}
	n, err := s.currentFile.Write(data)
	if err != nil {
		return err
	}
	s.recvBytes += int64(n)
	s.updateMonitor("Receiving", s.recvName, s.recvBytes, s.recvSize)

	if s.recvSize > 0 {
		pct := float64(s.recvBytes) / float64(s.recvSize) * 100.0
		fmt.Printf("\rReceiving %s: %.1f%% (%d/%d)...", s.recvName, pct, s.recvBytes, s.recvSize)

		if s.recvBytes >= s.recvSize {
			return s.finishCurrentFile()
		}
	} else {
		fmt.Printf("\rReceiving %s: %d bytes...", s.recvName, s.recvBytes)
	}
	return nil
}

func (s *Session) handleEOB() error {
	if s.currentFile != nil {
		if err := s.finishCurrentFile(); err != nil {
			return err
		}
	}

	log.Println(logutil.Info("-> Remote finished sending (M_EOB)."))

	s.eobMutex.Lock()
	s.theySentEOB = true
	weAreDone := s.weSentEOB
	s.eobMutex.Unlock()

	if weAreDone {
		log.Println(logutil.Debug("Both sides sent EOB, closing connection."))
		// We can close the connection here, but returning errSessionFinished
		// allows for a more graceful shutdown of the reader loop.
		return errSessionFinished
	}

	return nil
}

func (s *Session) finishCurrentFile() error {
	fmt.Println() // Finish progress bar line
	s.currentFile.Close()
	s.currentFile = nil

	var destDir string

	// If not explicitly authenticated, check if we match a password-less link
	if s.activeLink == nil {
		if l := s.findLink(); l != nil && l.Password == "" {
			s.activeLink = l
		}
	}

	if s.activeLink != nil {
		// Protected session
		if s.config.SecureInbound != "" {
			destDir = s.config.SecureInbound
		} else {
			destDir = s.config.InsecureInbound
		}
	} else {
		// Unprotected session
		destDir = s.config.InsecureInbound
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	src := filepath.Join(s.config.TempInbound, s.recvName)
	dst := filepath.Join(destDir, s.recvName)

	if err := os.Rename(src, dst); err != nil {
		log.Println(logutil.Error("Error moving file to inbound: %v", err))
	} else {
		log.Println(logutil.Success("Received %s successfully.", s.recvName))

		s.filesReceived++
		s.bytesReceived += s.recvBytes

		s.executeTriggers(dst)

		// Send M_GOT to confirm receipt
		// M_GOT args: filename size timestamp
		return s.writeCommand(M_GOT, fmt.Sprintf("%s %d %s", s.recvName, s.recvBytes, s.recvTimestamp))
	}
	return nil
}

func (s *Session) executeTriggers(path string) {
	filename := filepath.Base(path)
	for _, t := range s.config.Triggers {
		for _, mask := range t.Masks {
			matched, err := filepath.Match(mask, filename)
			if err != nil {
				log.Println(logutil.Error("Error matching trigger mask '%s': %v\n", mask, err))
				continue
			}
			if matched && t.Command != "" {
				// Use forward slashes for paths in commands for better cross-platform compatibility.
				cmdStr := strings.ReplaceAll(t.Command, "{file}", filepath.ToSlash(path))
				if t.RunAfterSession {
					log.Println(logutil.Debug("Trigger match '%s': queueing for later '%s'\n", mask, cmdStr))
					s.pendingCommands = append(s.pendingCommands, cmdStr)
				} else {
					log.Println(logutil.Debug("Trigger match '%s': executing '%s'\n", mask, cmdStr))
					s.runCommand(cmdStr)
				}
				break
			}
		}
	}
}

func (s *Session) executePendingTriggers() {
	if len(s.pendingCommands) > 0 {
		log.Println(logutil.Debug("Executing pending triggers..."))
		for _, cmdStr := range s.pendingCommands {
			s.runCommand(cmdStr)
		}
	}
}

func (s *Session) runCommand(cmdStr string) {
	cmd := getShellCommand(cmdStr)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Println(logutil.Error("Trigger execution failed for command \"%s\": %v\nOutput: %s", cmdStr, err, string(output)))
	}
}

func (s *Session) sendFiles() error {
	// If not explicitly authenticated, check if we match a password-less link
	if s.activeLink == nil {
		if l := s.findLink(); l != nil && l.Password == "" {
			s.activeLink = l
		}
	}

	if s.config.RefuseInsecureSending && s.activeLink == nil && s.remoteLink == nil {
		log.Println(logutil.Warn("Session is insecure (unauthenticated). Skipping outbound files."))
		if err := s.writeCommand(M_EOB, ""); err != nil {
			return err
		}
		return fmt.Errorf("insecure session, outbound skipped")
	}

	// 1. Send requested files (FREQ)
	for _, req := range s.requestedFiles {
		if err := s.sendFile(req.Path, req.SendAs, req.Offset); err != nil {
			return err
		}
		if req.DeleteAfter {
			os.Remove(req.Path)
		}
	}

	// Iterate over all addresses the remote node claims to have
	for _, addr := range s.remoteAddrs {
		flowFiles := FindFlowFiles(s.config, addr)
		for _, flowFile := range flowFiles {
			if err := s.processFlowFile(flowFile.Path); err != nil {
				return err
			}
		}
	}

	// We are done sending.
	log.Println(logutil.Info("-> Finished sending outbound queue."))
	if err := s.writeCommand(M_EOB, ""); err != nil {
		return err
	}
	return nil
}

// FlowFileInfo holds information about a found flow file.
type FlowFileInfo struct {
	Path  string
	State string // "Crash", "Normal", "Direct", "Hold"
}

// FindFlowFiles locates Binkley-style flow files.
func FindFlowFiles(cfg *config.Config, remote *ftn.FidoAddress) []FlowFileInfo {
	var flows []FlowFileInfo

	// 1. Determine Base Directory
	// If zones match: outbound/outbound
	// If zones differ: outbound/outbound.ZZZ (hex zone)
	var baseDir string
	if remote.Zone == cfg.DefaultZone {
		baseDir = filepath.Join(cfg.Outbound, "outbound")
	} else {
		baseDir = filepath.Join(cfg.Outbound, fmt.Sprintf("outbound.%03x", remote.Zone))
	}

	// 2. Determine Filename Base
	var nameBase string
	if remote.Point != 0 {
		// Points: outbound/NNNNFFFF.pnt/0000PPPP
		// NNNN=Net, FFFF=Node, PPPP=Point (all hex)
		dir := filepath.Join(baseDir, fmt.Sprintf("%04x%04x.pnt", remote.Net, remote.Node))
		baseDir = dir
		nameBase = fmt.Sprintf("0000%04x", remote.Point)
	} else {
		// Nodes: outbound/NNNNFFFF
		nameBase = fmt.Sprintf("%04x%04x", remote.Net, remote.Node)
	}

	// 3. Check extensions in priority order
	extMap := map[string]string{
		".clo": "Crash",
		".cut": "Crash",
		".dlo": "Direct",
		".dut": "Direct",
		".hlo": "Hold",
		".hut": "Hold",
		".lo":  "Normal",
		".out": "Normal",
		".flo": "Normal",
		".fut": "Normal",
	}
	// Check in priority order
	for _, ext := range []string{".clo", ".cut", ".dlo", ".dut", ".hlo", ".hut", ".lo", ".out", ".flo", ".fut"} {
		path := filepath.Join(baseDir, nameBase+ext)
		if _, err := os.Stat(path); err == nil {
			flows = append(flows, FlowFileInfo{Path: path, State: extMap[ext]})
		}
	}
	return flows
}

func (s *Session) processFlowFile(flowPath string) error {
	log.Println(logutil.Debug("Processing flow file: %s", flowPath))
	f, err := os.Open(flowPath)
	if err != nil {
		return nil // Skip if we can't open
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		deleteAfter := false
		path := line
		if strings.HasPrefix(line, "^") {
			deleteAfter = true
			path = line[1:]
		}

		// Send the file
		if err := s.sendFile(path, "", 0); err != nil {
			// If the file is missing, we just skip it (and it will be removed from flow later)
			// If it's a network error, we return it to abort the session.
			if os.IsNotExist(err) {
				log.Println(logutil.Warn("File not found: %s", path))
				continue
			}
			return err
		}

		// If successful and marked for deletion
		if deleteAfter {
			os.Remove(path)
		}
	}

	// If we processed the whole file without network error, delete the flow file
	f.Close()
	os.Remove(flowPath)
	return nil
}

func (s *Session) sendFile(path string, sendAs string, offset int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	name := sendAs
	if name == "" {
		name = filepath.Base(path)
	}

	if offset > 0 {
		log.Println(logutil.Warn("Resuming send for %s (%d bytes) at offset %d...", name, info.Size(), offset))
	} else {
		log.Println(logutil.Warn("Sending %s (%d bytes)...", name, info.Size()))
	}
	s.updateMonitor("Sending", name, offset, info.Size())

	// M_FILE: "name size timestamp offset"
	fileArgs := fmt.Sprintf("%s %d %d %d", name, info.Size(), info.ModTime().Unix(), offset)
	if err := s.writeCommand(M_FILE, fileArgs); err != nil {
		return err
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek in file %s: %w", path, err)
		}
	}

	buf := make([]byte, 4096)
	sentBytes := offset
	for {
		n, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := s.writeData(buf[:n]); err != nil {
			return err
		}
		s.bytesSent += int64(n)
		sentBytes += int64(n)
		s.updateMonitor("Sending", name, sentBytes, info.Size())
	}
	s.filesSent++
	return nil
}

func (s *Session) writeData(data []byte) error {
	// Frame: [Header (2 bytes)] [Data...]
	// Header = Length (15 bits). High bit is 0 for data.
	payloadLen := len(data)
	buf := make([]byte, 2+payloadLen)

	binary.BigEndian.PutUint16(buf[0:2], uint16(payloadLen))
	copy(buf[2:], data)

	_, err := s.conn.Write(buf)
	return err
}

func (s *Session) logSummary(err error) {
	duration := time.Since(s.startTime)
	direction := "IN"
	directionColored := logutil.Debug("IN")
	if s.remoteLink != nil {
		direction = "OUT"
		directionColored = logutil.Debug("OUT")
	}

	status := "OK"
	statusColored := logutil.Success("OK")
	if err != nil && err != errSessionFinished {
		status = "ERR"
		statusColored = logutil.Error("ERR")
	}

	remote := "unknown"
	if len(s.remoteAddrs) > 0 {
		remote = s.remoteAddrs[0].String()
	} else if s.remoteLink != nil {
		remote = s.remoteLink.Address
	}

	remoteColored := logutil.Remote(remote)

	summary := fmt.Sprintf("SESSION %s %s %s R:%d/%d S:%d/%d Time:%s",
		directionColored, remoteColored, statusColored,
		s.filesReceived, s.bytesReceived,
		s.filesSent, s.bytesSent,
		duration.Round(time.Millisecond))

	if s.config.SessionLog != "" {
		// Use RotatableWriter to ensure session log is rotated if it exceeds size
		f, err := logutil.NewRotatableWriter(s.config.SessionLog, s.config.LogMaxSize)
		if err != nil {
			log.Println(logutil.Error("Failed to open session log: %v", err))
			// Fallback to main log on error
			log.Println(summary)
		} else {
			// Write to session log without color
			fmt.Fprintf(f, "%s SESSION %s %s %s R:%d/%d S:%d/%d Time:%s\n", time.Now().Format("2006/01/02 15:04:05"), direction, remote, status, s.filesReceived, s.bytesReceived, s.filesSent, s.bytesSent, duration.Round(time.Millisecond))
			f.Close()
		}
	} else {
		// No separate session log, write summary to main log
		log.Println(summary)
	}
}

func (s *Session) updateMonitor(state, file string, pos, size int64) {
	direction := "IN"
	if s.remoteLink != nil {
		direction = "OUT"
	}

	remote := s.conn.RemoteAddr().String()
	if len(s.remoteAddrs) > 0 {
		remote = s.remoteAddrs[0].String()
	} else if s.remoteLink != nil {
		remote = s.remoteLink.Address
	}

	var endTime time.Time
	if state == "Finished" || state == "Error" {
		endTime = time.Now()
	}

	monitor.RegisterSession(s.id, monitor.SessionInfo{
		ID:          s.id,
		Remote:      remote,
		Direction:   direction,
		State:       state,
		CurrentFile: file,
		FilePos:     pos,
		FileSize:    size,
		StartedAt:   s.startTime,
		EndTime:     endTime,
	})
}
