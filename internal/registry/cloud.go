package registry

import (
	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func awsVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("aws"),
		"inspect_aws_version",
		"aws --version",
		"Shows the installed AWS CLI version.",
		"AWS CLI version information, usually including bundled Python details.",
		"aws",
	)
}

func awsIdentityIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("aws"),
		"inspect_aws_identity",
		"aws sts get-caller-identity",
		"Shows the currently authenticated AWS identity, including account, ARN, and user or role ID.",
		"JSON identity data for the active AWS credentials and account context.",
		"aws",
	)
}

func awsProfilesIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("aws"),
		"inspect_aws_profiles",
		"aws configure list-profiles",
		"Lists configured AWS CLI profiles so you can confirm what named credential sets are available locally.",
		"A profile name per line for locally configured AWS CLI profiles.",
		"aws",
	)
}

func azVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("az"),
		"inspect_azure_version",
		"az version",
		"Shows the installed Azure CLI version and component details.",
		"JSON or table-like Azure CLI version details for the local installation.",
		"az",
	)
}

func azAccountIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("az"),
		"inspect_azure_account",
		"az account show",
		"Shows the currently selected Azure subscription and tenant context.",
		"JSON account data including subscription ID, tenant ID, and default subscription name.",
		"az",
	)
}

func azSubscriptionsIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("az"),
		"inspect_azure_subscriptions",
		"az account list --output table",
		"Lists available Azure subscriptions and highlights which one is currently active.",
		"A subscription table with names, states, tenant IDs, and default subscription markers.",
		"az",
	)
}

func gcloudVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("gcloud"),
		"inspect_gcloud_version",
		"gcloud version",
		"Shows the installed Google Cloud CLI version and component details.",
		"Google Cloud CLI version information for the local installation.",
		"gcloud",
	)
}

func gcloudAccountIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("gcloud"),
		"inspect_gcloud_account",
		"gcloud auth list",
		"Lists authenticated Google Cloud accounts and marks the currently active account.",
		"An account list showing active and available authenticated identities.",
		"gcloud",
	)
}

func gcloudProjectIntent(collector evidence.Collector) (models.Response, error) {
	return simpleCloudIntent(
		collector.Lookup("gcloud"),
		"inspect_gcloud_project",
		"gcloud config get-value project",
		"Shows the currently selected Google Cloud project in the active gcloud configuration.",
		"The active project ID or an empty result when no project is configured.",
		"gcloud",
	)
}

func simpleCloudIntent(ev evidence.Evidence, intentID, command, explanation, expected, evidenceName string) (models.Response, error) {
	response := models.Response{
		IntentID:       intentID,
		Command:        command,
		Explanation:    explanation,
		ExpectedOutput: expected,
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, evidenceName)
	return response, nil
}
