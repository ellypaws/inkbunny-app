<p align="center">
  <img src="https://inkbunny.net/images81/elephant/logo/bunny.png" width="100" />
  <img src="https://inkbunny.net/images81/elephant/logo/text.png" width="300" />
  <br>
  <h1 align="center">Inkbunny Extensions</h1>
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

This project is designed to detect AI-generated images made with stable diffusion in Inkbunny submissions. It processes files and descriptions and uses heuristics to determine if the submission follows the [guidelines](https://wiki.inkbunny.net/wiki/ACP#AI).

By using crafted [heuristics](https://github.com/ellypaws/inkbunny-sd),
as well as the potential to use an LLM to inference the parameters.
A general purpose [API](https://github.com/ellypaws/inkbunny-app) library is available to integrate with your own program logic.

## Inkbunny AI Bridge

The Inkbunny AI Bridge extends the functionality of your browser through a userscript that creates a ticket ready for your review. Based on advanced heuristics, the script prepares everything you need to ensure the content meets Inkbunny's standards.

It displays a badge on each submission to quickly notify you of any potential flagged submission worth verifying.

![Inkbunny AI Bridge](doc/screenshot.png)

It constructs a prepared ticket based on the heuristics for you to audit and modify to then submit to Inkbunny.

![Ticket](doc/ticket.png)

<details>
<summary>Full api server</summary>

Additionally, there's a [full api server](../server) that provides additional tools.

A demo app is available either at [https://inkbunny.keiau.space](https://inkbunny.keiau.space/app/audits) or in [retool](https://inkbunny.retool.com).
![Inkbunny Ticket Auditor](../server/doc/screenshot.png?raw=true)
</details>

## Installation Instructions

> *Make sure you have api turned on in your Inkbunny account settings. You will need your API key and SID to
use the Inkbunny API. You can change this in
your [account settings](https://inkbunny.net/account.php#:~:text=API%20(External%20Scripting))*

There are two parts to the Inkbunny AI Bridge, the [server](#server) and the [userscript](#userscript).

You will need to install a userscript manager extension in your web browser. You can use tampermonkey, greasemonkey or any similar userscript extension.

## Userscript

After installing a userscript manager, you can install the Inkbunny AI Bridge userscript.

1. The current version of the [userscript](https://github.com/ellypaws/inkbunny-extension/blob/main/scripts/Inkbunny%20AI%20bridge.user.js) is available in [https://github.com/ellypaws/inkbunny-extension](https://github.com/ellypaws/inkbunny-extension/tree/main/scripts)
2. Click on the "[Raw](https://github.com/ellypaws/inkbunny-extension/raw/main/scripts/Inkbunny%20AI%20bridge.user.js)" button, your userscript manager will recognize this as a userscript and ask for confirmation to install it.
3. Alternatively you can either download or copy the content of the userscript and paste it in your userscript manager.

A simpler [userscript](https://github.com/ellypaws/inkbunny-extension/blob/main/scripts/Inkbunny%20AI%20detector.user.js) is available if you do need the other features and only want the labelling, blurring or removal of AI generated images.

Todo:
 - [x] Fix blurring and removal of AI generated images (the old script does this but the new one is currently broken) 
 - [ ] Allow editing of the prepared ticket
 - [ ] Highlight more relevant metadata and print generation objects (e.g. model, prompt, etc). Currently you can view this in the console debug.
 - [ ] Better styling

#### Configuring the Userscript

After installing the userscript, you need to configure it to match your server URL. If you're running the server locally, the default URL is `http://localhost:1323`.

![menu](doc/login.png)

1. Click on the Tampermonkey icon in your browser to open the menu.
2. Find the Inkbunny AI Bridge and <kbd>Set API URL</kbd> if you're using a different server.
3. In the same menu, you can login using the <kbd>User menu (login)</kbd> button
4. You can choose between three different options for AI thumbnails:
    - <kbd>Label</kbd> - This will add a label to the submission.
    - <kbd>Blur</kbd> - This will blur the submission.
    - <kbd>Remove</kbd> - This will remove the submission.

Now, the Inkbunny AI Bridge should be ready to use.

## Server

Set the environment variables for the server to run. You can set the following environment variables:

```bash
export PORT "your_port" # default is 1323
export API_HOST "your_api_host"
export SD_HOST "your_sd_host" # default is "http://localhost:7860"
export REDIS_HOST "your_redis_host" # default is "localhost:6379", when not set, uses local memory cache
export REDIS_PASSWORD "your_redis_password"
export REDIS_USER "your_redis_user" # when not set, uses 'default'

./inkbunny-ai-bridge
```

An optional Redis server can be used for caching.
If not set, it will fall back to local memory cache.
You can always override this behavior for most request by setting the `Cache-Control` header to `no-cache`.

### Building from Source

If you're building from source, you will need to install the dependencies:
Download Go 1.22.3 or later from the [official website](https://golang.org/dl/).

> When cloning from the repository, make sure to use `--recurse-submodules` to initialize inkbunny-sd.

```bash
git clone --recurse-submodules https://github.com/ellypaws/inkbunny-app.git
cd inkbunny-app/cmd/extension

go build -o inkbunny-ai-bridge
./inkbunny-ai-bridge
```

And when pulling, make sure to update the submodules:

```bash
git pull --recurse-submodules

# or if you forgot to clone with submodules
git pull
git submodule update --init --recursive
```

You can also use the pre-built binaries from the [releases page](https://github.com/ellypaws/inkbunny-extension/releases).