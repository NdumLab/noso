package registry

import (
	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/pkg/models"
)

// confidenceFor scores evidence quality from multiple local sources.
// Each source that confirms the command adds to the score:
//   - command exists on PATH        → +1
//   - help snippet captured         → +1
//   - man page available            → +1
//   - shell completion script found → +1
func confidenceFor(ev evidence.Evidence) string {
	if !ev.Exists {
		return "Low"
	}
	score := 1 // exists
	if len(ev.HelpSnippet) > 0 {
		score++
	}
	if ev.ManAvailable {
		score++
	}
	if ev.CompletionPath != "" {
		score++
	}
	switch {
	case score >= 3:
		return "High"
	case score >= 2:
		return "Medium"
	default:
		return "Low"
	}
}

func addHelpEvidence(response *models.Response, ev evidence.Evidence, command string) {
	// VerificationSources already contains --help when a snippet was captured,
	// so we only add a warning for missing commands here.
	if !ev.Exists {
		appendWarning(response, command+" is not currently installed on this host")
	}
}

func appendWarning(response *models.Response, warning string) {
	for _, existing := range response.Warnings {
		if existing == warning {
			return
		}
	}
	response.Warnings = append(response.Warnings, warning)
}
