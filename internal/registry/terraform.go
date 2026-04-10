package registry

import (
	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

func terraformVersionIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("terraform")
	command := "terraform version"
	response := models.Response{
		IntentID:       "inspect_terraform_version",
		Command:        command,
		Explanation:    "Shows the installed Terraform CLI version and provider lockfile compatibility information.",
		ExpectedOutput: "Terraform version information for the local CLI and sometimes provider or platform metadata.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "terraform")
	return response, nil
}

func terraformFmtCheckIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("terraform")
	command := "terraform fmt -check -recursive"
	response := models.Response{
		IntentID:       "validate_terraform_formatting",
		Command:        command,
		Explanation:    "Checks whether Terraform files are correctly formatted without rewriting them.",
		ExpectedOutput: "No output when formatting is already correct, or file paths that need formatting when changes are required.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "terraform")
	return response, nil
}

func terraformValidateIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("terraform")
	command := "terraform validate"
	response := models.Response{
		IntentID:       "validate_terraform_configuration",
		Command:        command,
		Explanation:    "Checks Terraform configuration syntax and internal consistency without applying infrastructure changes.",
		ExpectedOutput: "A success message when the configuration is valid, or validation errors describing broken references or invalid blocks.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "terraform")
	return response, nil
}

func terraformPlanIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("terraform")
	command := "terraform plan"
	response := models.Response{
		IntentID:       "preview_terraform_plan",
		Command:        command,
		Explanation:    "Builds an execution plan showing what Terraform would add, change, or destroy without applying those changes.",
		ExpectedOutput: "A detailed execution plan showing proposed resource changes and whether the configuration is up to date.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "terraform")
	return response, nil
}

func terraformWorkspaceListIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("terraform")
	command := "terraform workspace list"
	response := models.Response{
		IntentID:       "inspect_terraform_workspaces",
		Command:        command,
		Explanation:    "Lists available Terraform workspaces and marks the current active workspace.",
		ExpectedOutput: "A workspace list with an asterisk next to the current workspace.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "terraform")
	return response, nil
}

func terraformStateListIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("terraform")
	command := "terraform state list"
	response := models.Response{
		IntentID:       "inspect_terraform_state",
		Command:        command,
		Explanation:    "Lists resource addresses tracked in the current Terraform state without modifying that state.",
		ExpectedOutput: "One Terraform resource address per line when state is available in the current working directory or backend.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "terraform")
	return response, nil
}
