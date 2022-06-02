package discover_op

import (
	"fmt"
	"io"
	"os"

	"os/exec"

	nethttp "net/http"

	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func auto_discover() {
	fileUrl := "https://raw.githubusercontent.com/accuknox/tools/main/install.sh"
	err := DownloadFile("install.sh", fileUrl)
	if err != nil {
		log.Warn(err)
	}
	fmt.Println("Downloaded: " + fileUrl)
	command_query := "install.sh"
	cmd := exec.Command("/bin/bash", command_query)
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(stdout))
	log.Info(stdout)
	e := os.Remove(command_query)
	if e != nil {
		log.Fatal(e)
	}
	ad_dir = current_dir + "/ad-policy"
	if _, err := os.Stat(ad_dir); os.IsNotExist(err) {
		os.Mkdir(ad_dir, os.ModeDir|0755)
	}

	os.Chdir(ad_dir)
	log.Info("ad directory :" + ad_dir)

	fileUrl = "https://raw.githubusercontent.com/accuknox/tools/main/get_discovered_yamls.sh"
	err = DownloadFile("get_discovered_yamls.sh", fileUrl)
	if err != nil {
		log.Warn(err)
	}
	fmt.Println("Downloaded: " + fileUrl)
	command_query = "get_discovered_yamls.sh"
	cmd = exec.Command("/bin/bash", command_query)
	stdout, err = cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	e = os.Remove(command_query)
	if e != nil {
		log.Fatal(e)
	}
	fmt.Println(string(stdout))
	log.Info(stdout)
	os.Chdir(current_dir)

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
