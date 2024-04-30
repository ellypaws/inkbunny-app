module github.com/ellypaws/inkbunny-app/extension

go 1.22.2

replace github.com/ellypaws/inkbunny-app => ../..

replace github.com/ellypaws/inkbunny-sd => ../mod/github.com/ellypaws/inkbunny-sd

require (
	github.com/ellypaws/inkbunny-app v0.0.0
	github.com/ellypaws/inkbunny-sd v0.0.0
	github.com/labstack/echo/v4 v4.12.0
	github.com/labstack/gommon v0.4.2
	github.com/muesli/termenv v0.15.2
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coocood/freecache v1.2.4 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ellypaws/inkbunny/api v0.0.0-20240411110242-d491ced97f23 // indirect
	github.com/gitsight/go-echo-cache v1.0.1 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mcuadros/go-defaults v1.2.0 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/redis/go-redis/v9 v9.5.1 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.22.0 // indirect
	golang.org/x/image v0.15.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	modernc.org/gc/v3 v3.0.0-20240304020402-f0dba7c97c2b // indirect
	modernc.org/libc v1.50.4 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.29.8 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)
