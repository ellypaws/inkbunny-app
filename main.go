package main

import (
	"github.com/ellypaws/inkbunny-app/api"
	"github.com/ellypaws/inkbunny-app/cmd/db"
	sd "github.com/ellypaws/inkbunny-sd/stable_diffusion"
	"log"
	"net/url"
	"os"
)

var (
	database *db.Sqlite
	sdHost   = sd.DefaultHost
	apiHost  *url.URL
	port     = "1323"
)

func main() {
	api.Run(database, sdHost, apiHost, port)
}

func init() {
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	envApiHost := os.Getenv("API_HOST")
	if envApiHost == "" {
		log.Printf("API_HOST is not set, using default localhost:%s\n", port)
		api.ServerHost = &url.URL{
			Scheme: "http",
			Host:   "localhost:" + port,
		}
	} else {
		apiHost, err := url.Parse(envApiHost)
		if err != nil {
			log.Fatal(err)
		}
		api.ServerHost = apiHost
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

	log.Printf("api host: %s\n", envApiHost)
	log.Printf("sd host: %s\n", sdHost)

	if sdHost == nil || !sdHost.Alive() {
		log.Println("warning: host is not alive")
	}

	var err error
	database, err = db.New(nil)
	if err != nil {
		log.Fatal(err)
	}
}
