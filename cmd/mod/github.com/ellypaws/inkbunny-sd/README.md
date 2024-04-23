<p align="center">
  <img src="https://inkbunny.net/images81/elephant/logo/bunny.png" width="100" />
  <img src="https://inkbunny.net/images81/elephant/logo/text.png" width="300" />
  <br>
  <h1 align="center">Inkbunny ML</h1>
</p>

<p align="center">
  <a href="https://inkbunny.net/">
    <img alt="Inkbunny" src="https://img.shields.io/badge/website-inkbunny.net-blue">
  </a>
  <a href="https://wiki.inkbunny.net/wiki/API">
    <img alt="API" src="https://img.shields.io/badge/api-inkbunny.net-blue">
  </a>
  <a href="https://pkg.go.dev/github.com/ellypaws/inkbunny/api">
    <img alt="api reference" src="https://img.shields.io/badge/api-inkbunny/api-007d9c?logo=go&logoColor=white">
  </a>
  <a href="https://github.com/ellypaws/inkbunny">
    <img alt="api github" src="https://img.shields.io/badge/github-inkbunny/api-007d9c?logo=github&logoColor=white">
  </a>
  <a href="https://goreportcard.com/report/github.com/ellypaws/inkbunny-sd">
    <img src="https://goreportcard.com/badge/github.com/ellypaws/inkbunny-sd" alt="Go Report Card" />
  </a>
  <br>
  <a href="https://github.com/ellypaws/inkbunny-sd/graphs/contributors">
    <img alt="Inkbunny ML contributors" src="https://img.shields.io/github/contributors/ellypaws/inkbunny-sd">
  </a>
  <a href="https://github.com/ellypaws/inkbunny-sd/commits/main">
    <img alt="Commit Activity" src="https://img.shields.io/github/commit-activity/m/ellypaws/inkbunny-sd">
  </a>
  <a href="https://github.com/ellypaws/inkbunny-sd">
    <img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/ellypaws/inkbunny-sd?style=social">
  </a>
</p>

--------------

<p align="right"><i>Disclaimer: This project is not affiliated or endorsed by Inkbunny.</i></p>

<img src="https://go.dev/images/gophers/ladder.svg" width="48" alt="Go Gopher climbing a ladder." align="right">

This project is designed to detect AI-generated images made with stable diffusion in Inkbunny submissions. It processes
the descriptions of submissions and extracts prompt details through a Language Learning Model (LLM). The processed data
is then structured into a text-to-image format.

## Usage

You will be prompted for your Inkbunny username and password.
Environment variables can also be used to set the SID directly if
you [login manually](https://wiki.inkbunny.net/wiki/API#Login) and get the SID.
You can login as a guest by entering "`guest`" (or leaving it blank) as the username and leaving the password blank.

```bash
  Enter username [guest]:
  Enter password (hidden): 
  
  Logged in as your_username, sid: your_sid
  Enter submission IDs (comma separated) or a tag [tag:ai_generated]: your_submission_ids
```

It's also possible to search for tags such as "`tag:ai_generated`". Then this will return the submission IDs for you to
check.
If you leave the field empty, it will search for the tag "`tag:ai_generated`" by default.

This will return 5 submissions with the tag "ai_generated" and prompt you to select one to process.

After you've inputted submission IDs, it will ask if you want to use an LLM to infer parameters.
If you select `n` (default), it will only use heuristics and simple regex to find the prompt.

```bash
  Use an LLM to infer parameters? (y/[n]): y
  Inferencing from http://localhost:7869/v1/chat/completions
  Inferencing text to image (1/3)
  Accumulated tokens: 512
  Text to image saved to text_to_image.json
```

Then finally it will save to a json file in the current directory. You will be prompted if you want to log out.

### Building from Source

Prerequisites: Make sure you have api turned on in your Inkbunny account settings. You will need your API key and SID to
use the Inkbunny API. You can change this in
your [account settings](https://inkbunny.net/account.php#:~:text=API%20(External%20Scripting))

If you're building from source, you will need to install the dependencies:
Download Go 1.22.0 or later from the [official website](https://golang.org/dl/).

```bash
git clone https://github.com/ellypaws/inkbunny-sd.git
cd inkbunny-sd

go build .\cmd\
```

You can also use the pre-built binaries from the [releases page](https://github.com/ellypaws/inkbunny-sd/releases).

### Using Localhost

```go
package main

import (
	"github.com/ellypaws/inkbunny-sd/llm"
	"log"
	"net/url"
)

func main() {
	localhost := llm.Localhost()

	config := llm.Config{
		APIKey: "if needed",
		Endpoint: url.URL{
			Scheme: "http",
			Host:   "localhost:7860",
			Path:   "/v1/chat/completions",
		},
	}

	request := llm.DefaultRequest("your content here")
	response, _ := config.Infer(request)
	response, _ = localhost.Infer(request)

	log.Println(response.Choices[0].Message.Content)
}

```

### JSON System prompt

You can prefill the system prompt by using the helper
function [`PrefillSystemDump`](llm/defaults.go#L51)`(request entities.TextToImageRequest)`

```go
package main

import (
	"github.com/ellypaws/inkbunny-sd/entities"
	"github.com/ellypaws/inkbunny-sd/utils"
	"log"
)

func main() {
	request := entities.TextToImageRequest{
		Steps:       50,
		SamplerName: "DPM++ 2M Karras",
		OverrideSettings: entities.Config{
			SDModelCheckpoint: &alternateModel,
			SDCheckpointHash:  "70b33002f4",
		},
		CFGScale:          12,
		DenoisingStrength: 0.45,
		Prompt:            utils.ExtractPositivePrompt("description from submission"),
		NegativePrompt:    utils.ExtractNegativePrompt("description from submission"),
	}

	message, err := PrefillSystemDump(request)
	if err != nil {
		log.Fatalf("Error prefilling system dump: %v", err)
	}

	log.Println(message)
}

```

## Inkbunny API Library

This project uses the Inkbunny API library to log in and get submission details. Here is a sample usage:

```go
package main

import (
	"github.com/ellypaws/inkbunny/api"
	"log"
)

func main() {
	user := &api.Credentials{
		Username: "your_username",
		Password: "your_password",
	}

	user, err := user.Login()
	if err != nil {
		log.Printf("Error logging in: %v", err)
		return
	}
	
	log.Printf("Logged in with session ID: %s", user.Sid)
}

```

Altogether, we can use the [Inkbunny API library](https://github.com/ellypaws/inkbunny) to log in, get submission details, and then use the LLM to process the
descriptions of the submissions.
Then we can use the description to make an inference request to the LLM.