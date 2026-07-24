package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"cloud-client/internal/app"
	"cloud-client/internal/browser"
	"cloud-client/internal/cli"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/gui"
	"cloud-client/internal/gui/controller"
	"cloud-client/internal/proxy"
	"cloud-client/internal/runtime"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

func main() {
	cfg := config.Load()

	if len(os.Args) <= 1 {
		// Launch Fyne Desktop GUI mode when run without arguments
		log := logger.New(false)
		runtimeMgr := runtime.NewManager(log)
		cloudClient := cloud.NewClient(cfg, log)
		tsService := tailscale.NewService(runtimeMgr)
		proxyService := proxy.NewService(log, runtimeMgr.Socks5Addr())
		browserOpener := browser.New()

		ctrl := controller.NewConnectController(
			cfg,
			log,
			runtimeMgr,
			cloudClient,
			tsService,
			proxyService,
			browserOpener,
		)

		gui.RunApp(ctrl)
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
	proxyService := proxy.NewService(log, runtimeMgr.Socks5Addr())
	browserOpener := browser.New()

	switch cmd.Name {
	case "connect":
		connectUC := app.NewConnectUseCase(cloudClient, tsService, proxyService, browserOpener, log)
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
