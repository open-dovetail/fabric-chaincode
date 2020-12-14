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
	assert.Nil(t, err, "unmarshal setting config should not throw error")

	settings := &Settings{}
	settings.FromMap(sMap)
	assert.Equal(t, 2, len(settings.CompositeKeys), "number of configured composite key should be 2")

	act := &Activity{compositeKeys: settings.CompositeKeys}
	tc := test.NewActivityContext(act.Metadata())
	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	data := `{
		"docType": "marble",
		"name": "marble1",
		"color", "blue",
		"size": 50,
		"owner": "tom"
	}`
	input := &Input{StateKey: "marble1", StateData: data}
	err = tc.SetInputObject(input)
	assert.Nil(t, err)

	done, err := act.Eval(tc)
	assert.False(t, done)
	assert.NotNil(t, err)

	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.Nil(t, err)
	assert.Equal(t, 500, output.Code)
}
