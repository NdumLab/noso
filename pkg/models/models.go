package models

type CommandInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path,omitempty"`
	Type   string `json:"type,omitempty"`
	Exists bool   `json:"exists"`
}

type Environment struct {
	OSID           string                 `json:"os_id"`
	VersionID      string                 `json:"version_id"`
	PrettyName     string                 `json:"pretty_name"`
	Distro         string                 `json:"distro"`          // normalised family: rhel, debian, fedora, arch, unknown
	PackageManager string                 `json:"package_manager"` // dnf, apt, pacman, zypper, unknown
	Shell          string                 `json:"shell"`
	IsRHEL9        bool                   `json:"is_rhel9"`
	KubeConfig     string                 `json:"kube_config,omitempty"`
	KubeContext    string                 `json:"kube_context,omitempty"`
	KubeServer     string                 `json:"kube_server,omitempty"`
	Commands       map[string]CommandInfo `json:"commands"`
}

type Response struct {
	IntentID       string   `json:"intent_id"`
	Command        string   `json:"command,omitempty"`
	Explanation    string   `json:"explanation"`
	ExpectedOutput string   `json:"expected_output"`
	Risk           string   `json:"risk"`
	Confidence     string   `json:"confidence"`
	AdoptedTarget  string   `json:"adopted_target,omitempty"`
	ContainerHint  string   `json:"container_hint,omitempty"`
	Discovery      []string `json:"discovery,omitempty"`
	Findings       []string `json:"findings,omitempty"`
	LikelyCauses   []string `json:"likely_causes,omitempty"`
	VerifiedFrom   []string `json:"verified_from,omitempty"`
	NextSteps      []string `json:"next_steps,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

type AuditRecord struct {
	Timestamp string   `json:"timestamp"`
	Query     string   `json:"query"`
	Response  Response `json:"response"`
}

type LLMInterpretHints struct {
	MaxCandidates      int  `json:"max_candidates"`
	AllowClarification bool `json:"allow_clarification"`
}

type LLMInterpretRequest struct {
	Version     string            `json:"version"`
	Query       string            `json:"query"`
	Mode        string            `json:"mode"`
	Environment LLMEnvironment    `json:"environment"`
	Hints       LLMInterpretHints `json:"hints"`
}

type LLMEnvironment struct {
	OSFamily       string   `json:"os_family"`
	PackageManager string   `json:"package_manager"`
	Shell          string   `json:"shell"`
	AvailableTools []string `json:"available_tools"`
	IsRHEL9        bool     `json:"is_rhel9"`
}

type LLMIntentCandidate struct {
	Intent     string  `json:"intent"`
	Target     string  `json:"target,omitempty"`
	Namespace  string  `json:"namespace,omitempty"`
	ToolHint   string  `json:"tool_hint,omitempty"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning,omitempty"`
}

type LLMInterpretResponse struct {
	Status                string               `json:"status"`
	NeedsClarification    bool                 `json:"needs_clarification"`
	ClarificationQuestion string               `json:"clarification_question,omitempty"`
	Candidates            []LLMIntentCandidate `json:"candidates,omitempty"`
}
