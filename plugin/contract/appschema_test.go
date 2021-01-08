package contract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppSchema(t *testing.T) {
	spec, err := ReadContract(testContract)
	assert.NoError(t, err, "read sample contract should not throw error")
	err = spec.ConvertAppSchemas()
	assert.NoError(t, err, "convert contract to app config should not throw error")
	defs, err := getAppSchemas()
	assert.NoError(t, err, "collect contract to app config should not throw error")
	assert.Equal(t, 3, len(defs), "there should be 3 schemas in the sample contract")
	for k, v := range defs {
		fmt.Printf("%s => %s\n", k, v.Value)
		assert.NotContains(t, v.Value, "$ref", "app schema defintion should not contain refs")
	}
}

func TestJSON2Schema(t *testing.T) {
	sample := `[
		{
			"stringVar": "text",
			"numberVar": 1.1,
			"intVar": 10,
			"boolVar": true,
			"intSliceVar": [1, 2, 3, 4],
			"strSliceVar": ["a", "b", "c"],
			"objVar": {"foo": "bar"},
			"loop": {
				"@foreach(xyz)": {
					"x": "a",
					"y": 1,
					"z": false
				}
			}
		}
	]`
	schm, err := json2schema(sample)
	assert.NoError(t, err, "json2schema should not throw error")
	//fmt.Println(result)
	var data interface{}
	err = json.Unmarshal([]byte(schm), &data)
	assert.NoError(t, err, "schema should be a valid JSON object")
	lookup := lookupJSONPath(data, "$.items.properties.loop.items.properties.x")
	assert.Equal(t, 1, len(lookup), "should find 1 schema type")
}

func TestLookupJSONPath(t *testing.T) {
	config, _, err := ReadAppConfig(testConfig)
	assert.NoError(t, err)
	jsonbytes, err := marshalNoEscape(config)
	assert.NoError(t, err)
	var doc interface{}
	err = json.Unmarshal(jsonbytes, &doc)
	assert.NoError(t, err)
	result := lookupJSONPath(doc, "$.resources.data.tasks.activity")
	count := 0
	for _, v := range result {
		data := v.(map[string]interface{})
		if matched, err := regexp.Match(data["ref"].(string), []byte("#put|#get|#delete")); err == nil && matched {
			count++
		}
	}
	assert.Equal(t, 14, count, "ledger activity count should be 14")
}
