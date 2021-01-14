/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package get

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

var act *Activity
var tc *test.TestActivityContext
var stub *shimtest.MockStub
var queryStmt string

type Marble struct {
	DocType string `json:"docType"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	Size    int    `json:"size"`
	Owner   string `json:"owner"`
}

func setup() error {
	// settings exported by Web UI contains extra nesting of mapping, which maybe a bug
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
	if err := json.Unmarshal([]byte(config), &settings); err != nil {
		return err
	}

	mf := mapper.NewFactory(resolve.GetBasicResolver())
	ctx := test.NewActivityInitContext(settings, mf)
	iAct, err := New(ctx)
	if err != nil {
		return err
	}
	var ok bool
	if act, ok = iAct.(*Activity); !ok {
		return errors.Errorf("activity type %T is not *Activity", iAct)
	}

	queryStmt = act.query
	tc = test.NewActivityContext(act.Metadata())
	stub = shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// setup mock state and composite keys to be deleted
	sample := `[{
			"docType": "marble",
			"name": "marble1",
			"color": "blue",
			"size": 50,
			"owner": "tom"
		},
		{
			"docType": "marble",
			"name": "marble2",
			"color": "red",
			"size": 60,
			"owner": "tom"
		},
		{
			"docType": "marble",
			"name": "marble3",
			"color": "blue",
			"size": 70,
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"name": "marble4",
			"color": "red",
			"size": 80,
			"owner": "jerry"
		}
	]`
	stub.MockTransactionStart("1")
	var data []*Marble
	err = json.Unmarshal([]byte(sample), &data)
	for _, d := range data {
		v, _ := json.Marshal(d)
		err = stub.PutState(d.Name, v)
		ck, _ := stub.CreateCompositeKey("owner~name", []string{d.DocType, d.Owner, d.Name})
		err = stub.PutState(ck, []byte{0x00})
		ck, err = stub.CreateCompositeKey("color~name", []string{d.DocType, d.Color, d.Name})
		err = stub.PutState(ck, []byte{0x00})
	}
	stub.MockTransactionEnd("1")
	return err
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		logger.Errorf("FAILED %v", err)
		os.Exit(1)
	}
	logger.Info("Setup successful")
	os.Exit(m.Run())
}

func TestGetByKey(t *testing.T) {
	logger.Info("TestGetByKey")
	act.keysOnly = false

	input := &Input{Data: "marble1"}
	err := tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("2")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("2")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 1, len(output.Result), "GetByKey should return 1 record")
	//logger.Infof("result %v", output.Result)
	rec, ok := output.Result[0].(map[string]interface{})
	assert.True(t, ok, "record should be a JSON object")
	assert.Equal(t, "marble1", rec["key"].(string), "record key should be 'marble1'")
	val, ok := rec["value"].(map[string]interface{})
	assert.True(t, ok, "value should be a JSON object")
	assert.Equal(t, "marble1", val["name"].(string), "value should contain name attribute of marble1")
}

func TestGetByKey2(t *testing.T) {
	logger.Info("TestGetByKey2")
	act.keysOnly = false

	input := &Input{Data: []interface{}{"marble1", "marble3"}}
	err := tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("3")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("3")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 2, len(output.Result), "GetByKey should return 2 records")
	//logger.Infof("result %v", output.Result)
	count := 0
	for _, result := range output.Result {
		rec, ok := result.(map[string]interface{})
		assert.True(t, ok, "record should be a JSON object")
		val, ok := rec["value"].(map[string]interface{})
		assert.Equal(t, rec["key"].(string), val["name"].(string), "value should contain name matching the record key")
		if strings.HasPrefix(val["name"].(string), "marble") {
			count++
		}
	}
	assert.Equal(t, 2, count, "should have verified name of 2 records")
}

func TestGetByPartialKey(t *testing.T) {
	logger.Info("TestGetByPartialKey")
	act.keysOnly = false
	act.query = ""

	sample := `{
		"docType": "marble",
		"owner": "jerry"
	}`
	var data interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "input data should be valid JSON")

	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("4")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("4")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 2, len(output.Result), "jerry should own 2 marbles")
	//logger.Infof("result %v", output.Result)
	count := 0
	for _, result := range output.Result {
		rec, ok := result.(map[string]interface{})
		assert.True(t, ok, "record should be a JSON object")
		val, ok := rec["value"].(map[string]interface{})
		assert.Equal(t, rec["key"].(string), val["name"].(string), "value should contain name matching the record key")
		assert.Equal(t, "jerry", val["owner"].(string), "owner should be jerry")
		count++
	}
	assert.Equal(t, 2, count, "should have verified name of 2 records")
}

func TestGetByPartialKey2(t *testing.T) {
	logger.Info("TestGetByPartialKey2")
	act.keysOnly = false
	act.query = ""

	sample := `[{
			"docType": "marble",
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"owner": "tom"
		}
	]`
	var data []interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "input data should be valid JSON")

	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("5")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("5")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 4, len(output.Result), "jerry & tom should own total 4 marbles")
	//logger.Infof("result %v", output.Result)
	count := 0
	for _, result := range output.Result {
		rec, ok := result.(map[string]interface{})
		assert.True(t, ok, "record should be a JSON object")
		val, ok := rec["value"].(map[string]interface{})
		assert.Equal(t, rec["key"].(string), val["name"].(string), "value should contain name matching the record key")
		if strings.HasPrefix(val["name"].(string), "marble") {
			count++
		}
	}
	assert.Equal(t, 4, count, "should have verified name of 4 records")
}

func TestGetByRange(t *testing.T) {
	logger.Info("TestGetByRange")
	act.keysOnly = false
	act.query = ""

	sample := `{
		"start": "marble1",
		"end": "marble4"
	}`
	var data interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "input data should be valid JSON")

	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("6")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("6")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 3, len(output.Result), "marble count in range should be 3")
	//logger.Infof("result %v", output.Result)
	count := 0
	for _, result := range output.Result {
		rec, ok := result.(map[string]interface{})
		assert.True(t, ok, "record should be a JSON object")
		val, ok := rec["value"].(map[string]interface{})
		assert.Equal(t, rec["key"].(string), val["name"].(string), "value should contain name matching the record key")
		if strings.HasPrefix(val["name"].(string), "marble") {
			count++
		}
	}
	assert.Equal(t, 3, count, "should have verified name of 3 records")
}

func TestGetByOpenRange(t *testing.T) {
	logger.Info("TestGetByOpenRange")
	act.keysOnly = false
	act.query = ""

	sample := `{
		"start": ""
	}`
	var data interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "input data should be valid JSON")

	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("7")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("7")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 4, len(output.Result), "marble count in range should be 4")
	//logger.Infof("result %v", output.Result)
	count := 0
	for _, result := range output.Result {
		rec, ok := result.(map[string]interface{})
		assert.True(t, ok, "record should be a JSON object")
		val, ok := rec["value"].(map[string]interface{})
		assert.Equal(t, rec["key"].(string), val["name"].(string), "value should contain name matching the record key")
		if strings.HasPrefix(val["name"].(string), "marble") {
			count++
		}
	}
	assert.Equal(t, 4, count, "should have verified name of 4 records")
}

func TestGetByKeyOnly(t *testing.T) {
	logger.Info("TestGetByKeyOnly")
	act.keysOnly = true
	act.query = ""

	sample := `[{
			"docType": "marble",
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"owner": "tom"
		}
	]`
	var data []interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "input data should be valid JSON")

	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("8")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("8")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 1, len(output.Result), "only 1 composite key is defined")
	//logger.Infof("result %v", output.Result)
	rec, ok := output.Result[0].(map[string]interface{})
	assert.True(t, ok, "record should be a JSON object")
	assert.Equal(t, "owner~name", rec["name"].(string), "key name should be 'owner~name'")
	keys, ok := rec["keys"].([]interface{})
	assert.True(t, ok, "keys should be an array")
	assert.Equal(t, 4, len(keys), "returned key count should be 4")
	count := 0
	for _, key := range keys {
		k, ok := key.(map[string]interface{})
		assert.True(t, ok, "key should be a JSON object")
		fields, ok := k["fields"].([]interface{})
		assert.True(t, ok, "key fields should be an array")
		assert.Equal(t, 2, len(fields), "key field count should be 2")
		if strings.HasPrefix(k["key"].(string), "marble") {
			count++
		}
	}
	assert.Equal(t, 4, count, "all 4 key names should have prefix 'marble'")
}

func TestPrepareQueryStatement(t *testing.T) {
	logger.Info("TestPrepareQueryStatement")
	query := `{
        "selector": {
            "sParam": "$sparam",
            "iParam": {
                "$gt": "$iparam"
            },
            "bParam": "$bparam"
		}
	}`
	params := `{
        "sparam": "hello",
        "iparam": 100,
		"bparam": true,
		"foo": "bar"
	}`
	result := `{
        "selector": {
            "sParam": "hello",
            "iParam": {
                "$gt": 100
            },
            "bParam": true
		}
	}`
	var queryParams map[string]interface{}
	err := json.Unmarshal([]byte(params), &queryParams)
	assert.NoError(t, err, "failed to parse queryParams")
	stmt := prepareQueryStatement(query, queryParams)
	assert.Equal(t, result, stmt, "unexpected resulting query statement")
}

func TestGetByQuery(t *testing.T) {
	logger.Info("TestGetByQuery")
	act.keysOnly = false
	act.query = queryStmt

	// sample test message
	sample := `{
		"size": 40,
		"owner": "tom"
	}`
	var data map[string]interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "input params should be valid JSON object")
	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("9")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("9")
	assert.False(t, done, "action eval should fail")
	assert.Contains(t, err.Error(), "not implemented", "error message should show not implemented by mock")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "get action output should not throw error")
	assert.Equal(t, 500, output.Code, "action output status should be 500")
	assert.Contains(t, output.Message, "\"tom\"", "response error should show failed query")
}

func TestGetHistory(t *testing.T) {
	logger.Info("TestGetHistory")
	act.keysOnly = false
	act.history = true

	input := &Input{Data: []interface{}{"marble1", "marble3"}}
	err := tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("10")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("10")
	assert.False(t, done, "action eval should fail")
	assert.Contains(t, err.Error(), "not implemented", "error message should show not implemented by mock")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "get action output should not throw error")
	assert.Equal(t, 500, output.Code, "action output status should be 500")
	assert.Contains(t, output.Message, "marble", "response error shows failed query")
}
