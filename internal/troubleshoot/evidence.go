package troubleshoot

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/NdumLab/noso/internal/interpret"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

const probeTimeout = 3 * time.Second

type commandRunner func(context.Context, string) (string, error)

// EnrichWithLiveEvidence runs a small read-only evidence loop for troubleshoot
// responses. It currently supports systemd service probes and journal review.
func EnrichWithLiveEvidence(response models.Response) models.Response {
	return enrichWithRunner(response, runReadOnlyCommand)
}

func ApplyThreadContext(response models.Response, thread StateThread) models.Response {
	if len(thread.Executed) == 0 {
		return response
	}
	if response.Command != "" && alreadyExecuted(thread, response.Command) {
		response.Warnings = appendUnique(response.Warnings, "this primary probe already ran in the current troubleshoot thread")
		next := nextSuggestedCommand(response.NextSteps, thread)
		if next != "" {
			response.Command = next
			response.Explanation = response.Explanation + " Advancing to the next unread probe from the current troubleshoot thread."
		}
	}
	for _, finding := range thread.LastFindings {
		response.Findings = appendUnique(response.Findings, "Previous finding: "+finding)
	}
	for _, warning := range thread.LastWarnings {
		response.Warnings = appendUnique(response.Warnings, "previous thread warning: "+warning)
	}
	return response
}

func enrichWithRunner(response models.Response, runner commandRunner) models.Response {
	if safety.Classify(response.Command) != safety.RiskLow {
		return response
	}
	lower := strings.ToLower(strings.TrimSpace(response.Command))
	switch {
	case strings.HasPrefix(lower, "systemctl status "):
		return enrichServiceEvidence(response, runner)
	case strings.HasPrefix(lower, "docker ps"), strings.HasPrefix(lower, "podman ps"):
		return enrichRuntimeEvidence(response, runner)
	case strings.HasPrefix(lower, "kubectl get pods"):
		return enrichKubernetesPodsEvidence(response, runner)
	case strings.HasPrefix(lower, "kubectl describe pod"):
		return enrichKubernetesEvidence(response, runner)
	default:
		return response
	}
}

func runReadOnlyCommand(parent context.Context, command string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, probeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	output := strings.TrimSpace(buf.String())
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("read-only probe timed out for %q", command)
	}
	if err != nil && output == "" {
		return "", fmt.Errorf("read-only probe failed for %q: %w", command, err)
	}
	return output, nil
}

func serviceFromStatusCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) < 3 {
		return ""
	}
	return fields[2]
}

func shouldQueryJournalctl(statusOutput string) bool {
	lower := strings.ToLower(statusOutput)
	return strings.Contains(lower, "active: failed") ||
		strings.Contains(lower, "active: inactive (dead)") ||
		strings.Contains(lower, "result: exit-code")
}

func enrichServiceEvidence(response models.Response, runner commandRunner) models.Response {
	statusOutput, err := runner(context.Background(), response.Command)
	if err != nil {
		response.Warnings = append(response.Warnings, "live probe failed: "+err.Error())
		return response
	}
	statusInterpret, err := interpret.Output(response.Command, statusOutput)
	if err != nil {
		response.Warnings = append(response.Warnings, "live probe interpretation failed: "+err.Error())
		return response
	}
	if commandUnavailableOutput(statusOutput) {
		response.Warnings = append(response.Warnings, "live probe unavailable: "+strings.TrimSpace(statusOutput))
		return response
	}

	service := serviceFromStatusCommand(response.Command)
	response.Findings = appendUnique(response.Findings, "Live service evidence: "+statusInterpret.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+response.Command)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, statusInterpret.NextSteps...)

	if !shouldQueryJournalctl(statusOutput) || service == "" {
		return response
	}

	journalCommand := fmt.Sprintf("journalctl -u %s -n 50 --no-pager", service)
	journalOutput, err := runner(context.Background(), journalCommand)
	if err != nil {
		response.Warnings = append(response.Warnings, "live journal probe failed: "+err.Error())
		return response
	}
	journalInterpret, err := interpret.Output(journalCommand, journalOutput)
	if err != nil {
		response.Warnings = append(response.Warnings, "live journal interpretation failed: "+err.Error())
		return response
	}

	response.Findings = appendUnique(response.Findings, "Journal evidence: "+journalInterpret.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+journalCommand)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, journalInterpret.NextSteps...)
	return response
}

func enrichRuntimeEvidence(response models.Response, runner commandRunner) models.Response {
	output, err := runner(context.Background(), response.Command)
	if err != nil {
		response.Warnings = append(response.Warnings, "live runtime probe failed: "+err.Error())
		return response
	}
	interpreted, err := interpret.Output(response.Command, output)
	if err != nil {
		response.Warnings = append(response.Warnings, "live runtime interpretation failed: "+err.Error())
		return response
	}
	if commandUnavailableOutput(output) {
		response.Warnings = append(response.Warnings, "live runtime probe unavailable: "+strings.TrimSpace(output))
		return response
	}
	response.Findings = appendUnique(response.Findings, "Runtime evidence: "+interpreted.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+response.Command)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, interpreted.NextSteps...)

	target := runtimeTargetFromPlan(response)
	if target == "" || !shouldQueryRuntimeLogs(output, target) {
		return response
	}
	runtime := runtimeFromPSCommand(response.Command)
	logCommand := fmt.Sprintf("%s logs --tail 100 %s", runtime, target)
	logOutput, err := runner(context.Background(), logCommand)
	if err != nil {
		response.Warnings = append(response.Warnings, "live runtime log probe failed: "+err.Error())
		return response
	}
	logInterpret, err := interpret.Output(logCommand, logOutput)
	if err != nil {
		response.Warnings = append(response.Warnings, "live runtime log interpretation failed: "+err.Error())
		return response
	}
	response.Findings = appendUnique(response.Findings, "Runtime log evidence: "+logInterpret.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+logCommand)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, logInterpret.NextSteps...)
	return response
}

func enrichKubernetesEvidence(response models.Response, runner commandRunner) models.Response {
	output, err := runner(context.Background(), response.Command)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes probe failed: "+err.Error())
		return response
	}
	interpreted, err := interpret.Output(response.Command, output)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes interpretation failed: "+err.Error())
		return response
	}
	if commandUnavailableOutput(output) {
		response.Warnings = append(response.Warnings, "live kubernetes probe unavailable: "+strings.TrimSpace(output))
		return response
	}
	response.Findings = appendUnique(response.Findings, "Kubernetes evidence: "+interpreted.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+response.Command)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, interpreted.NextSteps...)

	pod, namespace := podFromDescribeCommand(response.Command)
	if pod == "" || !shouldQueryKubectlLogs(output) {
		return response
	}
	logCommand := fmt.Sprintf("kubectl logs %s --tail=100", pod)
	if namespace != "" {
		logCommand = fmt.Sprintf("kubectl logs -n %s %s --tail=100", namespace, pod)
	}
	logOutput, err := runner(context.Background(), logCommand)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes log probe failed: "+err.Error())
		return response
	}
	logInterpret, err := interpret.Output(logCommand, logOutput)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes log interpretation failed: "+err.Error())
		return response
	}
	response.Findings = appendUnique(response.Findings, "Kubernetes log evidence: "+logInterpret.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+logCommand)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, logInterpret.NextSteps...)
	return response
}

func enrichKubernetesPodsEvidence(response models.Response, runner commandRunner) models.Response {
	output, err := runner(context.Background(), response.Command)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes probe failed: "+err.Error())
		return response
	}
	interpreted, err := interpret.Output(response.Command, output)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes interpretation failed: "+err.Error())
		return response
	}
	if commandUnavailableOutput(output) {
		response.Warnings = append(response.Warnings, "live kubernetes probe unavailable: "+strings.TrimSpace(output))
		return response
	}
	response.Findings = appendUnique(response.Findings, "Kubernetes evidence: "+interpreted.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+response.Command)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, interpreted.NextSteps...)

	pod, namespace := firstUnhealthyPodFromGetPods(output)
	if pod == "" {
		return response
	}
	describeCommand := fmt.Sprintf("kubectl describe pod %s", pod)
	if namespace != "" {
		describeCommand = fmt.Sprintf("kubectl describe pod -n %s %s", namespace, pod)
	}
	describeOutput, err := runner(context.Background(), describeCommand)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes describe probe failed: "+err.Error())
		return response
	}
	describeInterpret, err := interpret.Output(describeCommand, describeOutput)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes describe interpretation failed: "+err.Error())
		return response
	}
	response.Findings = appendUnique(response.Findings, "Kubernetes describe evidence: "+describeInterpret.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+describeCommand)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, describeInterpret.NextSteps...)

	if !shouldQueryKubectlLogs(describeOutput) {
		return response
	}
	logCommand := fmt.Sprintf("kubectl logs %s --tail=100", pod)
	if namespace != "" {
		logCommand = fmt.Sprintf("kubectl logs -n %s %s --tail=100", namespace, pod)
	}
	logOutput, err := runner(context.Background(), logCommand)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes log probe failed: "+err.Error())
		return response
	}
	logInterpret, err := interpret.Output(logCommand, logOutput)
	if err != nil {
		response.Warnings = append(response.Warnings, "live kubernetes log interpretation failed: "+err.Error())
		return response
	}
	response.Findings = appendUnique(response.Findings, "Kubernetes log evidence: "+logInterpret.Explanation)
	response.VerifiedFrom = appendUnique(response.VerifiedFrom, "live:"+logCommand)
	response.NextSteps = appendEvidenceSteps(response.NextSteps, logInterpret.NextSteps...)
	return response
}

func runtimeFromPSCommand(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "docker"
	}
	return fields[0]
}

func runtimeTargetFromPlan(response models.Response) string {
	for _, step := range response.NextSteps {
		if strings.Contains(step, "logs --tail 100 ") {
			parts := strings.Split(step, "logs --tail 100 ")
			if len(parts) < 2 {
				continue
			}
			target := strings.TrimSpace(parts[1])
			target = strings.TrimPrefix(target, "`")
			if idx := strings.Index(target, "`"); idx >= 0 {
				target = target[:idx]
			}
			if idx := strings.Index(target, " "); idx >= 0 {
				target = target[:idx]
			}
			if target != "" {
				return target
			}
		}
	}
	return ""
}

func shouldQueryRuntimeLogs(output, target string) bool {
	lower := strings.ToLower(output)
	target = strings.ToLower(target)
	return strings.Contains(lower, target) &&
		(strings.Contains(lower, "exited") || strings.Contains(lower, "restarting") || strings.Contains(lower, "created") || strings.Contains(lower, "unhealthy"))
}

func podFromDescribeCommand(command string) (string, string) {
	fields := strings.Fields(command)
	if len(fields) < 4 {
		return "", ""
	}
	if len(fields) >= 6 && fields[3] == "-n" {
		return fields[5], fields[4]
	}
	return fields[3], ""
}

func shouldQueryKubectlLogs(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "crashloopbackoff") ||
		strings.Contains(lower, "back-off restarting failed container") ||
		strings.Contains(lower, "oomkilled") ||
		strings.Contains(lower, "state: waiting")
}

func firstUnhealthyPodFromGetPods(output string) (string, string) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 4 {
			continue
		}
		if strings.EqualFold(fields[0], "NAME") {
			continue
		}
		namespace := ""
		name := fields[0]
		status := ""
		if strings.HasPrefix(name, "NAMESPACE") {
			continue
		}
		// kubectl get pods -A includes namespace as the first column.
		if len(fields) >= 5 && !strings.Contains(fields[1], "/") {
			namespace = fields[0]
			name = fields[1]
			status = fields[3]
		} else {
			status = fields[2]
		}
		switch status {
		case "Running", "Completed":
			continue
		default:
			return name, namespace
		}
	}
	return "", ""
}

func commandUnavailableOutput(output string) bool {
	lower := strings.ToLower(strings.TrimSpace(output))
	return strings.Contains(lower, "command not found") || strings.Contains(lower, "not recognized as an internal or external command")
}

func appendEvidenceSteps(existing []string, extra ...string) []string {
	out := append([]string{}, existing...)
	for _, step := range extra {
		if strings.TrimSpace(step) == "" {
			continue
		}
		out = appendUnique(out, "Evidence follow-up: "+step)
	}
	return out
}

func alreadyExecuted(thread StateThread, command string) bool {
	for _, existing := range thread.Executed {
		if strings.TrimSpace(existing) == strings.TrimSpace(command) {
			return true
		}
	}
	return false
}

func nextSuggestedCommand(steps []string, thread StateThread) string {
	preferredFamilies := nextPreferredFamilies(thread)
	for _, family := range preferredFamilies {
		for _, step := range steps {
			command := extractBacktickCommand(step)
			if command == "" || alreadyExecuted(thread, command) {
				continue
			}
			if commandFamily(command) == family {
				return command
			}
		}
	}
	for _, step := range steps {
		command := extractBacktickCommand(step)
		if command == "" {
			continue
		}
		if !alreadyExecuted(thread, command) {
			return command
		}
	}
	return ""
}

func nextPreferredFamilies(thread StateThread) []string {
	if len(thread.FamilyScores) > 0 {
		type scoredFamily struct {
			name  string
			score float64
		}
		var families []scoredFamily
		for _, name := range []string{"service", "runtime", "kubernetes", "other"} {
			families = append(families, scoredFamily{name: name, score: thread.FamilyScores[name]})
		}
		sort.SliceStable(families, func(i, j int) bool {
			return families[i].score > families[j].score
		})
		var ordered []string
		for _, family := range families {
			ordered = append(ordered, family.name)
		}
		return ordered
	}

	text := strings.ToLower(strings.Join(append(append([]string{}, thread.LastFindings...), thread.LastWarnings...), "\n"))
	switch {
	case strings.Contains(text, "unit could not be found"), strings.Contains(text, "service name is wrong"), strings.Contains(text, "not managed by systemd"):
		return []string{"runtime", "kubernetes", "service"}
	case strings.Contains(text, "runtime probe unavailable"), strings.Contains(text, "docker is not currently installed"), strings.Contains(text, "no clear container state"):
		return []string{"kubernetes", "service", "runtime"}
	case strings.Contains(text, "kubernetes probe unavailable"), strings.Contains(text, "kubectl is not currently installed"), strings.Contains(text, "no current kubernetes context"):
		return []string{"service", "runtime", "kubernetes"}
	default:
		return []string{"service", "runtime", "kubernetes", "other"}
	}
}

func commandFamily(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command))
	switch {
	case strings.HasPrefix(lower, "systemctl"), strings.HasPrefix(lower, "journalctl"):
		return "service"
	case strings.HasPrefix(lower, "docker "), strings.HasPrefix(lower, "podman "), strings.HasPrefix(lower, "ctr "), strings.HasPrefix(lower, "crictl "), strings.HasPrefix(lower, "nerdctl "):
		return "runtime"
	case strings.HasPrefix(lower, "kubectl "):
		return "kubernetes"
	default:
		return "other"
	}
}

func extractBacktickCommand(value string) string {
	start := strings.Index(value, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(value[start+1:], "`")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(value[start+1 : start+1+end])
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
