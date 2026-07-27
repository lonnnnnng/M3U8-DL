package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "M3U8-DL",
		Width:     1180,
		Height:    760,
		MinWidth:  1040,
		MinHeight: 660,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Mac: &mac.Options{
			// long: 标题栏内容嵌入应用工具栏，深色主题不再残留独立的白色原生标题栏。
			TitleBar: mac.TitleBarHiddenInset(),
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
