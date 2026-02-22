package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	"uncleeugene.kz/momail/ftn"
)

// Link defines a configuration for a remote system.
type Link struct {
	Address      string `yaml:"address"`
	Host         string `yaml:"host"`
	Password     string `yaml:"password"`
	WorkingTime  string `yaml:"working_time"`
	LookupMethod string `yaml:"lookup_method"`

	// ParsedAddress is the parsed version of the Address field.
	// It is not loaded from yaml directly.
	ParsedAddress *ftn.FidoAddress `yaml:"-"`
}

// Trigger defines an action to execute when a file matching the mask is received.
type Trigger struct {
	Masks           []string `yaml:"masks"`
	Command         string   `yaml:"command"`
	RunAfterSession bool     `yaml:"run_after_session"`
}

// ScheduledTask defines a command to be run on a schedule.
type ScheduledTask struct {
	// CronSpec is a cron-style specification for when to run the command.
	CronSpec string `yaml:"cron_spec"`
	// Command is the shell command to execute.
	Command string `yaml:"command"`
}

// Config holds the application's configuration.
type Config struct {
	// Your primary FidoNet address (AKA)
	Address string `yaml:"address"`

	// System Name (e.g. "My BBS")
	SystemName string `yaml:"system_name"`

	// Sysop Name (e.g. "John Doe")
	Sysop string `yaml:"sysop"`

	// Location (e.g. "City, Country")
	Location string `yaml:"location"`

	// NodelistFlags defines capabilities sent in NDL handshake (default: "TCP,BINKP")
	NodelistFlags string `yaml:"nodelist_flags"`

	// RefuseInsecureSending, if true, prevents sending files to unauthenticated nodes.
	RefuseInsecureSending bool `yaml:"refuse_insecure_sending"`

	// DisableDNS prevents looking up addresses via DNS.
	DisableDNS bool `yaml:"disable_dns"`

	// DNSRoot is the root domain for BinkP DNS lookups (default: "binkp.net").
	DNSRoot string `yaml:"dns_root"`

	// LookupMethod defines the global preference for address lookup ("nodelist" or "dns").
	LookupMethod string `yaml:"lookup_method"`

	// LogFile is the path to the log file.
	LogFile string `yaml:"log_file"`

	// LogMaxSize is the maximum size of log files in kilobytes (KB). 0 means no limit.
	LogMaxSize int `yaml:"log_max_size"`

	// NodelistFile is the path to the FidoNet nodelist file (e.g. NODELIST.007)
	NodelistFile string `yaml:"nodelist_file"`

	// NodelistDir is the directory containing nodelist files.
	// If set, the mailer will load the file with the highest numeric extension.
	NodelistDir string `yaml:"nodelist_dir"`

	// Nodelists is a list of filename bases to look for in NodelistDir.
	// Example: ["nodelist", "r50pnt"] will look for the latest nodelist.xxx and r50pnt.xxx
	Nodelists []string `yaml:"nodelists"`

	// SessionLog is the path to the separate session log file.
	SessionLog string `yaml:"session_log"`

	// Port for the BinkP server (default: 24554)
	Port int `yaml:"port"`

	// APIPort is the port for the HTTP API server (optional).
	APIPort int `yaml:"api_port"`

	// APIToken is the secret token required to access the API.
	// If empty, the API is unprotected.
	APIToken string `yaml:"api_token"`

	// RescanPeriod is the interval in seconds to scan for outbound mail (default: 60)
	RescanPeriod int `yaml:"rescan_period"`

	// DialTimeout is the timeout in seconds for establishing an outgoing connection (default: 20)
	DialTimeout int `yaml:"dial_timeout"`

	// MaxDialAttempts is the number of failed attempts before suspending a node (default: 3)
	MaxDialAttempts int `yaml:"max_dial_attempts"`

	// HoldTime is the suspension duration in minutes (default: 60)
	HoldTime int `yaml:"hold_time"`

	// StaleBusyTimeout is the age in hours after which a busy flag is considered stale and removed (default: 24).
	StaleBusyTimeout int `yaml:"stale_busy_timeout"`

	// Directory for insecure incoming files (unprotected sessions)
	InsecureInbound string `yaml:"insecure_inbound"`

	// Directory for outgoing mail bundles
	Outbound string `yaml:"outbound"`

	// Directory for temporary files during sessions
	TempInbound string `yaml:"temp_inbound"`

	// Directory for secure inbound files
	SecureInbound string `yaml:"secure_inbound"`

	// FreqDir is the directory for public files available for request (FREQ).
	FreqDir string `yaml:"freq_dir"`

	// DefaultZone for outbound structure
	DefaultZone uint16 `yaml:"default_zone"`

	// Links holds configurations for remote systems.
	Links []Link `yaml:"links"`

	// Triggers holds file processing rules.
	Triggers []Trigger `yaml:"triggers"`

	// Tasks holds scheduled commands to run.
	Tasks []ScheduledTask `yaml:"tasks"`

	// ParsedAddress is the parsed version of the Address field.
	// It is not loaded from yaml directly.
	ParsedAddress *ftn.FidoAddress `yaml:"-"`
}

// Load reads a YAML file from the given path and returns a Config struct.
// It also parses the FidoNet address.
func Load(path string) (*Config, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(f))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file %s: %w", path, err)
	}

	if cfg.LogFile == "" {
		cfg.LogFile = "momail.log"
	}

	if cfg.NodelistFlags == "" {
		cfg.NodelistFlags = "TCP,BINKP"
	}

	if cfg.DNSRoot == "" {
		cfg.DNSRoot = "binkp.net"
	}

	if cfg.Port == 0 {
		cfg.Port = 24554
	}

	if cfg.RescanPeriod == 0 {
		cfg.RescanPeriod = 60
	}

	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 20
	}

	if cfg.MaxDialAttempts == 0 {
		cfg.MaxDialAttempts = 3
	}

	if cfg.HoldTime == 0 {
		cfg.HoldTime = 60
	}

	if cfg.StaleBusyTimeout == 0 {
		cfg.StaleBusyTimeout = 24
	}

	if cfg.LogMaxSize < 0 {
		cfg.LogMaxSize = 0
	}

	// The primary address for the mailer must contain a zone.
	if !strings.Contains(cfg.Address, ":") {
		return nil, fmt.Errorf("config 'address' must include a zone")
	}
	addr, err := ftn.ParseFidoAddress(cfg.Address, 0)
	if err != nil {
		return nil, fmt.Errorf("could not parse 'address' in config: %w", err)
	}
	cfg.ParsedAddress = addr

	if cfg.DefaultZone == 0 {
		cfg.DefaultZone = cfg.ParsedAddress.Zone
	}

	for i := range cfg.Links {
		// The default zone for a link address should be our own zone.
		addr, err := ftn.ParseFidoAddress(cfg.Links[i].Address, cfg.ParsedAddress.Zone)
		if err != nil {
			return nil, fmt.Errorf("could not parse address for link %s: %w", cfg.Links[i].Address, err)
		}
		cfg.Links[i].ParsedAddress = addr
	}

	return &cfg, nil
}
