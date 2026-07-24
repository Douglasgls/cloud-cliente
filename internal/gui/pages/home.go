package pages

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"cloud-client/internal/gui/controller"
	"cloud-client/internal/gui/widgets"
)

type HomePage struct {
	Content      *fyne.Container
	statusCard   *widgets.StatusCard
	tokenForm    *widgets.TokenForm
	loading      *widgets.LoadingWidget
	controller   *controller.ConnectController
	onConnected  func(info *controller.ConnectionInfo)
}

func NewHomePage(ctrl *controller.ConnectController, onConnected func(info *controller.ConnectionInfo)) *HomePage {
	hp := &HomePage{
		controller:  ctrl,
		onConnected: onConnected,
	}

	hp.statusCard = widgets.NewStatusCard()
	hp.loading = widgets.NewLoadingWidget()

	hp.tokenForm = widgets.NewTokenForm(func(token string) {
		hp.startConnect(token)
	})

	header := widget.NewLabelWithStyle("Cloud Client", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	hp.Content = container.NewVBox(
		container.NewPadded(header),
		container.NewPadded(hp.statusCard),
		widget.NewSeparator(),
		container.NewPadded(hp.tokenForm),
		container.NewPadded(hp.loading),
	)

	return hp
}

func (hp *HomePage) startConnect(token string) {
	if token == "" {
		hp.statusCard.SetState(widgets.StatusError, "Informe um token válido")
		return
	}

	hp.tokenForm.SetEnabled(false)
	hp.statusCard.SetState(widgets.StatusConnecting, "Conectando...")
	hp.loading.Reset()

	hp.controller.ConnectAsync(
		token,
		func(step string) {
			fyne.Do(func() {
				hp.loading.ShowProgress(step)
			})
		},
		func(info *controller.ConnectionInfo) {
			fyne.Do(func() {
				hp.loading.ShowSuccess("Conectado")
				hp.statusCard.SetState(widgets.StatusConnected, "Conectado")
				if hp.onConnected != nil {
					hp.onConnected(info)
				}
			})
		},
		func(err error) {
			fyne.Do(func() {
				hp.loading.ShowError(err.Error())
				hp.statusCard.SetState(widgets.StatusError, err.Error())
				hp.tokenForm.SetEnabled(true)
			})
		},
	)
}

func (hp *HomePage) Reset() {
	hp.tokenForm.SetEnabled(true)
	hp.statusCard.SetState(widgets.StatusDisconnected, "Desconectado")
	hp.loading.Reset()
}
