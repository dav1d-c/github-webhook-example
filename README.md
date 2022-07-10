# [go-]github-webhook-example

## Introduction

### The Challenge

1. Automate protection of the default (`main`) branch so that code reviews are required in order to push or merge into the aforementioned branch whenever a new Repository is created within a specific GitHub Organization.
1. Automate creation of a new Issue within the new Repository that mentions the protection which was added, mentioning yourself as a heads up that this automation executed successfully.

### The Implementetion

I have been meaning to freshen up my **GOLang** skills, so let's choose that as the programming language for implemention of the solution. A couple of thoughts spring to mind after some cursory research:

* GitHub offers a Branch Protection feature, which can be used to enforce controls on certain Repository branches.
* Branch Protections can be applied programmatically using the GitHub API.
* We can register Webhooks with GitHub which can then be used to receive events such as Repository create events *(that could trigger our automation).*

Google has a wonderful GO module for intereacting with the GitHub v3 API: 
* https://github.com/google/go-github

Githubevents is a GO module (built ontop of `go-github`) that will allow us to handle the desired webhook events easily using the `.OnRepositoryEventCreated()` function:
* https://github.com/cbrgm/githubevents

It seems like we should able to combine these GO modules into something that can solve the challenge outlined above quite eloquently.

## Quick Start

After making yourself a copy of the contents of this Repository, one should be able to get this code up and running fairly quickly. I make pretty extensive use of **direnv** to allow me to customize environment variables quickly based on my current working directory *(and the .envrc file contained therein).* I have provided an example file (`.envrc.example`) in this Repository which can be quickly customized *(.envrc also exists inside the `.gitignore` file to prevent accidental commit of the "secrets" file):*

```
$ git clone https://github.com/dav1d-c/github-webhook-example.git
$ cd github-webhook-example
$ cp .envrc.example .envrc
$ <editor-of-choice> .envrc
<customize values & save...>
$ direnv allow
direnv: loading ~/Development/git/github-webhook-example/.envrc
direnv: export +GITHUB_COMMENT_MENTION +GITHUB_EMAIL_PRIVATE +GITHUB_ORG_NAME +GITHUB_PERSONAL_ACCESS_TOKEN +GITHUB_WEBHOOK_SECRET
```

If one doesn't have direnv and wanted to run the webhook receiver without it, then the .envrc file can be sourced into your running shell *(assumes BASH or equivilent):*

```
source .envrc
```

Next, let's ensure we have the required GO modules and start up the webhook listener:

```
go get
go run main.go
```

Also setup some Ingress *(using `ngrok`)* in another terminal window:

```
ngrok http 8080
```

Then take the resulting `[random-bits-your-ip-ad-dr].ngrok.io` FQDN from ngrok and use it to configure a webhook receiver in the GitHub UI of your GitHub Organization. Please ensure that Repository create events are contained within your events selection *(otherwise the desired events will not reach the webhook receiver for processing).* Also select `application/json` as the mime-type for the content delivery.

When creating a new GitHub repositories under your Organization, it is important to make the following selections:
* **Public** *(A known limitation of my Free service tier GitHub Organization)*

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

### Initializing a new Repository Through the GitHub v3 API

Although hints/clues were provided regarding how to ensure the Repository's first branch is initialized during creation through the Web UI by checking the **Add README.md file** box during creation, there is no practical way to enforce this every time a member of our Organization creates a new Repository. I concluded that there must be a better way using the GitHub API *(and I was determined to find it).* During my research I managed to find this discussion, which goes back 10 years:

* https://stackoverflow.com/questions/9670604/github-v3-api-how-to-create-initial-commit-for-my-shiny-new-repository

Which also contains a recent update from last year showing that there is now a GitHub API end point which can accomplish this, and after searching through the `go-github` module documentation I landed on the following function:

* https://pkg.go.dev/github.com/google/go-github/v45@v45.0.0/github#RepositoriesService.CreateFile

The above function appears to be, and is indeed the missing function to invoke the method outlined in the earlier discussion in order to initialize the branch with its very first commit. When unable to locate the base Reference using the API, we can post a fresh `README.md` file and use that commit to initialize the `main` branch. There also exists code to update the contents of `README.md` if it already exists. I added this code so that the "Welcome" `README.md` contents can be customized the same way in either code path, so that the user will have a consistent experience regardless of which path is taken. Now that the branch exists, it can be protected!

### Error Handling

An astutue observer would notice that this code example has redundant error handling. The errors are handled in both the callback to handle the event, and also in a more generic error handler callback which is also registered. Both are technically not required, but I had used both during the testing phase of my development to ensure that I was not missing any errors, which would then be unhandled as a result.

## Other/Future Considerations

* **Test Coverage Needed!** Clearly `TDD` *(Test Driven Development)* was not in play while cobbling together this proof of concept example.
* How to identify and apply Branch Protections to already created Repositories within the Organization? *(migration of existing Repositories?)*
* Should creation of the webhook be configured via the API at some point? *(instead of relying on manual configuration using GitHub Web UI?)*
* Replace secrets read from Environment variables with an integration to a proper secret store *(i.e. Vault).*
* Refactor error handling into a single mechanism, once we are certain all errors are properly caught and handled in our code *(see previous section for more details).*
* Process logs shipped to an external log collection system, so that we have a long running and auditable history of automated protection events.

## Reference Links

* https://go.dev/
* https://direnv.net/
* https://ngrok.com/
* https://docs.github.com/en/rest
* https://github.com/google/go-github
* https://github.com/cbrgm/githubevents
* https://docs.github.com/en/developers/webhooks-and-events/webhooks/creating-webhooks
* https://stackoverflow.com/questions/9670604/github-v3-api-how-to-create-initial-commit-for-my-shiny-new-repository
* https://pkg.go.dev/github.com/google/go-github/v45@v45.0.0/github#RepositoriesService.CreateFile
