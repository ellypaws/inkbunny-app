module github.com/ellypaws/inkbunny-app/cmd/cli

go 1.22.3

replace github.com/ellypaws/inkbunny-app => ../..

replace github.com/ellypaws/inkbunny-app/cmd/cli => .

replace github.com/ellypaws/inkbunny-sd => ./../mod/github.com/ellypaws/inkbunny-sd

require (
	github.com/76creates/stickers v1.3.1-0.20230410064447-c0cf398aec57
	github.com/TheZoraiz/ascii-image-converter v1.13.1
	github.com/charmbracelet/bubbles v0.18.0
	github.com/charmbracelet/bubbletea v0.26.4
	github.com/charmbracelet/lipgloss v0.11.0
	github.com/ellypaws/inkbunny-app v0.0.0-20240530024600-3a2355ea6793
	github.com/ellypaws/inkbunny-sd v0.0.0-20240612150455-0a759fecc348
	github.com/ellypaws/inkbunny/api v0.0.0-20240521065300-7d34160ddf2d
	github.com/lrstanley/bubblezone v0.0.0-20240602193536-d14e6b69d7f9
	golang.org/x/sys v0.21.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/charmbracelet/x/ansi v0.1.2 // indirect
	github.com/charmbracelet/x/input v0.1.2 // indirect
	github.com/charmbracelet/x/term v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.1.2 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/makeworld-the-better-one/dither/v2 v2.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/nathan-fiscaletti/consolesize-go v0.0.0-20220204101620-317176b6684d // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20240525044651-4c93da0ed11d // indirect
	golang.org/x/image v0.17.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)
