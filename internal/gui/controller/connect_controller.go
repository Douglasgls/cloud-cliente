package controller

import (
	"context"
	"fmt"
	"sync"

	"cloud-client/internal/browser"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/proxy"
	"cloud-client/internal/runtime"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

type ConnectionInfo struct {
	LocalURL     string
	TargetHost   string
	ConnectionID string
	Hostname     string
}

type ConnectController struct {
	cfg         *config.Config
	log         *logger.Logger
	runtimeMgr  runtime.RuntimeManager
	cloudClient cloud.CloudClient
	tsService   tailscale.TailscaleService
	proxySvc    proxy.ProxyService
	browser     browser.Opener

	mu            sync.Mutex
	isConnected   bool
	currentInfo   *ConnectionInfo
	cancelConnect context.CancelFunc
}

func NewConnectController(
	cfg *config.Config,
	log *logger.Logger,
	runtimeMgr runtime.RuntimeManager,
	cloudClient cloud.CloudClient,
	tsService tailscale.TailscaleService,
	proxySvc proxy.ProxyService,
	browserOpener browser.Opener,
) *ConnectController {
	return &ConnectController{
		cfg:         cfg,
		log:         log,
		runtimeMgr:  runtimeMgr,
		cloudClient: cloudClient,
		tsService:   tsService,
		proxySvc:    proxySvc,
		browser:     browserOpener,
	}
}

func (c *ConnectController) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *ConnectController) GetConnectionInfo() *ConnectionInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentInfo
}

func (c *ConnectController) ConnectAsync(
	token string,
	onProgress func(step string),
	onSuccess func(info *ConnectionInfo),
	onError func(err error),
) {
	c.mu.Lock()
	if c.isConnected {
		c.mu.Unlock()
		if onSuccess != nil && c.currentInfo != nil {
			onSuccess(c.currentInfo)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConnect = cancel
	c.mu.Unlock()

	go func() {
		defer cancel()

		report := func(step string) {
			if onProgress != nil {
				onProgress(step)
			}
		}

		report("Preparando runtime...")
		if err := c.runtimeMgr.Prepare(); err != nil {
			if onError != nil {
				onError(fmt.Errorf("falha ao preparar runtime: %w", err))
			}
			return
		}

		report("Iniciando daemon...")
		if err := c.runtimeMgr.EnsureDaemonRunning(ctx); err != nil {
			if onError != nil {
				onError(fmt.Errorf("falha ao iniciar daemon: %w", err))
			}
			return
		}

		report("Solicitando autorização...")
		resp, err := c.cloudClient.Connect(ctx, token)
		if err != nil {
			if onError != nil {
				onError(err)
			}
			return
		}

		report("Executando Tailscale...")
		if err := c.tsService.Up(ctx, resp.LoginServer, resp.PreauthKey, resp.Hostname); err != nil {
			if onError != nil {
				onError(fmt.Errorf("falha no Tailscale: %w", err))
			}
			return
		}

		report("Confirmando conexão...")
		if _, err := c.cloudClient.Confirm(ctx, resp.ConnectionID.String()); err != nil {
			if onError != nil {
				onError(fmt.Errorf("falha na confirmação: %w", err))
			}
			return
		}

		report("Criando proxy...")
		localURL, err := c.proxySvc.Start(resp.Hostname)
		if err != nil {
			if onError != nil {
				onError(fmt.Errorf("falha ao iniciar proxy: %w", err))
			}
			return
		}

		report("Abrindo navegador...")
		if c.browser != nil {
			_ = c.browser.Open(localURL)
		}

		report("✓ Conectado")

		info := &ConnectionInfo{
			LocalURL:     localURL,
			TargetHost:   c.proxySvc.TargetURL(),
			ConnectionID: resp.ConnectionID.String(),
			Hostname:     resp.Hostname,
		}

		c.mu.Lock()
		c.isConnected = true
		c.currentInfo = info
		c.mu.Unlock()

		if onSuccess != nil {
			onSuccess(info)
		}
	}()
}

func (c *ConnectController) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelConnect != nil {
		c.cancelConnect()
		c.cancelConnect = nil
	}

	if c.proxySvc != nil {
		_ = c.proxySvc.Stop(ctx)
	}

	c.isConnected = false
	c.currentInfo = nil
	return nil
}

func (c *ConnectController) OpenBrowser(targetURL string) error {
	if c.browser == nil {
		return fmt.Errorf("browser opener not available")
	}
	return c.browser.Open(targetURL)
}
