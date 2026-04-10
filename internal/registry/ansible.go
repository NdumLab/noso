package registry

import (
	"fmt"
	"strings"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

func ansibleVersionIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ansible")
	command := "ansible --version"
	response := models.Response{
		IntentID:       "inspect_ansible_version",
		Command:        command,
		Explanation:    "Shows the installed Ansible version, Python runtime, and module search paths.",
		ExpectedOutput: "Version information for the Ansible CLI plus environment details such as config and Python paths.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ansible")
	return response, nil
}

func ansibleInventoryIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ansible-inventory")
	command := "ansible-inventory --list"
	if !ev.Exists {
		ev = collector.Lookup("ansible")
		command = "ansible-inventory --list"
	}
	response := models.Response{
		IntentID:       "inspect_ansible_inventory",
		Command:        command,
		Explanation:    "Prints the resolved Ansible inventory so you can confirm groups, hosts, and variables before any playbook run.",
		ExpectedOutput: "Inventory data in JSON or YAML form, depending on configuration and output options.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	if collector.Lookup("ansible-inventory").Exists {
		addHelpEvidence(&response, collector.Lookup("ansible-inventory"), "ansible-inventory")
	} else {
		appendWarning(&response, "ansible-inventory is not currently installed on this host")
		appendWarning(&response, "inventory inspection assumes the standalone ansible-inventory helper is available")
	}
	return response, nil
}

func ansibleSyntaxCheckIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ansible-playbook")
	playbook := extractPlaybook(query)
	command := fmt.Sprintf("ansible-playbook --syntax-check %s", playbook)
	response := models.Response{
		IntentID:       "validate_ansible_playbook_syntax",
		Command:        command,
		Explanation:    fmt.Sprintf("Checks the syntax of playbook %s without running any tasks on target hosts.", playbook),
		ExpectedOutput: "No output or a success message when syntax is valid, or parser and YAML errors when the playbook is invalid.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ansible-playbook")
	return response, nil
}

func ansibleCheckModeIntent(query string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ansible-playbook")
	playbook := extractPlaybook(query)
	command := fmt.Sprintf("ansible-playbook --check %s", playbook)
	response := models.Response{
		IntentID:       "preview_ansible_check_mode",
		Command:        command,
		Explanation:    fmt.Sprintf("Runs playbook %s in Ansible check mode so tasks report what they would change without making most changes.", playbook),
		ExpectedOutput: "A play recap and task output showing intended changes, skipped tasks, and modules that do or do not support check mode.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ansible-playbook")
	return response, nil
}

func extractPlaybook(query string) string {
	lower := strings.ToLower(query)
	if idx := strings.Index(lower, "playbook "); idx >= 0 {
		fields := strings.Fields(query[idx+len("playbook "):])
		if len(fields) > 0 {
			return strings.Trim(fields[0], "`\"'")
		}
	}
	return "site.yml"
}
