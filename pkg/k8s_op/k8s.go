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

var autoapply bool
var label_count int
var repo_path_git, git_policy_name string

func checkWorkloadIsMapped(workloadName string) {

	if resources.USEDWORKLOADMAP[workloadName] {
		return // Already in the map
	}
	resources.USEDWORKLOAD = append(resources.USEDWORKLOAD, workloadName)
	resources.USEDWORKLOADMAP[workloadName] = true

}

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
		fmt.Printf("[%s][%s] Failed to connect to Kubernetes cluster. Please try again.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))
	}

	return clientset
}

// Function to create strings from key-value pairs
func createKeyValuePairs(m map[string]string, disp bool, namespace string) string {

	log.Info("Started map to string conversion on labels")

	b := new(bytes.Buffer)
	if disp == true {
		log.Info("Started map to string conversion on labels with display value= " + strconv.FormatBool(disp))
		for key, value := range m {
			for i := 0; i < len(resources.WORKLOADS); i++ {
				if strings.Contains(key, resources.WORKLOADS[i]) || strings.Contains(value, resources.WORKLOADS[i]) {
					checkWorkloadIsMapped(resources.WORKLOADS[i])
					fmt.Fprintf(b, "%s: %s\n\t\t\t\t\t", key, value)
				}
			}
		}

	} else {
		log.Info("Started map to string conversion on labels with display value= " + strconv.FormatBool(disp))
		for key, value := range m {
			for i := 0; i < len(resources.WORKLOADS); i++ {
				if strings.Contains(key, resources.WORKLOADS[i]) || strings.Contains(value, resources.WORKLOADS[i]) {
					checkWorkloadIsMapped(resources.WORKLOADS[i])
					fmt.Fprintf(b, "%s: %s\n      ", key, value)
				}
			}
		}
	}
	return b.String()
}

func policy_read_ad(policy_name string, namespace string, labels string, search string) {

	var nameCount = 0
	var namePrefix string

	log.Info("Started AD Policy search : " + policy_name + " with labels '" + labels + "' and search '" + search + "'")

	content, err := os.ReadFile(policy_name)
	if err != nil {
		log.Error(err)
	}

	if strings.Contains(string(content), search) {

		log.Info("Found AD policy " + policy_name + " with search '" + search + "'")

		var repo_path string

		for i := 0; i < len(resources.USEDWORKLOAD); i++ {
			if strings.Contains(search, resources.USEDWORKLOAD[i]) {
				repo_path = repo_path_git + "/" + strings.ToLower(resources.USEDWORKLOAD[i]) + "/"
				break
			}
		}

		if _, err := os.Stat(repo_path); os.IsNotExist(err) {
			os.Mkdir(repo_path, 0755)
		}
		repo_path = repo_path + "/ad-policy"
		if _, err := os.Stat(repo_path); os.IsNotExist(err) {
			os.Mkdir(repo_path, 0755)
		}

		file, err := os.Open(policy_name)
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		var text []string
		for scanner.Scan() {
			if strings.Contains(string(scanner.Text()), resources.CILIUM_CLUSTER_POLICY) {
				namePrefix = "ccnp-"
			} else if strings.Contains(string(scanner.Text()), resources.CILIUM_POLICY) {
				namePrefix = "cnp-"
			} else if strings.Contains(string(scanner.Text()), resources.KUBEARMOR_POLICY) {
				namePrefix = "ksp-"
			} else if strings.Contains(string(scanner.Text()), resources.KUBEARMORHOST_POLICY) {
				namePrefix = "hsp-"
			}
			if strings.Contains(string(scanner.Text()), "name:") && nameCount == 0 {
				policy_val := strings.FieldsFunc(string(scanner.Text()), Split)
				policy_val[1] = strings.Replace(policy_val[1], " ", "", -1)
				git_policy_name = strings.Replace(namePrefix+policy_val[1], "\"", "", -1)
				git_policy_name = strings.Replace(git_policy_name, "block", "audit", -1)
				git_policy_name = strings.Replace(git_policy_name, "restrict", "audit", -1)
				git_policy_name = strings.Replace(git_policy_name, "allow", "audit", -1)
				nameCount = 1
				text = append(text, "  name: "+git_policy_name)
			} else {
				if strings.Contains(string(scanner.Text()), "action:") {
					actionVal := strings.FieldsFunc(string(scanner.Text()), Split)
					text = append(text, actionVal[0]+" : Audit")
					scanner.Scan()
					if strings.Contains(string(scanner.Text()), "Audit") || strings.Contains(string(scanner.Text()), "Block") || strings.Contains(string(scanner.Text()), "Allow") {
						scanner.Scan()
					}

				}
				text = append(text, scanner.Text())
			}
		}

		file.Close()
		policy_updated, err := os.OpenFile(repo_path+"/"+git_policy_name+".yaml", os.O_CREATE|os.O_WRONLY, 0644)
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

}

// FUnction to search files with .yaml extension under policy-template folder
func policy_search(namespace string, labels string, search string, git_dir string, ad_dir string) {

	log.Info("Started searching for files with .yaml extension under " + git_dir)

	err := filepath.Walk(git_dir, func(path string, info os.FileInfo, err error) error {
		log.Info("git directory accessed : " + git_dir)
		if err != nil {
			log.Error(err)
			return err
		}
		if strings.Contains(path, ".yaml") {
			content, err := os.ReadFile(path)
			if err != nil {
				log.Error(err)
			}

			if strings.Contains(string(content), search) {
				label_count = 0
				policy_read_templates(path, namespace, labels, search)
			}
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s][%s] Oops! No files found with .yaml extension. Please try again later.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))
	}

	log.Info("Started searching for files with .yaml extension under " + ad_dir)

	err = filepath.Walk(ad_dir, func(path string, info os.FileInfo, err error) error {
		log.Info("AD directory accessed : " + ad_dir)
		if err != nil {
			log.Error(err)
			return err
		}
		if strings.Contains(path, ".yaml") {
			label_count = 0
			policy_read_ad(path, namespace, labels, search)
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		fmt.Printf("[%s][%s] Oops! No files found with .yaml extension. Please try again later.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))
	}
}

func policy_read_templates(policy_name string, namespace string, labels string, search string) {

	log.Info("Started Template Policy search : " + policy_name + " with labels '" + labels + "' and search '" + search + "'")

	content, err := os.ReadFile(policy_name)
	if err != nil {
		log.Error(err)
	}

	if strings.Contains(string(content), search) {
		var repo_path, temp_path string

		for i := 0; i < len(resources.USEDWORKLOAD); i++ {
			if strings.Contains(search, resources.USEDWORKLOAD[i]) {
				repo_path = repo_path_git + "/" + strings.ToLower(resources.USEDWORKLOAD[i])
				break
			}
		}
		for i := 0; i < len(resources.COMPLIANCE); i++ {
			if strings.Contains(string(content), resources.COMPLIANCE[i]) {
				temp_path = "/compliance"
				break
			}
		}

		if _, err := os.Stat(repo_path); os.IsNotExist(err) {
			os.Mkdir(repo_path, 0755)
		}
		if _, err := os.Stat(repo_path + temp_path); os.IsNotExist(err) {
			os.Mkdir(repo_path+temp_path, 0755)
		}

		file, err := os.Open(policy_name)
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		var text []string
		for scanner.Scan() {
			if strings.Contains(string(scanner.Text()), "name:") {
				policy_val := strings.FieldsFunc(string(scanner.Text()), Split)
				policy_val[1] = strings.Replace(policy_val[1], " ", "", -1)
				git_policy_name = strings.Replace(policy_val[1], "\"", "", -1)
				git_policy_name = strings.Replace(git_policy_name, "block", "audit", -1)
				git_policy_name = strings.Replace(git_policy_name, "restrict", "audit", -1)
				git_policy_name = strings.Replace(git_policy_name, "allow", "audit", -1)

				text = append(text, "  name: "+git_policy_name)
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

			}
			if strings.Contains(string(scanner.Text()), "matchLabels:") && label_count == 0 {
				text = append(text, "    matchLabels:\n      "+labels)
				label_count = 1
				for scanner.Scan() {
					if strings.Contains(string(scanner.Text()), "file:") || strings.Contains(string(scanner.Text()), "process:") || strings.Contains(string(scanner.Text()), "network:") || strings.Contains(string(scanner.Text()), "capabilities:") || strings.Contains(string(scanner.Text()), "ingress:") || strings.Contains(string(scanner.Text()), "egress:") {
						break
					}
				}
			}
			if strings.Contains(string(scanner.Text()), "action:") {
				actionVal := strings.FieldsFunc(string(scanner.Text()), Split)
				text = append(text, actionVal[0]+" : Audit")
				scanner.Scan()
				if strings.Contains(string(scanner.Text()), "Audit") || strings.Contains(string(scanner.Text()), "Block") || strings.Contains(string(scanner.Text()), "Allow") {
					scanner.Scan()
				}

			}
			text = append(text, scanner.Text())

		}

		file.Close()
		if repo_path != "" && temp_path != "" {
			repo_path = repo_path + temp_path + "/"
		} else {
			if _, err := os.Stat(repo_path + "/hardening"); os.IsNotExist(err) {
				os.Mkdir(repo_path+"/hardening", 0755)
			}
			repo_path = repo_path + "/hardening/"
		}

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
		policy_updated.Close()

	}

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
			fmt.Printf("[%s][%s] Failed to create Kubernetes Clientset.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))

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
			fmt.Printf("[%s][%s] Oops! discovery client creation failed\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))
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
			} else if gvk.Kind == resources.CILIUM_CLUSTER_POLICY {
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

func K8s_Labels(flag bool, git_repo_path string, repo_path string, ad_dir string) {

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
		fmt.Printf("[%s][%s] No Predefined workloads found in the cluster. Gracefully exiting program.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))
		os.Exit(1)
	} else {
		fmt.Printf("[%s][%s] Found %d Label(s)\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("Label Count"), count)
	}
	for _, item := range temp {
		val := strings.Trim(item, "\n\t\t\t")

		fmt.Printf("[%s][%s]    %s\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("Label Details"), val)
		log.Info("Label values: ", item)
	}
	for _, pod := range pods.Items {
		labels := createKeyValuePairs(pod.GetLabels(), false, pod.GetNamespace())
		labels = strings.TrimSuffix(labels, "\n      ")
		log.Info("disp=false Label values: ", labels)
		searchVal := strings.FieldsFunc(labels, Split)
		log.Info("disp=false searchval values: ", searchVal)
		if labels != "" {
			//	fmt.Printf("[%s][%s] Pod: %s || Labels: %s || Namespace: %s\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("Label Details"), pod.GetName(), labels, pod.GetNamespace())
			/*
				for i := 0; i < len(searchVal); i++ {
					log.Info("disp=false parameter i's value : ", i)
					i++
					for j := 0; j < len(resources.USEDWORKLOAD); j++ {
						log.Info("disp=false parameter j's value : ", j)
						searchVal[i] = strings.Replace(searchVal[i], " ", "", -1)
						if strings.Contains(searchVal[i], resources.USEDWORKLOAD[j]) {
							policy_search(pod.GetNamespace(), labels, searchVal[i], git_repo_path, ad_dir)
						}
					}
				}
			*/
			for j := 0; j < len(resources.USEDWORKLOAD); j++ {
				if strings.Contains(labels, resources.USEDWORKLOAD[j]) {
					policy_search(pod.GetNamespace(), labels, resources.USEDWORKLOAD[j], git_repo_path, ad_dir)
				}

			}

		}
	}
	if flag == false {
		log.Info("Received flag value false")

		fmt.Printf("[%s][%s] Halting execution because auto-apply is not enabled\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.CyanString("WRN"))
	} else {
		fmt.Printf("[%s][%s] Started applying policies\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.BlueString("INIT"))

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
			fmt.Printf("[%s][%s] Oops! No files found with .yaml extension. Please try again later.\n", color.BlueString(time.Now().Format("01-02-2006 15:04:05")), color.RedString("ERR"))
		}
	}

}

func Split(r rune) bool {
	return r == ':' || r == '\n'
}
