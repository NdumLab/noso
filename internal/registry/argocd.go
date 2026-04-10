package registry

import (
	"fmt"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func argocdVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleArgoIntent(
		collector.Lookup("argocd"),
		"inspect_argocd_version",
		"argocd version --client",
		"Shows the installed Argo CD CLI version without requiring a server-side change.",
		"Client version details for the local Argo CD CLI.",
	)
}

func argocdAccountIntent(collector evidence.Collector) (models.Response, error) {
	return simpleArgoIntent(
		collector.Lookup("argocd"),
		"inspect_argocd_account",
		"argocd account get-user-info",
		"Shows the currently authenticated Argo CD account and capability information.",
		"Current user or account details for the active Argo CD login context.",
	)
}

func argocdAppsIntent(collector evidence.Collector) (models.Response, error) {
	return simpleArgoIntent(
		collector.Lookup("argocd"),
		"inspect_argocd_applications",
		"argocd app list",
		"Lists Argo CD applications and their sync and health status.",
		"A table of Argo CD applications including project, sync state, health, and destination.",
	)
}

func argocdAppGetIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("argocd")
	app := extractNamedObject(query, "app", "application-name")
	command := fmt.Sprintf("argocd app get %s", app)
	response := models.Response{
		IntentID:       "inspect_argocd_application",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows detailed status for Argo CD application %s, including sync state, health, and resource summaries.", app),
		ExpectedOutput: "A detailed application report showing source, destination, sync status, health, history, and managed resources.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "argocd")
	return response, nil
}

func argocdProjectsIntent(collector evidence.Collector) (models.Response, error) {
	return simpleArgoIntent(
		collector.Lookup("argocd"),
		"inspect_argocd_projects",
		"argocd proj list",
		"Lists Argo CD projects and their repository or destination boundaries.",
		"A project table showing configured projects and high-level scope details.",
	)
}

func argocdClustersIntent(collector evidence.Collector) (models.Response, error) {
	return simpleArgoIntent(
		collector.Lookup("argocd"),
		"inspect_argocd_clusters",
		"argocd cluster list",
		"Lists Kubernetes clusters registered with Argo CD.",
		"A cluster table showing server addresses, names, and connection state.",
	)
}

func simpleArgoIntent(ev evidence.Evidence, intentID, command, explanation, expected string) (models.Response, error) {
	response := models.Response{
		IntentID:       intentID,
		Command:        command,
		Explanation:    explanation,
		ExpectedOutput: expected,
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "argocd")
	return response, nil
}
