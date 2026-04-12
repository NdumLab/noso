package incident

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type AlertmanagerPayload struct {
	Alerts            []AlertmanagerAlert `json:"alerts"`
	CommonLabels      map[string]string   `json:"commonLabels,omitempty"`
	CommonAnnotations map[string]string   `json:"commonAnnotations,omitempty"`
	ExternalURL       string              `json:"externalURL,omitempty"`
}

type AlertmanagerAlert struct {
	Status      string            `json:"status,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Fingerprint string            `json:"fingerprint,omitempty"`
}

func LoadAlerts(inputPath string, stdin io.Reader) ([]Alert, error) {
	var data []byte
	var err error
	switch strings.TrimSpace(inputPath) {
	case "", "-":
		data, err = io.ReadAll(stdin)
	default:
		data, err = os.ReadFile(strings.TrimSpace(inputPath))
	}
	if err != nil {
		return nil, err
	}
	return ParseAlerts(data)
}

func ParseAlerts(data []byte) ([]Alert, error) {
	data = bytesTrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("alert payload is empty")
	}

	var many []Alert
	if err := json.Unmarshal(data, &many); err == nil && len(many) > 0 {
		return normalizeAlerts(many), nil
	}

	var one Alert
	if err := json.Unmarshal(data, &one); err == nil && (strings.TrimSpace(one.Query) != "" || strings.TrimSpace(one.Summary) != "" || len(one.Labels) > 0) {
		return normalizeAlerts([]Alert{one}), nil
	}

	var payload AlertmanagerPayload
	if err := json.Unmarshal(data, &payload); err == nil && len(payload.Alerts) > 0 {
		return alertsFromAlertmanager(payload), nil
	}

	return nil, fmt.Errorf("unsupported alert payload format")
}

func alertsFromAlertmanager(payload AlertmanagerPayload) []Alert {
	out := make([]Alert, 0, len(payload.Alerts))
	for _, item := range payload.Alerts {
		labels := mergeStringMaps(payload.CommonLabels, item.Labels)
		annotations := mergeStringMaps(payload.CommonAnnotations, item.Annotations)
		alert := Alert{
			Query:       firstNonEmptyString(annotations["summary"], labels["alertname"]),
			Source:      "alertmanager",
			Severity:    firstNonEmptyString(labels["severity"], item.Status),
			Summary:     firstNonEmptyString(annotations["summary"], annotations["description"], labels["alertname"]),
			Fingerprint: strings.TrimSpace(item.Fingerprint),
			Labels:      labels,
		}
		out = append(out, normalizeAlerts([]Alert{alert})...)
	}
	return out
}

func normalizeAlerts(alerts []Alert) []Alert {
	out := make([]Alert, 0, len(alerts))
	for _, alert := range alerts {
		alert.Query = strings.TrimSpace(alert.Query)
		alert.Source = strings.TrimSpace(alert.Source)
		alert.Severity = normalizeSeverity(alert.Severity)
		alert.Summary = strings.TrimSpace(alert.Summary)
		alert.Fingerprint = strings.TrimSpace(alert.Fingerprint)
		if alert.Query == "" {
			alert.Query = alert.Summary
		}
		out = append(out, alert)
	}
	return out
}

func mergeStringMaps(a, b map[string]string) map[string]string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func bytesTrimSpace(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}
