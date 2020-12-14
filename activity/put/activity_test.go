package put

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {

	ref := activity.GetRef(&Activity{})
	act := activity.Get(ref)

	assert.NotNil(t, act)
}

func TestCreate(t *testing.T) {

	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(Settings{}, mf)
	act, err := New(iCtx)
	assert.Nil(t, err)
	assert.NotNil(t, act, "activity should not be nil")
}

func TestEval(t *testing.T) {
	// config activity to add 2 composite keys for each ledger record
	sConfig := `{
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
        }
	}`
	sMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(sConfig), &sMap)
	assert.NoError(t, err, "unmarshal setting config should not throw error")

	settings := &Settings{}
	settings.FromMap(sMap)
	assert.Equal(t, 2, len(settings.CompositeKeys), "number of configured composite key should be 2")

	act := &Activity{compositeKeys: settings.CompositeKeys}
	tc := test.NewActivityContext(act.Metadata())
	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// sample test message
	data := `{
		"docType": "marble",
		"name": "marble1",
		"color": "blue",
		"size": 50,
		"owner": "tom"
	}`
	input := &Input{StateKey: "marble1", StateData: data}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
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
	assert.Equal(t, "marble1", output.Result["name"].(string), "result object should have a name attribute of marble1")

	// verify correct update of mock Fabric state
	stub.MockTransactionStart("2")
	val, err := stub.GetState("marble1")
	assert.NoError(t, err, "retrieve state of marble1 should not throw error")
	var rec map[string]interface{}
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
	stub.MockTransactionEnd("2")
}
