package resources

import (
	"errors"
	"os"

	"github.com/google/go-github/github"
)

var version string = "1.0.0"
var policy_updated *os.File

var (
	client                  *github.Client
	NotMergableError        = errors.New("Not mergable")
	BranchNotFoundError     = errors.New("Branch not found")
	NonDeletableBranchError = errors.New("Branch cannot be deleted")
	PullReqNotFoundError    = errors.New("Pull request not found")
)

var current_dir, git_dir, user_home, keyword, tags, ad_dir string

var policy_count int = 0
var label_count int = 0
var autoapply bool

var git_username, git_token, git_repo_url, git_branch_name, git_repo_path, git_policy_name, git_base_branch string

const repo_path = "/tmp/accuknox-client-repo"
