package gethistory

import (
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
	act := &Activity{}
	tc := test.NewActivityContext(act.Metadata())
	stub := shimtest.NewMockStub("mock", nil)
	err := tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)
	assert.NoError(t, err, "add mock stub should not throw error")

	// setup mock state for query by range
	sample := `{
		"docType": "marble",
		"name": "marble1",
		"color": "blue",
		"size": 50,
		"owner": "tom"
	}`
	stub.MockTransactionStart("1")
	err = stub.PutState("marble1", []byte(sample))
	assert.NoError(t, err, "insert mock data should not throw error")
	stub.MockTransactionEnd("1")

	// set query with state key 'marble1'
	err = tc.SetInputObject(&Input{StateKey: "marble1"})
	assert.NoError(t, err, "setting action input should not throw error")

	// process request using mock Fabric transaction
	stub.MockTransactionStart("2")
	done, err := act.Eval(tc)
	stub.MockTransactionEnd("2")
	assert.False(t, done, "action eval should fail")
	assert.Contains(t, err.Error(), "not implemented", "error message should show not implemented by mock")

	// verify activity output
	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.NoError(t, err, "get action output should not throw error")
	assert.Equal(t, 500, output.Code, "action output status should be 500")
	assert.Contains(t, output.Message, "marble1", "response error shows failed query")
}
