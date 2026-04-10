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
	Commands       map[string]CommandInfo `json:"commands"`
}

type Response struct {
	IntentID       string   `json:"intent_id"`
	Command        string   `json:"command,omitempty"`
	Explanation    string   `json:"explanation"`
	ExpectedOutput string   `json:"expected_output"`
	Risk           string   `json:"risk"`
	Confidence     string   `json:"confidence"`
	VerifiedFrom   []string `json:"verified_from,omitempty"`
	NextSteps      []string `json:"next_steps,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

type AuditRecord struct {
	Timestamp string   `json:"timestamp"`
	Query     string   `json:"query"`
	Response  Response `json:"response"`
}
