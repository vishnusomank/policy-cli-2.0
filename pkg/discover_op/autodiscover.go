package discover_op

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"os/exec"

	nethttp "net/http"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/vishnusomank/policy-cli-2.0/resources"
)

func Auto_Discover(installFileUrl string, discoverFileUrl string, ad_dir string, current_dir string) {

	err := DownloadFile("install.sh", installFileUrl)
	if err != nil {
		log.Warn(err)
	}

	command_query := "install.sh"
	cmd := exec.Command("/bin/bash", command_query)
	stdout, err := cmd.Output()
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s][%s] Failed to install autodiscovery tools: "+err.Error(), color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERROR"))
		return
	}
	fmt.Printf("[%s][%s] Installed necessary components for auto-discovery of policies.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.GreenString("DONE"))
	log.Info(stdout)
	e := os.Remove(command_query)
	if e != nil {
		log.Fatal(e)
	}
	if _, err := os.Stat(ad_dir); os.IsNotExist(err) {
		os.Mkdir(ad_dir, os.ModeDir|0755)
	}

	os.Chdir(ad_dir)
	log.Info("ad directory :" + ad_dir)

	err = DownloadFile("get_discovered_yamls.sh", discoverFileUrl)
	if err != nil {
		log.Warn(err)
	}
	command_query = "get_discovered_yamls.sh"
	cmd = exec.Command("/bin/bash", command_query)
	stdout, err = cmd.Output()
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s][%s] Failed to get autodiscovered policies: "+err.Error(), color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERROR"))
		return
	}
	fmt.Printf("[%s][%s] Successfully discovered policies.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.GreenString("DONE"))

	e = os.Remove(command_query)
	if e != nil {
		log.Fatal(e)
	}
	log.Info(stdout)

	split_cilium_file(ad_dir+"/cilium_policies.yaml", ad_dir)
	e = os.Remove(ad_dir + "/cilium_policies.yaml")
	if e != nil {
		log.Fatal(e)
	}

	os.Chdir(current_dir)

}

func shortID(length int) string {
	ll := len(resources.RAND_CHARS)
	b := make([]byte, length)
	rand.Read(b) // generates len(b) random bytes
	for i := 0; i < length; i++ {
		b[i] = resources.RAND_CHARS[int(b[i])%ll]
	}
	return string(b)
}

func Split(r rune) bool {
	return r == ':' || r == '\n'
}

func split_cilium_file(policy_name string, repo_path string) {

	file, err := os.Open(policy_name)
	if err != nil {
		fmt.Printf("[%s][%s] Oops! Failed to open "+policy_name+" file. Please try again.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))

	}
	scanner := bufio.NewScanner(file)
	var text []string
	for scanner.Scan() {

		if strings.Contains(string(scanner.Text()), "---") {

			git_policy_name := strings.Replace("cilium-policy-"+shortID(7), "\"", "", -1)
			git_policy_name = strings.Replace(git_policy_name, " ", "", -1)

			policy_updated, err := os.OpenFile(repo_path+"/"+git_policy_name+".yaml", os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Printf("[%s][%s] Oops! Failed to open "+git_policy_name+".yaml file. Please try again.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))

				return
			}

			for _, each_ln := range text {
				if !strings.Contains(each_ln, "---") {
					_, err = fmt.Fprintln(policy_updated, each_ln)
				}
				if err != nil {
					fmt.Printf("[%s][%s] Oops! Failed unable to write to "+git_policy_name+".yaml file. Please try again.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))

				}
			}
			text = text[:0]

		}

		text = append(text, scanner.Text())

	}
	file.Close()

}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := nethttp.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
