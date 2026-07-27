package controller

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"cloud-client/internal/browser"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/runtime"
	"cloud-client/internal/session"
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
	sessionSvc  session.SessionService
	browser     browser.Opener

	mu              sync.Mutex
	isConnected     bool
	isReconnecting  bool
	status          string
	targetIP        string
	lastLoginServer string
	lastAuthKey     string
	lastHostname    string
	currentInfo     *ConnectionInfo
	cancelConnect   context.CancelFunc
	cancelMonitor   context.CancelFunc
}

func NewConnectController(
	cfg *config.Config,
	log *logger.Logger,
	runtimeMgr runtime.RuntimeManager,
	cloudClient cloud.CloudClient,
	tsService tailscale.TailscaleService,
	fwdService forwarding.ForwardingService,
	dialer forwarding.Dialer,
	sessionSvc session.SessionService,
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
		sessionSvc:  sessionSvc,
		browser:     browserOpener,
		status:      "disconnected",
	}
}

func (c *ConnectController) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *ConnectController) HasSession() bool {
	return c.sessionSvc != nil && c.sessionSvc.HasSession()
}

func (c *ConnectController) ListSessions() ([]session.Session, error) {
	if c.sessionSvc == nil {
		return []session.Session{}, nil
	}
	return c.sessionSvc.List()
}

func (c *ConnectController) ForgetSession(id string) error {
	c.mu.Lock()
	if c.currentInfo != nil && (c.currentInfo.Hostname == id || c.currentInfo.ConnectionID == id) {
		c.mu.Unlock()
		_ = c.Disconnect(context.Background())
	} else {
		c.mu.Unlock()
	}

	if c.fwdService != nil {
		_ = c.fwdService.DeleteSessionForwardings(id)
	}
	if c.sessionSvc != nil {
		return c.sessionSvc.DeleteSession(id)
	}
	return nil
}

func (c *ConnectController) DeleteSession() error {
	if c.sessionSvc == nil {
		return nil
	}
	return c.sessionSvc.Delete()
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

func (c *ConnectController) ReconnectAsync(
	onProgress func(step string),
	onSuccess func(info *ConnectionInfo),
	onError func(err error),
) {
	if c.sessionSvc == nil || !c.sessionSvc.HasSession() {
		if onError != nil {
			onError(fmt.Errorf("Nenhuma sessão salva encontrada"))
		}
		return
	}

	sess, err := c.sessionSvc.Load()
	if err != nil || sess.AccessToken == "" {
		if onError != nil {
			onError(fmt.Errorf("Falha ao carregar sessão salva: %v", err))
		}
		return
	}

	c.ConnectAsync(sess.AccessToken, onProgress, onSuccess, onError)
}

func (c *ConnectController) ReconnectSessionAsync(
	sessionID string,
	onProgress func(step string),
	onSuccess func(info *ConnectionInfo),
	onError func(err error),
) {
	if c.sessionSvc == nil {
		if onError != nil {
			onError(fmt.Errorf("Serviço de sessão indisponível"))
		}
		return
	}

	sess, err := c.sessionSvc.Get(sessionID)
	if err != nil || sess.AccessToken == "" {
		if onError != nil {
			onError(fmt.Errorf("Falha ao carregar sessão salva: %v", err))
		}
		return
	}

	c.ConnectAsync(sess.AccessToken, onProgress, onSuccess, onError)
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
			if errors.Is(err, cloud.ErrInvalidToken) || errors.Is(err, cloud.ErrConnectionExpired) {
				if c.sessionSvc != nil {
					_ = c.sessionSvc.Delete()
				}
			}
			if onError != nil {
				onError(err)
			}
			return
		}

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

		if c.fwdService != nil {
			_ = c.fwdService.SwitchSession(sessionID)
		}

		if c.sessionSvc != nil {
			_, _ = c.sessionSvc.SaveWithDetails(token, containerName, resp.Hostname, resp.TailscaleIP)
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
		c.isReconnecting = false
		c.status = "connected"
		c.currentInfo = info
		c.targetIP = targetIP
		c.lastLoginServer = resp.LoginServer
		c.lastAuthKey = resp.PreauthKey
		c.lastHostname = resp.Hostname

		if c.cancelMonitor != nil {
			c.cancelMonitor()
		}
		monCtx, cancelMon := context.WithCancel(context.Background())
		c.cancelMonitor = cancelMon
		c.mu.Unlock()

		go c.startHealthMonitor(monCtx, report, onError)

		if onSuccess != nil {
			onSuccess(info)
		}
	}()
}

func (c *ConnectController) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status == "" {
		if c.isConnected {
			return "connected"
		}
		return "disconnected"
	}
	return c.status
}

func (c *ConnectController) IsReconnecting() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isReconnecting
}

func (c *ConnectController) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelMonitor != nil {
		c.cancelMonitor()
		c.cancelMonitor = nil
	}

	if c.cancelConnect != nil {
		c.cancelConnect()
		c.cancelConnect = nil
	}

	if c.fwdService != nil {
		_ = c.fwdService.StopAll()
	}

	c.isConnected = false
	c.isReconnecting = false
	c.status = "disconnected"
	c.currentInfo = nil
	return nil
}

func (c *ConnectController) startHealthMonitor(ctx context.Context, report func(step string), onError func(err error)) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			if !c.isConnected || c.isReconnecting {
				c.mu.Unlock()
				continue
			}
			targetIP := c.targetIP
			dialer := c.dialer
			loginServer := c.lastLoginServer
			authKey := c.lastAuthKey
			hostname := c.lastHostname
			c.mu.Unlock()

			if targetIP == "" || dialer == nil {
				continue
			}

			// Perform dial test
			dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(targetIP, "22"))
			cancel()

			if err != nil {
				if c.log != nil {
					c.log.Error("[Monitor] Oscilação de rede detectada: %v. Iniciando auto-reconexão...", err)
				}

				c.mu.Lock()
				c.isReconnecting = true
				c.status = "reconnecting"
				c.mu.Unlock()

				report("Reconectando...")

				reconnected := c.autoReconnectLoop(ctx, targetIP, loginServer, authKey, hostname, dialer, report)
				if !reconnected {
					if c.log != nil {
						c.log.Error("[Monitor] Não foi possível restabelecer a conexão de rede.")
					}
					_ = c.Disconnect(ctx)
					if onError != nil {
						onError(fmt.Errorf("Conexão de rede perdida e não pôde ser restabelecida"))
					}
					return
				}
			} else {
				conn.Close()
			}
		}
	}
}

func (c *ConnectController) autoReconnectLoop(
	ctx context.Context,
	targetIP, loginServer, authKey, hostname string,
	dialer forwarding.Dialer,
	report func(step string),
) bool {
	backoff := 2 * time.Second
	maxAttempts := 25

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
		}

		c.mu.Lock()
		if !c.isConnected {
			c.mu.Unlock()
			return false
		}
		c.mu.Unlock()

		report(fmt.Sprintf("Reconectando... (%d/%d)", attempt, maxAttempts))

		if c.tsService != nil && loginServer != "" {
			_ = c.tsService.Up(ctx, loginServer, authKey, hostname)
		}

		dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(targetIP, "22"))
		cancel()

		if err == nil {
			conn.Close()
			if c.log != nil {
				c.log.Info("[Monitor] Conexão restabelecida com sucesso na tentativa %d!", attempt)
			}

			if c.fwdService != nil {
				_ = c.fwdService.StartAll(targetIP, dialer)
			}

			c.mu.Lock()
			c.isReconnecting = false
			c.status = "connected"
			c.mu.Unlock()

			report("✓ Conectado")
			return true
		}

		backoff *= 2
		if backoff > 12*time.Second {
			backoff = 12 * time.Second
		}
	}

	return false
}

func (c *ConnectController) OpenBrowser(targetURL string) error {
	if c.browser == nil {
		return fmt.Errorf("browser opener not available")
	}
	return c.browser.Open(targetURL)
}
