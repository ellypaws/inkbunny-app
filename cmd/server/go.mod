module github.com/ellypaws/inkbunny-app/server

go 1.23.0

replace github.com/ellypaws/inkbunny-app => ../..

replace github.com/ellypaws/inkbunny-sd => ../../pkg/mod/github.com/ellypaws/inkbunny-sd

require (
	github.com/ellypaws/inkbunny-app v0.0.0-20250307162652-bd25669a153c
	github.com/ellypaws/inkbunny-sd v0.0.0-20250307145449-b2576215e847
	github.com/joho/godotenv v1.5.1
	github.com/labstack/echo/v4 v4.13.3
	github.com/labstack/gommon v0.4.2
	github.com/muesli/termenv v0.16.0
)

require (
	github.com/EndFirstCorp/peekingReader v0.0.0-20171012052444-257fb6f1a1a6 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ellypaws/inkbunny/api v0.0.0-20240521065300-7d34160ddf2d // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/lu4p/cat v0.1.6-0.20231019140758-acd8306e6645 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/redis/go-redis/v9 v9.7.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/image v0.25.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	modernc.org/libc v1.61.13 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.8.2 // indirect
	modernc.org/sqlite v1.36.0 // indirect
)
