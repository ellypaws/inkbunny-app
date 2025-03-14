package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	level "github.com/labstack/gommon/log"
	"github.com/muesli/termenv"

	"github.com/ellypaws/inkbunny-app/pkg/api"
	"github.com/ellypaws/inkbunny-app/pkg/api/cache"
	"github.com/ellypaws/inkbunny-app/pkg/db"

	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

var (
	database *db.Sqlite
	sdHost   = sd.DefaultHost
	apiHost  *url.URL
	port     uint = 1323
)

func main() {
	api.Run(api.RunConfig{
		Database:    database,
		SDHost:      sdHost,
		ServerHost:  apiHost,
		Port:        port,
		LogLevel:    level.DEBUG,
		Middlewares: middlewares,
		Extra:       extra,
	})
}

var middlewares = []echo.MiddlewareFunc{
	middleware.LoggerWithConfig(
		middleware.LoggerConfig{
			Skipper:          nil,
			Format:           `${time_custom}     	${status} ${method}  ${host}${uri} in ${latency_human} from ${remote_ip} ${error}` + "\n",
			CustomTimeFormat: time.DateTime,
		},
	),
	middleware.RemoveTrailingSlash(),
	middleware.Gzip(),
	middleware.Decompress(),
	middleware.NonWWWRedirect(),
	middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}),
}

var extra = []func(e *echo.Echo){
	startupMessage,
	func(e *echo.Echo) {
		e.GET("/", redirect, api.StaticMiddleware...)
		e.GET("/*", echo.StaticDirectoryHandler(
			echo.MustSubFS(e.Filesystem, "public"),
			false,
		), api.StaticMiddleware...)
	},
}

func redirect(c echo.Context) error {
	return c.Redirect(http.StatusTemporaryRedirect, "https://github.com/ellypaws/inkbunny-app")
}

func startupMessage(e *echo.Echo) {
	colors := []struct {
		text  string
		color string
	}{
		{"M", "#447294"},
		{"a", "#4f7d9e"},
		{"i", "#5987a8"},
		{"n", "#6492b2"},
		{"t", "#6f9cbd"},
		{"a", "#7aa7c7"},
		{"i", "#84b1d1"},
		{"n", "#8fbcdb"},
		{"e", "#a0c0d6"},
		{"d", "#b1c5d1"},
		{" ", "#c2c9cc"},
		{"b", "#d2cdc6"},
		{"y", "#e3d2c1"},
		{":", "#f4d6bc"},
	}

	var coloredText strings.Builder
	for _, ansi := range colors {
		coloredText.WriteString(termenv.String(ansi.text).Foreground(termenv.RGBColor(ansi.color)).Bold().String())
	}

	e.Logger.Infof("%s %s", coloredText.String(), "https://github.com/ellypaws")
	e.Logger.Infof("Post issues at %s", "https://github.com/ellypaws/inkbunny-app/issues")

	e.Logger.Infof("     api host: %s", api.ServerHost)
	e.Logger.Infof("      sd host: %s", api.SDHost)

	if api.SDHost.Alive() {
		e.Logger.Infof("      sd host: %s", api.SDHost)
	} else {
		e.Logger.Warnf("      sd host: %s (not running)", api.SDHost)
	}
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cache.Init()

	if p := os.Getenv("PORT"); p != "" {
		i, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			log.Fatal(err)
		}
		port = uint(i)
	}

	envApiHost := os.Getenv("API_HOST")
	if envApiHost == "" {
		log.Printf("API_HOST is not set, using default localhost:%d", port)
		apiHost = &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", port),
		}
	} else {
		var err error
		apiHost, err = url.Parse(envApiHost)
		if err != nil {
			log.Fatal(err)
		}
	}

	if h := os.Getenv("SD_HOST"); h != "" {
		u, err := url.Parse(h)
		if err != nil {
			log.Fatal(err)
		}
		sdHost = (*sd.Host)(u)
	} else {
		log.Println("warning: SD_HOST not set, using default localhost:7860")
	}

	var err error
	database, err = db.New(nil)
	if err != nil {
		log.Fatal(err)
	}
}
