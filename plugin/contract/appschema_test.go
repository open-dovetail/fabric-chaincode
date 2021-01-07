package contract

import (
	"fmt"
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
	result, err := json2schema(sample)
	assert.NoError(t, err, "test")
	fmt.Println(result)
	assert.Fail(t, "test")
}
