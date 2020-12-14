package get

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
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// setup mock ledger
	data := `{
		"docType": "marble",
		"name": "marble1",
		"color": "blue",
		"size": 50,
		"owner": "tom"
	}`
	stub.MockTransactionStart("1")
	stub.PutState("marble1", []byte(data))
	stub.MockTransactionEnd("1")

	input := &Input{StateKey: "marble1"}
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
	assert.Equal(t, "marble1", output.Result["name"].(string), "result object should have a name attribute of marble1")
}
