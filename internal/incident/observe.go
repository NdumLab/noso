package incident

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/NdumLab/noso/internal/interpret"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/internal/troubleshoot"
	"github.com/NdumLab/noso/pkg/models"
)

const observeTimeout = 3 * time.Second

type runner func(context.Context, string) (string, error)

var allowedObservePrefixes = []string{
	"systemctl status ",
	"journalctl -u ",
	"podman ps -a",
	"podman logs ",
	"docker ps -a",
	"docker logs ",
	"kubectl get pods",
	"kubectl get events",
	"kubectl describe pod ",
	"kubectl describe deployment ",
	"kubectl describe service ",
	"kubectl describe pvc ",
	"kubectl describe secret ",
	"kubectl describe configmap ",
	"kubectl describe node ",
	"kubectl logs ",
	"dig +short ",
	"nslookup ",
	"nc -vz ",
	"ss -ltnp",
}

func ObserveNext(record Record) (models.Response, string, error) {
	return observeNextWithRunner(record, runReadOnlyCommand)
}

func observeNextWithRunner(record Record, run runner) (models.Response, string, error) {
	command, err := NextObserveCommand(record)
	if err != nil {
		return models.Response{}, "", err
	}
	response := models.Response{
		IntentID:       "incident_observe",
		Command:        command,
		Explanation:    "Executed the next approved read-only incident probe and interpreted the result into updated findings.",
		ExpectedOutput: "Updated incident findings and follow-up guidance from the observed command output.",
		Risk:           safety.Classify(command),
		Confidence:     "High",
		NextSteps:      append([]string{}, record.NextSteps...),
	}

	if enriched := troubleshoot.EnrichWithLiveEvidence(response); len(enriched.Findings) > 0 || len(enriched.VerifiedFrom) > 0 || len(enriched.Warnings) > 0 {
		return enriched, command, nil
	}

	output, err := run(context.Background(), command)
	if err != nil {
		response.Warnings = append(response.Warnings, "incident observe failed: "+err.Error())
		return response, command, nil
	}
	interpreted, err := interpret.Output(command, output)
	if err != nil {
		response.Warnings = append(response.Warnings, "incident observe interpretation failed: "+err.Error())
		return response, command, nil
	}
	response.IntentID = interpreted.IntentID
	response.Explanation = interpreted.Explanation
	response.ExpectedOutput = interpreted.ExpectedOutput
	response.Findings = append([]string{}, interpreted.Findings...)
	response.VerifiedFrom = append([]string{}, "live:"+command)
	response.NextSteps = append([]string{}, interpreted.NextSteps...)
	response.ContainerHint = interpreted.ContainerHint
	if strings.TrimSpace(output) == "" {
		response.Warnings = append(response.Warnings, "incident observe produced no output")
	}
	return response, command, nil
}

func NextObserveCommand(record Record) (string, error) {
	executed := map[string]bool{}
	for _, probe := range record.ProbeHistory {
		if strings.TrimSpace(probe.Command) != "" {
			executed[strings.TrimSpace(probe.Command)] = true
		}
	}
	for _, step := range record.NextSteps {
		command := extractBacktickCommand(step)
		if command == "" || executed[command] {
			continue
		}
		if !ObserveAllowed(command) {
			continue
		}
		return command, nil
	}
	if command := strings.TrimSpace(record.LastCommand); command != "" && !executed[command] && ObserveAllowed(command) {
		return command, nil
	}
	return "", fmt.Errorf("no unread approved read-only probe is available for this incident")
}

func ObserveAllowed(command string) bool {
	command = strings.ToLower(strings.TrimSpace(command))
	if safety.Classify(command) != safety.RiskLow {
		return false
	}
	for _, prefix := range allowedObservePrefixes {
		if strings.HasPrefix(command, prefix) {
			return true
		}
	}
	return false
}

func runReadOnlyCommand(parent context.Context, command string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, observeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	output := strings.TrimSpace(buf.String())
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("incident observe timed out for %q", command)
	}
	if err != nil && output == "" {
		return "", fmt.Errorf("incident observe failed for %q: %w", command, err)
	}
	return output, nil
}

func extractBacktickCommand(step string) string {
	start := strings.Index(step, "`")
	if start < 0 {
		return ""
	}
	rest := step[start+1:]
	end := strings.Index(rest, "`")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}
