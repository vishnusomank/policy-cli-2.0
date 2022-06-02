package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vishnusomank/policy-cli-2.0/pkg/discover_op"
	"github.com/vishnusomank/policy-cli-2.0/pkg/git_op"
)

func banner() {

	figure.NewFigure("Knox AutoPol", "standard", true).Print()
	fmt.Println()
	fmt.Println()
	fmt.Printf("[%s] Uses KubeConfig file to connect to cluster.\n", color.CyanString("WRN"))
	fmt.Printf("[%s] Creates files and folders in current directory.\n", color.CyanString("WRN"))

}

var current_dir, git_dir, user_home, keyword, tags, ad_dir, git_username, git_token, git_repo_url, git_branch_name, git_repo_path, git_policy_name, git_base_branch string
var policy_count int = 0
var label_count int = 0
var autoapply bool
var version string = "1.0.0"
var policy_updated *os.File

const repo_path = "/tmp/accuknox-client-repo"

func main() {

	// logging function generating following output
	// log.Info("") --> {"level":"info","msg":"","time":"2022-03-17T14:51:30+05:30"}
	// log.Warn("") --> {"level":"warning","msg":"","time":"2022-03-17T14:51:30+05:30"}
	// log.Error("") -- {"level":"error","msg":"","time":"2022-03-17T14:51:30+05:30"}

	log.SetFormatter(&log.JSONFormatter{})

	log_file, err := os.OpenFile("logs.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(log_file)

	// to get the current working directory
	current_dir, err = os.Getwd()
	if err != nil {
		log.Error(err)
	}

	// adding policy-template directory to current working directory
	git_dir = current_dir + "/policy-template"

	ad_dir = current_dir + "/ad-policy"

	log.Info("Current Working directory: " + current_dir)
	log.Info("Github clone directory: " + git_dir)

	myFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "git_username",
			Aliases:     []string{"git_user"},
			Usage:       "GitHub username",
			EnvVars:     []string{},
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			TakesFile:   false,
			Value:       "",
			DefaultText: "",
			Destination: new(string),
			HasBeenSet:  false,
		},
		&cli.StringFlag{
			Name:        "git_repo_url",
			Aliases:     []string{"git_url"},
			Usage:       "GitHub URL to push the updates",
			EnvVars:     []string{},
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			TakesFile:   false,
			Value:       "",
			DefaultText: "",
			Destination: new(string),
			HasBeenSet:  false,
		},
		&cli.StringFlag{
			Name:        "git_token",
			Aliases:     []string{"token"},
			Usage:       "GitHub token for authentication",
			EnvVars:     []string{},
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			TakesFile:   false,
			Value:       "",
			DefaultText: "",
			Destination: new(string),
			HasBeenSet:  false,
		},
		&cli.StringFlag{
			Name:        "git_branch_name",
			Aliases:     []string{"branch"},
			Usage:       "GitHub branch name for pushing updates",
			EnvVars:     []string{},
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			TakesFile:   false,
			Value:       "",
			DefaultText: "",
			Destination: new(string),
			HasBeenSet:  false,
		},
		&cli.StringFlag{
			Name:        "git_base_branch",
			Aliases:     []string{"basebranch"},
			Usage:       "GitHub base branch name for PR creation",
			EnvVars:     []string{},
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			TakesFile:   false,
			Value:       "",
			DefaultText: "",
			Destination: new(string),
			HasBeenSet:  false,
		},
		&cli.BoolFlag{
			Name:        "auto-apply",
			Aliases:     []string{"auto"},
			Usage:       "If true, modifed YAML will be applied to the cluster",
			EnvVars:     []string{},
			FilePath:    "",
			Required:    false,
			Hidden:      false,
			Value:       false,
			DefaultText: "",
			Destination: new(bool),
			HasBeenSet:  false,
		},
	}
	app := &cli.App{
		Name:      "knox-autopol",
		Usage:     "A simple CLI tool to automatically generate and apply policies or push to GitHub",
		Version:   version,
		UsageText: "knox-autopol [Flags]\nEg. knox-autopol --git_base_branch=deploy-branch --auto-apply=false --git_branch_name=temp-branch --git_token=gh_token123 --git_repo_url= https://github.com/testuser/demo.git --git_username=testuser",
		Flags:     myFlags,
		Action: func(c *cli.Context) error {
			git_username = c.String("git_username")
			git_token = c.String("git_token")
			git_repo_url = c.String("git_repo_url")
			git_branch_name = c.String("git_branch_name")
			autoapply = c.Bool("auto-apply")
			git_base_branch = c.String("git_base_branch")
			banner()
			fileUrl := "https://raw.githubusercontent.com/accuknox/tools/main/install.sh"
			discoverFileUrl := "https://raw.githubusercontent.com/accuknox/tools/main/get_discovered_yamls.sh"
			git_op.Git_Operation(git_dir)
			discover_op.Auto_Discover(fileUrl, discoverFileUrl, ad_dir, current_dir)
			git_op.Init_Git(git_username, git_token, git_repo_url, git_branch_name, git_base_branch, repo_path, ad_dir, current_dir, autoapply)
			return nil
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	err = app.Run(os.Args)
	if err != nil {
		log.Error(err)
	}

}
