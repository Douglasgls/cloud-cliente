package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"cloud-client/internal/gui/controller"
	"cloud-client/internal/gui/pages"
)

type MainWindow struct {
	app           fyne.App
	window        fyne.Window
	controller    *controller.ConnectController
	mainContainer *fyne.Container
	homePage      *pages.HomePage
	connectedPage *pages.ConnectedPage
}

func NewMainWindow(ctrl *controller.ConnectController) *MainWindow {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(theme.DarkTheme())

	w := fyneApp.NewWindow("Cloud Client")
	w.Resize(fyne.NewSize(700, 450))

	mw := &MainWindow{
		app:        fyneApp,
		window:     w,
		controller: ctrl,
	}

	var home *pages.HomePage
	var connected *pages.ConnectedPage

	home = pages.NewHomePage(ctrl, func(info *controller.ConnectionInfo) {
		connected.SetConnectionInfo(info)
		mw.mainContainer.Objects = []fyne.CanvasObject{connected.Content}
		mw.mainContainer.Refresh()
	})

	connected = pages.NewConnectedPage(ctrl, func() {
		home.Reset()
		mw.mainContainer.Objects = []fyne.CanvasObject{home.Content}
		mw.mainContainer.Refresh()
	})

	mw.homePage = home
	mw.connectedPage = connected
	mw.mainContainer = container.NewStack(home.Content)

	w.SetContent(mw.mainContainer)
	return mw
}

func (mw *MainWindow) ShowAndRun() {
	mw.window.ShowAndRun()
}
