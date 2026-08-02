package system

import "strings"

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
	Observations    map[string]ObservedValue `json:"observations,omitempty"`
	Facts           map[string]Fact          `json:"facts,omitempty"`
}

type Fact struct {
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type ObservedValue struct {
	Name       string
	Value      string
	Source     string
	Confidence string
}

func (host Host) Observed(name string) ObservedValue {
	if observed, ok := host.Observations[name]; ok {
		return observed
	}
	value := host.valueForObservation(name)
	fact := host.Facts[name]
	source := fact.Source
	if source == "" {
		source = "unknown"
	}
	confidence := fact.Confidence
	if confidence == "" {
		confidence = confidenceForObservedValue(value)
	}
	return ObservedValue{
		Name:       name,
		Value:      value,
		Source:     source,
		Confidence: confidence,
	}
}

func (host *Host) Observe(name string, value string, fact Fact) {
	if host.Observations == nil {
		host.Observations = map[string]ObservedValue{}
	}
	if host.Facts == nil {
		host.Facts = map[string]Fact{}
	}
	if fact.Source == "" {
		fact.Source = "unknown"
	}
	if fact.Confidence == "" {
		fact.Confidence = confidenceForObservedValue(value)
	}
	host.Observations[name] = ObservedValue{Name: name, Value: value, Source: fact.Source, Confidence: fact.Confidence}
	host.Facts[name] = fact
}

func (host *Host) RefreshObservations() {
	for _, name := range ObservationNames() {
		fact := host.Facts[name]
		host.Observe(name, host.valueForObservation(name), fact)
	}
}

func ObservationNames() []string {
	return []string{"os", "package_manager", "init_system", "firewall_backend", "ssh_service", "ssh_port", "provider", "architecture"}
}

func (host *Host) ProjectObservedFields() {
	for _, name := range ObservationNames() {
		observed := host.Observed(name)
		host.setObservationField(name, observed.Value)
	}
}

func (host Host) valueForObservation(name string) string {
	switch name {
	case "os":
		return strings.TrimSpace(host.OSID + " " + host.OSVersion)
	case "package_manager":
		return host.PackageManager
	case "init_system":
		return host.InitSystem
	case "firewall_backend":
		return host.FirewallBackend
	case "ssh_service":
		return host.SSHService
	case "ssh_port":
		return host.SSHPort
	case "provider":
		return host.Provider
	case "architecture":
		return host.Architecture
	default:
		return ""
	}
}

func (host *Host) setObservationField(name string, value string) {
	switch name {
	case "os":
		parts := strings.Fields(value)
		if len(parts) > 0 {
			host.OSID = parts[0]
		}
		if len(parts) > 1 {
			host.OSVersion = strings.Join(parts[1:], " ")
		}
	case "package_manager":
		host.PackageManager = value
	case "init_system":
		host.InitSystem = value
	case "firewall_backend":
		host.FirewallBackend = value
	case "ssh_service":
		host.SSHService = value
	case "ssh_port":
		host.SSHPort = value
	case "provider":
		host.Provider = value
	case "architecture":
		host.Architecture = value
	}
}

func confidenceForObservedValue(value string) string {
	if strings.TrimSpace(value) == "" || value == "unknown" {
		return "low"
	}
	return "high"
}
