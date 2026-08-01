package apply

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dotbrains/ares/internal/plan"
	"github.com/dotbrains/ares/internal/plugins"
	"github.com/dotbrains/ares/internal/reports"
)

type TransactionSummary = reports.TransactionSummary

func BuildTransaction(hardeningPlan plan.Plan) TransactionSummary {
	var summary TransactionSummary
	for _, plugin := range hardeningPlan.Plugins {
		addPluginTransaction(&summary, hardeningPlan, plugin)
	}
	summary.Files = uniqueStrings(summary.Files)
	summary.Commands = uniqueStrings(summary.Commands)
	summary.Backups = uniqueStrings(summary.Backups)
	summary.RollbackSteps = uniqueStrings(summary.RollbackSteps)
	return summary
}

func addPluginTransaction(summary *TransactionSummary, hardeningPlan plan.Plan, plugin plugins.Plugin) {
	if slices.Contains(plugin.Categories, "distro") || strings.HasPrefix(plugin.ID, "provider-") {
		return
	}
	summary.Files = append(summary.Files, plugin.ManagedFiles...)
	summary.Backups = append(summary.Backups, plugin.BackupFiles...)
	summary.RollbackSteps = append(summary.RollbackSteps, plugin.RollbackSteps...)
	switch plugin.ID {
	case "ssh-hardening":
		summary.Commands = append(summary.Commands, "sshd -t", "systemctl reload "+hardeningPlan.Host.SSHService)
	case "firewall-ufw":
		summary.Commands = append(summary.Commands,
			hardeningPlan.Host.PackageManager+" update",
			installCommandString(hardeningPlan.Host.PackageManager, "ufw"),
			"ufw allow "+hardeningPlan.Host.SSHPort+"/tcp",
			"ufw default deny incoming",
			"ufw default allow outgoing",
			"ufw --force enable",
			"ufw status",
		)
		summary.RollbackSteps = append(summary.RollbackSteps, "review firewall rules and run ufw disable if needed")
	case "firewall-firewalld":
		summary.Commands = append(summary.Commands,
			installCommandString(hardeningPlan.Host.PackageManager, "firewalld"),
			"systemctl enable --now firewalld",
			"firewall-cmd --permanent --add-port="+hardeningPlan.Host.SSHPort+"/tcp",
			"firewall-cmd --set-default-zone=public",
			"firewall-cmd --reload",
			"firewall-cmd --list-all",
		)
		summary.RollbackSteps = append(summary.RollbackSteps, "review firewalld ports/services and reload after manual rollback")
	case "firewall-nftables":
		summary.Commands = append(summary.Commands, installCommandString(hardeningPlan.Host.PackageManager, "nftables"), "nft -c -f /etc/nftables.conf", "systemctl enable --now nftables", "nft list ruleset")
	case "fail2ban":
		summary.Commands = append(summary.Commands, installCommandString(hardeningPlan.Host.PackageManager, "fail2ban"), "systemctl enable --now fail2ban")
	case "unattended-upgrades":
		summary.Commands = append(summary.Commands, hardeningPlan.Host.PackageManager+" update", installCommandString(hardeningPlan.Host.PackageManager, "unattended-upgrades"))
	case "dnf-automatic":
		summary.Commands = append(summary.Commands, installCommandString(hardeningPlan.Host.PackageManager, "dnf-automatic"), "systemctl enable --now dnf-automatic.timer")
	case "pacman-upgrade":
		summary.Commands = append(summary.Commands, "pacman -Syu --noconfirm")
	case "zypper-patches":
		summary.Commands = append(summary.Commands, "zypper --non-interactive patch")
	case "apk-upgrade":
		summary.Commands = append(summary.Commands, "apk update", "apk upgrade")
	case "sysctl-baseline":
		summary.Commands = append(summary.Commands, "sysctl --system")
	case "web-profile":
		addWebTransaction(summary, hardeningPlan)
	case "strict-profile":
	default:
		if plugin.Kind == "custom" {
			addCustomTransaction(summary, plugin)
		}
	}
}

func addWebTransaction(summary *TransactionSummary, hardeningPlan plan.Plan) {
	switch hardeningPlan.Host.FirewallBackend {
	case "firewalld":
		summary.Commands = append(summary.Commands, "firewall-cmd --permanent --add-service=http", "firewall-cmd --permanent --add-service=https", "firewall-cmd --reload", "firewall-cmd --list-all")
	case "nftables":
		summary.Files = append(summary.Files, "/etc/nftables.conf")
		summary.Backups = append(summary.Backups, "/etc/nftables.conf")
		summary.Commands = append(summary.Commands, "nft -c -f /etc/nftables.conf", "nft -f /etc/nftables.conf", "nft list ruleset")
	case "ufw":
		summary.Commands = append(summary.Commands, "ufw allow 80/tcp", "ufw allow 443/tcp", "ufw status")
	}
}

func addCustomTransaction(summary *TransactionSummary, plugin plugins.Plugin) {
	if plugin.Probe != "" {
		summary.Commands = append(summary.Commands, "custom "+plugin.ID+" probe: "+plugin.Probe)
	}
	if plugin.Apply != "" {
		summary.Commands = append(summary.Commands, "custom "+plugin.ID+" apply: "+plugin.Apply)
	}
	if plugin.Verify != "" {
		summary.Commands = append(summary.Commands, "custom "+plugin.ID+" verify: "+plugin.Verify)
	}
	if plugin.Rollback != "" {
		summary.RollbackSteps = append(summary.RollbackSteps, "custom "+plugin.ID+" rollback: "+plugin.Rollback)
	}
}

func installCommandString(packageManager string, packages ...string) string {
	name, args, err := installCommand(packageManager, packages...)
	if err != nil {
		return fmt.Sprintf("install %s with unsupported package manager %s", strings.Join(packages, " "), packageManager)
	}
	return name + " " + strings.Join(args, " ")
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
