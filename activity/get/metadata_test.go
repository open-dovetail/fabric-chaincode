/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package get

import (
	"encoding/json"
	"testing"

	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/stretchr/testify/assert"
)

func TestOSSMetadata(t *testing.T) {
	logger.Info("TestOSSMetadata")
	config := `{
        "compositeKeys": {
            "mapping": {
                "owner~name": [
                    "docType",
                    "owner",
                    "name"
                ]
            }
        },
		"query": {
            "mapping": {
				"selector": {
					"docType": "marble",
					"owner": "$owner",
					"size": {
						"$gt": "$size"
					}
				}
			}
		},
		"keysOnly": false
	}`
	settings := make(map[string]interface{})
	err := json.Unmarshal([]byte(config), &settings)
	assert.NoError(t, err, "unmarshal config should not throw error")
	compKeys, err := common.MapToObject(settings["compositeKeys"])
	assert.NoError(t, err, "convert compositeKeys should not throw error")
	ownerKey, ok := compKeys["owner~name"].([]interface{})
	assert.True(t, ok, "onwer key should contains an array")
	assert.Equal(t, 3, len(ownerKey), "owner key should contain 3 attribnutes")
}

func TestFEMetadata(t *testing.T) {
	logger.Info("TestFEMetadata")
	config := `{
        "compositeKeys": "{\"owner~name\": [\"docType\",\"owner\",\"name\"]}",
		"query": "{\"selector\": {\"docType\": \"marble\",\"owner\": \"$owner\",\"size\": {\"$gt\": \"$size\"}}}",
		"keysOnly": false
	}`
	settings := make(map[string]interface{})
	err := json.Unmarshal([]byte(config), &settings)
	assert.NoError(t, err, "unmarshal config should not throw error")
	compKeys, err := common.MapToObject(settings["compositeKeys"])
	assert.NoError(t, err, "convert compositeKeys should not throw error")
	ownerKey, ok := compKeys["owner~name"].([]interface{})
	assert.True(t, ok, "onwer key should contains an array")
	assert.Equal(t, 3, len(ownerKey), "owner key should contain 3 attribnutes")
}
