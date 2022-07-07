# github-webhook-example

## Introduction

@TODO

## Quick Start

After making yourself a copy of the contents of this repo, you should be able to get it up and running fairly quickly. I make pretty extensive use of **direnv** to allow me to customize environment variables quickly based on my current location (and the .envrc file contained therein). I have provided an example file (.envrc.example) in this repo which can be quickly customized *(.envrc also exists inside the .gitignore file to prevent accidental commit of the "secrets" file):*

```
$ git clone https://github.com/dav1d-c/github-webhook-example.git
$ cd github-webhook-example
$ cp .envrc.example .envrc
$ <editor-of-choice> .envrc
<customize values...>
$ direnv allow
direnv: loading ~/Development/git/github-webhook-example/.envrc
direnv: export +GITHUB_COMMENT_MENTION +GITHUB_EMAIL_PRIVATE +GITHUB_ORG_NAME +GITHUB_PERSONAL_ACCESS_TOKEN +GITHUB_WEBHOOK_SECRET
```

If you don't have direnv and want to run the webhook reciever without it, then the .envrc file can be sourced into your running shell *(assumes BASH or equivilent):*

```
source .envrc
```

Next, let's ensure we have required GO modules and start up the webhook listener:

```
go get
go run main.c
```

Also setup some Ingress (using ngrok) in another terminal window:

```
ngrok http 8080
```

Then take the resulting `[random-bits-your-rev-ip].ngrok.io` FQDN from ngrok and use it to configure a webhook reciever in the GitHub UI of your GitHub Organization. Please ensure that Repository create events are contained within your events selection *(otherwise the desired events will not reach the webhook reciever for processing).*

When creating new GitHub repositories under your Organization, it is important to make the following selections:
* Public *(A limitation of my free service tier GitHub Organization)*
* Add a README file *(initializes the the default `main` branch, so that the code can protect it)*

## Process Diagram

@TODO

## Interesting

@TODO

## Links

* https://direnv.net/
* https://github.com/google/go-github
* https://github.com/cbrgm/githubevents
