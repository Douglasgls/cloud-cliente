package main

import (
	"context"
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"cloud-client/internal/app"
	"cloud-client/internal/bridge"
	"cloud-client/internal/browser"
	"cloud-client/internal/cli"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/gui/controller"
	"cloud-client/internal/runtime"
	"cloud-client/internal/session"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.Load()

	if len(os.Args) <= 1 {
		// Launch Wails Desktop GUI mode when run without arguments
		log := logger.New(false)
		runtimeMgr := runtime.NewManager(log)
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
		browserOpener := browser.New()

		ctrl := controller.NewConnectController(
			cfg,
			log,
			runtimeMgr,
			cloudClient,
			tsService,
			fwdService,
			dialer,
			sessionSvc,
			browserOpener,
		)

		bridgeApp := bridge.NewApp(
			cfg,
			log,
			runtimeMgr,
			cloudClient,
			tsService,
			fwdService,
			dialer,
			sessionSvc,
			browserOpener,
			ctrl,
		)

		err = wails.Run(&options.App{
			Title:  "Cloud Client",
			Width:  800,
			Height: 600,
			AssetServer: &assetserver.Options{
				Assets: assets,
			},
			BackgroundColour: &options.RGBA{R: 9, G: 9, B: 11, A: 255},
			OnStartup:        bridgeApp.Startup,
			Bind: []interface{}{
				bridgeApp,
			},
		})
		if err != nil {
			log.Error("Wails error: %v", err)
			os.Exit(1)
		}
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
		if err := connectUC.Execute(context.Background(), cmd.Token); err != nil {
			log.Error("Error: %v", err)
			os.Exit(1)
		}
	case "debug":
		_ = runtimeMgr.PrintDebugInfo(context.Background())
		log.Info("")
		log.Info("Status:")
		status, err := tsService.Status(context.Background())
		if err != nil {
			log.Error("  Error fetching Tailscale status: %v", err)
		} else {
			log.Info("  %s", status)
		}
	}
}
