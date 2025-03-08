package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	logger "github.com/labstack/gommon/log"
	"github.com/muesli/termenv"

	"github.com/ellypaws/inkbunny-app/cmd/api"
	"github.com/ellypaws/inkbunny-app/cmd/api/cache"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
)

var (
	sdHost    = sd.DefaultHost   // SD_HOST
	port      = "1323"           // PORT
	redisHost = "localhost:6379" // REDIS_HOST

	e = echo.New()
)

func main() {
	e.Use(middleware.Recover())

	for _, m := range middlewares {
		e.Use(m)
	}

	reducedMiddleware := append(slices.Clone(api.WithRedis),
		[]echo.MiddlewareFunc{
			api.RequireSID,
			api.TryAuditor,
		}...)

	e.GET("/review/:id", api.GetReviewHandler, reducedMiddleware...)
	e.POST("/review/:id", api.GetReviewHandler, reducedMiddleware...)

	e.GET("/report/:id/:key", api.GetReportKeyHandler, api.StaticMiddleware...)
	e.PATCH("/report", api.PatchReport, reducedMiddleware...)

	e.GET("/", redirect, api.StaticMiddleware...)
	e.GET("/*", echo.StaticDirectoryHandler(
		echo.MustSubFS(e.Filesystem, "public"),
		false,
	), api.StaticMiddleware...)

	startupMessage(e)
	e.Logger.Infof("Starting server on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

func redirect(c echo.Context) error {
	return c.Redirect(http.StatusTemporaryRedirect, "https://github.com/ellypaws/inkbunny-app")
}

func startupMessage(e *echo.Echo) {
	e.HideBanner = true
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

//go:embed artists.json
var artistsJSON []byte

//go:embed auditors.json
var auditorsJSON []byte

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

func init() {
	e.Logger.SetLevel(logger.DEBUG)
	e.Logger.SetHeader(`${time_rfc3339} ${level}	${short_file}:${line}	`)

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cache.Init()

	if h := os.Getenv("SD_HOST"); h != "" {
		u, err := url.Parse(h)
		if err != nil {
			e.Logger.Fatal(err)
		}
		sdHost = (*sd.Host)(u)
	} else {
		e.Logger.Warn("warning: SD_HOST not set, using default localhost:7860")
	}

	if p := os.Getenv("PORT"); p != "" {
		port = p
	} else {
		e.Logger.Warn("env PORT not set, using default 1323")
	}

	envApiHost := os.Getenv("API_HOST")
	if envApiHost == "" {
		e.Logger.Warnf("env API_HOST is not set, using default localhost:%s\n", port)
		api.ServerHost = &url.URL{
			Scheme: "http",
			Host:   "localhost:" + port,
		}
	} else {
		apiHost, err := url.Parse(envApiHost)
		if err != nil {
			e.Logger.Fatal(err)
		}
		api.ServerHost = apiHost
	}

	var err error
	api.Database, err = db.New(nil)
	if err != nil {
		e.Logger.Fatal(err)
	}

	var artists []db.Artist
	err = json.Unmarshal(artistsJSON, &artists)
	if err != nil {
		e.Logger.Fatal(err)
	}
	err = api.Database.UpsertArtist(artists...)
	if err != nil {
		e.Logger.Fatal(err)
	}

	var auditors []db.Auditor
	err = json.Unmarshal(auditorsJSON, &auditors)
	if err != nil {
		e.Logger.Fatal(err)
	}
	for i := range auditors {
		err = api.Database.InsertAuditor(auditors[i])
		if err != nil {
			e.Logger.Fatal(err)
		}
	}
}
