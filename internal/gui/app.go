package gui

import (
	"cloud-client/internal/gui/controller"
)

func RunApp(ctrl *controller.ConnectController) {
	window := NewMainWindow(ctrl)
	window.ShowAndRun()
}
