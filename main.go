package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cbrgm/githubevents/githubevents"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

// globals
var gh_webhook_secret_key string = ""
var gh_personal_access_token string = ""
var gh_username_issue_mention string = ""
var gh_private_email string = ""
var gh_code_review_min int = 2
var gh_readme_contents string = "\nYour Organization **loves <3 documentation,** please don't forget to update this file with specific information about this project!\n"

// main
func main() {
	// Say hello
	log.Println("Example GitHub Webhook Handler is Starting Up Now...")

	// Read in parameters from ENV (ideally set using direnv)
	readValuesFromEnv()
	checkValuesFromEnv()

	// Create new instance of githubevents using gh_webhook_secret_key read in above
	handle := githubevents.New(gh_webhook_secret_key)
	// Create an instance of the github-go API client using the gh_personal_access_token read in above
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: gh_personal_access_token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Print some information about the rate limit associated with the user whose Personal Access Token we are using here
	printRateLimitUserInfo(client, ctx)
	// Attempt to fill-in user information from the current client context
	autoLoadUserValues(client, ctx)
	// Some helpful information about loaded configuration values
	reportLoadedConfigValues()

	// Pass the eventHandler to funcs that define callbacks to do the things
	setupErorrCallback(handle)
	setupProtectCallback(handle, client, ctx)

	// Setup a path to handle callbacks with
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		err := handle.HandleEventRequest(r)
		if err != nil {
			log.Printf("error %v", err)
		}
	})

	// Finally let's start listening on port 8080 on every available interface
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// setupProtectCallback
func setupProtectCallback(handle *githubevents.EventHandler, client *github.Client, ctx context.Context) {
	// Registrer a callback to handle the event when a Repo is created in this Org
	handle.OnRepositoryEventCreated(
		func(deliveryID string, eventName string, event *github.RepositoryEvent) error {
			// DEBUG
			log.Println("[setupProtectCallback] Processing " + eventName + " event with ID " + deliveryID + "...")
			//log.Println(github.Stringify(event))

			gh_organization_name := event.GetOrg().GetLogin()
			var repo *github.Repository = event.GetRepo()
			// Create the first branch via first commit of README.md
			var baseRef *github.Reference
			baseRef, _, err := client.Git.GetRef(ctx, gh_organization_name, repo.GetName(), "refs/heads/"+repo.GetDefaultBranch())
			if err != nil {
				log.Printf("Error retrieving Reference: %v\n", err)

				// Hmmmm, if the branch is not initialized, maybe we could do that now??
				init_err := initRepoWithReadMe(client, ctx, repo, gh_organization_name)
				if init_err != nil {
					log.Println("Unable to Initiailize " + repo.GetDefaultBranch() + " with a first commit to that branch.")
					return err
				}
			} else {
				// If there is no error from the above, then what likely happened is the branch is already init'ed (and we can customize the README.me file instead)
				updateRepoReadMe(client, ctx, baseRef, repo, gh_organization_name)
			}

			// Setup Branch Protection via ProtectionRequest
			prr := &github.PullRequestReviewsEnforcementRequest{RequiredApprovingReviewCount: gh_code_review_min, RequireCodeOwnerReviews: true, DismissStaleReviews: false}
			pr := &github.ProtectionRequest{RequiredPullRequestReviews: prr, AllowForcePushes: github.Bool(false)}
			protections, _, err := client.Repositories.UpdateBranchProtection(ctx, gh_organization_name, repo.GetName(), repo.GetDefaultBranch(), pr)
			if err != nil {
				log.Println(err)
				return err
			}
			log.Println(github.Stringify(protections.GetRequiredPullRequestReviews()))

			// Let's create an Issue alerting us to what has been done (but only if we made it this far!)
			issue_title := "New Repository Protection Applied Successfully"
			issue_body := "After the main branch was created, it was protected so that only properly reviewed code (with " + strconv.Itoa(gh_code_review_min) + " or more reviews) can be commited to the main branch\n\nCC @" + gh_username_issue_mention
			issue_repo := event.GetRepo().GetName()

			err = createIssue(client, ctx, gh_organization_name, issue_repo, issue_title, issue_body)
			if err != nil {
				return err
			}

			return nil
		})
}

// setupErrorCallback
func setupErorrCallback(handle *githubevents.EventHandler) {
	// catch any unhandle errors, so we can certain that we have at least captured and logged them for analysis later
	// perhaps this will double report some errors, but I would rather have some errors double reported over the chance of missing some
	handle.OnError(
		func(deliveryID string, eventName string, event interface{}, err error) error {
			// DEBUG
			log.Println("[setupErrorCallback] Encountered ERROR while processing " + eventName + " event with ID " + deliveryID + "...")
			log.Println(err)
			return nil
		})
}

// createIssue
func createIssue(client *github.Client, ctx context.Context, org string, repo string, title string, body string) error {
	issue_title := title
	issue_body := body

	// create a new IssueRequest using the provided inputs
	i := &github.IssueRequest{Title: &issue_title, Body: &issue_body}
	new_issue, _, err := client.Issues.Create(ctx, org, repo, i)
	if err != nil {
		log.Println("Problem encountered while attempting to create a GitHub Issue in the Repo " + repo)
		log.Println(err)
		return err
	}

	// DEBUG
	log.Printf("Successfully created new issue: %v in repo: %v\n", new_issue.GetTitle(), repo)
	//log.Println(github.Stringify(new_issue))

	return nil
}

// initRepoWithReadMe
func initRepoWithReadMe(client *github.Client, ctx context.Context, repo *github.Repository, org string) error {
	// Create a simple little README.md file that we will use to initialize the main branch with
	fileContent := []byte("# " + repo.GetName() + gh_readme_contents)

	// Note: the file needs to be absent from the repository as you are not
	// specifying a SHA reference here.
	opts := &github.RepositoryContentFileOptions{
		Message:   github.String("First Commit (Initialize Branch " + repo.GetDefaultBranch() + ")"),
		Content:   fileContent,
		Branch:    github.String(repo.GetDefaultBranch()),
		Committer: &github.CommitAuthor{Name: github.String(gh_username_issue_mention), Email: github.String(gh_private_email)},
	}
	_, _, err := client.Repositories.CreateFile(ctx, org, repo.GetName(), "README.md", opts)
	if err != nil {
		log.Printf("Error Initializing Default Branch: %v\n", err)
		return err
	}
	// DEBUG
	log.Println("OH MY GOODNESS, did we really just init the main branch?!?!")

	return nil
}

// updateRepoReadMe
func updateRepoReadMe(client *github.Client, ctx context.Context, baseRef *github.Reference, repo *github.Repository, gh_organization_name string) error {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}
	entries = append(entries, &github.TreeEntry{Path: github.String(string("README.md")), Type: github.String("blob"),
		Content: github.String(string("# " + repo.GetName() + gh_readme_contents)),
		Mode:    github.String("100644")})
	var tree *github.Tree
	tree, _, err := client.Git.CreateTree(ctx, gh_organization_name, repo.GetName(), *baseRef.Object.SHA, entries)
	if err != nil {
		log.Printf("Error creating Tree Entry: %v\n", err)
		return err
	}

	// Get the parent commit to attach the commit to.
	parent, _, err := client.Repositories.GetCommit(ctx, gh_organization_name, repo.GetName(), *baseRef.Object.SHA, nil)
	if err != nil {
		log.Printf("Error retrieiving commit: %v\n", err)
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	commit_msg := "Setting up Branch Protection for " + repo.GetName()
	commit_login := gh_username_issue_mention
	commit_email := gh_private_email
	author := &github.CommitAuthor{Date: &date, Name: &commit_login, Email: &commit_email}
	commit := &github.Commit{Author: author, Message: &commit_msg, Tree: tree, Parents: []*github.Commit{parent.Commit}}
	newCommit, _, err := client.Git.CreateCommit(ctx, gh_organization_name, repo.GetName(), commit)
	if err != nil {
		log.Printf("Error creating commit: %v\n", err)
		return err
	}

	// Attach the commit to the desired branch.
	baseRef.Object.SHA = newCommit.SHA
	_, _, err = client.Git.UpdateRef(ctx, gh_organization_name, repo.GetName(), baseRef, false)
	if err != nil {
		log.Printf("Error Updating Reference: %v\n", err)
		return err
	}

	return nil
}

// printRateLimitUserInfo
func printRateLimitUserInfo(client *github.Client, ctx context.Context) {
	// get the GitHub user object
	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		log.Printf("[printRateLimitUserInfo] Error retrieving User object: %v\n", err)
		return
	}
	log.Printf("Effective User: %v\n", user.GetLogin())
	// Rate.Limit should most likely be 5000 when authorized.
	log.Printf("Rate: %#v\n", resp.Rate)
	log.Println("")
}

func autoLoadUserValues(client *github.Client, ctx context.Context) {
	// get the GitHub user object
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		log.Printf("[autoLoadUserValues] Error retrieving User object: %v\n", err)
		return
	}

	// Can we attempt to replace username and email values, if none where provided via the Environment?
	if user.GetLogin() != "" {
		if gh_username_issue_mention == "no-such-user" {
			gh_username_issue_mention = user.GetLogin()
		}
	}
	if user.GetEmail() != "" {
		if gh_private_email == "private@email.com" {
			gh_private_email = user.GetEmail()
		}
	}
}

// readValuesFromEnv
func readValuesFromEnv() {
	gh_webhook_secret_key = os.Getenv("GITHUB_WEBHOOK_SECRET")
	gh_personal_access_token = os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	gh_username_issue_mention = os.Getenv("GITHUB_COMMENT_MENTION")
	gh_private_email = os.Getenv("GITHUB_EMAIL_PRIVATE")
	tmp_int, err := strconv.Atoi(os.Getenv("GITHUB_REVIEW_MIN_COUNT"))
	if err != nil {
		// Default to 3 if we are unable to parse the Int from a string
		gh_code_review_min = 3
		// DEBUG
		//log.Println(err)
		log.Println("Unable to determine Int value from Environment vairable GITHUB_REVIEW_MIN_COUNT, defaulting to 3.")
	} else {
		gh_code_review_min = tmp_int
		// DEBUG
		//log.Printf("Read in %v as the desired minimum number of code reviewes", gh_code_review_min)
	}
}

// checkValuesFromEnv
func checkValuesFromEnv() {
	if gh_webhook_secret_key == "" {
		log.Fatal("Could not read GITHUB_WEBHOOK_SECRET value in from the environment")
	}
	if gh_personal_access_token == "" {
		log.Fatal("Could not read GITHUB_PERSONAL_ACCESS_TOKEN value in from the environment")
	}
	// Set reasonable defaults for the last 2 inputs, should they be nil
	if gh_username_issue_mention == "" {
		gh_username_issue_mention = "no-such-user"
	}
	if gh_private_email == "" {
		gh_private_email = "private@email.com"
	}
}

// reportLoadedConfigValues
func reportLoadedConfigValues() {
	log.Println("The process is running with the following configuration values:")
	log.Println("GITHUB_WEBHOOK_SECRET=*************** [REDACTED]")
	log.Println("GITHUB_PERSONAL_ACCESS_TOKEN=*************** [REDACTED]")
	log.Printf("GITHUB_COMMENT_MENTION=%v", gh_username_issue_mention)
	log.Printf("GITHUB_EMAIL_PRIVATE=%v", gh_private_email)
	log.Printf("GITHUB_REVIEW_MIN_COUNT=%v", gh_code_review_min)
	log.Println()
}

// FIN
