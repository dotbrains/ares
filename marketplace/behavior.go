package marketplace

import "fmt"

var knownBehaviors = map[string]bool{
	"distro":            true,
	"provider-advisory": true,
	"ssh-hardening":     true,
	"firewall":          true,
	"fail2ban":          true,
	"security-updates":  true,
	"sysctl":            true,
	"web-profile":       true,
	"strict-profile":    true,
}

var knownBehaviorVariants = map[string]map[string]bool{
	"firewall": {
		"ufw":       true,
		"firewalld": true,
		"nftables":  true,
	},
	"security-updates": {
		"apt":           true,
		"dnf-automatic": true,
		"pacman":        true,
		"zypper":        true,
		"apk":           true,
	},
	"web-profile": {
		"web": true,
	},
}

var knownVerifiers = map[string]bool{
	"path":              true,
	"command":           true,
	"firewall":          true,
	"provider-advisory": true,
	"custom":            true,
	"none":              true,
}

func KnownBehavior(behavior string) bool {
	return knownBehaviors[behavior]
}

func KnownBehaviorVariant(behavior string, variant string) bool {
	variants := knownBehaviorVariants[behavior]
	return variants != nil && variants[variant]
}

func KnownVerifier(verifier string) bool {
	return knownVerifiers[verifier]
}

func ValidateBehavior(plugin Plugin) error {
	if !KnownBehavior(plugin.Behavior) {
		return fmt.Errorf("plugin %s has unknown behavior %q", plugin.ID, plugin.Behavior)
	}
	if plugin.BehaviorVariant != "" && !KnownBehaviorVariant(plugin.Behavior, plugin.BehaviorVariant) {
		return fmt.Errorf("plugin %s has unknown behavior variant %q for behavior %q", plugin.ID, plugin.BehaviorVariant, plugin.Behavior)
	}
	if plugin.Verifier != "" && !KnownVerifier(plugin.Verifier) {
		return fmt.Errorf("plugin %s has unknown verifier %q", plugin.ID, plugin.Verifier)
	}
	return nil
}
