package main

import (
	"fmt"
	api "github.com/ellypaws/inkbunny-app/cmd/cli/requests"
	"github.com/ellypaws/inkbunny-sd/entities"
	"os"
)

func main() {
	config := api.New()

	go config.Run()

	resp := config.AddToQueue(&entities.TextToImageRequest{
		Prompt: "A cat",
		Steps:  20,
	})
	images, err := api.ToImages(<-resp)
	if err != nil {
		fmt.Println(err)
	}

	for i, img := range images {
		_ = os.WriteFile(fmt.Sprintf("image_%d.png", i), img, 0644)
	}
}
