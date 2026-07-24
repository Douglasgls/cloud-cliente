package controller

import (
	"context"
	"fmt"
	"sync"

	"cloud-client/internal/browser"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/runtime"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

type ConnectionInfo struct {
	ConnectionID  string
	Hostname      string
	TailscaleIP   string
	TailscaleIPv6 string
}

type ConnectController struct {
	cfg         *config.Config
	log         *logger.Logger
	runtimeMgr  runtime.RuntimeManager
	cloudClient cloud.CloudClient
	tsService   tailscale.TailscaleService
	fwdService  forwarding.ForwardingService
	dialer      forwarding.Dialer
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
	fwdService forwarding.ForwardingService,
	dialer forwarding.Dialer,
	browserOpener browser.Opener,
) *ConnectController {
	return &ConnectController{
		cfg:         cfg,
		log:         log,
		runtimeMgr:  runtimeMgr,
		cloudClient: cloudClient,
		tsService:   tsService,
		fwdService:  fwdService,
		dialer:      dialer,
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

func (c *ConnectController) ForwardingService() forwarding.ForwardingService {
	return c.fwdService
}

func (c *ConnectController) ListForwardings() []forwarding.ForwardingState {
	if c.fwdService == nil {
		return nil
	}
	return c.fwdService.List()
}

func (c *ConnectController) AddForwarding(name string, remotePort, localPort int) (forwarding.Forwarding, error) {
	if c.fwdService == nil {
		return forwarding.Forwarding{}, fmt.Errorf("forwarding service not available")
	}
	return c.fwdService.Add(name, remotePort, localPort)
}

func (c *ConnectController) UpdateForwarding(f forwarding.Forwarding) error {
	if c.fwdService == nil {
		return fmt.Errorf("forwarding service not available")
	}
	return c.fwdService.Update(f)
}

func (c *ConnectController) DeleteForwarding(id string) error {
	if c.fwdService == nil {
		return fmt.Errorf("forwarding service not available")
	}
	return c.fwdService.Delete(id)
}

func (c *ConnectController) ToggleForwarding(id string, enabled bool) error {
	if c.fwdService == nil {
		return fmt.Errorf("forwarding service not available")
	}
	return c.fwdService.Toggle(id, enabled)
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

		report("Iniciando serviços...")
		targetIP := resp.TailscaleIP
		if targetIP == "" {
			targetIP = resp.Hostname
		}
		if c.fwdService != nil && c.dialer != nil {
			if err := c.fwdService.StartAll(targetIP, c.dialer); err != nil {
				if onError != nil {
					onError(fmt.Errorf("falha ao iniciar serviços: %w", err))
				}
				return
			}
		}

		report("✓ Conectado")

		info := &ConnectionInfo{
			ConnectionID:  resp.ConnectionID.String(),
			Hostname:      resp.Hostname,
			TailscaleIP:   resp.TailscaleIP,
			TailscaleIPv6: resp.TailscaleIPv6,
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

	if c.fwdService != nil {
		_ = c.fwdService.StopAll()
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
