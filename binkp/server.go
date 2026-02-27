package binkp

import (
	"context"
	"fmt"
	"log"
	"net"

	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/logutil"
)

// Serve starts the BinkP TCP listener and blocks.
func Serve(ctx context.Context, cfg *config.Config) error {
	addr := fmt.Sprintf(":%d", cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	log.Println(logutil.Info("BinkP server listening on %s", addr))

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Check if the context was cancelled, meaning a graceful shutdown
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Println(logutil.Error("Accept error: %v", err))
			}
			continue
		}

		go handleConnection(conn, cfg)
	}
}

func handleConnection(conn net.Conn, cfg *config.Config) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	log.Println(logutil.Info("Incoming connection from %s", remoteAddr))

	session := NewSession(conn, cfg, nil, "incoming")
	if err := session.Run(); err != nil {
		log.Println(logutil.Error("Session error with %s: %v", remoteAddr, err))
	}
}
