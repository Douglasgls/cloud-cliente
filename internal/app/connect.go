package app

import (
	"context"
	"fmt"

	"cloud-client/internal/browser"
	"cloud-client/internal/cloud"
	"cloud-client/internal/proxy"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

type ConnectUseCase struct {
	cloudClient cloud.CloudClient
	tsService   tailscale.TailscaleService
	proxySvc    proxy.ProxyService
	browser     browser.Opener
	logger      *logger.Logger
}

func NewConnectUseCase(
	cloudClient cloud.CloudClient,
	tsService tailscale.TailscaleService,
	proxySvc proxy.ProxyService,
	browserOpener browser.Opener,
	log *logger.Logger,
) *ConnectUseCase {
	return &ConnectUseCase{
		cloudClient: cloudClient,
		tsService:   tsService,
		proxySvc:    proxySvc,
		browser:     browserOpener,
		logger:      log,
	}
}

func (uc *ConnectUseCase) Execute(ctx context.Context, token string) error {
	uc.logger.Info("Connecting...")
	uc.logger.Info("Requesting authorization...")

	resp, err := uc.cloudClient.Connect(ctx, token)
	if err != nil {
		return err
	}

	uc.logger.Info("Authorization received")
	uc.logger.Info("Executing tailscale...")

	err = uc.tsService.Up(ctx, resp.LoginServer, resp.PreauthKey, resp.Hostname)
	if err != nil {
		return err
	}

	_, err = uc.cloudClient.Confirm(ctx, resp.ConnectionID.String())
	if err != nil {
		return fmt.Errorf("failed to confirm connection: %w", err)
	}

	uc.logger.Info("Connection confirmed")
	uc.logger.Info("")
	uc.logger.Info("Starting local reverse proxy...")

	localURL, err := uc.proxySvc.Start(resp.Hostname)
	if err != nil {
		return fmt.Errorf("failed to start local proxy: %w", err)
	}

	uc.logger.Info("")
	uc.logger.Info("Listening on:")
	uc.logger.Info("%s", localURL)
	uc.logger.Info("")
	uc.logger.Info("Forward target:")
	uc.logger.Info("%s", uc.proxySvc.TargetURL())
	uc.logger.Info("")

	if uc.browser != nil {
		uc.logger.Info("Opening browser...")
		if err := uc.browser.Open(localURL); err != nil {
			uc.logger.Error("Failed to open browser: %v", err)
		}
	}

	uc.logger.Info("Proxy ready.")

	// Keep proxy active until context is canceled or signal received
	<-ctx.Done()
	_ = uc.proxySvc.Stop(context.Background())

	return nil
}
