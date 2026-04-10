package safety

import "testing"

func TestClassifyHighRisk(t *testing.T) {
	cases := []struct {
		command string
	}{
		{"rm -rf /var/log"},
		{"rm -r /tmp/work"},
		{"rm -f important.conf"},
		{"git reset --hard HEAD~1"},
		{"git clean -f"},
		{"git push --force origin main"},
		{"git push -f"},
		{"docker system prune -a"},
		{"podman system prune --all"},
		{"docker rm mycontainer"},
		{"docker rmi myimage:latest"},
		{"kubectl delete pod mypod -n prod"},
		{"kubectl delete deployment myapp"},
		{"terraform destroy"},
		{"terraform apply -auto-approve"},
		{"ansible-playbook site.yml"},
		{"ansible-playbook -i inventory deploy.yml"},
		{"dd if=/dev/zero of=/dev/sda"},
		{"shred -u secrets.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			if got := Classify(tc.command); got != RiskHigh {
				t.Errorf("Classify(%q) = %q, want High", tc.command, got)
			}
		})
	}
}

func TestClassifyMediumRisk(t *testing.T) {
	cases := []struct {
		command string
	}{
		{"systemctl restart nginx"},
		{"systemctl stop sshd"},
		{"systemctl start httpd"},
		{"systemctl enable firewalld"},
		{"systemctl disable chronyd"},
		{"service restart mysql"},
		{"git reset HEAD~1"},
		{"git revert abc123"},
		{"git stash drop"},
		{"docker restart mycontainer"},
		{"podman stop mycontainer"},
		{"kubectl rollout restart deployment/myapp"},
		{"kubectl scale deployment myapp --replicas=0"},
		{"dnf install nginx"},
		{"dnf remove httpd"},
		{"yum update kernel"},
		{"apt install curl"},
		{"apt remove vim"},
		{"rpm -e oldpackage"},
		{"chmod 644 /etc/ssh/sshd_config"},
		{"chown root:root /etc/cron.d/myjob"},
		{"firewall-cmd --add-port=443/tcp"},
		{"setenforce 0"},
		{"helm install myapp ./chart"},
		{"helm upgrade myapp ./chart"},
		{"helm rollback myapp 1"},
		// dry-run / safe sub-commands that contain high-risk substrings
		{"terraform plan"},
		{"ansible-playbook --check site.yml"},
		{"ansible-playbook --syntax-check site.yml"},
		{"helm template myapp ./chart"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			if got := Classify(tc.command); got != RiskMedium {
				t.Errorf("Classify(%q) = %q, want Medium", tc.command, got)
			}
		})
	}
}

func TestClassifyLowRisk(t *testing.T) {
	cases := []struct {
		command string
	}{
		{"df -h"},
		{"du -sh /var"},
		{"ps aux --sort=-%mem | head -n 10"},
		{"ss -ltnp | grep :8080"},
		{"ip addr show"},
		{"systemctl status nginx --no-pager -l"},
		{"journalctl -u nginx -n 50 --no-pager"},
		{"kubectl get pods -n prod"},
		{"kubectl describe pod mypod"},
		{"kubectl logs mypod --previous"},
		{"helm list"},
		{"helm status myapp"},
		{"git status"},
		{"git log --oneline -10"},
		{"git diff HEAD"},
		{"git branch -a"},
		{"rpm -qi bash"},
		{"dpkg -s curl"},
		{"find /var -type f -size +1G"},
		{"grep -Rni error /var/log"},
		{"tar -tf archive.tar.gz"},
		{"ping -c 4 8.8.8.8"},
		{"curl -I https://example.com"},
		{"tail -n 100 /var/log/messages"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			if got := Classify(tc.command); got != RiskLow {
				t.Errorf("Classify(%q) = %q, want Low", tc.command, got)
			}
		})
	}
}

// TestClassifySafeOverridePrecedence verifies that safe dry-run variants
// of otherwise high-risk tools are never classified as High.
func TestClassifySafeOverridePrecedence(t *testing.T) {
	cases := map[string]string{
		"terraform plan -out=tfplan":            RiskMedium,
		"ansible-playbook --check deploy.yml":   RiskMedium,
		"ansible-playbook --syntax-check a.yml": RiskMedium,
		"helm template myapp ./charts/myapp":    RiskMedium,
	}
	for cmd, want := range cases {
		if got := Classify(cmd); got != want {
			t.Errorf("Classify(%q) = %q, want %q", cmd, got, want)
		}
	}
}
