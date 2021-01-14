/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package contract

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testConfig = "../../samples/marble/marble.json"

func TestAppConfig(t *testing.T) {
	fmt.Println("TestAppConfig")
	config, _, err := ReadAppConfig(testConfig)
	assert.NoError(t, err, "read sample app config should not throw error")
	assert.Equal(t, 1, len(config.Triggers), "sample file should contain 1 trigger")
	assert.Equal(t, 10, len(config.Triggers[0].Handlers), "sample trigger should have 10 handlers")
	assert.Equal(t, 10, len(config.Resources), "sample file should contain 10 resources")

	err = WriteAppConfig(config, "out.json")
	assert.NoError(t, err, "write app config should not throw error")
}

func TestContractToAppConfig(t *testing.T) {
	fmt.Println("TestContractToAppConfig")
	spec, err := ReadContract(testContract)
	assert.NoError(t, err, "read sample contract should not throw error")
	config, err := spec.ToAppConfig(true)
	assert.NoError(t, err, "convert contract to app config should not throw error")

	err = WriteAppConfig(config, "contract-app.json")
	assert.NoError(t, err, "write app config should not throw error")
	//assert.Fail(t, "test")
}

func TestToSnakeCase(t *testing.T) {
	fmt.Println("TestToSnakeCase")
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

func TestFixActivitySchema(t *testing.T) {
	fmt.Println("TestFixActivitySchema")
	spec, err := ReadContract(testContract)
	assert.NoError(t, err, "read sample contract should not throw error")
	config, err := spec.ToAppConfig(true)
	assert.NoError(t, err, "convert contract to app config should not throw error")

	doc, err := fixTriggerConfig(config)
	assert.NoError(t, err)
	fixActivitySchema(doc)
	result := lookupJSONPath(doc, "$.resources.data.tasks.activity.schemas.settings.query")
	assert.Equal(t, 1, len(result), "query schema setting")
	result = lookupJSONPath(doc, "$.resources.data.tasks.activity.schemas.settings.compositeKeys")
	assert.Equal(t, 6, len(result), "compositeKeys schema setting")
}
