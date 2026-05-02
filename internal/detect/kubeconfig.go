package detect

import (
	"os"

	"gopkg.in/yaml.v3"
)

type kubeConfigDetails struct {
	CurrentContext string
	Server         string
	contextCluster map[string]string
	clusterServer  map[string]string
}

type kubeConfigFile struct {
	CurrentContext string              `yaml:"current-context"`
	Contexts       []kubeConfigContext `yaml:"contexts"`
	Clusters       []kubeConfigCluster `yaml:"clusters"`
}

type kubeConfigContext struct {
	Name    string `yaml:"name"`
	Context struct {
		Cluster string `yaml:"cluster"`
	} `yaml:"context"`
}

type kubeConfigCluster struct {
	Name    string `yaml:"name"`
	Cluster struct {
		Server string `yaml:"server"`
	} `yaml:"cluster"`
}

func parseKubeConfig(path string) (kubeConfigDetails, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return kubeConfigDetails{}, err
	}
	return parseKubeConfigContent(content)
}

func parseKubeConfigs(paths []string) (kubeConfigDetails, error) {
	merged := kubeConfigDetails{
		contextCluster: map[string]string{},
		clusterServer:  map[string]string{},
	}
	var firstErr error
	for _, path := range paths {
		details, err := parseKubeConfig(path)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if merged.CurrentContext == "" {
			merged.CurrentContext = details.CurrentContext
		}
		for name, cluster := range details.contextCluster {
			if _, exists := merged.contextCluster[name]; !exists {
				merged.contextCluster[name] = cluster
			}
		}
		for name, server := range details.clusterServer {
			if _, exists := merged.clusterServer[name]; !exists {
				merged.clusterServer[name] = server
			}
		}
	}
	merged.Server = serverForContext(merged.CurrentContext, merged.contextCluster, merged.clusterServer)
	if merged.Server == "" && len(merged.clusterServer) == 1 {
		for _, server := range merged.clusterServer {
			merged.Server = server
		}
	}
	return merged, firstErr
}

func parseKubeConfigContent(content []byte) (kubeConfigDetails, error) {
	var parsed kubeConfigFile
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return kubeConfigDetails{}, err
	}

	details := kubeConfigDetails{
		CurrentContext: parsed.CurrentContext,
		contextCluster: map[string]string{},
		clusterServer:  map[string]string{},
	}
	for _, context := range parsed.Contexts {
		if context.Name != "" && context.Context.Cluster != "" {
			details.contextCluster[context.Name] = context.Context.Cluster
		}
	}
	for _, cluster := range parsed.Clusters {
		if cluster.Name != "" && cluster.Cluster.Server != "" {
			details.clusterServer[cluster.Name] = cluster.Cluster.Server
		}
	}
	details.Server = serverForContext(details.CurrentContext, details.contextCluster, details.clusterServer)
	if details.Server == "" && len(details.clusterServer) == 1 {
		for _, server := range details.clusterServer {
			details.Server = server
		}
	}
	return details, nil
}

func serverForContext(context string, contextCluster, clusterServer map[string]string) string {
	if context == "" {
		return ""
	}
	clusterName := contextCluster[context]
	if clusterName == "" {
		return ""
	}
	return clusterServer[clusterName]
}
