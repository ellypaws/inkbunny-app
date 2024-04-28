package main

import (
	_ "embed"
	"encoding/json"
	"github.com/ellypaws/inkbunny-app/api"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	logger "github.com/labstack/gommon/log"
	"net/url"
	"os"
	"time"
)

var (
	sdHost    = sd.DefaultHost   // SD_HOST
	port      = "1323"           // PORT
	redisHost = "localhost:6379" // REDIS_HOST

	e = echo.New()
)

func main() {
	e.Use(middleware.LoggerWithConfig(
		middleware.LoggerConfig{
			Skipper:          nil,
			Format:           `${time_custom}     	${status} ${method}  ${host}${uri} in ${latency_human} from ${remote_ip} ${error}` + "\n",
			CustomTimeFormat: time.DateTime,
		},
	))
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	config := append(api.WithRedis,
		[]echo.MiddlewareFunc{
			api.SIDMiddleware,
			api.Anonymous,
		}...)

	e.GET("/review/:id", api.GetReviewHandler, config...)
	e.POST("/review/:id", api.GetReviewHandler, config...)

	e.File("/favicon.ico", "../api/public/16930_inkbunny_inkbunnylogo_trans_rev_outline.ico")

	e.Logger.Info("Starting...")
	e.Logger.Fatal(e.Start(":" + port))
}

//go:embed artists.json
var artistsJSON []byte

//go:embed auditors.json
var auditorsJSON []byte

func init() {
	e.Logger.SetLevel(logger.DEBUG)
	e.Logger.SetHeader(`${time_rfc3339} ${level}	${short_file}:${line}	`)

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

	e.Logger.Infof("api host: %s\n", api.ServerHost)

	if sdHost == nil || !sdHost.Alive() {
		e.Logger.Warn("warning: host is not alive")
	}

	e.Logger.Infof("sd host: %s\n", sdHost)

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
