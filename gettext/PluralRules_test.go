package gettext

import "testing"

func TestAllDefaultPluralRules(t *testing.T) {
	for _, def := range defaultPluralRulesDefinitions {
		_ = def.Parse() //panics if parsing or sample validation fails
	}
}
