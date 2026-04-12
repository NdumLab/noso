package troubleshoot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NdumLab/noso/pkg/models"
)

type State struct {
	UpdatedAt string        `json:"updated_at"`
	Threads   []StateThread `json:"threads"`
}

type StateThread struct {
	Query            string             `json:"query"`
	IntentID         string             `json:"intent_id"`
	ActiveFamily     string             `json:"active_family,omitempty"`
	ActiveTarget     string             `json:"active_target,omitempty"`
	ActiveNamespace  string             `json:"active_namespace,omitempty"`
	ActiveContainer  string             `json:"active_container,omitempty"`
	RuntimeHint      string             `json:"runtime_hint,omitempty"`
	LastCommand      string             `json:"last_command,omitempty"`
	Executed         []string           `json:"executed,omitempty"`
	LastDiscovery    []string           `json:"last_discovery,omitempty"`
	SuggestedTargets []SuggestedTarget  `json:"suggested_targets,omitempty"`
	LastFindings     []string           `json:"last_findings,omitempty"`
	LastWarnings     []string           `json:"last_warnings,omitempty"`
	FamilyScores     map[string]float64 `json:"family_scores,omitempty"`
	CauseScores      map[string]float64 `json:"cause_scores,omitempty"`
	History          []ProbeRecord      `json:"history,omitempty"`
}

type ProbeRecord struct {
	Timestamp string   `json:"timestamp"`
	Command   string   `json:"command,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Findings  []string `json:"findings,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type SuggestedTarget struct {
	Family    string `json:"family"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Command   string `json:"command"`
}

func LoadState(path string) (State, error) {
	if strings.TrimSpace(path) == "" {
		return State{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var state State
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func SaveState(path string, state State) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func UpdateState(state State, query string, response models.Response) State {
	normalized := normalizeQuery(query)
	var updated []StateThread
	thread := PreviewThread(StateThread{}, query, response)
	updated = append(updated, thread)
	for _, existing := range state.Threads {
		if normalizeQuery(existing.Query) == normalized {
			thread = PreviewThread(existing, query, response)
			continue
		}
		updated = append(updated, existing)
		if len(updated) >= 10 {
			break
		}
	}
	updated[0] = thread
	state.Threads = updated
	return state
}

func PreviewThread(existing StateThread, query string, response models.Response) StateThread {
	thread := StateThread{
		Query:            query,
		IntentID:         response.IntentID,
		LastCommand:      response.Command,
		LastDiscovery:    append([]string{}, response.Discovery...),
		SuggestedTargets: parseSuggestedTargets(response.NextSteps),
		LastFindings:     append([]string{}, response.Findings...),
		LastWarnings:     append([]string{}, response.Warnings...),
		FamilyScores:     map[string]float64{},
		CauseScores:      map[string]float64{},
	}
	record := newProbeRecord(response)
	if record.Timestamp != "" {
		thread.History = []ProbeRecord{record}
	}
	if response.Command != "" {
		thread.Executed = append(thread.Executed, response.Command)
	}

	thread.Executed = appendUniqueStrings(thread.Executed, existing.Executed...)
	thread.LastDiscovery = appendUniqueStrings(thread.LastDiscovery, existing.LastDiscovery...)
	thread.SuggestedTargets = mergeSuggestedTargets(existing.SuggestedTargets, thread.SuggestedTargets)
	thread.LastFindings = appendUniqueStrings(thread.LastFindings, existing.LastFindings...)
	thread.LastWarnings = appendUniqueStrings(thread.LastWarnings, existing.LastWarnings...)
	thread.FamilyScores = mergeFamilyScores(existing.FamilyScores, thread.FamilyScores)
	thread.CauseScores = mergeFamilyScores(existing.CauseScores, thread.CauseScores)
	thread.History = append(thread.History, existing.History...)
	thread.ActiveFamily = existing.ActiveFamily
	thread.ActiveTarget = existing.ActiveTarget
	thread.ActiveNamespace = existing.ActiveNamespace
	thread.ActiveContainer = existing.ActiveContainer
	thread.RuntimeHint = existing.RuntimeHint

	mergeCommandContext(&thread, response.Command)
	if strings.TrimSpace(response.ContainerHint) != "" {
		thread.ActiveContainer = strings.TrimSpace(response.ContainerHint)
	}

	applyFamilyScoreDelta(&thread, response)
	applyCauseScoreDelta(&thread, response)
	return thread
}

func ResetState(state State, query string) State {
	if strings.TrimSpace(query) == "" {
		state.Threads = nil
		return state
	}

	normalized := normalizeQuery(query)
	filtered := make([]StateThread, 0, len(state.Threads))
	for _, thread := range state.Threads {
		if normalizeQuery(thread.Query) == normalized {
			continue
		}
		filtered = append(filtered, thread)
	}
	state.Threads = filtered
	return state
}

func UpdateThread(state State, query string, updatedThread StateThread) State {
	normalized := normalizeQuery(query)
	replaced := false
	for i, thread := range state.Threads {
		if normalizeQuery(thread.Query) == normalized {
			state.Threads[i] = updatedThread
			replaced = true
			break
		}
	}
	if !replaced {
		state.Threads = append([]StateThread{updatedThread}, state.Threads...)
		if len(state.Threads) > 10 {
			state.Threads = state.Threads[:10]
		}
	}
	return state
}

func FindThread(state State, query string) (StateThread, bool) {
	normalized := normalizeQuery(query)
	for _, thread := range state.Threads {
		if normalizeQuery(thread.Query) == normalized {
			return thread, true
		}
	}
	return StateThread{}, false
}

func normalizeQuery(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func appendUniqueStrings(values []string, extra ...string) []string {
	out := append([]string{}, values...)
	for _, value := range extra {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen := false
		for _, existing := range out {
			if existing == value {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, value)
		}
	}
	return out
}

func mergeFamilyScores(base, overlay map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func mergeSuggestedTargets(base, overlay []SuggestedTarget) []SuggestedTarget {
	out := append([]SuggestedTarget{}, base...)
	for _, candidate := range overlay {
		seen := false
		for _, existing := range out {
			if existing.Family == candidate.Family && existing.Name == candidate.Name && existing.Command == candidate.Command {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, candidate)
		}
	}
	return out
}

func newProbeRecord(response models.Response) ProbeRecord {
	record := ProbeRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Command:   strings.TrimSpace(response.Command),
		Findings:  append([]string{}, response.Findings...),
		Warnings:  append([]string{}, response.Warnings...),
		Summary:   summarizeProbe(response),
	}
	if record.Command == "" && record.Summary == "" && len(record.Findings) == 0 && len(record.Warnings) == 0 {
		return ProbeRecord{}
	}
	return record
}

func AttachLikelyCauses(response models.Response, thread StateThread) models.Response {
	response.LikelyCauses = topLikelyCauses(thread.CauseScores, 3)
	for _, step := range topCauseFollowUps(thread.CauseScores, 2) {
		response.NextSteps = appendEvidenceSteps(response.NextSteps, step)
	}
	return response
}

func summarizeProbe(response models.Response) string {
	switch {
	case len(response.Findings) > 0:
		return strings.TrimSpace(response.Findings[0])
	case len(response.Warnings) > 0:
		return strings.TrimSpace(response.Warnings[0])
	case strings.TrimSpace(response.Explanation) != "":
		return strings.TrimSpace(response.Explanation)
	case strings.TrimSpace(response.IntentID) != "":
		return "Intent: " + strings.TrimSpace(response.IntentID)
	default:
		return ""
	}
}

func mergeCommandContext(thread *StateThread, command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return
	}
	switch {
	case strings.HasPrefix(command, "systemctl status ") && len(fields) >= 3:
		thread.ActiveFamily = "service"
		thread.ActiveTarget = fields[2]
		thread.ActiveNamespace = ""
		thread.ActiveContainer = ""
	case (strings.HasPrefix(command, "podman ") || strings.HasPrefix(command, "docker ")) && len(fields) >= 1:
		thread.ActiveFamily = "runtime"
		thread.RuntimeHint = fields[0]
		thread.ActiveContainer = ""
		if strings.HasPrefix(command, "podman logs ") || strings.HasPrefix(command, "docker logs ") || strings.HasPrefix(command, "podman inspect ") || strings.HasPrefix(command, "docker inspect ") {
			thread.ActiveTarget = fields[len(fields)-1]
		}
	case strings.HasPrefix(command, "kubectl describe pod ") || strings.HasPrefix(command, "kubectl logs "):
		thread.ActiveFamily = "kubernetes"
		thread.RuntimeHint = ""
		thread.ActiveNamespace = ""
		thread.ActiveContainer = ""
		for i := 0; i < len(fields); i++ {
			if fields[i] == "-n" && i+1 < len(fields) {
				thread.ActiveNamespace = fields[i+1]
			}
		}
		switch {
		case strings.HasPrefix(command, "kubectl describe pod ") && len(fields) >= 4:
			thread.ActiveTarget = fields[len(fields)-1]
		case strings.HasPrefix(command, "kubectl logs ") && len(fields) >= 3:
			for i := 1; i < len(fields); i++ {
				if fields[i] == "-n" {
					i++
					continue
				}
				if strings.HasPrefix(fields[i], "--") {
					continue
				}
				thread.ActiveTarget = fields[i]
				break
			}
		}
	case strings.HasPrefix(command, "kubectl describe pvc "), strings.HasPrefix(command, "kubectl describe secret "), strings.HasPrefix(command, "kubectl describe configmap "), strings.HasPrefix(command, "kubectl describe deployment "), strings.HasPrefix(command, "kubectl describe service "):
		thread.RuntimeHint = ""
		thread.ActiveContainer = ""
		thread.ActiveNamespace = ""
		switch {
		case strings.HasPrefix(command, "kubectl describe pvc "):
			thread.ActiveFamily = "kubernetes-pvc"
		case strings.HasPrefix(command, "kubectl describe secret "):
			thread.ActiveFamily = "kubernetes-secret"
		case strings.HasPrefix(command, "kubectl describe configmap "):
			thread.ActiveFamily = "kubernetes-configmap"
		case strings.HasPrefix(command, "kubectl describe deployment "):
			thread.ActiveFamily = "kubernetes-deployment"
		case strings.HasPrefix(command, "kubectl describe service "):
			thread.ActiveFamily = "kubernetes-service"
		}
		for i := 0; i < len(fields); i++ {
			if fields[i] == "-n" && i+1 < len(fields) {
				thread.ActiveNamespace = fields[i+1]
				i++
				continue
			}
		}
		thread.ActiveTarget = fields[len(fields)-1]
	case strings.HasPrefix(command, "kubectl describe node ") && len(fields) >= 4:
		thread.ActiveFamily = "kubernetes-node"
		thread.ActiveTarget = fields[3]
		thread.ActiveNamespace = ""
		thread.RuntimeHint = ""
		thread.ActiveContainer = ""
	}
}

func applyFamilyScoreDelta(thread *StateThread, response models.Response) {
	if thread.FamilyScores == nil {
		thread.FamilyScores = map[string]float64{}
	}
	family := commandFamily(response.Command)
	if family != "" && family != "other" {
		thread.FamilyScores[family] += 0.2
	}

	text := strings.ToLower(strings.Join(currentEvidenceLines(response), "\n"))
	switch {
	case strings.Contains(text, "unit could not be found"), strings.Contains(text, "unit not found"), strings.Contains(text, "service name is wrong"), strings.Contains(text, "not managed by systemd"):
		thread.FamilyScores["service"] -= 2.0
		thread.FamilyScores["runtime"] += 1.0
		thread.FamilyScores["kubernetes"] += 0.7
	case strings.Contains(text, "unit is in a failed state"), strings.Contains(text, "exit-code"):
		thread.FamilyScores["service"] += 1.2
	case strings.Contains(text, "runtime output was recognized, but no clear container state could be classified"):
		thread.FamilyScores["runtime"] -= 0.6
	case strings.Contains(text, "runtime container list shows non-running or unhealthy containers"), strings.Contains(text, "runtime log evidence"):
		thread.FamilyScores["runtime"] += 1.2
	case strings.Contains(text, "kubernetes evidence: at least one pod is not healthy"), strings.Contains(text, "kubernetes describe evidence"), strings.Contains(text, "kubernetes log evidence"):
		thread.FamilyScores["kubernetes"] += 1.2
	}

	if strings.Contains(text, "runtime probe unavailable") || strings.Contains(text, "docker is not currently installed") || strings.Contains(text, "podman is not currently installed") {
		thread.FamilyScores["runtime"] -= 1.5
	}
	if strings.Contains(text, "kubernetes probe unavailable") || strings.Contains(text, "kubectl is not currently installed") {
		thread.FamilyScores["kubernetes"] -= 1.5
	}

	discoveryText := strings.ToLower(strings.Join(response.Discovery, "\n"))
	if strings.Contains(discoveryText, "found matching systemd unit name") {
		thread.FamilyScores["service"] += 1.5
	}
	if strings.Contains(discoveryText, "no matching systemd unit name") {
		thread.FamilyScores["service"] -= 1.0
	}
	if strings.Contains(discoveryText, "found matching podman container name") || strings.Contains(discoveryText, "found matching docker container name") {
		thread.FamilyScores["runtime"] += 1.5
	}
	if strings.Contains(discoveryText, "no matching podman container name") || strings.Contains(discoveryText, "no matching docker container name") {
		thread.FamilyScores["runtime"] -= 0.8
	}
	if strings.Contains(discoveryText, "found matching kubernetes pod name") {
		thread.FamilyScores["kubernetes"] += 1.6
	}
	if strings.Contains(discoveryText, "no matching kubernetes pod name") {
		thread.FamilyScores["kubernetes"] -= 0.8
	}
}

func applyCauseScoreDelta(thread *StateThread, response models.Response) {
	if thread.CauseScores == nil {
		thread.CauseScores = map[string]float64{}
	}
	lines := currentEvidenceLines(response)
	lines = append(lines, response.Explanation)
	text := strings.ToLower(strings.Join(lines, "\n"))

	switch {
	case strings.Contains(text, "unit could not be found"), strings.Contains(text, "unit not found"), strings.Contains(text, "service name is wrong"), strings.Contains(text, "not managed by systemd"):
		adjustCauseScore(thread.CauseScores, "service_unit_missing", 2.4)
		adjustCauseScore(thread.CauseScores, "service_process_failure", -2.0)
	case strings.Contains(text, "unit is in a failed state"), strings.Contains(text, "exit-code"):
		adjustCauseScore(thread.CauseScores, "service_process_failure", 1.4)
		adjustCauseScore(thread.CauseScores, "service_unit_missing", -1.5)
	}
	if strings.Contains(text, "appears healthy and running") || strings.Contains(text, "does not indicate a current service failure") {
		adjustCauseScore(thread.CauseScores, "service_process_failure", -2.0)
		adjustCauseScore(thread.CauseScores, "service_unit_missing", -1.2)
	}
	if strings.Contains(text, "permission denied") {
		adjustCauseScore(thread.CauseScores, "permission_or_access_denied", 1.8)
		adjustCauseScore(thread.CauseScores, "dependency_database_connectivity", -0.4)
	}
	if strings.Contains(text, "failed to connect to database") || strings.Contains(text, "connect to database") {
		adjustCauseScore(thread.CauseScores, "dependency_database_connectivity", 2.1)
		adjustCauseScore(thread.CauseScores, "permission_or_access_denied", -0.3)
	}
	if strings.Contains(text, "runtime container list shows non-running or unhealthy containers") {
		adjustCauseScore(thread.CauseScores, "runtime_container_failure", 1.5)
		adjustCauseScore(thread.CauseScores, "service_unit_missing", -0.4)
	}
	if strings.Contains(text, "no clear container state could be classified") {
		adjustCauseScore(thread.CauseScores, "runtime_container_failure", -0.8)
	}
	if strings.Contains(text, "crashloopbackoff") || strings.Contains(text, "back-off restarting failed container") {
		adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", 2.2)
		adjustCauseScore(thread.CauseScores, "kubernetes_image_pull", -1.2)
		adjustCauseScore(thread.CauseScores, "kubernetes_scheduling_capacity", -1.2)
	}
	if strings.Contains(text, "imagepullbackoff") || strings.Contains(text, "errimagepull") {
		adjustCauseScore(thread.CauseScores, "kubernetes_image_pull", 2.2)
		adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", -1.5)
		adjustCauseScore(thread.CauseScores, "kubernetes_scheduling_capacity", -1.0)
	}
	if strings.Contains(text, "unschedulable") || strings.Contains(text, "insufficient cpu") || strings.Contains(text, "insufficient memory") {
		adjustCauseScore(thread.CauseScores, "kubernetes_scheduling_capacity", 2.0)
		adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", -1.5)
		adjustCauseScore(thread.CauseScores, "kubernetes_image_pull", -1.2)
	}
	if strings.Contains(text, "all parsed pods are either running or completed") {
		adjustCauseScore(thread.CauseScores, "kubernetes_crashloop", -1.8)
		adjustCauseScore(thread.CauseScores, "kubernetes_image_pull", -1.8)
		adjustCauseScore(thread.CauseScores, "kubernetes_scheduling_capacity", -1.8)
	}
}

func currentEvidenceLines(response models.Response) []string {
	var lines []string
	for _, finding := range response.Findings {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(finding)), "previous finding:") {
			continue
		}
		lines = append(lines, finding)
	}
	for _, warning := range response.Warnings {
		lower := strings.ToLower(strings.TrimSpace(warning))
		if strings.HasPrefix(lower, "previous thread warning:") {
			continue
		}
		lines = append(lines, warning)
	}
	return lines
}

func adjustCauseScore(scores map[string]float64, id string, delta float64) {
	scores[id] += delta
	if scores[id] < 0 {
		scores[id] = 0
	}
}

func topLikelyCauses(scores map[string]float64, limit int) []string {
	type scoredCause struct {
		id    string
		score float64
	}
	var ranked []scoredCause
	for id, score := range scores {
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scoredCause{id: id, score: score})
	}
	if len(ranked) == 0 {
		return nil
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].id < ranked[j].id
		}
		return ranked[i].score > ranked[j].score
	})
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	out := make([]string, 0, len(ranked))
	for _, candidate := range ranked {
		summary, ok := causeSummary(candidate.id)
		if !ok {
			continue
		}
		out = append(out, confidenceLabel(candidate.score)+": "+summary)
	}
	return out
}

func causeSummary(id string) (string, bool) {
	switch id {
	case "service_unit_missing":
		return "the named systemd unit is missing or the workload is not managed by systemd", true
	case "service_process_failure":
		return "the service process is exiting or failing during startup", true
	case "permission_or_access_denied":
		return "the workload is failing because of a permission or access-denied error", true
	case "dependency_database_connectivity":
		return "the workload is failing because it cannot reach its database dependency", true
	case "runtime_container_failure":
		return "a container is present but exiting, restarting, or otherwise unhealthy", true
	case "kubernetes_crashloop":
		return "the pod is crashing repeatedly after startup", true
	case "kubernetes_image_pull":
		return "the cluster cannot pull the required container image", true
	case "kubernetes_scheduling_capacity":
		return "the pod cannot be scheduled because of capacity or placement constraints", true
	default:
		return "", false
	}
}

func confidenceLabel(score float64) string {
	switch {
	case score >= 2.0:
		return "High confidence"
	case score >= 1.0:
		return "Medium confidence"
	default:
		return "Low confidence"
	}
}

func topCauseFollowUps(scores map[string]float64, limit int) []string {
	type scoredCause struct {
		id    string
		score float64
	}
	var ranked []scoredCause
	for id, score := range scores {
		if score <= 0 {
			continue
		}
		ranked = append(ranked, scoredCause{id: id, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].id < ranked[j].id
		}
		return ranked[i].score > ranked[j].score
	})
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}

	steps := make([]string, 0, len(ranked))
	for _, candidate := range ranked {
		if step, ok := causeFollowUp(candidate.id); ok {
			steps = append(steps, step)
		}
	}
	return steps
}

func causeFollowUp(id string) (string, bool) {
	switch id {
	case "service_unit_missing":
		return "Cause follow-up: Confirm the real unit name with `systemctl list-units --type=service | grep <name>` before probing systemd again.", true
	case "service_process_failure":
		return "Cause follow-up: Inspect the startup path in `journalctl -u <service> -n 50 --no-pager` and look for the first fatal error before attempting a restart.", true
	case "permission_or_access_denied":
		return "Cause follow-up: Check ownership, mode bits, SELinux context, and the service account permissions on the referenced files or sockets.", true
	case "dependency_database_connectivity":
		return "Cause follow-up: Verify the database endpoint, credentials source, DNS resolution, and network reachability from the workload host or container.", true
	case "runtime_container_failure":
		return "Cause follow-up: Inspect the container exit reason with `<runtime> logs --tail 100 <name>` and compare it with the configured entrypoint and environment.", true
	case "kubernetes_crashloop":
		return "Cause follow-up: Use `kubectl describe pod` and `kubectl logs --previous` to isolate the first failing container exit in the crash loop.", true
	case "kubernetes_image_pull":
		return "Cause follow-up: Check image name, tag, registry credentials, and node egress reachability before retrying the pod.", true
	case "kubernetes_scheduling_capacity":
		return "Cause follow-up: Review pod events, node allocatable resources, taints, and affinity rules to confirm the scheduling blocker.", true
	default:
		return "", false
	}
}
