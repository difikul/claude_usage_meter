package main

import (
	"embed"
	"log"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIconData []byte

func main() {
	app := application.New(application.Options{
		Name: "Claude Usage Meter",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		Services: []application.Service{
			application.NewService(&UsageService{}),
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:          "Claude Usage Meter",
		Width:          340,
		Height:         340,
		Frameless:      true,
		AlwaysOnTop:    true,
		Hidden:         true,
		BackgroundType: application.BackgroundTypeTransparent,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})

	systray := app.SystemTray.New()

	if runtime.GOOS == "darwin" {
		systray.SetTemplateIcon(trayIconData)
	} else {
		systray.SetIcon(trayIconData)
	}
	systray.SetTooltip("Claude Usage Meter")

	menu := app.Menu.New()
	menu.Add("Show/Hide Widget").OnClick(func(ctx *application.Context) {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show()
			window.Focus()
		}
	})
	menu.Add("Refresh Now").OnClick(func(ctx *application.Context) {
		window.EmitEvent("usage-updated", nil)
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	systray.SetMenu(menu)

	systray.AttachWindow(window).WindowOffset(5)

	// Auto-refresh goroutine (every 60s)
	go func() {
		// Initial show after small delay
		time.Sleep(500 * time.Millisecond)
		window.Show()
		window.Focus()

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			window.EmitEvent("usage-updated", nil)
		}
	}()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
