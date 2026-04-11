package registry

import (
	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func gitStatusIntent(env models.Environment, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("git")
	command := "git status --short --branch"
	response := models.Response{
		IntentID:       "inspect_git_status",
		Command:        command,
		Explanation:    "Shows the current branch and a compact summary of tracked and untracked file changes in the current repository.",
		ExpectedOutput: "The current branch plus short status entries when the current directory is inside a Git repository.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "git")
	if !env.Commands["git"].Exists {
		appendWarning(&response, "git is not currently installed on this host")
	}
	return response, nil
}

func gitLogIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("git")
	command := "git log --oneline -n 10"
	response := models.Response{
		IntentID:       "inspect_git_log",
		Command:        command,
		Explanation:    "Shows the ten most recent commits in a compact one-line format.",
		ExpectedOutput: "Commit hashes and summaries from newest to oldest when run inside a Git repository.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "git")
	return response, nil
}

func gitDiffIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("git")
	command := "git diff --stat"
	response := models.Response{
		IntentID:       "inspect_git_diff",
		Command:        command,
		Explanation:    "Shows a summary of uncommitted file changes without printing the full patch.",
		ExpectedOutput: "Changed file names plus added and removed line counts for the current working tree.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "git")
	return response, nil
}

func gitPushIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("git")
	command := "git push"
	response := models.Response{
		IntentID:       "push_git_changes",
		Command:        command,
		Explanation:    "Pushes committed changes on the current branch to its configured upstream remote.",
		ExpectedOutput: "Progress lines ending with a summary of the ref that was updated, e.g. 'main -> main'.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps:      []string{"Run 'git status' first to confirm all intended commits are included before pushing."},
		Warnings:       []string{"Avoid 'git push --force' on shared branches — it rewrites remote history and can cause data loss for other contributors."},
	}
	addHelpEvidence(&response, ev, "git")
	return response, nil
}

func gitBranchIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("git")
	command := "git branch -vv"
	response := models.Response{
		IntentID:       "inspect_git_branches",
		Command:        command,
		Explanation:    "Lists local branches with their upstream tracking information and recent commit summaries.",
		ExpectedOutput: "One line per local branch, marking the current branch and showing ahead or behind state when upstreams are configured.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "git")
	return response, nil
}
