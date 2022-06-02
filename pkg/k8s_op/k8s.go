package k8s_op

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/vishnusomank/policy-cli-2.0/resources"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	kubernetes "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/restmapper"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

var workloads = []string{"mysql", "elastic", "postgres", "kafka", "ngnix", "percona", "cassandra", "wordpress", "django", "mongodb", "mariadb", "redis", "pinot"}

var autoapply bool
var label_count int
var repo_path_git, git_policy_name string

func connectToK8s() *kubernetes.Clientset {

	log.Info("Trying to establish connection to k8s")
	home, exists := os.LookupEnv("HOME")
	if !exists {
		home = "/root"
	}

	configPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		log.Error("failed to create K8s config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("Failed to create K8s clientset")
		fmt.Printf("[%s] Failed to connect to Kubernetes cluster. Please try again.\n", color.RedString("ERR"))
	}

	return clientset
}

// Function to create strings from key-value pairs
func createKeyValuePairs(m map[string]string, disp bool, namespace string) string {

	log.Info("Started map to string conversion on labels")

	b := new(bytes.Buffer)
	if disp == true {
		for key, value := range m {
			for i := 0; i < len(workloads); i++ {
				if strings.Contains(key, workloads[i]) || strings.Contains(value, workloads[i]) {
					fmt.Fprintf(b, "%s: %s\n\t\t\t\t\t", key, value)
				}
			}
		}

	} else {
		for key, value := range m {
			for i := 0; i < len(workloads); i++ {
				if strings.Contains(key, workloads[i]) || strings.Contains(value, workloads[i]) {
					fmt.Fprintf(b, "%s: %s\n\t\t\t\t\t", key, value)
				}
			}
		}
	}
	return b.String()
}

// FUnction to search files with .yaml extension under policy-template folder
func policy_search(namespace string, labels string, search string, git_dir string) {

	log.Info("Started searching for files with .yaml extension under policy-template folder")

	err := filepath.Walk(git_dir, func(path string, info os.FileInfo, err error) error {
		log.Info("git directory accessed : " + git_dir)
		if err != nil {
			log.Error(err)
			return err
		}
		if strings.Contains(path, ".yaml") {
			label_count = 0
			policy_read(path, namespace, labels, search)
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s] Oops! No files found with .yaml extension. Please try again later.\n", color.RedString("ERR"))
	}
}

func policy_read(policy_name string, namespace string, labels string, search string) {

	log.Info("Started Policy search : " + policy_name + " with labels '" + labels + "' and search '" + search + "'")

	content, err := os.ReadFile(policy_name)
	if err != nil {
		log.Error(err)
	}

	if strings.Contains(string(content), search) {
		repo_path := repo_path_git + "/" + strings.ToLower(search)

		if _, err := os.Stat(repo_path); os.IsNotExist(err) {
			os.Mkdir(repo_path, 0755)
		}

		file, err := os.Open(policy_name)
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		var text []string
		text = append(text, "---")
		for scanner.Scan() {
			if strings.Contains(string(scanner.Text()), "name:") {
				policy_val := strings.FieldsFunc(string(scanner.Text()), Split)
				git_policy_name = strings.Replace(policy_val[1]+"-"+shortID(7), "\"", "", -1)

				text = append(text, string(scanner.Text())+"-"+shortID(7))
				for scanner.Scan() {
					if strings.Contains(string(scanner.Text()), "namespace:") {
						break
					}
				}
			}
			if strings.Contains(string(scanner.Text()), "namespace:") {
				text = append(text, "  namespace: "+namespace)
				for scanner.Scan() {
					if strings.Contains(string(scanner.Text()), "spec:") {
						break
					}
				}

			} else if strings.Contains(string(scanner.Text()), "matchLabels:") && label_count == 0 {
				text = append(text, "    matchLabels:\n      "+labels)
				label_count = 1
				for scanner.Scan() {
					if strings.Contains(string(scanner.Text()), "file:") || strings.Contains(string(scanner.Text()), "process:") || strings.Contains(string(scanner.Text()), "network:") || strings.Contains(string(scanner.Text()), "capabilities:") || strings.Contains(string(scanner.Text()), "ingress:") || strings.Contains(string(scanner.Text()), "egress:") {
						break
					}
				}
			}
			text = append(text, scanner.Text())
		}

		file.Close()

		policy_updated, err := os.OpenFile(repo_path+git_policy_name+".yaml", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Error(err)
			return
		}

		for _, each_ln := range text {
			_, err = fmt.Fprintln(policy_updated, each_ln)
			if err != nil {
				log.Error(err)
			}
		}

	}
	/*

		if strings.Contains(string(content), keyword) && keyword != "" && tags == "" {

			file, err := os.Open(policy_name)
			if err != nil {
				log.Fatal(err)
			}
			scanner := bufio.NewScanner(file)
			var text []string
			text = append(text, "---")
			for scanner.Scan() {
				if strings.Contains(string(scanner.Text()), "namespace:") {

					text = append(text, "  namespace: "+namespace)
					for scanner.Scan() {
						if strings.Contains(string(scanner.Text()), "spec:") {
							break
						}
					}

				} else if strings.Contains(string(scanner.Text()), "matchLabels:") && label_count == 0 {
					text = append(text, "    matchLabels:\n      "+labels)
					label_count = 1
					for scanner.Scan() {
						if strings.Contains(string(scanner.Text()), "file:") || strings.Contains(string(scanner.Text()), "process:") || strings.Contains(string(scanner.Text()), "network:") || strings.Contains(string(scanner.Text()), "capabilities:") || strings.Contains(string(scanner.Text()), "ingress") || strings.Contains(string(scanner.Text()), "egress") {
							break
						}
					}
				}
				text = append(text, scanner.Text())
			}

			file.Close()

			policy_updated, err = os.OpenFile(git_repo_path+git_policy_name+".yaml", os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Error(err)
				return
			}

			for _, each_ln := range text {
				_, err = fmt.Fprintln(policy_updated, each_ln)
				if err != nil {
					log.Error(err)
				}
			}
		}
	*/

}

var chars = "abcdefghijklmnopqrstuvwxyz1234567890-"

func shortID(length int) string {
	ll := len(chars)
	b := make([]byte, length)
	rand.Read(b) // generates len(b) random bytes
	for i := 0; i < length; i++ {
		b[i] = chars[int(b[i])%ll]
	}
	return string(b)
}

func k8s_apply(path string) {

	if autoapply == true {
		log.Info("auto-apply = " + strconv.FormatBool(autoapply))

		log.Info("Trying to establish connection to k8s")
		home, exists := os.LookupEnv("HOME")
		if !exists {
			home = "/root"
		}

		configPath := filepath.Join(home, ".kube", "config")

		config, err := clientcmd.BuildConfigFromFlags("", configPath)
		if err != nil {
			log.Error("failed to create K8s config")
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Error("Failed to create K8s clientset")
			fmt.Printf("[%s] Failed to create Kubernetes Clientset.\n", color.RedString("ERR"))

		}
		log.Info(clientset)

		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.Error(err)
		}

		dd, err := dynamic.NewForConfig(config)
		if err != nil {
			log.Error(err)
		}
		discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
		if err != nil {
			fmt.Printf("[%s] Oops! discovery client creation failed\n", color.RedString("ERR"))
			log.Warn(err.Error())
		}
		log.Info(discoveryClient)

		var namespaceNames string
		namespaceNames = ""
		//Fetch the NamespaceNames using NamespaceID

		Decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 98)
		for {
			var rawObject runtime.RawExtension
			if err = Decoder.Decode(&rawObject); err != nil {
				log.Warn("decoding not possible because " + err.Error())

			}
			//decode yaml into unstructured.Unstructured and get Group version kind
			object, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObject.Raw, nil, nil)
			unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
			if err != nil {
				log.Warn("Error in Unstructuredmap because " + err.Error())
			}

			unstructuredObject := &unstructured.Unstructured{Object: unstructuredMap}
			grs, err := restmapper.GetAPIGroupResources(clientset.DiscoveryClient)
			if err != nil {
				log.Warn("Unable to Get API Group resource because " + err.Error())
			}

			//Get Group version resource using Group version kind
			rMapper := restmapper.NewDiscoveryRESTMapper(grs)
			log.Info("Group  Kind :  " + fmt.Sprint(gvk.GroupKind()))
			log.Info("Version :  " + fmt.Sprint(gvk.Version))
			mapping, err := rMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				log.Warn("unexpected error getting mapping for Group version resource " + err.Error())
			}

			// Obtain REST interface for the Group version resource and checking for namespace or cluster wide resource
			var dri dynamic.ResourceInterface

			if gvk.Kind == resources.KUBEARMORHOST_POLICY {
				if namespaceNames != "" {
					dri = dd.Resource(mapping.Resource)

				} else {

					dri = dd.Resource(mapping.Resource)
				}
			} else if gvk.Kind == resources.CILIUM_KIND_NODE_LABEL {
				if namespaceNames != "" {

					dri = dd.Resource(mapping.Resource)
				}
			} else {

				dri = dd.Resource(mapping.Resource).Namespace(unstructuredObject.GetNamespace())
			}

			//To Create or update the policy get the name of the policy which is to be applied and check exist or not

			getObj, err := dri.Get(context.TODO(), unstructuredObject.GetName(), v1.GetOptions{})

			// if policy is not applied or found, create the policy in cluster
			if err != nil && errors.IsNotFound(err) {
				_, err = dri.Create(context.Background(), unstructuredObject, v1.CreateOptions{})
				if err != nil {
					log.Warn("Policy Creation is failed " + err.Error())
				}
				log.Info("Policy Apply Successfully")
			} else {
				//Update the policy in cluster
				unstructuredObject.SetResourceVersion(getObj.GetResourceVersion())
				_, err = dri.Update(context.TODO(), unstructuredObject, v1.UpdateOptions{})
				if err != nil {
					log.Warn("Policy Updation is failed " + err.Error())
				}
				log.Info("Policy Updated Successfully")
			}

		}

	} else {
		log.Warn("auto-apply = " + strconv.FormatBool(autoapply))
	}

}

func K8s_Labels(flag bool, git_repo_path string, repo_path string) {

	autoapply = flag
	repo_path_git = repo_path

	clientset := connectToK8s()
	// access the API to list pods
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		log.Error(err)
	}
	var temp []string
	var count int = 0
	for _, pod := range pods.Items {
		if createKeyValuePairs(pod.GetLabels(), true, pod.GetNamespace()) != "" {
			temp = append(temp, createKeyValuePairs(pod.GetLabels(), true, pod.GetNamespace()))
			count++
		}
	}
	if count == 0 {
		fmt.Printf("[%s] No Predefined workloads found in the cluster. Gracefully exiting program.\n", color.RedString("ERR"))
		os.Exit(1)
	} else {
		fmt.Printf("[%s][%s] Found %d Labels\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("Label Count"), count)
	}
	for _, item := range temp {
		val := strings.TrimSuffix(item, "\n\t\t\t\t\t")

		fmt.Printf("[%s][%s]    %s\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("Label Details"), val)
		log.Info("Label values: ", item)
	}
	for _, pod := range pods.Items {
		labels := createKeyValuePairs(pod.GetLabels(), false, pod.GetNamespace())
		labels = strings.TrimSuffix(labels, "\n      ")
		searchVal := strings.FieldsFunc(labels, Split)
		if labels != "" {
			//	fmt.Printf("[%s][%s] Pod: %s || Labels: %s || Namespace: %s\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("Label Details"), pod.GetName(), labels, pod.GetNamespace())
			for i := 0; i < len(searchVal); i++ {
				i++
				policy_search(pod.GetNamespace(), labels, searchVal[i], git_repo_path)
			}
		}
	}
	if flag == false {
		log.Info("Received flag value false")

		fmt.Printf("[%s][%s] Halting execution because auto-apply is not enabled\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.CyanString("WRN"))
	} else {
		fmt.Printf("[%s][%s] Started applying policies\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("INIT"))
	}
	err = filepath.Walk(git_repo_path, func(path string, info os.FileInfo, err error) error {
		log.Info("git directory accessed : " + git_repo_path)
		if err != nil {
			log.Error(err)
			return err
		}
		if strings.Contains(path, ".yaml") {
			k8s_apply(path)
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s] Oops! No files found with .yaml extension. Please try again later.\n", color.RedString("ERR"))
	}

}

func Split(r rune) bool {
	return r == ':' || r == '\n'
}
