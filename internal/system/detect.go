package system

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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

type Prober interface {
	Env(string) string
	ReadFile(string) ([]byte, error)
	Stat(string) error
	LookPath(string) bool
	GOARCH() string
}

type RealProber struct{}

func (RealProber) Env(name string) string {
	return os.Getenv(name)
}

func (RealProber) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (RealProber) Stat(path string) error {
	_, err := os.Stat(path)
	return err
}

func (RealProber) LookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (RealProber) GOARCH() string {
	return runtime.GOARCH
}

func Detect() (Host, error) {
	return DetectWithProber(RealProber{})
}

func DetectWithProber(prober Prober) (Host, error) {
	root := prober.Env("ARES_ROOT")
	osReleaseOverride := prober.Env("ARES_OS_RELEASE") != ""
	osReleasePath := prober.Env("ARES_OS_RELEASE")
	if osReleasePath == "" {
		osReleasePath = rootPath(root, "/etc/os-release")
	}
	osRelease, err := readOSReleaseWithProber(prober, osReleasePath)
	if err != nil {
		return Host{}, err
	}

	host := Host{
		OSID:           osRelease["ID"],
		OSName:         osRelease["PRETTY_NAME"],
		OSVersion:      osRelease["VERSION_ID"],
		IDLike:         strings.Fields(osRelease["ID_LIKE"]),
		Provider:       detectProviderWithProber(prober, root),
		SSHPort:        detectSSHPortWithProber(prober, rootPath(root, "/etc/ssh/sshd_config")),
		RunningOverSSH: prober.Env("SSH_CONNECTION") != "" || prober.Env("SSH_CLIENT") != "",
		Architecture:   prober.GOARCH(),
	}
	probeHostCommands := root == "" && !osReleaseOverride
	host.PackageManager = packageManagerWithProber(prober, host, probeHostCommands)
	host.InitSystem = initSystemWithProber(prober, host, root)
	host.SSHService = sshServiceName(host)
	host.FirewallBackend = firewallBackendWithProber(prober, host, probeHostCommands)

	return host, nil
}

func readOSRelease(path string) (map[string]string, error) {
	return readOSReleaseWithProber(RealProber{}, path)
}

func readOSReleaseWithProber(prober Prober, path string) (map[string]string, error) {
	data, err := prober.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	values := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
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

func detectProviderWithProber(prober Prober, root string) string {
	if provider := strings.TrimSpace(prober.Env("ARES_PROVIDER")); provider != "" {
		return normalizeProvider(provider)
	}
	probeFiles := []string{
		"/sys/class/dmi/id/sys_vendor",
		"/sys/class/dmi/id/product_name",
		"/sys/class/dmi/id/board_vendor",
	}
	var values []string
	for _, path := range probeFiles {
		data, err := prober.ReadFile(rootPath(root, path))
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
	return packageManagerWithProber(RealProber{}, host, probeHostCommands)
}

func packageManagerWithProber(prober Prober, host Host, probeHostCommands bool) string {
	if probeHostCommands {
		if detected := firstCommandWithProber(prober, "apt-get", "dnf", "yum", "pacman", "zypper", "apk"); detected != "unknown" {
			return detected
		}
	}
	if plugin, ok := distroPlugin(host); ok && plugin.PackageManager != "" {
		return plugin.PackageManager
	}
	return "unknown"
}

func firewallBackend(host Host, probeHostCommands bool) string {
	return firewallBackendWithProber(RealProber{}, host, probeHostCommands)
}

func firewallBackendWithProber(prober Prober, host Host, probeHostCommands bool) string {
	if probeHostCommands && prober.LookPath("ufw") {
		return "ufw"
	}
	if probeHostCommands && prober.LookPath("firewall-cmd") {
		return "firewalld"
	}
	if probeHostCommands && prober.LookPath("nft") {
		return "nftables"
	}
	if plugin, ok := distroPlugin(host); ok && plugin.FirewallBackend != "" {
		return plugin.FirewallBackend
	}
	return "unknown"
}

func firstCommandWithProber(prober Prober, names ...string) string {
	for _, name := range names {
		if prober.LookPath(name) {
			return name
		}
	}
	return "unknown"
}

func detectInitSystemWithProber(prober Prober, root string) string {
	if err := prober.Stat(rootPath(root, "/run/systemd/system")); err == nil {
		return "systemd"
	}
	if err := prober.Stat(rootPath(root, "/run/openrc")); err == nil {
		return "openrc"
	}
	return "unknown"
}

func initSystem(host Host, root string) string {
	return initSystemWithProber(RealProber{}, host, root)
}

func initSystemWithProber(prober Prober, host Host, root string) string {
	if detected := detectInitSystemWithProber(prober, root); detected != "unknown" {
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
	return detectSSHPortWithProber(RealProber{}, path)
}

func detectSSHPortWithProber(prober Prober, path string) string {
	data, err := prober.ReadFile(path)
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
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Port") && validPort(fields[1]) {
			return fields[1]
		}
	}
	return "22"
}

func validPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}

func sshServiceName(host Host) string {
	if plugin, ok := distroPlugin(host); ok && plugin.SSHService != "" {
		return plugin.SSHService
	}
	return "sshd"
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
