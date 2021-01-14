/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package common

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/stretchr/testify/assert"
)

func TestMockLedger(t *testing.T) {
	stub := shimtest.NewMockStub("mock", nil)
	ck, err := stub.CreateCompositeKey("owner~name", []string{"marble", "tom", "marble1"})
	assert.NoError(t, err, "create composite key should not throw error")
	assert.True(t, IsCompositeKey(ck), "composite key should verify to true")
	kv, err := SplitCompositeKey(stub, ck)
	assert.NoError(t, err, "split composite key should succeed")
	assert.Equal(t, "marble1", kv.Key, "state key should be 'marble1'")
	assert.Equal(t, 3, len(kv.Fields), "key fields should count to 3")

	// sample composite key defs
	ckdef := `{
        "color~name": [
            "$.docType",
            "$.color",
            "$.name"
        ],
        "owner~name": [
            "$.docType",
            "$.owner",
            "$.name"
        ]
	}`
	def := make(map[string][]string)
	err = json.Unmarshal([]byte(ckdef), &def)
	assert.NoError(t, err, "composite key def should be valid")
	assert.Equal(t, 2, len(def), "composite keys should count to 2")

	// sample data
	data := `{
		"docType": "marble",
		"color": "blue",
		"owner": "tom"
	}`
	var state map[string]interface{}
	err = json.Unmarshal([]byte(data), &state)
	assert.NoError(t, err, "sample data should be valid")
	assert.Equal(t, "tom", state["owner"].(string), "sample owner should be 'tom'")

	// test extract composite keys
	keys := ExtractCompositeKeys(stub, def, "marble1", state)
	assert.Equal(t, 2, len(keys), "it should extract 2 composite keys")
	assert.Equal(t, ck, keys[1], "first key should be identical to marble1 with owner tom")

	data = `{
		"docType": "marble",
		"name": "marble1",
		"color": "blue",
		"size": 50,
		"owner": "tom"
	}`

	// test PutData
	stub.MockTransactionStart("1")
	err = PutData(stub, "", "marble1", []byte(data))
	assert.NoError(t, err, "put state data should not throw error")
	err = PutData(stub, "", keys[1], nil)
	assert.NoError(t, err, "put composite key should not throw error")
	stub.MockTransactionEnd("1")

	// test GetData
	stub.MockTransactionStart("2")
	k, v, err := GetData(stub, "", "marble1", false)
	assert.NoError(t, err, "get state data should not throw error")
	assert.Equal(t, "marble1", k, "state key should be 'marble1'")
	err = json.Unmarshal([]byte(v), &state)
	assert.NoError(t, err, "state data should be valid")
	assert.Equal(t, "marble1", state["name"], "name of the record should be 'marble1'")

	// test GetData using a composite key
	k2, v2, err := GetData(stub, "", keys[1], false)
	assert.NoError(t, err, "get state using composite key should not throw error")
	assert.Equal(t, k, k2, "composite key should return the same state key")
	assert.Equal(t, 0, bytes.Compare(v2, v), "composite key should return the same state value")
	stub.MockTransactionEnd("2")

	// test DeleteData
	stub.MockTransactionStart("3")
	err = DeleteData(stub, "", "marble1")
	assert.NoError(t, err, "delete state data should not throw error")
	err = DeleteData(stub, "", keys[1])
	assert.NoError(t, err, "delete composite key should not throw error")
	stub.MockTransactionEnd("3")

	// verify result of deletion
	stub.MockTransactionStart("4")
	_, v, err = GetData(stub, "", "marble1", false)
	assert.NoError(t, err, "retrieve non-existing state data should not throw error")
	assert.Nil(t, v, "retrieve non-existing state data should return nil")
	_, v, err = GetData(stub, "", keys[1], false)
	assert.NoError(t, err, "retrieve non-existing composite key should not throw error")
	assert.Nil(t, v, "retrieve non-existing composite key should return nil")
	stub.MockTransactionEnd("4")
}

func TestExtractDataAttributes(t *testing.T) {
	attrs := []string{"$.docType", "$.color", "$.name"}
	sample := `{
		"color": "blue",
		"docType": "marble"
	}`
	var data interface{}
	json.Unmarshal([]byte(sample), &data)
	result := ExtractDataAttributes(attrs, data)
	assert.Equal(t, 2, len(result), "it should extract 2 attribute fields")
	assert.Equal(t, "blue", result[1], "second attribute should be 'blue'")
}
