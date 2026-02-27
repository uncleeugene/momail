package binkp

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/logutil"
)

// Dial initiates a BinkP connection to a remote link.
func Dial(cfg *config.Config, link *config.Link) error {
	host := link.Host
	if !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:24554", host)
	}

	log.Println(logutil.Info("Dialing %s (%s)...", link.Address, host))
	timeout := time.Duration(cfg.DialTimeout) * time.Second
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", host, err)
	}

	session := NewSession(conn, cfg, link, "outgoing")
	return session.Run()
}
