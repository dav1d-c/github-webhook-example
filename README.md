# [go-]github-webhook-example

## Introduction

### The Challenge

1. Automate protection of the default (`main`) branch so that code reviews are required in order to puch or merge into the aforementioned branch whenever a new Repository is created within a GitHub Organization.
1. Automate creation of a new Issue within the new Repository that mentions the protection which was added, mentioning yourself as a heads up that it ran successfully.

### The Implementetion

I have been meaning to freshen up my GOLang skills, so let's choose that as the programming language for implemention of the solution.

Google has a wonderful GO module for intereacting with the GitHub v3 API: 
* https://github.com/google/go-github

Githubevents is a GO module that will allow us to handle the desired webhook events easily using `.OnRepositoryEventCreated()`: 
* https://github.com/cbrgm/githubevents

It seems like we ought to be able to combine these GO modules into something that can solve the challenge outlined above quite eloquently.

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
go run main.go
```

Also setup some Ingress *(using `ngrok`)* in another terminal window:

```
ngrok http 8080
```

Then take the resulting `[random-bits-your-ip-ad-dr].ngrok.io` FQDN from ngrok and use it to configure a webhook reciever in the GitHub UI of your GitHub Organization. Please ensure that Repository create events are contained within your events selection *(otherwise the desired events will not reach the webhook reciever for processing).*

When creating new GitHub repositories under your Organization, it is important to make the following selections:
* **Public** *(A limitation of my free service tier GitHub Organization)*

Testing pushes to the `main` branch of the new Repoisitory using the git cli should now restrict direct pushes by non-owners:

```
ERROR: Permission to [Your-Org]/[Your-New-Repo].git denied to [non-owner-username].
fatal: Could not read from remote repository.

Please make sure you have the correct access rights
and the repository exists.
```

## Process Diagram

The following diagram shows the process flow, starting from the Developer requesting a new Repository within the GitHub Organization using the GitHub Web UI:

![Process Flow Diagram](https://github.com/dav1d-c/github-webhook-example/blob/main/imgs/process-flow.jpg?raw=true "Process Flow Diagram")

## Interesting

### Initialzing a new Repository Through the GitHub v3 API

Although the code used to update the README.md file contents *(when it already exists)* is technically not required for the solution, however I was using this code path as an experiment in order to challenge myself and see if I could create the very first commit within the Repository (aka initialize the `main` branch) if it had not been done as part of the creation step. I managed to find this discussion, which goes back 10 years:

* https://stackoverflow.com/questions/9670604/github-v3-api-how-to-create-initial-commit-for-my-shiny-new-repository

Which also contains a recent update from last year showing that there is now a GitHub API end point which can accomplish this, after searching through the `go-github` module documentation I landed on the following function:

* https://pkg.go.dev/github.com/google/go-github/v45@v45.0.0/github#RepositoriesService.CreateFile

The above function appears to be, and is indeed the missing function to invoke the method outlined in the earlier discussion in order to initialize the branch with it's very first commit. Stretch goal accomplished! :)

### Error Handling

An astutue observer would notice that this code example has redundant error handling. The errors are handled in both the callback to handle the event, plus a more generic error handler callback is also registered. Both are not required, but I had used this during testing of my code to ensure that I was not missing errors which would then be unhandled as a result.

## Other/Future Considerations

* Test Coverage Needed! Clearly `TDD` *(Test Driven Development)* was not in play while cobbling together this proof of concept example.
* How to identify and apply Branch Protections to already created Repositories? *(migration of existing Repos?)*
* Should creation of the webhook be configure via the API at some point? *(instead of relying on manual configuration?)*

## Reference Links

* https://direnv.net/
* https://github.com/google/go-github
* https://github.com/cbrgm/githubevents
* https://docs.github.com/en/developers/webhooks-and-events/webhooks/creating-webhooks
* https://stackoverflow.com/questions/9670604/github-v3-api-how-to-create-initial-commit-for-my-shiny-new-repository
* https://pkg.go.dev/github.com/google/go-github/v45@v45.0.0/github#RepositoriesService.CreateFile
