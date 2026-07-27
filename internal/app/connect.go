package app

import (
	"context"
	"errors"
	"fmt"

	"cloud-client/internal/cloud"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/session"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

type ConnectUseCase struct {
	cloudClient cloud.CloudClient
	tsService   tailscale.TailscaleService
	fwdService  forwarding.ForwardingService
	dialer      forwarding.Dialer
	sessionSvc  session.SessionService
	logger      *logger.Logger
}

func NewConnectUseCase(
	cloudClient cloud.CloudClient,
	tsService tailscale.TailscaleService,
	fwdService forwarding.ForwardingService,
	dialer forwarding.Dialer,
	sessionSvc session.SessionService,
	log *logger.Logger,
) *ConnectUseCase {
	return &ConnectUseCase{
		cloudClient: cloudClient,
		tsService:   tsService,
		fwdService:  fwdService,
		dialer:      dialer,
		sessionSvc:  sessionSvc,
		logger:      log,
	}
}

func (uc *ConnectUseCase) Execute(ctx context.Context, token string) error {
	uc.logger.Info("Connecting...")
	uc.logger.Info("Requesting authorization...")

	resp, err := uc.cloudClient.Connect(ctx, token)
	if err != nil {
		if errors.Is(err, cloud.ErrInvalidToken) || errors.Is(err, cloud.ErrConnectionExpired) {
			if uc.sessionSvc != nil {
				_ = uc.sessionSvc.Delete()
			}
		}
		return err
	}

	uc.logger.Info("Authorization received")

	containerName := resp.ContainerName
	if containerName == "" {
		containerName = resp.Name
	}
	if containerName == "" {
		containerName = resp.Hostname
	}

	sessionID := resp.Hostname
	if sessionID == "" {
		sessionID = resp.ConnectionID.String()
	}

	if uc.fwdService != nil {
		_ = uc.fwdService.SwitchSession(sessionID)
	}

	if uc.sessionSvc != nil {
		if _, err := uc.sessionSvc.SaveWithDetails(token, containerName, resp.Hostname, resp.TailscaleIP); err != nil {
			uc.logger.Error("Failed to save session: %v", err)
		}
	}

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
	uc.logger.Info("Starting local forwardings...")

	if err := uc.fwdService.StartAll(resp.TailscaleIP, uc.dialer); err != nil {
		return fmt.Errorf("failed to start forwardings: %w", err)
	}

	uc.logger.Info("Active forwardings:")
	for _, item := range uc.fwdService.List() {
		status := "Inativo"
		if item.Running {
			status = "Ativo"
		} else if item.LastError != "" {
			status = "Erro: " + item.LastError
		}
		uc.logger.Info("  - %s (%d -> %d): %s", item.Forwarding.Name, item.Forwarding.RemotePort, item.Forwarding.LocalPort, status)
	}
	uc.logger.Info("")
	uc.logger.Info("Forwardings ready.")

	// Keep active until context is canceled
	<-ctx.Done()
	_ = uc.fwdService.StopAll()

	return nil
}
