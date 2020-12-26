package put

import (
	"encoding/json"
	"os"
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

func setup() error {
	// settings exported by Web UI contains extra nesting of mapping, which maybe a bug
	config := `{
        "compositeKeys": {
			"mapping": {
            	"owner~name": [
                	"docType",
                	"owner",
                	"name"
            	],
            	"color~name": [
                	"docType",
                	"color",
                	"name"
				]
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

	tc = test.NewActivityContext(act.Metadata())
	return nil
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		logger.Errorf("FAILED %v", err)
		os.Exit(1)
	}
	logger.Info("Setup successful")
	os.Exit(m.Run())
}

func TestPutData(t *testing.T) {
	logger.Info("TestPutData")
	act.keysOnly = false
	act.createOnly = false

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// sample test message
	data := `{
		"key": "marble1",
		"value": {
			"docType": "marble",
			"name": "marble1",
			"color": "blue",
			"size": 50,
			"owner": "tom"
		}
	}`
	var state map[string]interface{}
	err := json.Unmarshal([]byte(data), &state)
	assert.NoError(t, err, "smaple data should be valid JSON object")
	input := &Input{Data: state}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request to store data in mock Fabric transaction
	stub.MockTransactionStart("1")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("1")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 1, len(output.Result), "result should contain 1 record")
	rec := output.Result[0].(map[string]interface{})
	assert.Equal(t, "marble1", rec["key"].(string), "updated record should contain key 'marble1'")

	// verify correct update of mock Fabric state
	stub.MockTransactionStart("2")
	val, err := stub.GetState("marble1")
	assert.NoError(t, err, "retrieve state of marble1 should not throw error")
	err = json.Unmarshal(val, &rec)
	assert.NoError(t, err, "unmarshal stored record should not throw error")
	assert.Equal(t, "blue", rec["color"].(string), "stored record should have color = 'blue'")

	// verify correct update of composite key for owner
	iter, err := stub.GetStateByPartialCompositeKey("owner~name", []string{"marble", "tom"})
	assert.NoError(t, err, "composite key query for owner should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	v, err := iter.Next()
	assert.NoError(t, err, "composite key query should return a key")
	_, cp, err := stub.SplitCompositeKey(v.Key)
	assert.NoError(t, err, "composite key query should return a valid key")
	assert.Equal(t, "marble1", cp[len(cp)-1], "returned composite key should be 'marble1'")
	iter.Close()

	// verify correct update of composite key for color
	iter, err = stub.GetStateByPartialCompositeKey("color~name", []string{"marble", "blue"})
	assert.NoError(t, err, "composite key query for color should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	v, err = iter.Next()
	assert.NoError(t, err, "composite key query should return a key")
	_, cp, err = stub.SplitCompositeKey(v.Key)
	assert.NoError(t, err, "composite key query should return a valid key")
	assert.Equal(t, "marble1", cp[len(cp)-1], "returned composite key should be 'marble1'")
	iter.Close()
	stub.MockTransactionEnd("2")
}

func TestPutData2(t *testing.T) {
	logger.Info("TestPutData2")
	act.keysOnly = false
	act.createOnly = false

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// sample test message
	sample := `[{
			"key": "marble1",
			"value": {
				"docType": "marble",
				"name": "marble1",
				"color": "blue",
				"size": 50,
				"owner": "tom"
			}
		},
		{
			"key": "marble2",
			"value": {
				"docType": "marble",
				"name": "marble2",
				"color": "red",
				"size": 60,
				"owner": "tom"
			}
		}
	]`
	var data []interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "smaple data should be valid JSON object")
	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request to store data in mock Fabric transaction
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
	assert.Equal(t, 2, len(output.Result), "result should contain 2 record")
	for _, v := range output.Result {
		rec := v.(map[string]interface{})
		value := rec["value"].(map[string]interface{})
		assert.Equal(t, "tom", value["owner"].(string), "updated records should have owner 'tom'")
	}

	// verify correct update of mock Fabric state
	stub.MockTransactionStart("6")
	for _, k := range []string{"marble1", "marble2"} {
		val, err := stub.GetState(k)
		assert.NoError(t, err, "retrieve state of %s should not throw error", k)
		rec := make(map[string]interface{})
		err = json.Unmarshal(val, &rec)
		assert.NoError(t, err, "unmarshal stored record should not throw error")
		assert.Equal(t, "tom", rec["owner"].(string), "stored record should have owner = 'tom'")
	}
	// verify correct update of composite key for owner
	iter, err := stub.GetStateByPartialCompositeKey("owner~name", []string{"marble", "tom"})
	assert.NoError(t, err, "composite key query for owner should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	count := 0
	for iter.HasNext() {
		v, err := iter.Next()
		count++
		assert.NoError(t, err, "composite key query should return a key")
		_, cp, err := stub.SplitCompositeKey(v.Key)
		assert.NoError(t, err, "composite key query should return a valid key")
		assert.Equal(t, "tom", cp[1], "owner field of composite key should be 'tom'")
	}
	iter.Close()
	assert.Equal(t, 2, count, "tom should own 2 marbles")
	stub.MockTransactionEnd("6")
}

func TestPutCreateOnly(t *testing.T) {
	logger.Info("TestPutCreateOnly")
	act.keysOnly = false
	act.createOnly = true

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// setup mock ledger
	sample := `{
		"docType": "marble",
		"name": "marble1",
		"color": "purple",
		"size": 40,
		"owner": "jerry"
	}`
	stub.MockTransactionStart("7")
	stub.PutState("marble1", []byte(sample))
	stub.MockTransactionEnd("7")

	// sample test message
	sample = `[{
			"key": "marble1",
			"value": {
				"docType": "marble",
				"name": "marble1",
				"color": "blue",
				"size": 50,
				"owner": "tom"
			}
		},
		{
			"key": "marble2",
			"value": {
				"docType": "marble",
				"name": "marble2",
				"color": "red",
				"size": 60,
				"owner": "tom"
			}
		}
	]`
	var data []interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "smaple data should be valid JSON object")
	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request to store data in mock Fabric transaction
	stub.MockTransactionStart("8")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("8")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 206, output.Code, "action output status should be 206, partial content")
	assert.Equal(t, 1, len(output.Result), "result should contain 1 new record")
	for _, v := range output.Result {
		rec := v.(map[string]interface{})
		value := rec["value"].(map[string]interface{})
		assert.Equal(t, "marble2", value["name"].(string), "new record should be 'marble2'")
	}

	// verify correct update of mock Fabric state
	stub.MockTransactionStart("9")
	for _, k := range []string{"marble1", "marble2"} {
		val, err := stub.GetState(k)
		assert.NoError(t, err, "retrieve state of %s should not throw error", k)
		rec := make(map[string]interface{})
		err = json.Unmarshal(val, &rec)
		assert.NoError(t, err, "unmarshal stored record should not throw error")
		if k == "marble1" {
			assert.Equal(t, "jerry", rec["owner"].(string), "marble1 should have owner = 'jerry'")
		} else {
			assert.Equal(t, "tom", rec["owner"].(string), "marble2 should have owner = 'tom'")
		}
	}
	// verify correct update of composite key for owner
	iter, err := stub.GetStateByPartialCompositeKey("owner~name", []string{"marble", "tom"})
	assert.NoError(t, err, "composite key query for owner should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	count := 0
	for iter.HasNext() {
		v, err := iter.Next()
		count++
		assert.NoError(t, err, "composite key query should return a key")
		_, cp, err := stub.SplitCompositeKey(v.Key)
		assert.NoError(t, err, "composite key query should return a valid key")
		assert.Equal(t, "marble2", cp[2], "name field of composite key should be 'marble2'")
	}
	iter.Close()
	assert.Equal(t, 1, count, "tom should own 1 marble")
	stub.MockTransactionEnd("9")
}

func TestPutCompositeKey(t *testing.T) {
	logger.Info("TestPutCompositeKey")
	act.keysOnly = true

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// sample test message
	sample := `{
		"docType": "marble",
		"name": "marble2",
		"color": "blue",
		"owner": "jerry"
	}`
	var data map[string]interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "smaple data should be valid JSON object")
	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request to store data in mock Fabric transaction
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
	assert.Equal(t, 2, len(output.Result), "result should contain 2 composite keys")
	for _, v := range output.Result {
		rec, ok := v.(map[string]interface{})
		assert.True(t, ok, "composite key should be a JSON object")
		keys, ok := rec["keys"].([]interface{})
		assert.True(t, ok, "composite key bag should contain array of keys")
		assert.Equal(t, 1, len(keys), "composite key bag should contain 1 key")
		key, ok := keys[0].(map[string]interface{})
		assert.True(t, ok, "composite key should be a JSON object")
		assert.Equal(t, "marble2", key["key"].(string), "state key value of %s should be marble2", rec["name"])
	}
	// verify no mock Fabric state is updated
	stub.MockTransactionStart("4")
	val, err := stub.GetState("marble2")
	assert.NoError(t, err, "retrieve state of marble2 should not throw error")
	assert.Nil(t, val, "marble2 should not exist in the ledger")

	// verify correct update of composite key for owner
	iter, err := stub.GetStateByPartialCompositeKey("owner~name", []string{"marble", "jerry"})
	assert.NoError(t, err, "composite key query for owner should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	v, err := iter.Next()
	assert.NoError(t, err, "composite key query should return a key")
	_, cp, err := stub.SplitCompositeKey(v.Key)
	assert.NoError(t, err, "composite key query should return a valid key")
	assert.Equal(t, "marble2", cp[len(cp)-1], "returned composite key should be 'marble2'")
	iter.Close()

	// verify correct update of composite key for color
	iter, err = stub.GetStateByPartialCompositeKey("color~name", []string{"marble", "blue"})
	assert.NoError(t, err, "composite key query for color should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	v, err = iter.Next()
	assert.NoError(t, err, "composite key query should return a key")
	_, cp, err = stub.SplitCompositeKey(v.Key)
	assert.NoError(t, err, "composite key query should return a valid key")
	assert.Equal(t, "marble2", cp[len(cp)-1], "returned composite key should be 'marble2'")
	iter.Close()

	stub.MockTransactionEnd("4")
}

func TestPutCompositeKey2(t *testing.T) {
	logger.Info("TestPutCompositeKey2")
	act.keysOnly = true

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// sample test message
	sample := `[{
			"docType": "marble",
			"name": "marble1",
			"color": "blue",
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"name": "marble2",
			"color": "red",
			"owner": "jerry"
		}
	]`
	var data []interface{}
	err := json.Unmarshal([]byte(sample), &data)
	assert.NoError(t, err, "smaple data should be valid JSON object")
	input := &Input{Data: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request to store data in mock Fabric transaction
	stub.MockTransactionStart("10")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("10")
	assert.True(t, done, "action eval should be successful")
	assert.NoError(t, err, "action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "action output should not be error")
	assert.Equal(t, 200, output.Code, "action output status should be 200")
	assert.Equal(t, 2, len(output.Result), "result should contain 2 composite keys")
	for _, v := range output.Result {
		rec, ok := v.(map[string]interface{})
		assert.True(t, ok, "composite key should be a JSON object")
		keys, ok := rec["keys"].([]interface{})
		assert.True(t, ok, "composite key bag should contain array of keys")
		assert.Equal(t, 2, len(keys), "composite key bag for owner should contain 2 keys")
	}
	// verify no mock Fabric state is updated
	stub.MockTransactionStart("11")
	for _, k := range []string{"marble1", "marble2"} {
		val, err := stub.GetState(k)
		assert.NoError(t, err, "retrieve state of %s should not throw error", k)
		assert.Nil(t, val, "%s should not exist in the ledger", k)
	}

	// verify correct update of composite key for owner
	iter, err := stub.GetStateByPartialCompositeKey("owner~name", []string{"marble", "jerry"})
	assert.NoError(t, err, "composite key query for owner should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	count := 0
	for iter.HasNext() {
		v, err := iter.Next()
		count++
		assert.NoError(t, err, "composite key query should return a key")
		_, cp, err := stub.SplitCompositeKey(v.Key)
		assert.NoError(t, err, "composite key query should return a valid key")
		assert.Equal(t, "jerry", cp[1], "returned composite key should have owner 'jerry'")
	}
	iter.Close()
	assert.Equal(t, 2, count, "jerry should own 2 marbles")

	// verify correct update of composite key for color
	iter, err = stub.GetStateByPartialCompositeKey("color~name", []string{"marble", "blue"})
	assert.NoError(t, err, "composite key query for color should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	count = 0
	for iter.HasNext() {
		v, err := iter.Next()
		count++
		assert.NoError(t, err, "composite key query should return a key")
		_, cp, err := stub.SplitCompositeKey(v.Key)
		assert.NoError(t, err, "composite key query should return a valid key")
		assert.Equal(t, "blue", cp[1], "returned composite key should have color 'blue'")
	}
	iter.Close()
	assert.Equal(t, 1, count, "only 1 marble should be blue")

	stub.MockTransactionEnd("11")
}
