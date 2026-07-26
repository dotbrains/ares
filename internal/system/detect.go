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
)

type Host struct {
	OSID            string
	OSName          string
	OSVersion       string
	IDLike          []string
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
		InitSystem:     detectInitSystem(root),
		SSHPort:        detectSSHPort(rootPath(root, "/etc/ssh/sshd_config")),
		RunningOverSSH: os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != "",
		Architecture:   runtime.GOARCH,
	}
	host.PackageManager = packageManager(host)
	host.SSHService = sshServiceName(host)
	host.FirewallBackend = firewallBackend(host)

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

func firstCommand(names ...string) string {
	for _, name := range names {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return "unknown"
}

func packageManager(host Host) string {
	if detected := firstCommand("apt-get", "dnf", "yum", "pacman", "zypper", "apk"); detected != "unknown" {
		return detected
	}
	switch host.OSID {
	case "ubuntu", "debian":
		return "apt-get"
	case "fedora", "almalinux", "rocky", "rhel":
		return "dnf"
	case "arch":
		return "pacman"
	case "opensuse-leap", "opensuse-tumbleweed":
		return "zypper"
	case "alpine":
		return "apk"
	default:
		return "unknown"
	}
}

func detectInitSystem(root string) string {
	if _, err := os.Stat(rootPath(root, "/run/systemd/system")); err == nil {
		return "systemd"
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
	switch host.OSID {
	case "ubuntu", "debian":
		return "ssh"
	default:
		return "sshd"
	}
}

func firewallBackend(host Host) string {
	if commandExists("ufw") {
		return "ufw"
	}
	if commandExists("firewall-cmd") {
		return "firewalld"
	}
	if commandExists("nft") {
		return "nftables"
	}
	switch host.PackageManager {
	case "apt-get":
		return "ufw"
	case "dnf", "yum":
		return "firewalld"
	default:
		return "unknown"
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
