package system

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dotbrains/ares/internal/plugins"
)

type Host struct {
	OSID            string
	OSName          string
	OSVersion       string
	IDLike          []string
	Provider        string
	PackageManager  string
	InitSystem      string
	FirewallBackend string
	SSHService      string
	SSHPort         string
	RunningOverSSH  bool
	Architecture    string
}

func Detect() (Host, error) {
	root := os.Getenv("ARES_ROOT")
	osReleaseOverride := os.Getenv("ARES_OS_RELEASE") != ""
	osReleasePath := os.Getenv("ARES_OS_RELEASE")
	if osReleasePath == "" {
		osReleasePath = rootPath(root, "/etc/os-release")
	}
	osRelease, err := readOSRelease(osReleasePath)
	if err != nil {
		return Host{}, err
	}

	host := Host{
		OSID:           osRelease["ID"],
		OSName:         osRelease["PRETTY_NAME"],
		OSVersion:      osRelease["VERSION_ID"],
		IDLike:         strings.Fields(osRelease["ID_LIKE"]),
		Provider:       detectProvider(root),
		SSHPort:        detectSSHPort(rootPath(root, "/etc/ssh/sshd_config")),
		RunningOverSSH: os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "",
		Architecture:   runtime.GOARCH,
	}
	probeHostCommands := root == "" && !osReleaseOverride
	host.PackageManager = packageManager(host, probeHostCommands)
	host.InitSystem = initSystem(host, root)
	host.SSHService = sshServiceName(host)
	host.FirewallBackend = firewallBackend(host, probeHostCommands)

	return host, nil
}

func readOSRelease(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), "\"")
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if values["ID"] == "" {
		return nil, errors.New("unable to detect distro: /etc/os-release has no ID")
	}
	return values, nil
}

func detectProvider(root string) string {
	if provider := strings.TrimSpace(os.Getenv("ARES_PROVIDER")); provider != "" {
		return normalizeProvider(provider)
	}
	probeFiles := []string{
		"/sys/class/dmi/id/sys_vendor",
		"/sys/class/dmi/id/product_name",
		"/sys/class/dmi/id/board_vendor",
	}
	var values []string
	for _, path := range probeFiles {
		data, err := os.ReadFile(rootPath(root, path))
		if err == nil {
			values = append(values, string(data))
		}
	}
	return providerFromText(strings.Join(values, "\n"))
}

func providerFromText(value string) string {
	normalized := strings.ToLower(value)
	switch {
	case strings.Contains(normalized, "digitalocean"):
		return "digitalocean"
	case strings.Contains(normalized, "hostinger"):
		return "hostinger"
	case strings.Contains(normalized, "hetzner"):
		return "hetzner"
	case strings.Contains(normalized, "vultr"):
		return "vultr"
	case strings.Contains(normalized, "linode") || strings.Contains(normalized, "akamai"):
		return "linode"
	case strings.Contains(normalized, "ovh"):
		return "ovh"
	case strings.Contains(normalized, "amazon") || strings.Contains(normalized, "lightsail"):
		return "lightsail"
	default:
		return "unknown"
	}
}

func normalizeProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "do", "digital-ocean", "digitalocean":
		return "digitalocean"
	case "aws-lightsail", "amazon-lightsail", "lightsail":
		return "lightsail"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func packageManager(host Host, probeHostCommands bool) string {
	if probeHostCommands {
		if detected := firstCommand("apt-get", "dnf", "yum", "pacman", "zypper", "apk"); detected != "unknown" {
			return detected
		}
	}
	if plugin, ok := distroPlugin(host); ok && plugin.PackageManager != "" {
		return plugin.PackageManager
	}
	return "unknown"
}

func firewallBackend(host Host, probeHostCommands bool) string {
	if probeHostCommands && commandExists("ufw") {
		return "ufw"
	}
	if probeHostCommands && commandExists("firewall-cmd") {
		return "firewalld"
	}
	if probeHostCommands && commandExists("nft") {
		return "nftables"
	}
	if plugin, ok := distroPlugin(host); ok && plugin.FirewallBackend != "" {
		return plugin.FirewallBackend
	}
	return "unknown"
}

func firstCommand(names ...string) string {
	for _, name := range names {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return "unknown"
}

func detectInitSystem(root string) string {
	if _, err := os.Stat(rootPath(root, "/run/systemd/system")); err == nil {
		return "systemd"
	}
	if _, err := os.Stat(rootPath(root, "/run/openrc")); err == nil {
		return "openrc"
	}
	return "unknown"
}

func initSystem(host Host, root string) string {
	if detected := detectInitSystem(root); detected != "unknown" {
		return detected
	}
	if plugin, ok := distroPlugin(host); ok && plugin.InitSystem != "" {
		return plugin.InitSystem
	}
	return "unknown"
}

func rootPath(root string, path string) string {
	if root == "" {
		return path
	}
	return filepath.Join(root, strings.TrimPrefix(path, "/"))
}

func detectSSHPort(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "22"
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Port") {
			return fields[1]
		}
	}
	return "22"
}

func sshServiceName(host Host) string {
	if plugin, ok := distroPlugin(host); ok && plugin.SSHService != "" {
		return plugin.SSHService
	}
	return "sshd"
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func distroPlugin(host Host) (plugins.Plugin, bool) {
	return plugins.DistroAdapter(plugins.HostMatcher{
		OSID:            host.OSID,
		IDLike:          host.IDLike,
		Provider:        host.Provider,
		PackageManager:  host.PackageManager,
		FirewallBackend: host.FirewallBackend,
	})
}
