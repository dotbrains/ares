package plugins

import "github.com/dotbrains/ares/marketplace"

type BehaviorSpec struct {
	Name     string
	Variant  string
	Verifier string
}

func Behavior(plugin Plugin) BehaviorSpec {
	return BehaviorSpec{
		Name:     plugin.Behavior,
		Variant:  plugin.BehaviorVariant,
		Verifier: plugin.Verifier,
	}
}

func (behavior BehaviorSpec) Is(name string) bool {
	return behavior.Name == name
}

func (behavior BehaviorSpec) IsVariant(name string) bool {
	return behavior.Variant == name
}

func (behavior BehaviorSpec) IsVerifier(name string) bool {
	return behavior.Verifier == name
}

func (behavior BehaviorSpec) IsDistro() bool {
	return behavior.Is("distro")
}

func (behavior BehaviorSpec) IsProviderAdvisory() bool {
	return behavior.Is("provider-advisory")
}

func (behavior BehaviorSpec) IsPlanningOnly() bool {
	return behavior.IsDistro() || behavior.IsProviderAdvisory()
}

func KnownBehavior(name string) bool {
	return marketplace.KnownBehavior(name)
}

func KnownBehaviorVariant(name string, variant string) bool {
	return marketplace.KnownBehaviorVariant(name, variant)
}

func KnownVerifier(name string) bool {
	return marketplace.KnownVerifier(name)
}
