package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cbrgm/githubevents/githubevents"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

// Globals
var gh_webhook_secret_key string = ""
var gh_personal_access_token string = ""
var gh_organization_name string = ""
var gh_username_issue_mention string = ""
var gh_private_email string = ""

// main
func main() {
	// Say hello
	fmt.Println("Example GitHub Webhook Handler is Starting Up Now...")

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
	printRateLimitInfo(client, ctx)

	// Pass the eventHandler to funcs that define callbacks to do the things
	setupProtectCallback(handle, client, ctx)
	setupIssueCallback(handle, client, ctx)

	// Setup a path to handle callbacks with
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		err := handle.HandleEventRequest(r)
		if err != nil {
			fmt.Println("error")
		}
	})

	// Finally let's start listening on port 8080 on every available interface
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// setupIssueCallback
func setupIssueCallback(handle *githubevents.EventHandler, client *github.Client, ctx context.Context) {

	// Registrer a callback to handle the event when a Repo is created in this Org
	handle.OnRepositoryEventCreated(
		func(deliveryID string, eventName string, event *github.RepositoryEvent) error {
			// DEBUG
			fmt.Println("Processing " + eventName + " event with ID " + deliveryID + "...")
			//fmt.Println(github.Stringify(event))
			// Let's create an Issue alerting us to what has been done
			issue_title := "New Repo Permisions Applied Successfully"
			issue_body := "The main branch was created, and it was protected so that only properly reviewed code can be commited to the main branch\n\nCC @dav1d-c"
			issue_repo := event.GetRepo().GetName()
			//issue_user := event.GetSender().GetLogin()
			//issue_org := event.GetOrg().GetName()
			i := &github.IssueRequest{Title: &issue_title, Body: &issue_body}
			new_issue, _, err := client.Issues.Create(ctx, gh_organization_name, issue_repo, i)
			if err != nil {
				log.Println(err)
			}
			fmt.Printf("Successfully created new issue: %v in repo: %v\n", new_issue.GetTitle(), new_issue.GetRepository().GetName())
			fmt.Println(github.Stringify(new_issue))
			return nil
		})

}

// setupProtectCallback
func setupProtectCallback(handle *githubevents.EventHandler, client *github.Client, ctx context.Context) {
	// Registrer a callback to handle the event when a Repo is created in this Org
	handle.OnRepositoryEventCreated(
		func(deliveryID string, eventName string, event *github.RepositoryEvent) error {
			// DEBUG
			fmt.Println("Processing " + eventName + " event with ID " + deliveryID + "...")
			//fmt.Println(github.Stringify(event))

			// repo, _, err := client.Repositories.Get(ctx, gh_organization_name, event.GetRepo().GetName())
			// if err != nil {
			// 	log.Fatal(err)
			// }
			// // DEBUG
			// //fmt.Printf("Request Response Code: %v %v\n", resp.Response.StatusCode, resp.Response.Status)
			// fmt.Printf("Master Branch of %v is %v.\n", repo.GetName(), repo.GetDefaultBranch())
			// //fmt.Println(github.Stringify(repo))

			var repo *github.Repository
			repo = event.GetRepo()

			// Create the first branch via first commit of README.md
			var baseRef *github.Reference
			baseRef, _, err := client.Git.GetRef(ctx, gh_organization_name, repo.GetName(), "refs/heads/"+repo.GetDefaultBranch())
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
				return err
			}

			fmt.Println("Created BaseRef")

			// Create a tree with what to commit.
			entries := []*github.TreeEntry{}
			entries = append(entries, &github.TreeEntry{Path: github.String(string("README.md")), Type: github.String("blob"),
				Content: github.String(string("# " + repo.GetName() + "\nYour Organization loves documentation, don't forget to update this file with specific information about this project!\n")),
				Mode:    github.String("100644")})
			var tree *github.Tree
			tree, _, err = client.Git.CreateTree(ctx, gh_organization_name, repo.GetName(), *baseRef.Object.SHA, entries)
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
				return err
			}

			fmt.Println("Created new Tree")

			// Get the parent commit to attach the commit to.
			parent, _, err := client.Repositories.GetCommit(ctx, gh_organization_name, repo.GetName(), *baseRef.Object.SHA, nil)
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
				return err
			}
			// This is not always populated, but is needed.
			parent.Commit.SHA = parent.SHA

			fmt.Println("Fetched Commit")

			// get the GitHub user object
			user, _, err := client.Users.Get(ctx, "")
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
				return err
			}

			fmt.Println("Fetched User")

			// Create the commit using the tree.
			date := time.Now()
			commit_msg := "Setting up Branch Protections for " + repo.GetName()
			commit_login := user.GetLogin()
			commit_email := user.GetEmail()
			// Has the user marked their email address as private?
			if commit_email == "" {
				commit_email = "private@email.com"
			}
			author := &github.CommitAuthor{Date: &date, Name: &commit_login, Email: &commit_email}
			commit := &github.Commit{Author: author, Message: &commit_msg, Tree: tree, Parents: []*github.Commit{parent.Commit}}
			newCommit, _, err := client.Git.CreateCommit(ctx, gh_organization_name, repo.GetName(), commit)
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
				return err
			}

			fmt.Println("Created Commit using Tree")

			// Attach the commit to the desired branch.
			baseRef.Object.SHA = newCommit.SHA
			_, _, err = client.Git.UpdateRef(ctx, gh_organization_name, repo.GetName(), baseRef, false)
			if err != nil {
				fmt.Printf("\nerror: %v\n", err)
				return err
			}

			fmt.Println("Attached Commit to Branch")

			// Setup Branch Protection via ProtectionRequest
			prr := &github.PullRequestReviewsEnforcementRequest{RequiredApprovingReviewCount: 2, RequireCodeOwnerReviews: true, DismissStaleReviews: false}
			pr := &github.ProtectionRequest{RequiredPullRequestReviews: prr, AllowForcePushes: github.Bool(false)}
			protections, _, err := client.Repositories.UpdateBranchProtection(ctx, gh_organization_name, repo.GetName(), repo.GetDefaultBranch(), pr)
			if err != nil {
				log.Println(err)
			}
			fmt.Println(github.Stringify(protections.GetRequiredPullRequestReviews()))
			return nil
		})
}

// printRateLimitInfo
func printRateLimitInfo(client *github.Client, ctx context.Context) {
	// get the GitHub user object
	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		fmt.Printf("\nerror: %v\n", err)
		return
	}
	fmt.Printf("Effective User: %v\n", user.GetLogin())
	// Rate.Limit should most likely be 5000 when authorized.
	log.Printf("Rate: %#v\n", resp.Rate)
	fmt.Println("")
}

// readValuesFromEnv
func readValuesFromEnv() {
	gh_webhook_secret_key = os.Getenv("GITHUB_WEBHOOK_SECRET")
	gh_personal_access_token = os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	gh_organization_name = os.Getenv("GITHUB_ORG_NAME")
	gh_username_issue_mention = os.Getenv("GITHUB_COMMENT_MENTION")
	gh_private_email = os.Getenv("GITHUB_EMAIL_PRIVATE")
}

// checkValuesFromEnv
func checkValuesFromEnv() {
	if gh_webhook_secret_key == "" {
		log.Fatal("Could not read GITHUB_WEBHOOK_SECRET value in from the environment")
	}
	if gh_personal_access_token == "" {
		log.Fatal("Could not read GITHUB_PERSONAL_ACCESS_TOKEN value in from the environment")
	}
	if gh_organization_name == "" {
		log.Fatal("Could not read GITHUB_ORG_NAME value in from the environment")
	}
	// Set reasonable defaults for the last 2 inputs, should they be nil
	if gh_username_issue_mention == "" {
		gh_username_issue_mention = "dav1d-c"
	}
	if gh_private_email == "" {
		gh_private_email = "private@email.com"
	}
}

// readFileContents
func readFileContents(fileName string) string {
	// take care of reading in "secrets", first for the webhook
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}
	// clean up the value we read in
	fileContents := strings.TrimSuffix(string(content), "\n")

	return fileContents
}
