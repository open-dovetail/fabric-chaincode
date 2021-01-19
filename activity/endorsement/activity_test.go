/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package endorsement

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric/common/policydsl"
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

	policy := rec["policy"].(map[string]interface{})
	assert.Equal(t, 2, len(policy["orgs"].([]interface{})), "result policy should include 2 organizations")
	rule := policy["rule"].(map[string]interface{})
	assert.Equal(t, int32(2), rule["outOf"].(int32), "result rule should require 2 signatures")
}

func TestSetPolicy(t *testing.T) {
	logger.Info("TestSetPolicy")
	act.operation = "SET"

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// input data
	req := `{
		"keys": "key1",
		"policy": "OutOf(2, 'org1.peer', 'org2.peer', 'org3.peer')"
	}`
	var data map[string]interface{}
	err := json.Unmarshal([]byte(req), &data)
	assert.NoError(t, err, "input data should be valid JSON object")

	input := &Input{}
	err = input.FromMap(data)
	assert.NoError(t, err, "create input from map should not throw error")
	assert.Equal(t, 1, len(input.StateKeys), "input should contain 1 state keys")

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
	assert.Equal(t, 1, len(output.Result), "result should contain 1 record")

	rec := output.Result[0].(map[string]interface{})
	assert.Equal(t, "key1", rec["key"].(string), "result map's key should be 'key1'")
	policy := rec["policy"].(map[string]interface{})
	assert.Equal(t, 3, len(policy["orgs"].([]interface{})), "result policy should include 3 organizations")
	rule := policy["rule"].(map[string]interface{})
	assert.Equal(t, int32(2), rule["outOf"].(int32), "result rule should require 2 signatures")
}

func setTestPolicy(stub shim.ChaincodeStubInterface, key string) {
	policy := "OutOf(1, 'org1.peer', 'org2.peer', 'org3.peer')"
	envelope, _ := policydsl.FromString(policy)
	ep, _ := proto.Marshal(envelope)
	stub.SetStateValidationParameter(key, ep)
}

func TestListPolicy(t *testing.T) {
	logger.Info("TestListPolicy")
	act.operation = "LIST"

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// input data
	req := `{
		"keys": "key1"
	}`
	var data map[string]interface{}
	err := json.Unmarshal([]byte(req), &data)
	assert.NoError(t, err, "input data should be valid JSON object")

	input := &Input{}
	err = input.FromMap(data)
	assert.NoError(t, err, "create input from map should not throw error")
	assert.Equal(t, 1, len(input.StateKeys), "input should contain 1 state keys")

	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// prepare endorsement policy
	setTestPolicy(stub, "key1")

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
	assert.Equal(t, 1, len(output.Result), "result should contain 1 record")

	rec := output.Result[0].(map[string]interface{})
	assert.Equal(t, "key1", rec["key"].(string), "result map's key should be 'key1'")
	policy := rec["policy"].(map[string]interface{})
	assert.Equal(t, 3, len(policy["orgs"].([]interface{})), "result policy should include 3 organizations")
	rule := policy["rule"].(map[string]interface{})
	assert.Equal(t, int32(1), rule["outOf"].(int32), "result rule should require 1 signature")
}

func TestDeleteOrgs(t *testing.T) {
	logger.Info("TestDeleteOrgs")
	act.operation = "DELETE"

	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	// input data
	req := `{
		"keys": "key1",
		"organizations": "org1"
	}`
	var data map[string]interface{}
	err := json.Unmarshal([]byte(req), &data)
	assert.NoError(t, err, "input data should be valid JSON object")

	input := &Input{}
	err = input.FromMap(data)
	assert.NoError(t, err, "create input from map should not throw error")
	assert.Equal(t, 1, len(input.StateKeys), "input should contain 1 state keys")

	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// prepare endorsement policy
	setTestPolicy(stub, "key1")

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
	assert.Equal(t, 1, len(output.Result), "result should contain 1 record")

	rec := output.Result[0].(map[string]interface{})
	assert.Equal(t, "key1", rec["key"].(string), "result map's key should be 'key1'")
	policy := rec["policy"].(map[string]interface{})
	assert.Equal(t, 2, len(policy["orgs"].([]interface{})), "result policy should include 2 organizations")
	rule := policy["rule"].(map[string]interface{})
	assert.Equal(t, int32(2), rule["outOf"].(int32), "result rule should require 2 signatures")
}
