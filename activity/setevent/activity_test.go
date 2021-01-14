/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package setevent

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

func TestSetEvent(t *testing.T) {

	mf := mapper.NewFactory(resolve.GetBasicResolver())
	ctx := test.NewActivityInitContext(Settings{}, mf)
	act, err := New(ctx)
	assert.NoError(t, err, "create action instance should not throw error")

	tc := test.NewActivityContext(act.Metadata())
	stub := shimtest.NewMockStub("mock", nil)
	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	input := &Input{
		Name: "test",
		Payload: map[string]interface{}{
			"data": "test event",
		},
	}
	err = tc.SetInputObject(input)
	assert.NoError(t, err, "setting action input should not throw error")

	// start mock Fabric transaction, and set event
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
	assert.Equal(t, "test", output.Result["name"].(string), "resulting event name should be 'test'")

	// verify event data
	event := <-stub.ChaincodeEventsChannel
	var evtData map[string]interface{}
	json.Unmarshal(event.GetPayload(), &evtData)
	assert.Equal(t, "test event", evtData["data"].(string), "event payload data should be 'test event'")
}
