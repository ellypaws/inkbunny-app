module github.com/ellypaws/inkbunny-app/cmd/cli

go 1.22.2

replace github.com/ellypaws/inkbunny-app => ../..

replace github.com/ellypaws/inkbunny-app/cmd/cli => .

replace github.com/ellypaws/inkbunny-sd => ./../mod/github.com/ellypaws/inkbunny-sd

require (
	github.com/76creates/stickers v1.3.1-0.20230410064447-c0cf398aec57
	github.com/TheZoraiz/ascii-image-converter v1.13.1
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/bubbletea v0.25.0
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/ellypaws/inkbunny-app v0.0.0-00010101000000-000000000000
	github.com/ellypaws/inkbunny-sd v0.0.0-20240421145525-f3b56afc12a5
	github.com/ellypaws/inkbunny/api v0.0.0-20240411110242-d491ced97f23
	github.com/lrstanley/bubblezone v0.0.0-20240125042004-b7bafc493195
	golang.org/x/sys v0.19.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/gookit/color v1.4.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/makeworld-the-better-one/dither/v2 v2.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/muesli/ansi v0.0.0-20211031195517-c9f0611b6c70 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/nathan-fiscaletti/consolesize-go v0.0.0-20210105204122-a87d9f614b9d // indirect
	github.com/rivo/uniseg v0.4.6 // indirect
	github.com/sahilm/fuzzy v0.1.1-0.20230530133925-c48e322e2a8f // indirect
	github.com/xo/terminfo v0.0.0-20210125001918-ca9a967f8778 // indirect
	golang.org/x/image v0.10.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/term v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
