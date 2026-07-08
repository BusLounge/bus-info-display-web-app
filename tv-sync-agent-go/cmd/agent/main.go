package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tv-sync-agent-go/internal/config"
	"tv-sync-agent-go/internal/server"
	syncagent "tv-sync-agent-go/internal/sync"
)

func main() {
	configPath := flag.String("config", "./config.json", "Path to config.json")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	a := syncagent.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.Run(ctx)

	srv := server.New(a, cfg.LocalBridgePort)
	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("local bridge: %v", err)
		}
	}()

	log.Printf("tv-sync-agent started for loungeId=%s group=%s", cfg.LoungeID, cfg.LoungeGroup)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Printf("shutdown requested")
	cancel()
}
