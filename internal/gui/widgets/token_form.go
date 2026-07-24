package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type TokenForm struct {
	widget.BaseWidget
	tokenEntry *widget.Entry
	connectBtn *widget.Button
	onSubmit   func(token string)
	box        *fyne.Container
}

func NewTokenForm(onSubmit func(token string)) *TokenForm {
	tokenEntry := widget.NewEntry()
	tokenEntry.SetPlaceHolder("Digite seu Token de Acesso...")

	connectBtn := widget.NewButton("Conectar", nil)
	connectBtn.Importance = widget.HighImportance

	tf := &TokenForm{
		tokenEntry: tokenEntry,
		connectBtn: connectBtn,
		onSubmit:   onSubmit,
	}

	connectBtn.OnTapped = func() {
		if tf.onSubmit != nil {
			tf.onSubmit(tf.tokenEntry.Text)
		}
	}

	tokenEntry.OnSubmitted = func(val string) {
		if tf.onSubmit != nil {
			tf.onSubmit(val)
		}
	}

	tf.box = container.NewVBox(
		widget.NewLabelWithStyle("Token de Acesso", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		tokenEntry,
		container.NewPadded(connectBtn),
	)

	tf.ExtendBaseWidget(tf)
	return tf
}

func (tf *TokenForm) SetEnabled(enabled bool) {
	if enabled {
		tf.tokenEntry.Enable()
		tf.connectBtn.Enable()
	} else {
		tf.tokenEntry.Disable()
		tf.connectBtn.Disable()
	}
}

func (tf *TokenForm) SetToken(token string) {
	tf.tokenEntry.SetText(token)
}

func (tf *TokenForm) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tf.box)
}
