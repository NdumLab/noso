package registry

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NdumLab/noso/internal/evidence"
	"github.com/NdumLab/noso/internal/safety"
	"github.com/NdumLab/noso/pkg/models"
)

func diskFreeIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("df")
	command := "df -h"
	response := models.Response{
		IntentID:       "inspect_disk_free_space",
		Command:        command,
		Explanation:    "Shows mounted filesystem usage and available space in human-readable units.",
		ExpectedOutput: "Each mounted filesystem with total size, used space, available space, and mount point.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "df")
	return response, nil
}

func processIntent(normalized string, collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ps")
	command := "ps aux --sort=-%mem | head -n 10"
	explanation := "Shows the top ten processes by memory usage, including owner, PID, CPU, memory, and command."
	if strings.Contains(normalized, "cpu") {
		command = "ps aux --sort=-%cpu | head -n 10"
		explanation = "Shows the top ten processes by CPU usage, which is useful when the host feels busy or slow."
	}
	response := models.Response{
		IntentID:       "inspect_top_processes",
		Command:        command,
		Explanation:    explanation,
		ExpectedOutput: "A process table sorted by the requested resource, with the busiest processes at the top.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ps")
	return response, nil
}

func ipAddressIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("ip")
	command := "ip addr show"
	response := models.Response{
		IntentID:       "inspect_ip_addresses",
		Command:        command,
		Explanation:    "Shows network interfaces, link states, MAC addresses, and assigned IP addresses on the host.",
		ExpectedOutput: "Per-interface blocks with state, MTU, MAC address, and IPv4 or IPv6 assignments.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ip")
	return response, nil
}

func pingIntent(query string, collector evidence.Collector) (models.Response, error) {
	host := "localhost"
	if matches := hostRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 2 {
		host = matches[1]
	}
	ev := collector.Lookup("ping")
	command := fmt.Sprintf("ping -c 4 %s", host)
	response := models.Response{
		IntentID:       "inspect_host_reachability",
		Command:        command,
		Explanation:    fmt.Sprintf("Sends four ICMP echo requests to %s to test basic reachability and latency.", host),
		ExpectedOutput: "Four replies with latency or a timeout or unreachable message if the host cannot be reached.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ping")
	return response, nil
}

func curlHeadIntent(query string, collector evidence.Collector) (models.Response, error) {
	url := "https://example.com"
	if matches := urlRegex.FindString(query); matches != "" {
		url = matches
	}
	ev := collector.Lookup("curl")
	command := fmt.Sprintf("curl -I %s", url)
	response := models.Response{
		IntentID:       "inspect_http_headers",
		Command:        command,
		Explanation:    "Fetches only HTTP response headers so you can confirm status codes and basic metadata without downloading the body.",
		ExpectedOutput: "HTTP status line plus headers such as server, date, content type, and cache-related fields.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "curl")
	return response, nil
}

func tailFileIntent(query string, collector evidence.Collector) (models.Response, error) {
	path := "/var/log/messages"
	if matches := fileRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 2 {
		path = matches[1]
	}
	ev := collector.Lookup("tail")
	command := fmt.Sprintf("tail -n 100 %s", path)
	response := models.Response{
		IntentID:       "inspect_file_tail",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the last 100 lines of %s, which is useful for recent log or file inspection.", path),
		ExpectedOutput: "The most recent 100 lines from the requested file.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "tail")
	return response, nil
}

func tarListIntent(query string, collector evidence.Collector) (models.Response, error) {
	archive := "archive.tar.gz"
	if matches := tarRegex.FindStringSubmatch(query); len(matches) == 2 {
		archive = matches[1]
	}
	ev := collector.Lookup("tar")
	command := fmt.Sprintf("tar -tf %s", archive)
	response := models.Response{
		IntentID:       "inspect_archive_contents",
		Command:        command,
		Explanation:    fmt.Sprintf("Lists the contents of %s without extracting it.", archive),
		ExpectedOutput: "One archive member per line, showing the paths stored inside the tar archive.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "tar")
	return response, nil
}

func portIntent(query string, collector evidence.Collector) (models.Response, error) {
	matches := portRegex.FindStringSubmatch(strings.ToLower(query))
	port := "8080"
	if len(matches) == 2 {
		if numericPort, err := strconv.Atoi(matches[1]); err == nil && numericPort > 0 && numericPort <= 65535 {
			port = matches[1]
		}
	}

	ev := collector.Lookup("ss")
	command := fmt.Sprintf("ss -ltnp | grep :%s", port)
	response := models.Response{
		IntentID:       "inspect_port_listener",
		Command:        command,
		Explanation:    fmt.Sprintf("Uses ss to list listening TCP sockets, then filters for port %s to identify the owning process.", port),
		ExpectedOutput: fmt.Sprintf("A matching line with the local address, state, and PID/program if something is listening on %s.", port),
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "ss")
	return response, nil
}

func serviceLogsIntent(query string, collector evidence.Collector) (models.Response, error) {
	service := "nginx"
	matches := serviceLogsRegex.FindStringSubmatch(strings.ToLower(query))
	if len(matches) == 2 {
		service = matches[1]
	}

	ev := collector.Lookup("journalctl")
	command := fmt.Sprintf("journalctl -u %s -n 50 --no-pager", service)
	response := models.Response{
		IntentID:       "inspect_service_logs",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the last 50 journal lines for the %s service without opening a pager.", service),
		ExpectedOutput: "Recent log lines, timestamps, and error messages for the requested service unit.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "journalctl")
	return response, nil
}

func serviceIntent(query string, collector evidence.Collector) (models.Response, error) {
	service := "nginx"
	matches := serviceRegex.FindStringSubmatch(strings.ToLower(query))
	if len(matches) == 2 {
		service = matches[1]
	}

	ev := collector.Lookup("systemctl")
	command := fmt.Sprintf("systemctl status %s --no-pager -l", service)
	response := models.Response{
		IntentID:       "inspect_service_status",
		Command:        command,
		Explanation:    fmt.Sprintf("Shows the current unit state, recent log lines, and failure context for the %s service without opening a pager.", service),
		ExpectedOutput: "Loaded/active state, recent journal lines, exit codes, and the main PID when the service exists.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "systemctl")
	return response, nil
}

func diskUsageIntent(query string, collector evidence.Collector) (models.Response, error) {
	path := "/var"
	if matches := pathRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 2 {
		path = matches[1]
	}

	ev := collector.Lookup("du")
	command := fmt.Sprintf("du -sh %s", path)
	response := models.Response{
		IntentID:       "inspect_directory_size",
		Command:        command,
		Explanation:    fmt.Sprintf("Summarizes total disk usage for %s in a human-readable format.", path),
		ExpectedOutput: "One line with the total size and the requested path.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "du")
	return response, nil
}

func largeFilesIntent(query string, collector evidence.Collector) (models.Response, error) {
	size := "1G"
	path := "/var"
	if matches := findLargeRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 3 {
		size = strings.ToUpper(matches[1])
		path = matches[2]
	}

	ev := collector.Lookup("find")
	command := fmt.Sprintf("find %s -type f -size +%s", path, size)
	response := models.Response{
		IntentID:       "find_large_files",
		Command:        command,
		Explanation:    fmt.Sprintf("Searches %s for regular files larger than %s using find's native size filter.", path, size),
		ExpectedOutput: "A path per matching file, one file per line.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "find")
	return response, nil
}

func packageInfoIntent(query string, env models.Environment, collector evidence.Collector) (models.Response, error) {
	packageName := "bash"
	if matches := packageRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 2 {
		packageName = matches[1]
	}

	type pkgSpec struct {
		tool, command, explanation, expectedOutput string
	}
	var spec pkgSpec
	switch env.PackageManager {
	case "apt":
		spec = pkgSpec{
			tool:           "dpkg",
			command:        fmt.Sprintf("dpkg -s %s", packageName),
			explanation:    fmt.Sprintf("Displays locally installed package metadata for %s from the dpkg database.", packageName),
			expectedOutput: "Package status, version, and description when installed; 'not installed' otherwise.",
		}
	case "pacman":
		spec = pkgSpec{
			tool:           "pacman",
			command:        fmt.Sprintf("pacman -Qi %s", packageName),
			explanation:    fmt.Sprintf("Displays locally installed package metadata for %s from the pacman database.", packageName),
			expectedOutput: "Package metadata fields when installed.",
		}
	case "zypper":
		spec = pkgSpec{
			tool:           "zypper",
			command:        fmt.Sprintf("zypper info %s", packageName),
			explanation:    fmt.Sprintf("Displays package information for %s from the zypper database.", packageName),
			expectedOutput: "Package details including version, description, and repository source.",
		}
	default: // dnf / rpm
		spec = pkgSpec{
			tool:           "rpm",
			command:        fmt.Sprintf("rpm -qi %s", packageName),
			explanation:    fmt.Sprintf("Displays locally installed package metadata for %s, including version, release, vendor, and summary.", packageName),
			expectedOutput: "RPM metadata fields when the package is installed, or an error if it is not installed.",
		}
	}

	ev := collector.Lookup(spec.tool)
	response := models.Response{
		IntentID:       "inspect_package_info",
		Command:        spec.command,
		Explanation:    spec.explanation,
		ExpectedOutput: spec.expectedOutput,
		Risk:           safety.Classify(spec.command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, spec.tool)
	return response, nil
}

func serviceControlIntent(query string, collector evidence.Collector) (models.Response, error) {
	action := "restart"
	service := "nginx"
	if matches := serviceControlRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 3 {
		action = matches[1]
		service = matches[2]
	}

	ev := collector.Lookup("systemctl")
	command := fmt.Sprintf("systemctl %s %s", action, service)

	var explanation string
	switch action {
	case "restart":
		explanation = fmt.Sprintf("Stops and immediately restarts the %s service. Any existing connections handled by the service will be dropped during the restart window.", service)
	case "start":
		explanation = fmt.Sprintf("Starts the %s service if it is not already running. Has no effect if the service is already active.", service)
	case "stop":
		explanation = fmt.Sprintf("Stops the %s service. It will not restart automatically unless a watchdog or dependency restarts it.", service)
	case "enable":
		explanation = fmt.Sprintf("Marks %s to start automatically at boot. Does not start the service immediately.", service)
	case "disable":
		explanation = fmt.Sprintf("Prevents %s from starting automatically at boot. Does not stop the service if it is currently running.", service)
	default:
		explanation = fmt.Sprintf("Runs systemctl %s on the %s unit.", action, service)
	}

	response := models.Response{
		IntentID:       "control_service",
		Command:        command,
		Explanation:    explanation,
		ExpectedOutput: fmt.Sprintf("No output on success. Run 'systemctl status %s' to confirm the new state.", service),
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps:      []string{fmt.Sprintf("Run 'systemctl status %s --no-pager -l' to confirm the service reached the expected state.", service)},
	}
	addHelpEvidence(&response, ev, "systemctl")
	return response, nil
}

func packageInstallIntent(query string, env models.Environment, collector evidence.Collector) (models.Response, error) {
	pkg := "curl"
	if matches := packageInstallRegex.FindStringSubmatch(strings.ToLower(query)); len(matches) == 2 {
		pkg = matches[1]
	}

	type pkgSpec struct {
		tool, command, explanation string
	}
	var spec pkgSpec
	switch env.PackageManager {
	case "apt":
		spec = pkgSpec{
			tool:        "apt-get",
			command:     fmt.Sprintf("apt-get install -y %s", pkg),
			explanation: fmt.Sprintf("Installs %s and its dependencies from the configured apt repositories.", pkg),
		}
	case "pacman":
		spec = pkgSpec{
			tool:        "pacman",
			command:     fmt.Sprintf("pacman -S --noconfirm %s", pkg),
			explanation: fmt.Sprintf("Installs %s from the pacman package database.", pkg),
		}
	case "zypper":
		spec = pkgSpec{
			tool:        "zypper",
			command:     fmt.Sprintf("zypper install -y %s", pkg),
			explanation: fmt.Sprintf("Installs %s from the configured zypper repositories.", pkg),
		}
	default: // dnf
		spec = pkgSpec{
			tool:        "dnf",
			command:     fmt.Sprintf("dnf install -y %s", pkg),
			explanation: fmt.Sprintf("Installs %s and its dependencies from the configured dnf repositories.", pkg),
		}
	}

	ev := collector.Lookup(spec.tool)
	response := models.Response{
		IntentID:       "install_package",
		Command:        spec.command,
		Explanation:    spec.explanation,
		ExpectedOutput: fmt.Sprintf("Download progress, dependency resolution, and a summary confirming %s is installed.", pkg),
		Risk:           safety.Classify(spec.command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps:      []string{fmt.Sprintf("Verify with: command -v %s", pkg)},
	}
	addHelpEvidence(&response, ev, spec.tool)
	return response, nil
}

func dnsLookupIntent(query string, collector evidence.Collector) (models.Response, error) {
	host := "example.com"
	if matches := dnsHostExtract.FindStringSubmatch(strings.ToLower(query)); len(matches) == 2 && matches[1] != "" {
		host = matches[1]
	}

	ev := collector.Lookup("dig")
	command := fmt.Sprintf("dig +short %s", host)
	if !ev.Exists {
		evNs := collector.Lookup("nslookup")
		if evNs.Exists {
			command = fmt.Sprintf("nslookup %s", host)
			ev = evNs
		}
	}

	response := models.Response{
		IntentID:       "inspect_dns_lookup",
		Command:        command,
		Explanation:    fmt.Sprintf("Resolves the DNS records for %s and returns the associated IP addresses.", host),
		ExpectedOutput: fmt.Sprintf("One IP address per line for %s, or an empty response if the name does not resolve.", host),
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "dig")
	return response, nil
}

func cronListIntent(collector evidence.Collector) (models.Response, error) {
	ev := collector.Lookup("crontab")
	command := "crontab -l"
	response := models.Response{
		IntentID:       "inspect_cron_jobs",
		Command:        command,
		Explanation:    "Lists the cron jobs for the current user. System-wide jobs live in /etc/cron.d/ and /etc/crontab.",
		ExpectedOutput: "One cron expression per scheduled job, or 'no crontab for <user>' if none are configured.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
		NextSteps:      []string{"To see system-wide jobs: cat /etc/crontab && ls /etc/cron.d/"},
	}
	addHelpEvidence(&response, ev, "crontab")
	return response, nil
}

func grepIntent(query string, collector evidence.Collector) (models.Response, error) {
	needle := "error"
	path := "/var/log"
	if matches := grepRegex.FindStringSubmatch(query); len(matches) == 3 {
		needle = strings.TrimSpace(matches[1])
		path = matches[2]
	}

	ev := collector.Lookup("grep")
	command := fmt.Sprintf("grep -Rni -- %q %s", needle, path)
	response := models.Response{
		IntentID:       "search_text_in_files",
		Command:        command,
		Explanation:    fmt.Sprintf("Recursively searches %s for the text %q, showing matching line numbers without changing any files.", path, needle),
		ExpectedOutput: "Matching file paths with line numbers and the lines that contain the requested text.",
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, "grep")
	return response, nil
}
