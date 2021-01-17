package endorsement

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
	settings := map[string]interface{}{
		"operation": "ADD",
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

func TestAddOrgs(t *testing.T) {
	logger.Info("TestAddOrgs")
	act.operation = "ADD"

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// input data
	req := `{
		"keys": ["key1", "key2"],
		"organizations": ["org1", "org2"]
	}`
	var data map[string]interface{}
	err := json.Unmarshal([]byte(req), &data)
	assert.NoError(t, err, "input data should be valid JSON object")

	input := &Input{}
	err = input.FromMap(data)
	assert.NoError(t, err, "create input from map should not throw error")
	assert.Equal(t, 2, len(input.Organizations), "input should contain 2 orgs")
	assert.Equal(t, 2, len(input.StateKeys), "input should contain 2 state keys")

	input = &Input{
		StateKeys:     []string{"key1", "key2"},
		Organizations: []string{"org1", "org2"},
	}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// process request
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
	assert.Equal(t, 2, len(output.Result), "result should contain 2 records")

	rec := output.Result[0].(map[string]interface{})
	assert.Equal(t, "key1", rec["key"].(string), "result map's key should be 'key1'")
	assert.Equal(t, 2, len(rec["organizations"].([]string)), "result policy should include 2 organizations")

	policy := rec["policy"].(map[string]interface{})
	rule := policy["rule"].(map[string]interface{})
	assert.Equal(t, int32(2), rule["outOf"].(int32), "result rule should require 2 signatures")
}
