package registry

import (
	"regexp"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/explain"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

var (
	portRegex             = regexp.MustCompile(`\bport\s+(\d{1,5})\b`)
	hostRegex             = regexp.MustCompile(`(?:ping|reachability|connect(?:ion)?(?: to)?)\s+([a-zA-Z0-9_.:-]+)`)
	serviceRegex          = regexp.MustCompile(`\b([a-zA-Z0-9_.@-]+)\s+service\b`)
	findLargeRegex        = regexp.MustCompile(`files?\s+larger\s+than\s+([0-9]+[kmgtp]?)\s+in\s+(\S+)`)
	pathRegex             = regexp.MustCompile(`(?:disk usage|size)\s+(?:of\s+)?(\S+)`)
	fileRegex             = regexp.MustCompile(`(?:tail|show)\s+(?:the\s+)?(?:log|file)\s+(\S+)`)
	serviceLogsRegex      = regexp.MustCompile(`logs?\s+(?:for|of)\s+([a-zA-Z0-9_.@-]+)(?:\s+service)?`)
	packageRegex          = regexp.MustCompile(`package\s+(?:info|details?)\s+(?:for\s+)?([a-zA-Z0-9+_.-]+)`)
	grepRegex             = regexp.MustCompile(`(?:find|search|grep)\s+(?:for\s+)?["']?([^"']+?)["']?\s+(?:in|under)\s+(\S+)`)
	urlRegex              = regexp.MustCompile(`https?://\S+`)
	tarRegex              = regexp.MustCompile(`(?:tar|archive).*(\S+\.tar(?:\.\w+)?)`)
	runtimeStatusRegex    = regexp.MustCompile(`(?:docker|podman)\s+status|status\s+(?:of\s+)?(?:docker|podman)|is\s+(docker|podman)\s+(?:running|installed)`)
	runtimeVersionRegex   = regexp.MustCompile(`(?:docker|podman)\s+version|version\s+of\s+(docker|podman)`)
	runtimePsRegex        = regexp.MustCompile(`(?:list|show)\s+(docker|podman)\s+containers|(?:docker|podman)\s+containers`)
	runtimeImagesRegex    = regexp.MustCompile(`(?:list|show)\s+(docker|podman)\s+images|(?:docker|podman)\s+images`)
	runtimeLogsRegex      = regexp.MustCompile(`(?:docker|podman)\s+logs(?:\s+for)?\s+([a-zA-Z0-9_.-]+)|logs?\s+(?:for|of)\s+([a-zA-Z0-9_.-]+)\s+(?:container|docker|podman)`)
	runtimeInspectRegex   = regexp.MustCompile(`(?:inspect|show)\s+(docker|podman)\s+container\s+([a-zA-Z0-9_.-]+)`)
	k8sVersionRegex       = regexp.MustCompile(`(?:kubectl|kubernetes)\s+version|version\s+of\s+kubectl`)
	k8sContextRegex       = regexp.MustCompile(`(?:show|current)\s+(?:kubernetes|kubectl|k8s)\s+context|kubectl\s+context`)
	k8sPodsRegex          = regexp.MustCompile(`(?:show|get|list)\s+pods?(?:\s+in\s+namespace\s+\S+)?|kubernetes pods?|kubectl get pods`)
	k8sDeploymentsRegex   = regexp.MustCompile(`(?:show|get|list)\s+deployments?(?:\s+in\s+namespace\s+\S+)?|kubectl get deployments`)
	k8sServicesRegex      = regexp.MustCompile(`(?:show|get|list)\s+services?(?:\s+in\s+namespace\s+\S+)?|kubectl get services`)
	k8sNamespacesRegex    = regexp.MustCompile(`(?:show|get|list)\s+namespaces|kubectl get namespaces`)
	k8sLogsRegex          = regexp.MustCompile(`(?:show|get)?\s*logs?\s+for\s+pod\s+\S+|kubectl logs`)
	k8sDescribeRegex      = regexp.MustCompile(`(?:describe|show)\s+pod\s+\S+|kubectl describe pod`)
	k8sEventsRegex        = regexp.MustCompile(`(?:show|get|list)\s+(?:kubernetes|cluster)?\s*events|kubectl get events`)
	helmVersionRegex      = regexp.MustCompile(`(?:helm)\s+version|version\s+of\s+helm`)
	helmReposRegex        = regexp.MustCompile(`(?:show|get|list)\s+helm\s+repos?|helm repo list`)
	helmReleasesRegex     = regexp.MustCompile(`(?:show|get|list)\s+helm\s+releases?(?:\s+in\s+namespace\s+\S+)?|helm list`)
	helmStatusRegex       = regexp.MustCompile(`(?:show|get)\s+helm\s+status\s+for\s+release\s+\S+|helm status`)
	helmHistoryRegex      = regexp.MustCompile(`(?:show|get)\s+helm\s+history\s+for\s+release\s+\S+|helm history`)
	helmValuesRegex       = regexp.MustCompile(`(?:show|get)\s+helm\s+values\s+for\s+release\s+\S+|helm get values`)
	helmTemplateRegex     = regexp.MustCompile(`(?:preview|render|show)\s+helm\s+template|helm template`)
	tfVersionRegex        = regexp.MustCompile(`(?:terraform)\s+version|version\s+of\s+terraform`)
	tfFmtRegex            = regexp.MustCompile(`(?:check|validate)\s+terraform\s+format(?:ting)?|terraform fmt`)
	tfValidateRegex       = regexp.MustCompile(`(?:validate)\s+terraform|terraform validate`)
	tfPlanRegex           = regexp.MustCompile(`(?:preview|show|run)\s+terraform\s+plan|terraform plan`)
	tfWorkspaceRegex      = regexp.MustCompile(`(?:show|get|list)\s+terraform\s+workspaces?|terraform workspace list`)
	tfStateRegex          = regexp.MustCompile(`(?:show|get|list)\s+terraform\s+state|terraform state list`)
	ansibleVersionRegex   = regexp.MustCompile(`(?:ansible)\s+version|version\s+of\s+ansible`)
	ansibleInventoryRegex = regexp.MustCompile(`(?:show|get|list)\s+ansible\s+inventory|ansible-inventory`)
	ansibleSyntaxRegex    = regexp.MustCompile(`(?:syntax\s+check|check\s+syntax)\s+(?:ansible\s+)?playbook|ansible-playbook --syntax-check`)
	ansibleCheckRegex     = regexp.MustCompile(`(?:dry\s+run|check\s+mode|preview)\s+(?:ansible\s+)?playbook|ansible-playbook --check`)
	sshVersionRegex       = regexp.MustCompile(`(?:ssh)\s+version|version\s+of\s+ssh`)
	sshConfigRegex        = regexp.MustCompile(`(?:show|get)\s+ssh\s+config|ssh config`)
	sshHostKeyRegex       = regexp.MustCompile(`(?:show|get|scan)\s+ssh\s+host\s+key|host key for`)
	sshPortRegex          = regexp.MustCompile(`(?:check|test)\s+ssh\s+(?:connectivity|port)|ssh connectivity`)
	rsyncPreviewRegex     = regexp.MustCompile(`(?:preview|dry run)\s+rsync|copy\s+\S+\s+to\s+\S+`)
	scpPreviewRegex       = regexp.MustCompile(`(?:preview)\s+scp|scp\s+preview`)
	awsVersionRegex       = regexp.MustCompile(`(?:aws)\s+version|version\s+of\s+aws`)
	awsIdentityRegex      = regexp.MustCompile(`(?:show|get)\s+aws\s+(?:identity|caller identity|account)|caller identity`)
	awsProfilesRegex      = regexp.MustCompile(`(?:show|get|list)\s+aws\s+profiles?|aws profiles`)
	azVersionRegex        = regexp.MustCompile(`(?:azure|az)\s+version|version\s+of\s+(?:azure|az)`)
	azAccountRegex        = regexp.MustCompile(`(?:show|get)\s+(?:azure|az)\s+account|azure account`)
	azSubscriptionsRegex  = regexp.MustCompile(`(?:show|get|list)\s+(?:azure|az)\s+subscriptions?|azure subscriptions`)
	gcloudVersionRegex    = regexp.MustCompile(`(?:gcloud|google cloud)\s+version|version\s+of\s+gcloud`)
	gcloudAccountRegex    = regexp.MustCompile(`(?:show|get|list)\s+gcloud\s+account|gcloud account|google cloud account`)
	gcloudProjectRegex    = regexp.MustCompile(`(?:show|get)\s+gcloud\s+project|gcloud project|google cloud project`)
	argocdVersionRegex    = regexp.MustCompile(`(?:argocd|argo cd)\s+version|version\s+of\s+argocd`)
	argocdAccountRegex    = regexp.MustCompile(`(?:show|get)\s+argocd\s+account|argocd account|argo cd account`)
	argocdAppsRegex       = regexp.MustCompile(`(?:show|get|list)\s+argocd\s+apps?|argocd app list|argo cd apps`)
	argocdAppRegex        = regexp.MustCompile(`(?:show|get)\s+argocd\s+app\s+\S+|argocd app get|argo cd app`)
	argocdProjectsRegex   = regexp.MustCompile(`(?:show|get|list)\s+argocd\s+projects?|argocd proj list|argo cd projects`)
	argocdClustersRegex   = regexp.MustCompile(`(?:show|get|list)\s+argocd\s+clusters?|argocd cluster list|argo cd clusters`)
	selinuxModeRegex      = regexp.MustCompile(`(?:show|get)\s+selinux\s+mode|selinux mode|getenforce`)
	selinuxStatusRegex    = regexp.MustCompile(`(?:show|get)\s+selinux\s+status|sestatus`)
	firewallRulesRegex    = regexp.MustCompile(`(?:show|get|list)\s+firewall\s+(?:rules|status)|firewall-cmd --list-all`)
	firewallZonesRegex    = regexp.MustCompile(`(?:show|get|list)\s+firewall\s+zones?|active firewall zones`)
	certInspectRegex      = regexp.MustCompile(`(?:inspect|show|decode)\s+(?:certificate|cert)\s+\S+|openssl x509`)
	cpuInfoRegex          = regexp.MustCompile(`(?:show|get|list)\s+(?:cpu|processor)\s+(?:info|details?)|cpu info|lscpu`)
	memoryInfoRegex       = regexp.MustCompile(`(?:show|get|list)\s+(?:memory|ram)\s+(?:info|usage|status)|memory info|free -h`)
	blockDevicesRegex     = regexp.MustCompile(`(?:show|get|list)\s+(?:block devices|disks|storage devices)|lsblk`)
	systemHardwareRegex   = regexp.MustCompile(`(?:show|get)\s+(?:system|hardware|firmware)\s+(?:info|details?)|dmidecode`)
	diskHealthRegex       = regexp.MustCompile(`(?:show|get|check)\s+(?:disk|drive)\s+health|smartctl -h|smart health`)
	ipmiInfoRegex         = regexp.MustCompile(`(?:show|get)\s+ipmi\s+(?:info|status)|ipmitool`)
	gpuInfoRegex          = regexp.MustCompile(`(?:show|get|list)\s+(?:gpu|nvidia)\s+(?:info|status|usage)|gpu status|nvidia-smi`)
	psqlVersionRegex      = regexp.MustCompile(`(?:postgres|postgresql|psql)\s+version|version\s+of\s+(?:postgres|postgresql|psql)`)
	psqlDatabasesRegex    = regexp.MustCompile(`(?:show|get|list)\s+(?:postgres|postgresql|psql)\s+databases?|postgres databases`)
	mysqlVersionRegex     = regexp.MustCompile(`(?:mysql)\s+version|version\s+of\s+mysql`)
	mysqlDatabasesRegex   = regexp.MustCompile(`(?:show|get|list)\s+mysql\s+databases?|mysql databases`)
	redisVersionRegex     = regexp.MustCompile(`(?:redis|redis-cli)\s+version|version\s+of\s+(?:redis|redis-cli)`)
	redisPingRegex        = regexp.MustCompile(`(?:check|show|get)\s+redis\s+(?:ping|health|status)|redis ping|redis-cli ping`)
	containerdLogsRegex   = regexp.MustCompile(`(?:containerd\s+logs|logs?\s+(?:for|of)\s+containerd)`)
	containerdStatusRegex = regexp.MustCompile(`(?:containerd\s+status|status\s+(?:of\s+)?containerd|is\s+containerd\s+(?:running|installed))`)
	containerdVerRegex    = regexp.MustCompile(`(?:containerd\s+version|version\s+of\s+containerd|ctr\s+version|crictl\s+version|nerdctl\s+version)`)
	// Word-boundary ping: avoids matching words like "shipping" that contain
	// "ping" as a substring.
	pingIntentRegex = regexp.MustCompile(`\bping\b`)
)

func Resolve(query string, env models.Environment, collector evidence.Collector) (models.Response, error) {
	normalized := strings.ToLower(strings.TrimSpace(query))

	if isExplainQuery(normalized) {
		return explain.Command(query, collector)
	}
	if response, ok := troubleshoot.Resolve(query, collector); ok {
		return response, nil
	}

	switch {
	case redisPingRegex.MatchString(normalized):
		return redisPingIntent(collector)
	case redisVersionRegex.MatchString(normalized):
		return redisVersionIntent(collector)
	case mysqlDatabasesRegex.MatchString(normalized):
		return mysqlDatabasesIntent(collector)
	case mysqlVersionRegex.MatchString(normalized):
		return mysqlVersionIntent(collector)
	case psqlDatabasesRegex.MatchString(normalized):
		return postgresDatabasesIntent(collector)
	case psqlVersionRegex.MatchString(normalized):
		return postgresVersionIntent(collector)
	case gpuInfoRegex.MatchString(normalized):
		return gpuInfoIntent(collector)
	case ipmiInfoRegex.MatchString(normalized):
		return ipmiInfoIntent(collector)
	case diskHealthRegex.MatchString(normalized):
		return diskHealthIntent(query, collector)
	case systemHardwareRegex.MatchString(normalized):
		return systemHardwareIntent(collector)
	case blockDevicesRegex.MatchString(normalized):
		return blockDevicesIntent(collector)
	case memoryInfoRegex.MatchString(normalized):
		return memoryInfoIntent(collector)
	case cpuInfoRegex.MatchString(normalized):
		return cpuInfoIntent(collector)
	case certInspectRegex.MatchString(normalized):
		return opensslCertIntent(query, collector)
	case firewallZonesRegex.MatchString(normalized):
		return firewallZonesIntent(collector)
	case firewallRulesRegex.MatchString(normalized):
		return firewallRulesIntent(collector)
	case selinuxStatusRegex.MatchString(normalized):
		return selinuxStatusIntent(collector)
	case selinuxModeRegex.MatchString(normalized):
		return selinuxModeIntent(collector)
	case argocdClustersRegex.MatchString(normalized):
		return argocdClustersIntent(collector)
	case argocdProjectsRegex.MatchString(normalized):
		return argocdProjectsIntent(collector)
	case argocdAppRegex.MatchString(normalized):
		return argocdAppGetIntent(query, collector)
	case argocdAppsRegex.MatchString(normalized):
		return argocdAppsIntent(collector)
	case argocdAccountRegex.MatchString(normalized):
		return argocdAccountIntent(collector)
	case argocdVersionRegex.MatchString(normalized):
		return argocdVersionIntent(collector)
	case gcloudProjectRegex.MatchString(normalized):
		return gcloudProjectIntent(collector)
	case gcloudAccountRegex.MatchString(normalized):
		return gcloudAccountIntent(collector)
	case gcloudVersionRegex.MatchString(normalized):
		return gcloudVersionIntent(collector)
	case azSubscriptionsRegex.MatchString(normalized):
		return azSubscriptionsIntent(collector)
	case azAccountRegex.MatchString(normalized):
		return azAccountIntent(collector)
	case azVersionRegex.MatchString(normalized):
		return azVersionIntent(collector)
	case awsProfilesRegex.MatchString(normalized):
		return awsProfilesIntent(collector)
	case awsIdentityRegex.MatchString(normalized):
		return awsIdentityIntent(collector)
	case awsVersionRegex.MatchString(normalized):
		return awsVersionIntent(collector)
	case scpPreviewRegex.MatchString(normalized):
		return scpPreviewIntent(query, collector)
	case rsyncPreviewRegex.MatchString(normalized):
		return rsyncDryRunIntent(query, collector)
	case sshPortRegex.MatchString(normalized):
		return sshPortCheckIntent(query, collector)
	case sshHostKeyRegex.MatchString(normalized):
		return sshHostKeyIntent(query, collector)
	case sshConfigRegex.MatchString(normalized):
		return sshConfigIntent(query, collector)
	case sshVersionRegex.MatchString(normalized):
		return sshVersionIntent(collector)
	case ansibleCheckRegex.MatchString(normalized):
		return ansibleCheckModeIntent(query, collector)
	case ansibleSyntaxRegex.MatchString(normalized):
		return ansibleSyntaxCheckIntent(query, collector)
	case ansibleInventoryRegex.MatchString(normalized):
		return ansibleInventoryIntent(query, collector)
	case ansibleVersionRegex.MatchString(normalized):
		return ansibleVersionIntent(collector)
	case tfStateRegex.MatchString(normalized):
		return terraformStateListIntent(collector)
	case tfWorkspaceRegex.MatchString(normalized):
		return terraformWorkspaceListIntent(collector)
	case tfPlanRegex.MatchString(normalized):
		return terraformPlanIntent(collector)
	case tfValidateRegex.MatchString(normalized):
		return terraformValidateIntent(collector)
	case tfFmtRegex.MatchString(normalized):
		return terraformFmtCheckIntent(collector)
	case tfVersionRegex.MatchString(normalized):
		return terraformVersionIntent(collector)
	case helmTemplateRegex.MatchString(normalized):
		return helmTemplateIntent(query, collector)
	case helmValuesRegex.MatchString(normalized):
		return helmValuesIntent(query, collector)
	case helmHistoryRegex.MatchString(normalized):
		return helmHistoryIntent(query, collector)
	case helmStatusRegex.MatchString(normalized):
		return helmStatusIntent(query, collector)
	case helmReleasesRegex.MatchString(normalized):
		return helmReleasesIntent(query, collector)
	case helmReposRegex.MatchString(normalized):
		return helmReposIntent(collector)
	case helmVersionRegex.MatchString(normalized):
		return helmVersionIntent(collector)
	case k8sEventsRegex.MatchString(normalized):
		return kubectlEventsIntent(query, collector)
	case k8sDescribeRegex.MatchString(normalized):
		return kubectlDescribePodIntent(query, collector)
	case k8sLogsRegex.MatchString(normalized):
		return kubectlLogsIntent(query, collector)
	case k8sNamespacesRegex.MatchString(normalized):
		return kubectlNamespacesIntent(collector)
	case k8sServicesRegex.MatchString(normalized):
		return kubectlServicesIntent(query, collector)
	case k8sDeploymentsRegex.MatchString(normalized):
		return kubectlDeploymentsIntent(query, collector)
	case k8sPodsRegex.MatchString(normalized):
		return kubectlPodsIntent(query, collector)
	case k8sContextRegex.MatchString(normalized):
		return kubectlContextIntent(env, collector)
	case k8sVersionRegex.MatchString(normalized):
		return kubectlVersionIntent(collector)
	case runtimeInspectRegex.MatchString(normalized):
		return runtimeInspectIntent(query, collector)
	case runtimeLogsRegex.MatchString(normalized):
		return runtimeLogsIntent(query, collector)
	case runtimeImagesRegex.MatchString(normalized):
		return runtimeImagesIntent(query, collector)
	case runtimePsRegex.MatchString(normalized):
		return runtimePsIntent(query, collector)
	case runtimeVersionRegex.MatchString(normalized):
		return runtimeVersionIntent(query, collector)
	case runtimeStatusRegex.MatchString(normalized):
		return runtimeStatusIntent(query, collector)
	case strings.Contains(normalized, "disk free") || strings.Contains(normalized, "free space") || normalized == "df -h":
		return diskFreeIntent(collector)
	case strings.Contains(normalized, "top processes") || strings.Contains(normalized, "memory usage by process") || strings.Contains(normalized, "cpu usage by process"):
		return processIntent(normalized, collector)
	case strings.Contains(normalized, "ip address") || strings.Contains(normalized, "network interfaces") || strings.Contains(normalized, "show interfaces"):
		return ipAddressIntent(collector)
	case pingIntentRegex.MatchString(normalized) || strings.Contains(normalized, "reachability"):
		return pingIntent(query, collector)
	case strings.Contains(normalized, "http headers") || strings.Contains(normalized, "check website") || strings.Contains(normalized, "curl headers"):
		return curlHeadIntent(query, collector)
	case strings.Contains(normalized, "tail ") || strings.Contains(normalized, "show log file"):
		return tailFileIntent(query, collector)
	case strings.Contains(normalized, "list tar") || strings.Contains(normalized, "archive contents"):
		return tarListIntent(query, collector)
	case strings.Contains(normalized, "process") && strings.Contains(normalized, "port"):
		return portIntent(query, collector)
	case strings.Contains(normalized, "git log") || strings.Contains(normalized, "recent commits"):
		return gitLogIntent(collector)
	case strings.Contains(normalized, "git diff") || strings.Contains(normalized, "uncommitted changes"):
		return gitDiffIntent(collector)
	case strings.Contains(normalized, "git branch") || strings.Contains(normalized, "branches in git"):
		return gitBranchIntent(collector)
	case containerdVerRegex.MatchString(normalized):
		return containerdVersionIntent(collector)
	case containerdLogsRegex.MatchString(normalized):
		return containerdLogsIntent(collector)
	case containerdStatusRegex.MatchString(normalized):
		return containerdStatusIntent(collector)
	case (strings.Contains(normalized, "logs for") || strings.Contains(normalized, "logs of")) && strings.Contains(normalized, "service"):
		return serviceLogsIntent(query, collector)
	case strings.Contains(normalized, "status") && strings.Contains(normalized, "service"):
		return serviceIntent(query, collector)
	case strings.Contains(normalized, "disk usage") || strings.Contains(normalized, "size of"):
		return diskUsageIntent(query, collector)
	case strings.Contains(normalized, "files larger than"):
		return largeFilesIntent(query, collector)
	case strings.Contains(normalized, "package info") || strings.Contains(normalized, "package details"):
		return packageInfoIntent(query, env, collector)
	case strings.Contains(normalized, "git status") || strings.HasPrefix(normalized, "status of git"):
		return gitStatusIntent(env, collector)
	case strings.Contains(normalized, "grep ") || strings.Contains(normalized, "search for") || strings.Contains(normalized, "find ") && strings.Contains(normalized, " in "):
		return grepIntent(query, collector)
	default:
		return models.Response{
			IntentID:       "unsupported_query",
			Explanation:    "This query did not match any supported intent. Try rephrasing as a plain-English question about a specific tool, service, or resource — for example: \"show disk free space\", \"nginx is not starting\", or \"explain kubectl delete\".",
			ExpectedOutput: "A response with command guidance once a matching intent is found.",
			Risk:           safety.RiskLow,
			Confidence:     "Low",
			Warnings:       []string{"query did not match any known intent"},
			NextSteps:      []string{"Run 'cli-helper --help' to see all subcommands and usage examples."},
		}, nil
	}
}

func isExplainQuery(normalized string) bool {
	return strings.HasPrefix(normalized, "explain ") || strings.HasPrefix(normalized, "what does ")
}
