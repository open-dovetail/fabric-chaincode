package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testConfig = "../../samples/marble/marble.json"

func TestAppConfig(t *testing.T) {
	config, err := ReadAppConfig(testConfig)
	assert.NoError(t, err, "read sample app config should not throw error")
	assert.Equal(t, 1, len(config.Config.Triggers), "sample file should contain 1 trigger")
	assert.Equal(t, 10, len(config.Config.Triggers[0].Handlers), "sample trigger should have 10 handlers")
	assert.Equal(t, 10, len(config.Resources), "sample file should contain 10 resources")

	err = config.WriteAppConfig("out.json")
	assert.NoError(t, err, "write app config should not throw error")
}

func TestContractToAppConfig(t *testing.T) {
	spec, err := ReadContract(testContract)
	assert.NoError(t, err, "read sample contract should not throw error")
	config, err := spec.ToAppConfig()
	assert.NoError(t, err, "convert contract to app config should not throw error")

	err = config.WriteAppConfig("contract-app.json")
	assert.NoError(t, err, "write app config should not throw error")
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"camelCase", "camel_case"},
		{"PascalCase", "pascal_case"},
		{"snake_case", "snake_case"},
		{"Pascal_Snake", "pascal_snake"},
		{"SCREAMING_SNAKE", "screaming_snake"},
		{"kebab-case", "kebab_case"},
		{"Pascal-Kebab", "pascal_kebab"},
		{"SCREAMING-KEBAB", "screaming_kebab"},
		{"A", "a"},
		{"AA", "aa"},
		{"AAA", "aaa"},
		{"AAAA", "aaaa"},
		{"AaAa", "aa_aa"},
		{"HTTPRequest", "http_request"},
		{"BatteryLifeValue", "battery_life_value"},
		{"Id0Value", "id0_value"},
		{"ID0Value", "id0_value"},
	}
	for _, test := range tests {
		have := ToSnakeCase(test.input)
		if have != test.want {
			t.Errorf("input=%q:\nhave: %q\nwant: %q", test.input, have, test.want)
		}
	}
}
