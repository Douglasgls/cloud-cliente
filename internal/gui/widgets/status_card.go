package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type StatusState int

const (
	StatusDisconnected StatusState = iota
	StatusConnecting
	StatusConnected
	StatusError
)

type StatusCard struct {
	widget.BaseWidget
	dot        *canvas.Circle
	statusText *widget.Label
	box        *fyne.Container
}

func NewStatusCard() *StatusCard {
	dot := canvas.NewCircle(color.RGBA{R: 150, G: 150, B: 150, A: 255})
	dot.Resize(fyne.NewSize(14, 14))

	lbl := widget.NewLabelWithStyle("Desconectado", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	card := &StatusCard{
		dot:        dot,
		statusText: lbl,
	}

	card.box = container.NewHBox(
		widget.NewLabel("Status:"),
		container.NewCenter(dot),
		lbl,
	)

	card.ExtendBaseWidget(card)
	card.SetState(StatusDisconnected, "Desconectado")
	return card
}

func (s *StatusCard) SetState(state StatusState, customMsg ...string) {
	msg := ""
	if len(customMsg) > 0 && customMsg[0] != "" {
		msg = customMsg[0]
	}

	switch state {
	case StatusDisconnected:
		s.dot.FillColor = color.RGBA{R: 150, G: 150, B: 150, A: 255} // Gray
		if msg == "" {
			msg = "Desconectado"
		}
	case StatusConnecting:
		s.dot.FillColor = color.RGBA{R: 245, G: 166, B: 35, A: 255} // Yellow / Orange
		if msg == "" {
			msg = "Conectando..."
		}
	case StatusConnected:
		s.dot.FillColor = color.RGBA{R: 46, G: 204, B: 113, A: 255} // Green
		if msg == "" {
			msg = "Conectado"
		}
	case StatusError:
		s.dot.FillColor = color.RGBA{R: 231, G: 76, B: 60, A: 255} // Red
		if msg == "" {
			msg = "Erro"
		}
	}

	s.statusText.SetText("● " + msg)
	s.dot.Refresh()
}

func (s *StatusCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.box)
}
