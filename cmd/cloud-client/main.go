package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"cloud-client/internal/app"
	"cloud-client/internal/cli"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/runtime"
	"cloud-client/internal/session"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

func main() {
	cfg := config.Load()

	if len(os.Args) <= 1 {
		// CLI fallback when no command specified
		log := logger.New(false)
		log.Info("Cloud Client CLI - para iniciar a GUI desktop, execute a aplicação Wails.")
		return
	}

	// CLI execution mode
	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		log := logger.New(false)
		log.Error("Error: %v", err)
		os.Exit(1)
	}

	isDebug := cmd.Debug || os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "1"
	log := logger.New(isDebug)

	runtimeMgr := runtime.NewManager(log)
	if err := runtimeMgr.Prepare(); err != nil {
		log.Error("Error preparing runtime: %v", err)
		os.Exit(1)
	}

	ctx, stopSignal := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignal()

	if err := runtimeMgr.EnsureDaemonRunning(ctx); err != nil {
		log.Error("Error starting background daemon: %v", err)
		os.Exit(1)
	}

	cloudClient := cloud.NewClient(cfg, log)
	tsService := tailscale.NewService(runtimeMgr)

	storage, err := forwarding.NewJSONStorage("")
	if err != nil {
		log.Error("Error initializing storage: %v", err)
		os.Exit(1)
	}
	fwdService, err := forwarding.NewService(storage, log)
	if err != nil {
		log.Error("Error initializing forwarding service: %v", err)
		os.Exit(1)
	}
	dialer := forwarding.NewSocks5Dialer(runtimeMgr.Socks5Addr(), log)

	sStorage, err := session.NewJSONStorage("")
	if err != nil {
		log.Error("Error initializing session storage: %v", err)
		os.Exit(1)
	}
	sessionSvc := session.NewService(sStorage)

	switch cmd.Name {
	case "connect":
		connectUC := app.NewConnectUseCase(cloudClient, tsService, fwdService, dialer, sessionSvc, log)
		if err := connectUC.Execute(ctx, cmd.Token); err != nil && err != context.Canceled {
			log.Error("Error: %v", err)
			os.Exit(1)
		}
	case "debug":
		_ = runtimeMgr.PrintDebugInfo(ctx)
		log.Info("")
		log.Info("Status:")
		status, err := tsService.Status(ctx)
		if err != nil {
			log.Error("  Error fetching Tailscale status: %v", err)
		} else {
			log.Info("  %s", status)
		}
	}
}
