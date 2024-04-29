<p align="center">
  <img src="https://inkbunny.net/images81/elephant/logo/bunny.png" width="100" />
  <img src="https://inkbunny.net/images81/elephant/logo/text.png" width="300" />
  <br>
  <h1 align="center">Inkbunny Auditor</h1>
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
  <a href="https://goreportcard.com/report/github.com/ellypaws/inkbunny-app">
    <img src="https://goreportcard.com/badge/github.com/ellypaws/inkbunny-app" alt="Go Report Card" />
  </a>
  <br>
  <a href="https://github.com/ellypaws/inkbunny-app/graphs/contributors">
    <img alt="Inkbunny ML contributors" src="https://img.shields.io/github/contributors/ellypaws/inkbunny-app">
  </a>
  <a href="https://github.com/ellypaws/inkbunny-app/commits/main">
    <img alt="Commit Activity" src="https://img.shields.io/github/commit-activity/m/ellypaws/inkbunny-app">
  </a>
  <a href="https://github.com/ellypaws/inkbunny-app">
    <img alt="GitHub Repo stars" src="https://img.shields.io/github/stars/ellypaws/inkbunny-app?style=social">
  </a>
</p>

--------------

<p align="right"><i>Disclaimer: This project is not affiliated or endorsed by Inkbunny.</i></p>

<img src="https://go.dev/images/gophers/ladder.svg" width="48" alt="Go Gopher climbing a ladder." align="right">

This project is designed to detect AI-generated images made with stable diffusion in Inkbunny submissions. It processes
the descriptions of submissions and extracts prompt details through a Language Learning Model (LLM). The processed data
is then structured into a text-to-image format.

By using crafted [heuristics](https://github.com/ellypaws/inkbunny-sd),
as well as the potential to use an LLM to inference the parameters.
A general purpose [API](cmd/api) library is available to integrate with your own program logic.

There are three different projects that aim to help in auditing and moderating AI generated content.

1. [Inkbunny AI Bridge](cmd/extension): A userscript server that constructs a prepared ticket based
   on [heuristics](https://github.com/ellypaws/inkbunny-sd)
   for you to audit and modify to then submit to Inkbunny.
   ![Inkbunny AI Bridge](cmd/extension/doc/ticket.png)
2. [Inkbunny ML](cmd/server): A general purpose [API](cmd/api) that features different tools to help in auditing and
   moderating AI generated content.

   It contains a database for managing tickets, auditors, artist lookup and auditing system.

   It also provides both a local and Redis cache layer for performance.
   The cache layer tries to be reasonably aggressive to make it performant and scalable.
   ![Inkbunny ML](cmd/server/doc/screenshot.png)
3. [CLI](cmd/cli): A command line interface that allows you to interact with the Inkbunny ML [API](cmd/api).
   It provides a way to interact with the API without needing to use the web interface.
   ![Inkbunny CLI](cmd/cli/doc/cli.gif)

## Usage

Prerequisites: Make sure you have api turned on in your Inkbunny account settings.
You will need your API key and SID to use the Inkbunny API.
You can change this in your [account settings](https://inkbunny.net/account.php#:~:text=API%20(External%20Scripting))

You can read the individual readme files for each project to get started.
An example usage for [Inkbunny AI Bridge](cmd/extension) is provided below.

### Building from Source

If you're building from source, you will need to install the dependencies:
Download Go 1.22.2 or later from the [official website](https://golang.org/dl/).

Set the environment variables for the server to run. You can set the following environment variables:

```bash
export PORT "your_port" # default is 1323
export API_HOST "your_api_host"
export SD_HOST "your_sd_host" # default is "http://localhost:7860"
export REDIS_HOST "your_redis_host" # default is "localhost:6379", when not set, uses local memory cache
export REDIS_PASSWORD "your_redis_password"
export REDIS_USER "your_redis_user" # when not set, uses 'default'
```

An optional Redis server can be used for caching.
If not set, it will fall back to local memory cache.
You can always override this behavior for most request by setting the `Cache-Control` header to `no-cache`.

```bash
git clone https://github.com/ellypaws/inkbunny-app.git
cd inkbunny-app/cmd/extension

go build -o inkbunny-ai-bridge
./inkbunny-ai-bridge
```

You can also use the pre-built binaries from the [releases page](https://github.com/ellypaws/inkbunny-sd/releases).
