package plugins

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
