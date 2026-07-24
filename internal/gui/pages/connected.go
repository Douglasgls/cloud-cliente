package pages

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"cloud-client/internal/gui/controller"
	"cloud-client/internal/gui/widgets"
)

type ConnectedPage struct {
	Content        *fyne.Container
	statusCard     *widgets.StatusCard
	urlLabel       *widget.Label
	containerLabel *widget.Label
	serverLabel    *widget.Label
	controller     *controller.ConnectController
	onDisconnected func()
	info           *controller.ConnectionInfo
}

func NewConnectedPage(ctrl *controller.ConnectController, onDisconnected func()) *ConnectedPage {
	cp := &ConnectedPage{
		controller:     ctrl,
		onDisconnected: onDisconnected,
	}

	cp.statusCard = widgets.NewStatusCard()
	cp.statusCard.SetState(widgets.StatusConnected, "Conectado")

	header := widget.NewLabelWithStyle("Cloud Client", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	cp.containerLabel = widget.NewLabelWithStyle("BD2", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	cp.serverLabel = widget.NewLabelWithStyle("Cloud Control", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	cp.urlLabel = widget.NewLabelWithStyle("http://127.0.0.1:8080", fyne.TextAlignLeading, fyne.TextStyle{Bold: true, Italic: true})

	openBtn := widget.NewButton("Abrir Aplicação", func() {
		if cp.info != nil && cp.info.LocalURL != "" {
			_ = cp.controller.OpenBrowser(cp.info.LocalURL)
		}
	})
	openBtn.Importance = widget.HighImportance

	disconnectBtn := widget.NewButton("Desconectar", func() {
		_ = cp.controller.Disconnect(context.Background())
		if cp.onDisconnected != nil {
			cp.onDisconnected()
		}
	})
	disconnectBtn.Importance = widget.DangerImportance

	infoBox := container.NewVBox(
		widget.NewLabel("Container:"),
		cp.containerLabel,
		widget.NewLabel("Servidor:"),
		cp.serverLabel,
		widget.NewLabel("Aplicação Local:"),
		cp.urlLabel,
	)

	cp.Content = container.NewVBox(
		container.NewPadded(header),
		container.NewPadded(cp.statusCard),
		widget.NewSeparator(),
		container.NewPadded(infoBox),
		widget.NewSeparator(),
		container.NewVBox(
			container.NewPadded(openBtn),
			container.NewPadded(disconnectBtn),
		),
	)

	return cp
}

func (cp *ConnectedPage) SetConnectionInfo(info *controller.ConnectionInfo) {
	cp.info = info
	if info != nil {
		cp.urlLabel.SetText(info.LocalURL)
		if info.Hostname != "" {
			cp.containerLabel.SetText("Container " + info.ConnectionID)
		}
	}
}
