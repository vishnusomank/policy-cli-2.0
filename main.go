package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vishnusomank/policy-cli-2.0/pkg/discover_op"
	"github.com/vishnusomank/policy-cli-2.0/pkg/git_op"
	"github.com/vishnusomank/policy-cli-2.0/resources"
)

func banner() {
	fmt.Println()
	fmt.Println()
	fmt.Printf(strings.TrimSuffix(figure.NewFigure("Auto Discovery", "slant", true).String(), "\n") + "   v2.0.0")
	fmt.Println()
	fmt.Println()

}

func removeResidues(repo_path string) {

	err := os.RemoveAll(repo_path)
	if err != nil {
		log.Fatal(err)
	}
}

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
	resources.CURRENT_DIR, err = os.Getwd()
	if err != nil {
		log.Error(err)
	}

	// adding policy-template directory to current working directory
	resources.GIT_DIR = resources.CURRENT_DIR + "/policy-template"

	resources.AD_DIR = resources.CURRENT_DIR + "/ad-policy"

	log.Info("Current Working directory: " + resources.CURRENT_DIR)
	log.Info("Github clone directory: " + resources.GIT_DIR)

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
		Name:      "Auto Discovery v2.0",
		Usage:     "A simple CLI tool to automatically generate and apply policies or push to GitHub",
		Version:   resources.CLI_VERSION,
		UsageText: "autodiscovery2.0 [Flags]\nEg. autodiscovery2.0 --git_base_branch=deploy-branch --auto-apply=false --git_branch_name=temp-branch --git_token=gh_token123 --git_repo_url= https://github.com/testuser/demo.git --git_username=testuser",
		Flags:     myFlags,
		Action: func(c *cli.Context) error {

			if c.String("git_username") == "" || c.String("git_token") == "" || c.String("git_repo_url") == "" || c.String("git_branch_name") == "" || c.String("git_base_branch") == "" {
				banner()
				fmt.Printf("[%s][%s] Parameters missing.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.CyanString("WARN"))
				fmt.Printf("[%s][%s] Please use autodisovery2.0 --help for help\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.CyanString("WARN"))

			} else {
				banner()
				fmt.Printf("[%s][%s] Uses KubeConfig file to connect to cluster.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.CyanString("WARN"))
				fmt.Printf("[%s][%s] Creates files and folders in current directory.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.CyanString("WARN"))
				fileUrl := "https://raw.githubusercontent.com/accuknox/samples/main/discover/install.sh"
				discoverFileUrl := "https://raw.githubusercontent.com/accuknox/samples/main/discover/get_discovered_yamls.sh"
				git_op.Git_Operation(resources.GIT_DIR)
				discover_op.Auto_Discover(fileUrl, discoverFileUrl, resources.AD_DIR, resources.CURRENT_DIR)
				resources.REPO_PATH = resources.CURRENT_DIR + resources.REPO_PATH
				log.Info("repo_path=" + resources.REPO_PATH)
				git_op.Init_Git(c.String("git_username"), c.String("git_token"), c.String("git_repo_url"), c.String("git_branch_name"), c.String("git_base_branch"), resources.REPO_PATH, resources.AD_DIR, resources.CURRENT_DIR, c.Bool("auto-apply"))
				removeResidues(resources.GIT_DIR)
				removeResidues(resources.AD_DIR)

				removeResidues("logs.log")
			}
			return nil
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	err = app.Run(os.Args)
	if err != nil {
		log.Error(err)
	}

}
