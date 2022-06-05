package resources

import "os"

const CREATE = "Create"
const APPLY = "Apply"
const DELETE = "Delete"
const LATEST = "latest"

const CILIUM_VESION = "cilium.io/v2"
const CILIUM_POLICY = "CiliumNetworkPolicy"
const CILIUM_CLUSTER_POLICY = "CiliumClusterwideNetworkPolicy"

const FORMAT_STRING = "%s:%s@tcp(%s:%s)/%s"

const UPDATE = "Update"

const SUCCESS = "success"

const SYSTEM_API_VERSION = "security.kubearmor.com/v1"
const KUBEARMORHOST_POLICY = "KubeArmorHostPolicy"
const KUBEARMOR_POLICY = "KubeArmorPolicy"
const GCP = "GCP"

var CLI_VERSION string = "2.0.0"

var CURRENT_DIR, GIT_DIR, USER_HOME, AD_DIR, GIT_REPO_PATH, GIT_POLICY_NAME string
var POLICY_COUNT int = 0
var LABEL_COUNT int = 0
var AUTOAPPLY bool
var POLICY_UPDATED *os.File

var REPO_PATH = "/accuknox-client-repo"

var RAND_CHARS = "abcdefghijklmnopqrstuvwxyz1234567890"

var WORKLOADS = []string{"mysql", "elastic", "postgres", "kafka", "ngnix", "percona", "cassandra", "wordpress", "django", "mongodb", "mariadb", "redis", "pinot"}

var COMPLIANCE = []string{"nist", "pci-dss", "stig"}

var USEDWORKLOAD []string

var USEDWORKLOADMAP = make(map[string]bool)
