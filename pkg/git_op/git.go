package git_op

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/fatih/color"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Git Functions

func Init_Git(username string, token string, repo_url string, branch_name string, git_base_branch string, repo_path string) {

	s := strings.Split(repo_url, "/")
	var repoName string

	for i := 0; i < len(s); i++ {
		if strings.Contains(s[i], ".git") {
			repoName = strings.Split(s[i], ".")[0]
		}
	}

	r := GitClone(username, token, repo_url, repo_path, git_base_branch)

	createBranch(r, username, token, branch_name)
	fmt.Printf("%v branch is created\n", branch_name)

	pushToGithub(r, username, token)
	fmt.Printf("Pushed to the github repo %v\n", repo_url)

	createPRToGit(token, branch_name, username, repoName)

	removeLocalRepo()
}

func GitClone(username string, token string, repo_url string, repo_path string, git_base_branch string) *git.Repository {

	if _, err := os.Stat(repo_path); os.IsNotExist(err) {
		os.Mkdir(repo_path, 0755)
	}

	auth := &http.BasicAuth{
		Username: username,
		Password: token,
	}

	r, _ := git.PlainClone(repo_path, false, &git.CloneOptions{
		URL:           repo_url,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", git_base_branch)),
		Auth:          auth,
	})

	return r
}

func createBranch(r *git.Repository, username string, token string, branch_name string) {

	w, _ := r.Worktree()

	err := w.Checkout(&git.CheckoutOptions{
		Create: true,
		Force:  false,
		Branch: plumbing.ReferenceName(git_base_branch),
	})

	checkError(err, "create branch: checkout "+git_base_branch)

	branchName := plumbing.ReferenceName("refs/heads/" + branch_name)

	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Force:  false,
		Branch: branchName,
	})

	checkError(err, "create branch: checkout "+branch_name)

	if _, err := os.Stat(git_repo_path); os.IsNotExist(err) {
		os.Mkdir(git_repo_path, 0755)
	}
	k8s_labels(autoapply)

	CopyDir(git_repo_path, repo_path)
	w.Add(".")

	author := &object.Signature{
		Name:  "KnoxAutoPol",
		Email: "vishnu@accuknox.com",
		When:  time.Now(),
	}

	w.Commit("Commit from KnoxAutoPol CLI", &git.CommitOptions{Author: author})
}

func pushToGithub(r *git.Repository, username, password string) {

	auth := &http.BasicAuth{
		Username: username,
		Password: password,
	}

	err := r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      true,
	})

	checkError(err, "pushtogit error")
}

func newClient(token string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)

}

func createPRToGit(token string, branchName string, username string, repoName string) {

	newPR := &github.NewPullRequest{
		Title:               github.String("PR from KnoxAutoPol CLI"),
		Head:                github.String(branchName),
		Base:                github.String(git_base_branch),
		Body:                github.String("This is an automated PR created by KnoxAutoPol CLI"),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := client.PullRequests.Create(context.Background(), username, repoName, newPR)
	if err != nil {
		fmt.Println("There is a Git Error " + err.Error())
		return
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	s := strings.Split(pr.GetHTMLURL(), "/")
	mergePullRequest(username, repoName, s[len(s)-1], token)

}

func stringToInt(number string) int {
	intVal, err := strconv.Atoi(number)
	if err != nil {
		fmt.Printf("[%s] Oops! String to integer conversion failed\n", color.RedString("ERR"))
		log.Warn(err)
	}
	return intVal
}

func mergePullRequest(owner, repo, number, token string) error {
	fmt.Printf("Attempting to merge PR #%s on %s/%s...\n", number, owner, repo)

	commitMsg := "Commit from Accuknox GitOps CLI"
	_, _, mergeErr := client.PullRequests.Merge(
		context.Background(),
		owner,
		repo,
		stringToInt(number),
		commitMsg,
		&github.PullRequestOptions{},
	)

	if mergeErr != nil {
		fmt.Println("Received an error!", mergeErr)
	} else {
		fmt.Printf("Successfully merged PR #%s on %s/%s...\n", number, owner, repo)

	}

	return nil
}

func removeLocalRepo() {

	err := os.RemoveAll(repo_path)
	checkError(err, "removelocalrepo function")
}

func checkError(err error, data string) {
	if err != nil {
		fmt.Printf("[%s] Oops! Error from \n"+data, color.RedString("ERR"))
		log.Warn(err)
	}
}

// END Git Functions

// function to clone policy-template repo to current working directory
func git_clone_policy_templates() {

	log.Info("Started Cloning policy-template repository")
	fmt.Printf("[%s][%s] Cloning policy-template repository\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("INIT"))
	r, err := git.PlainClone(git_dir, false, &git.CloneOptions{
		URL: "https://github.com/kubearmor/policy-templates",
	})

	if err != nil {
		log.Error(err)
	}
	log.Info(r)
	fmt.Printf("[%s][%s] Cloned policy-template repository\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.GreenString("DONE"))

}

// function to pull latest changes into policy-template folder
func git_pull_policy_templates() {

	log.Info("Started Pulling into policy-template repository")
	fmt.Printf("[%s][%s] Fetching updates from policy-template repository\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("INIT"))
	r, err := git.PlainOpen(git_dir)
	if err != nil {
		log.Error(err)
	}

	w, err := r.Worktree()
	if err != nil {
		log.Error(err)
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		log.Debug(err)
	}

	fmt.Printf("[%s][%s] Fetched updates from policy-template repository\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.GreenString("DONE"))

}

// Function to Create connection to kubernetes cluster

func git_operation() {

	//check if the policy-template directory exist
	// if exist pull down the latest changes
	// else clone the policy-templates repo
	if _, err := os.Stat(git_dir); !os.IsNotExist(err) {

		git_pull()

	} else {

		git_clone()

	}

}
func delete_all() {
	logs_path := current_dir + "/logs.log"
	err := os.RemoveAll(logs_path)
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s] Unable to remove file %s\n", color.RedString("ERR"), logs_path)
	}
	err = os.RemoveAll(git_dir)
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s] Unable to remove folder %s\n", color.RedString("ERR"), git_dir)
	}
}
