/*
 * This file is part of eLabFTW lite.
 *
 * @author Nicolas CARPi / Deltablot
 * @author Moustapha Camara / Deltablot
 * @copyright 2026 Nicolas CARPi
 * @see https://www.elabftw.net Official website
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

var AppVersion = "dev"

func main() {
	app := NewApp()

	wailsApp := application.New(application.Options{
		Name:        "elabftw-lite",
		Description: "Local-first eLabFTW lite client",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Go signal handlers crash the app on iOS
		DisableDefaultSignalHandler: runtime.GOOS == "ios",
	})

	about := func() {
		wailsApp.Dialog.Info().
			SetTitle("About eLabFTW lite").
			SetMessage("Version: " + AppVersion + "\n\nLocal-first eLabFTW lite client.\nDevelopment sponsored by CNRS.").
			Show()
	}

	toggleFullscreen := func() {
		if w := wailsApp.Window.Current(); w != nil {
			w.ToggleFullscreen()
		}
	}

	appMenu := wailsApp.NewMenu()
	if runtime.GOOS == "darwin" {
		appSubmenu := appMenu.AddSubmenu("eLabFTW lite")
		appSubmenu.Add("About...").OnClick(func(*application.Context) { about() })
		appSubmenu.AddSeparator()
		appSubmenu.Add("Quit").SetAccelerator("CmdOrCtrl+q").OnClick(func(*application.Context) { wailsApp.Quit() })

		appMenu.AddRole(application.EditMenu)

		viewMenu := appMenu.AddSubmenu("View")
		viewMenu.Add("Toggle Full Screen").SetAccelerator("CmdOrCtrl+Ctrl+f").OnClick(func(*application.Context) { toggleFullscreen() })
	} else {
		fileMenu := appMenu.AddSubmenu("File")
		fileMenu.Add("Quit").SetAccelerator("CmdOrCtrl+q").OnClick(func(*application.Context) { wailsApp.Quit() })

		viewMenu := appMenu.AddSubmenu("View")
		viewMenu.Add("Toggle Full Screen").SetAccelerator("F11").OnClick(func(*application.Context) { toggleFullscreen() })

		helpMenu := appMenu.AddSubmenu("Help")
		helpMenu.Add("About...").OnClick(func(*application.Context) { about() })
	}
	wailsApp.Menu.Set(appMenu)

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              "elabftw-lite",
		Width:              1200,
		Height:             900,
		BackgroundColour:   application.NewRGB(27, 38, 54),
		URL:                "/",
		UseApplicationMenu: true,
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
