package registry

import (
	"strings"
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestKubectlPodsIntentNamespace(t *testing.T) {
	response, err := Resolve("show pods in namespace kube-system", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_pods" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
	if !strings.Contains(response.Command, "-n kube-system") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestKubectlDescribeIntent(t *testing.T) {
	response, err := Resolve("describe pod api-123 in namespace prod", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_pod_describe" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestKubectlVersionIntent(t *testing.T) {
	response, err := Resolve("kubectl version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_version" {
		t.Fatalf("IntentID = %q, want inspect_k8s_version", response.IntentID)
	}
}

func TestKubectlDeploymentsIntent(t *testing.T) {
	response, err := Resolve("list deployments", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_deployments" {
		t.Fatalf("IntentID = %q, want inspect_k8s_deployments", response.IntentID)
	}
}

func TestKubectlServicesIntent(t *testing.T) {
	response, err := Resolve("list services", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_services" {
		t.Fatalf("IntentID = %q, want inspect_k8s_services", response.IntentID)
	}
}

func TestKubectlNamespacesIntent(t *testing.T) {
	response, err := Resolve("list namespaces", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_namespaces" {
		t.Fatalf("IntentID = %q, want inspect_k8s_namespaces", response.IntentID)
	}
}

func TestKubectlLogsIntent(t *testing.T) {
	response, err := Resolve("show logs for pod api-abc-123", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_logs" {
		t.Fatalf("IntentID = %q, want inspect_k8s_logs", response.IntentID)
	}
	if !strings.Contains(response.Command, "api-abc-123") {
		t.Fatalf("Command = %q", response.Command)
	}
}

func TestKubectlEventsIntent(t *testing.T) {
	response, err := Resolve("show cluster events", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_events" {
		t.Fatalf("IntentID = %q, want inspect_k8s_events", response.IntentID)
	}
}

func TestKubectlContextIntent(t *testing.T) {
	env := models.Environment{KubeContext: ""}
	response, err := Resolve("show kubernetes context", env, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_k8s_context" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
