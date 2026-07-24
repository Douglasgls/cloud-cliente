package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type LoadingWidget struct {
	widget.BaseWidget
	progressText *widget.Label
	spinner      *widget.ProgressBarInfinite
	box          *fyne.Container
}

func NewLoadingWidget() *LoadingWidget {
	lbl := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
	spinner := widget.NewProgressBarInfinite()

	lw := &LoadingWidget{
		progressText: lbl,
		spinner:      spinner,
	}

	lw.box = container.NewVBox(
		spinner,
		lbl,
	)
	lw.box.Hide()

	lw.ExtendBaseWidget(lw)
	return lw
}

func (lw *LoadingWidget) ShowProgress(step string) {
	lw.progressText.SetText("⏳ " + step)
	lw.box.Show()
	lw.spinner.Show()
	lw.Refresh()
}

func (lw *LoadingWidget) ShowSuccess(msg string) {
	lw.progressText.SetText("✓ " + msg)
	lw.spinner.Hide()
	lw.Refresh()
}

func (lw *LoadingWidget) ShowError(msg string) {
	lw.progressText.SetText("❌ " + msg)
	lw.spinner.Hide()
	lw.Refresh()
}

func (lw *LoadingWidget) Reset() {
	lw.progressText.SetText("")
	lw.box.Hide()
	lw.Refresh()
}

func (lw *LoadingWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(lw.box)
}
