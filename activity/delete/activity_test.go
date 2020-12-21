package delete

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
var stub *shimtest.MockStub
var tc *test.TestActivityContext

type Marble struct {
	DocType string `json:"docType"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	Size    int    `json:"size"`
	Owner   string `json:"owner"`
}

func setup() error {
	// config activity to add 2 composite keys for each ledger record
	config := `{
        "compositeKeys": {
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
			"color": "red",
			"size": 70,
			"owner": "tom"
		},
		{
			"docType": "marble",
			"name": "marble4",
			"color": "purple",
			"size": 50,
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"name": "marble5",
			"color": "green",
			"size": 60,
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"name": "marble6",
			"color": "green",
			"size": 70,
			"owner": "jerry"
		},
		{
			"docType": "marble",
			"name": "marble7",
			"color": "blue",
			"size": 70,
			"owner": "pluto"
		},
		{
			"docType": "marble",
			"name": "marble8",
			"color": "red",
			"size": 80,
			"owner": "pluto"
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

func TestDeleteByKey(t *testing.T) {
	logger.Info("TestDeleteByKey")
	act.keysOnly = false

	// process request to delete marble1
	err := tc.SetInputObject(&Input{Data: "marble1"})
	assert.NoError(t, err, "setting action input should not throw error")

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
	//logger.Infof("result %v", output.Result)
	rec, ok := output.Result[0].(map[string]interface{})
	assert.True(t, ok, "record should be a JSON object")
	assert.Equal(t, "marble1", rec["key"].(string), "record key should be 'marble1'")
	val, ok := rec["value"].(map[string]interface{})
	assert.True(t, ok, "value should be a JSON object")
	assert.Equal(t, "marble1", val["name"].(string), "value should contain name attribute of marble1")
}

func TestDeleteByKey2(t *testing.T) {
	logger.Info("TestDeleteByKey2")
	act.keysOnly = false

	input := &Input{Data: []interface{}{"marble2", "marble3"}}
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
	assert.Equal(t, 2, len(output.Result), "DeleteByKey should return 2 records")
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

func TestDeleteByPartialKey(t *testing.T) {
	logger.Info("TestDeleteByPartialKey")
	act.keysOnly = false

	// process request to delete by composite keys
	pkjson := `{
		"docType": "marble",
		"color": "green",
		"owner": "jerry"
	}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(pkjson), &data)
	assert.NoError(t, err, "parsing input data should not throw error")
	err = tc.SetInputObject(&Input{Data: data})
	assert.NoError(t, err, "setting action input should not throw error")
	stub.MockTransactionStart("4")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("4")
	assert.True(t, done, "partial key delete action eval should be successful")
	assert.NoError(t, err, "partial key delete action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "partial key delete action output should not be error")
	assert.Equal(t, 200, output.Code, "partial key delete action output status should be 200")
	assert.Equal(t, 3, len(output.Result), "result should list 3 state keys")
	//logger.Infof("result %v", output.Result)
	count := 0
	for _, result := range output.Result {
		rec, ok := result.(map[string]interface{})
		assert.True(t, ok, "record should be a JSON object")
		val, ok := rec["value"].(map[string]interface{})
		assert.Equal(t, rec["key"].(string), val["name"].(string), "value should contain name matching the record key")
		if strings.HasPrefix(val["owner"].(string), "jerry") {
			count++
		}
	}
	assert.Equal(t, 3, count, "all 3 records should be owned by 'jerry'")
}

func TestDeleteKeysOnly(t *testing.T) {
	logger.Info("TestDeleteKeysOnly")
	act.keysOnly = true

	// process request to delete by composite keys
	pkjson := `{
		"docType": "marble",
		"color": "nomatch",
		"owner": "pluto"
	}`
	data := make(map[string]interface{})
	err := json.Unmarshal([]byte(pkjson), &data)
	assert.NoError(t, err, "parsing input data should not throw error")
	err = tc.SetInputObject(&Input{Data: data})
	assert.NoError(t, err, "setting action input should not throw error")
	stub.MockTransactionStart("5")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("5")
	assert.True(t, done, "partial key delete action eval should be successful")
	assert.NoError(t, err, "partial key delete action eval should not throw error")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "partial key delete action output should not be error")
	assert.Equal(t, 200, output.Code, "partial key delete action output status should be 200")
	//logger.Infof("result %v", output.Result)
	assert.Equal(t, 1, len(output.Result), "result should list 1 matching key name")
	rec, ok := output.Result[0].(map[string]interface{})
	assert.True(t, ok, "record should be a JSON object")
	assert.Equal(t, "owner~name", rec["name"].(string), "key name should be 'owner~name'")
	keys, ok := rec["keys"].([]interface{})
	assert.True(t, ok, "keys should be an array")
	assert.Equal(t, 2, len(keys), "returned key count should be 2")
	for _, key := range keys {
		k, ok := key.(map[string]interface{})
		assert.True(t, ok, "key should be a JSON object")
		fields, ok := k["fields"].([]interface{})
		assert.True(t, ok, "key fields should be an array")
		assert.Equal(t, 2, len(fields), "key field count should be 2")
		assert.Equal(t, "pluto", fields[1].(string), "record owner should be 'pluto'")
	}

	// verify that state is not deleted
	stub.MockTransactionStart("6")
	val, err := stub.GetState("marble7")
	assert.NoError(t, err, "retrieve state of marble7 should not throw error")
	err = json.Unmarshal(val, &rec)
	assert.NoError(t, err, "unmarshal stored record should not throw error")
	assert.Equal(t, "pluto", rec["owner"].(string), "stored record should have owner pluto")

	// verify owner keys are deleted
	iter, err := stub.GetStateByPartialCompositeKey("owner~name", []string{"marble", "pluto"})
	assert.NoError(t, err, "composite key query for owner should not throw error")
	assert.NotNil(t, iter, "composite key query iterator should not be nil")
	assert.False(t, iter.HasNext(), "all pluto owner keys should have been deleted")
	iter.Close()

	// verify color keys still exist
	iter, err = stub.GetStateByPartialCompositeKey("color~name", []string{"marble", "red"})
	assert.NoError(t, err, "composite key query for color should not throw error")
	assert.NotNil(t, iter, "composite key query resultset should not be nil")
	v, err := iter.Next()
	assert.NoError(t, err, "composite key query should return a key")
	_, cp, err := stub.SplitCompositeKey(v.Key)
	assert.NoError(t, err, "composite key query should return a valid key")
	assert.Equal(t, "marble8", cp[len(cp)-1], "returned composite key should be 'marble8'")
	iter.Close()
	stub.MockTransactionEnd("6")
}
