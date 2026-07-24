package pages

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"cloud-client/internal/gui/controller"
	"cloud-client/internal/gui/widgets"
)

type HomePage struct {
	Content        *fyne.Container
	statusCard     *widgets.StatusCard
	tokenForm      *widgets.TokenForm
	loading        *widgets.LoadingWidget
	sessionBox     *fyne.Container
	formContainer  *fyne.Container
	controller     *controller.ConnectController
	onConnected    func(info *controller.ConnectionInfo)
	reconnectBtn   *widget.Button
	newConnBtn     *widget.Button
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

	// Session UI components
	sessionLabel := widget.NewLabelWithStyle("Última conexão encontrada.", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	hp.reconnectBtn = widget.NewButtonWithIcon("Reconectar", theme.ConfirmIcon(), func() {
		hp.startReconnect()
	})
	hp.reconnectBtn.Importance = widget.HighImportance

	hp.newConnBtn = widget.NewButtonWithIcon("Nova conexão", theme.ContentAddIcon(), func() {
		_ = hp.controller.DeleteSession()
		hp.showFormView()
	})

	hp.sessionBox = container.NewVBox(
		container.NewPadded(sessionLabel),
		container.NewPadded(hp.reconnectBtn),
		container.NewPadded(hp.newConnBtn),
	)

	hp.formContainer = container.NewPadded(hp.tokenForm)

	hp.Content = container.NewVBox(
		container.NewPadded(header),
		container.NewPadded(hp.statusCard),
		widget.NewSeparator(),
		hp.sessionBox,
		hp.formContainer,
		container.NewPadded(hp.loading),
	)

	hp.updateVisibility()
	return hp
}

func (hp *HomePage) updateVisibility() {
	if hp.controller != nil && hp.controller.HasSession() {
		hp.sessionBox.Show()
		hp.formContainer.Hide()
	} else {
		hp.sessionBox.Hide()
		hp.formContainer.Show()
	}
	hp.Content.Refresh()
}

func (hp *HomePage) showFormView() {
	hp.sessionBox.Hide()
	hp.tokenForm.SetEnabled(true)
	hp.formContainer.Show()
	hp.statusCard.SetState(widgets.StatusDisconnected, "Desconectado")
	hp.Content.Refresh()
}

func (hp *HomePage) startReconnect() {
	hp.reconnectBtn.Disable()
	hp.newConnBtn.Disable()
	hp.statusCard.SetState(widgets.StatusConnecting, "Reconectando...")
	hp.loading.Reset()

	hp.controller.ReconnectAsync(
		func(step string) {
			fyne.Do(func() {
				hp.loading.ShowProgress(step)
			})
		},
		func(info *controller.ConnectionInfo) {
			fyne.Do(func() {
				hp.loading.ShowSuccess("Conectado")
				hp.statusCard.SetState(widgets.StatusConnected, "Conectado")
				hp.reconnectBtn.Enable()
				hp.newConnBtn.Enable()
				if hp.onConnected != nil {
					hp.onConnected(info)
				}
			})
		},
		func(err error) {
			fyne.Do(func() {
				hp.loading.ShowError(err.Error())
				hp.statusCard.SetState(widgets.StatusError, err.Error())
				hp.reconnectBtn.Enable()
				hp.newConnBtn.Enable()
				if !hp.controller.HasSession() {
					hp.showFormView()
				}
			})
		},
	)
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
	hp.updateVisibility()
}
