package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"cloud-client/internal/browser"
	"cloud-client/internal/cloud"
	"cloud-client/internal/config"
	"cloud-client/internal/forwarding"
	"cloud-client/internal/gui/controller"
	"cloud-client/internal/runtime"
	"cloud-client/internal/session"
	"cloud-client/internal/tailscale"
	"cloud-client/pkg/logger"
)

type App struct {
	ctx         context.Context
	cfg         *config.Config
	log         *logger.Logger
	runtimeMgr  runtime.RuntimeManager
	cloudClient cloud.CloudClient
	tsService   tailscale.TailscaleService
	fwdService  forwarding.ForwardingService
	dialer      forwarding.Dialer
	sessionSvc  session.SessionService
	browser     browser.Opener
	ctrl        *controller.ConnectController
	mu          sync.Mutex
}

func NewApp(
	cfg *config.Config,
	log *logger.Logger,
	runtimeMgr runtime.RuntimeManager,
	cloudClient cloud.CloudClient,
	tsService tailscale.TailscaleService,
	fwdService forwarding.ForwardingService,
	dialer forwarding.Dialer,
	sessionSvc session.SessionService,
	browserOpener browser.Opener,
	ctrl *controller.ConnectController,
) *App {
	return &App{
		cfg:         cfg,
		log:         log,
		runtimeMgr:  runtimeMgr,
		cloudClient: cloudClient,
		tsService:   tsService,
		fwdService:  fwdService,
		dialer:      dialer,
		sessionSvc:  sessionSvc,
		browser:     browserOpener,
		ctrl:        ctrl,
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if a.fwdService != nil {
		a.fwdService.Subscribe(&bridgeListener{app: a})
	}
}

type bridgeListener struct {
	app *App
}

func (b *bridgeListener) OnForwardingStarted(id string)        { b.app.notifyForwardingsChanged() }
func (b *bridgeListener) OnForwardingStopped(id string)        { b.app.notifyForwardingsChanged() }
func (b *bridgeListener) OnForwardingError(id string, err error) { b.app.notifyForwardingsChanged() }

func (a *App) notifyForwardingsChanged() {
	if a.ctx != nil {
		wailsRuntime.EventsEmit(a.ctx, "forwardings_changed", a.ListForwardings())
	}
}

func (a *App) HasSession() bool {
	if a.sessionSvc == nil {
		return false
	}
	return a.sessionSvc.HasSession()
}

func (a *App) DeleteSession() error {
	if a.sessionSvc == nil {
		return nil
	}
	return a.sessionSvc.Delete()
}

func (a *App) IsConnected() bool {
	if a.ctrl == nil {
		return false
	}
	return a.ctrl.IsConnected()
}

func (a *App) GetConnectionInfo() ConnectionInfoDTO {
	if a.ctrl == nil {
		return ConnectionInfoDTO{}
	}
	info := a.ctrl.GetConnectionInfo()
	if info == nil {
		return ConnectionInfoDTO{}
	}
	return ConnectionInfoDTO{
		ConnectionID:  info.ConnectionID,
		Hostname:      info.Hostname,
		TailscaleIP:   info.TailscaleIP,
		TailscaleIPv6: info.TailscaleIPv6,
	}
}

func (a *App) Connect(token string) error {
	if token == "" {
		return fmt.Errorf("informe um token válido")
	}

	errChan := make(chan error, 1)

	a.ctrl.ConnectAsync(
		token,
		func(step string) {
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "connection_progress", step)
			}
		},
		func(info *controller.ConnectionInfo) {
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "connection_state_changed", true)
			}
			errChan <- nil
		},
		func(err error) {
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "connection_state_changed", false)
			}
			errChan <- err
		},
	)

	return <-errChan
}

func (a *App) Reconnect() error {
	if a.sessionSvc == nil || !a.sessionSvc.HasSession() {
		return fmt.Errorf("nenhuma sessão salva encontrada")
	}

	sess, err := a.sessionSvc.Load()
	if err != nil || sess.AccessToken == "" {
		return fmt.Errorf("falha ao carregar sessão salva: %w", err)
	}

	return a.Connect(sess.AccessToken)
}

func (a *App) Disconnect() error {
	if a.ctrl != nil {
		err := a.ctrl.Disconnect(context.Background())
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "connection_state_changed", false)
		}
		return err
	}
	return nil
}

func (a *App) ListForwardings() []ForwardingDTO {
	if a.fwdService == nil {
		return []ForwardingDTO{}
	}

	states := a.fwdService.List()
	result := make([]ForwardingDTO, len(states))
	for i, st := range states {
		result[i] = ForwardingDTO{
			ID:         st.Forwarding.ID,
			Name:       st.Forwarding.Name,
			RemotePort: st.Forwarding.RemotePort,
			LocalPort:  st.Forwarding.LocalPort,
			Enabled:    st.Forwarding.Enabled,
			IsDefault:  st.Forwarding.IsDefault,
			Running:    st.Running,
			LastError:  st.LastError,
		}
	}
	return result
}

func (a *App) AddForwarding(name string, remotePort, localPort int) (ForwardingDTO, error) {
	if a.fwdService == nil {
		return ForwardingDTO{}, fmt.Errorf("forwarding service not available")
	}

	fwd, err := a.fwdService.Add(name, remotePort, localPort)
	if err != nil {
		return ForwardingDTO{}, err
	}

	a.notifyForwardingsChanged()

	st, _ := a.fwdService.Get(fwd.ID)
	return ForwardingDTO{
		ID:         fwd.ID,
		Name:       fwd.Name,
		RemotePort: fwd.RemotePort,
		LocalPort:  fwd.LocalPort,
		Enabled:    fwd.Enabled,
		IsDefault:  fwd.IsDefault,
		Running:    st.Running,
		LastError:  st.LastError,
	}, nil
}

func (a *App) UpdateForwarding(id string, name string, remotePort, localPort int, enabled bool) error {
	if a.fwdService == nil {
		return fmt.Errorf("forwarding service not available")
	}

	st, err := a.fwdService.Get(id)
	if err != nil {
		return err
	}

	fwd := st.Forwarding
	fwd.Name = name
	fwd.RemotePort = remotePort
	fwd.LocalPort = localPort
	fwd.Enabled = enabled

	if err := a.fwdService.Update(fwd); err != nil {
		return err
	}

	a.notifyForwardingsChanged()
	return nil
}

func (a *App) DeleteForwarding(id string) error {
	if a.fwdService == nil {
		return fmt.Errorf("forwarding service not available")
	}

	if err := a.fwdService.Delete(id); err != nil {
		return err
	}

	a.notifyForwardingsChanged()
	return nil
}

func (a *App) ToggleForwarding(id string, enabled bool) error {
	if a.fwdService == nil {
		return fmt.Errorf("forwarding service not available")
	}

	if err := a.fwdService.Toggle(id, enabled); err != nil {
		return err
	}

	a.notifyForwardingsChanged()
	return nil
}

func (a *App) CopyToClipboard(text string) error {
	if a.ctx != nil {
		return wailsRuntime.ClipboardSetText(a.ctx, text)
	}
	return errors.New("wails context not ready")
}

func (a *App) OpenURL(url string) error {
	if a.browser != nil {
		return a.browser.Open(url)
	}
	if a.ctx != nil {
		wailsRuntime.BrowserOpenURL(a.ctx, url)
		return nil
	}
	return errors.New("browser opener not available")
}
