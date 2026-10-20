package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

func setupTray(app *application.App, window *application.WebviewWindow) {
	var quitting bool

	showWindow := func() {
		window.Show()
		window.Focus()
	}

	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(icons.DefaultWindowsIcon)
	}
	tray.SetTooltip("preset-wails")

	menu := app.NewMenu()
	menu.Add("显示窗口").OnClick(func(*application.Context) {
		showWindow()
	})
	menu.Add("隐藏窗口").OnClick(func(*application.Context) {
		window.Hide()
	})
	alwaysOnTop := menu.AddCheckbox("置顶", true)
	alwaysOnTop.OnClick(func(*application.Context) {
		window.SetAlwaysOnTop(alwaysOnTop.Checked())
	})
	menu.AddSeparator()
	menu.Add("退出").OnClick(func(*application.Context) {
		quitting = true
		app.Quit()
	})
	tray.SetMenu(menu)

	tray.OnClick(showWindow)
	tray.OnRightClick(tray.OpenMenu)

	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if quitting {
			return
		}
		event.Cancel()
		window.Hide()
	})

	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
			showWindow()
		})
	}
}
