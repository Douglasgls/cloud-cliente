package pages

import (
	"context"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"cloud-client/internal/forwarding"
	"cloud-client/internal/gui/controller"
	"cloud-client/internal/gui/widgets"
)

type ConnectedPage struct {
	Content        *fyne.Container
	window         fyne.Window
	statusCard     *widgets.StatusCard
	containerLabel *widget.Label
	hostLabel      *widget.Label
	controller     *controller.ConnectController
	onDisconnected func()
	info           *controller.ConnectionInfo

	defaultsContainer *fyne.Container
	customContainer   *fyne.Container
}

type guiListener struct {
	refreshFunc func()
}

func (g *guiListener) OnForwardingStarted(id string)        { fyne.Do(g.refreshFunc) }
func (g *guiListener) OnForwardingStopped(id string)        { fyne.Do(g.refreshFunc) }
func (g *guiListener) OnForwardingError(id string, err error) { fyne.Do(g.refreshFunc) }

func NewConnectedPage(w fyne.Window, ctrl *controller.ConnectController, onDisconnected func()) *ConnectedPage {
	cp := &ConnectedPage{
		window:         w,
		controller:     ctrl,
		onDisconnected: onDisconnected,
	}

	cp.statusCard = widgets.NewStatusCard()
	cp.statusCard.SetState(widgets.StatusConnected, "Conectado")

	header := widget.NewLabelWithStyle("Cloud Client", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	cp.containerLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	cp.hostLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	infoBox := container.NewVBox(
		container.NewHBox(widget.NewLabel("Container:"), cp.containerLabel),
		container.NewHBox(widget.NewLabel("Host Remote:"), cp.hostLabel),
	)

	cp.defaultsContainer = container.NewVBox()
	cp.customContainer = container.NewVBox()

	newServiceBtn := widget.NewButtonWithIcon("Novo Serviço", theme.ContentAddIcon(), func() {
		cp.showAddServiceDialog()
	})
	newServiceBtn.Importance = widget.HighImportance

	disconnectBtn := widget.NewButtonWithIcon("Desconectar", theme.LogoutIcon(), func() {
		_ = cp.controller.Disconnect(context.Background())
		if cp.onDisconnected != nil {
			cp.onDisconnected()
		}
	})
	disconnectBtn.Importance = widget.DangerImportance

	scrollableList := container.NewVScroll(
		container.NewVBox(
			container.NewPadded(header),
			container.NewPadded(cp.statusCard),
			container.NewPadded(infoBox),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Serviços Padrão", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			cp.defaultsContainer,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Serviços Personalizados", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			cp.customContainer,
			container.NewPadded(newServiceBtn),
			widget.NewSeparator(),
			container.NewPadded(disconnectBtn),
		),
	)

	cp.Content = container.NewBorder(nil, nil, nil, nil, scrollableList)

	// Subscribe to forwarding updates
	if ctrl.ForwardingService() != nil {
		ctrl.ForwardingService().Subscribe(&guiListener{
			refreshFunc: func() {
				cp.RefreshForwardings()
			},
		})
	}

	return cp
}

func (cp *ConnectedPage) SetConnectionInfo(info *controller.ConnectionInfo) {
	cp.info = info
	if info != nil {
		if info.Hostname != "" {
			cp.containerLabel.SetText(info.Hostname)
			cp.hostLabel.SetText(info.Hostname)
		} else {
			cp.containerLabel.SetText("Container " + info.ConnectionID)
			cp.hostLabel.SetText(info.ConnectionID)
		}
	}
	cp.RefreshForwardings()
}

func (cp *ConnectedPage) RefreshForwardings() {
	if cp.controller == nil {
		return
	}

	states := cp.controller.ListForwardings()

	cp.defaultsContainer.Objects = nil
	cp.customContainer.Objects = nil

	for _, state := range states {
		st := state
		itemCard := cp.createForwardingCard(st)
		if st.Forwarding.IsDefault {
			cp.defaultsContainer.Add(itemCard)
		} else {
			cp.customContainer.Add(itemCard)
		}
	}

	if len(cp.customContainer.Objects) == 0 {
		cp.customContainer.Add(widget.NewLabelWithStyle("Nenhum serviço personalizado cadastrado", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))
	}

	cp.defaultsContainer.Refresh()
	cp.customContainer.Refresh()
}

func (cp *ConnectedPage) createForwardingCard(state forwarding.ForwardingState) fyne.CanvasObject {
	fwd := state.Forwarding

	// Checkbox for enable/disable toggle
	check := widget.NewCheck("", func(checked bool) {
		_ = cp.controller.ToggleForwarding(fwd.ID, checked)
		cp.RefreshForwardings()
	})
	check.Checked = fwd.Enabled

	// Status badge
	var statusText string
	if !fwd.Enabled {
		statusText = "⚪ Desabilitado"
	} else if state.Running {
		statusText = "🟢 Executando"
	} else {
		statusText = "⚠ Erro ao iniciar"
		if state.LastError != "" {
			statusText += fmt.Sprintf(" (%s)", state.LastError)
		}
	}
	statusLabel := widget.NewLabelWithStyle(statusText, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	titleLabel := widget.NewLabelWithStyle(fwd.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	portsLabel := widget.NewLabel(fmt.Sprintf("Remota: %d  ➔  Local: %d", fwd.RemotePort, fwd.LocalPort))

	topLine := container.NewHBox(check, titleLabel, statusLabel)
	infoBox := container.NewVBox(topLine, portsLabel)

	buttonsBox := container.NewHBox()

	// Action buttons depending on service type
	switch fwd.ID {
	case "ssh":
		copyBtn := widget.NewButtonWithIcon("Copiar comando", theme.ContentCopyIcon(), func() {
			cmdStr := fmt.Sprintf("ssh root@127.0.0.1 -p %d", fwd.LocalPort)
			cp.window.Clipboard().SetContent(cmdStr)
		})
		editBtn := widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
			cp.showEditServiceDialog(fwd)
		})
		buttonsBox.Add(copyBtn)
		buttonsBox.Add(editBtn)

	case "http":
		openBtn := widget.NewButtonWithIcon("Abrir", theme.NavigateNextIcon(), func() {
			urlStr := fmt.Sprintf("http://127.0.0.1:%d", fwd.LocalPort)
			_ = cp.controller.OpenBrowser(urlStr)
		})
		openBtn.Importance = widget.HighImportance

		copyBtn := widget.NewButtonWithIcon("Copiar URL", theme.ContentCopyIcon(), func() {
			urlStr := fmt.Sprintf("http://127.0.0.1:%d", fwd.LocalPort)
			cp.window.Clipboard().SetContent(urlStr)
		})

		editBtn := widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
			cp.showEditServiceDialog(fwd)
		})

		buttonsBox.Add(openBtn)
		buttonsBox.Add(copyBtn)
		buttonsBox.Add(editBtn)

	case "https":
		openBtn := widget.NewButtonWithIcon("Abrir", theme.NavigateNextIcon(), func() {
			urlStr := fmt.Sprintf("https://127.0.0.1:%d", fwd.LocalPort)
			_ = cp.controller.OpenBrowser(urlStr)
		})
		openBtn.Importance = widget.HighImportance

		copyBtn := widget.NewButtonWithIcon("Copiar URL", theme.ContentCopyIcon(), func() {
			urlStr := fmt.Sprintf("https://127.0.0.1:%d", fwd.LocalPort)
			cp.window.Clipboard().SetContent(urlStr)
		})

		editBtn := widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
			cp.showEditServiceDialog(fwd)
		})

		buttonsBox.Add(openBtn)
		buttonsBox.Add(copyBtn)
		buttonsBox.Add(editBtn)

	default:
		editBtn := widget.NewButtonWithIcon("Editar", theme.DocumentCreateIcon(), func() {
			cp.showEditServiceDialog(fwd)
		})
		deleteBtn := widget.NewButtonWithIcon("Remover", theme.DeleteIcon(), func() {
			dialog.ShowConfirm("Remover Serviço", fmt.Sprintf("Deseja remover o serviço '%s'?", fwd.Name), func(ok bool) {
				if ok {
					_ = cp.controller.DeleteForwarding(fwd.ID)
					cp.RefreshForwardings()
				}
			}, cp.window)
		})
		deleteBtn.Importance = widget.DangerImportance

		buttonsBox.Add(editBtn)
		buttonsBox.Add(deleteBtn)
	}

	cardContent := container.NewVBox(infoBox, buttonsBox)
	return widget.NewCard("", "", container.NewPadded(cardContent))
}

func (cp *ConnectedPage) showAddServiceDialog() {
	nameEntry := widget.NewEntry()
	remotePortEntry := widget.NewEntry()
	localPortEntry := widget.NewEntry()

	items := []*widget.FormItem{
		{Text: "Nome", Widget: nameEntry},
		{Text: "Porta Remota", Widget: remotePortEntry},
		{Text: "Porta Local", Widget: localPortEntry},
	}

	dialog.ShowForm("Novo Serviço Personalizado", "Salvar", "Cancelar", items, func(ok bool) {
		if !ok {
			return
		}

		name := nameEntry.Text
		remotePort, errR := strconv.Atoi(remotePortEntry.Text)
		localPort, errL := strconv.Atoi(localPortEntry.Text)

		if name == "" || errR != nil || errL != nil {
			dialog.ShowError(fmt.Errorf("por favor preencha valores válidos para nome e portas"), cp.window)
			return
		}

		_, err := cp.controller.AddForwarding(name, remotePort, localPort)
		if err != nil {
			dialog.ShowError(err, cp.window)
			return
		}

		cp.RefreshForwardings()
	}, cp.window)
}

func (cp *ConnectedPage) showEditServiceDialog(fwd forwarding.Forwarding) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(fwd.Name)
	if fwd.IsDefault {
		nameEntry.Disable()
	}

	remotePortEntry := widget.NewEntry()
	remotePortEntry.SetText(strconv.Itoa(fwd.RemotePort))
	if fwd.IsDefault {
		remotePortEntry.Disable()
	}

	localPortEntry := widget.NewEntry()
	localPortEntry.SetText(strconv.Itoa(fwd.LocalPort))

	items := []*widget.FormItem{
		{Text: "Nome", Widget: nameEntry},
		{Text: "Porta Remota", Widget: remotePortEntry},
		{Text: "Porta Local", Widget: localPortEntry},
	}

	dialog.ShowForm(fmt.Sprintf("Editar %s", fwd.Name), "Salvar", "Cancelar", items, func(ok bool) {
		if !ok {
			return
		}

		name := nameEntry.Text
		remotePort, errR := strconv.Atoi(remotePortEntry.Text)
		localPort, errL := strconv.Atoi(localPortEntry.Text)

		if name == "" || errR != nil || errL != nil {
			dialog.ShowError(fmt.Errorf("por favor preencha valores válidos para nome e portas"), cp.window)
			return
		}

		fwd.Name = name
		fwd.RemotePort = remotePort
		fwd.LocalPort = localPort

		err := cp.controller.UpdateForwarding(fwd)
		if err != nil {
			dialog.ShowError(err, cp.window)
			return
		}

		cp.RefreshForwardings()
	}, cp.window)
}
