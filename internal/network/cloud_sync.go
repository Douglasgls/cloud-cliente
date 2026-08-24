package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud-client/internal/cloud"
	"cloud-client/pkg/logger"
)

type CloudSync struct {
	cloudClient cloud.CloudClient
	registry    *EndpointRegistry
	token       string
	interval    time.Duration
	logger      *logger.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewCloudSync(cloudClient cloud.CloudClient, registry *EndpointRegistry, token string, interval time.Duration, log *logger.Logger) *CloudSync {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &CloudSync{
		cloudClient: cloudClient,
		registry:    registry,
		token:       token,
		interval:    interval,
		logger:      log,
	}
}

// SyncOnce fetches network endpoints from Cloud and updates EndpointRegistry if version differs or on snapshot receipt.
func (cs *CloudSync) SyncOnce(ctx context.Context) error {
	if cs.cloudClient == nil {
		return fmt.Errorf("cloud client is nil")
	}

	resp, err := cs.cloudClient.GetNetworkEndpoints(ctx, cs.token)
	if err != nil {
		if cs.logger != nil {
			cs.logger.Error("[CloudSync] Failed to sync endpoints from cloud: %v", err)
		}
		return fmt.Errorf("cloud sync failed: %w", err)
	}

	currentVersion := cs.registry.Version()
	if resp.Version != currentVersion || currentVersion == -1 {
		if cs.logger != nil {
			cs.logger.Info("[CloudSync] Cloud catalog version changed (%d -> %d). Updating registry...", currentVersion, resp.Version)
		}
		cs.registry.Update(resp.Version, resp.Endpoints)
	} else if cs.logger != nil {
		cs.logger.Debug("[CloudSync] Catalog version %d unchanged. Skipping registry update.", resp.Version)
	}

	return nil
}

// Start launches initial sync and starts periodic background polling until context is canceled.
func (cs *CloudSync) Start(parentCtx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.cancel != nil {
		return nil // Already running
	}

	ctx, cancel := context.WithCancel(parentCtx)
	cs.cancel = cancel

	// Initial sync on startup
	if err := cs.SyncOnce(ctx); err != nil {
		if cs.logger != nil {
			cs.logger.Warn("[CloudSync] Initial sync warning: %v", err)
		}
	}

	go func() {
		ticker := time.NewTicker(cs.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				if cs.logger != nil {
					cs.logger.Debug("[CloudSync] Background sync loop stopped.")
				}
				return
			case <-ticker.C:
				_ = cs.SyncOnce(ctx)
			}
		}
	}()

	return nil
}

// Stop stops the background polling loop.
func (cs *CloudSync) Stop() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.cancel != nil {
		cs.cancel()
		cs.cancel = nil
	}
}
