/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package invokechaincode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/open-dovetail/fabric-chaincode/common"

	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

// MockCC implements the shim.Chaincode interface, which is invoked by the test here
type MockCC struct {
}

// Init is called during chaincode instantiation to initialize any data
func (c *MockCC) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode.
// mock transaction accepts 3 params: strParam, numParam, boolParam of 3 types,
//      returns them in JSON object
func (c *MockCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fn, args := stub.GetFunctionAndParameters()
	logger.Debugf("invoke transaction fn=%s, args=%+v", fn, args)

	// check input parameters
	var msg string
	var n float64
	var b bool
	var err error

	if len(args) != 3 {
		msg = "mock transaction requires 3 parameters"
	} else if n, err = strconv.ParseFloat(args[1], 64); err != nil {
		msg = fmt.Sprintf("second parameter %s is not a number", args[1])
	} else if b, err = strconv.ParseBool(args[2]); err != nil {
		msg = fmt.Sprintf("third parameter %s is not a boolean", args[2])
	}
	if len(msg) > 0 {
		return pb.Response{
			Status:  400,
			Message: msg,
			Payload: nil,
		}
	}

	result := map[string]interface{}{
		"transaction": fn,
		"strParam":    args[0],
		"numParam":    n,
		"boolParam":   b,
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return pb.Response{
			Status:  500,
			Message: "MockCC returned error " + err.Error(),
			Payload: nil,
		}
	}
	return pb.Response{
		Status:  200,
		Message: "MockCC called successfully",
		Payload: payload,
	}
}

func TestInvokeCC(t *testing.T) {

	mf := mapper.NewFactory(resolve.GetBasicResolver())
	ctx := test.NewActivityInitContext(Settings{}, mf)
	act, err := New(ctx)
	assert.NoError(t, err, "create action instance should not throw error")

	// create invocable test chaincode
	testStub := shimtest.NewMockStub("testMock", &MockCC{})

	// test invoke chaincode
	tc := test.NewActivityContext(act.Metadata())
	stub := shimtest.NewMockStub("mock", nil)
	stub.MockPeerChaincode("testCC", testStub, "testChannel")

	tc.ActivityHost().Scope().SetValue(common.FabricStub, stub)

	input := &Input{
		ChaincodeName:   "testCC",
		ChannelID:       "testChannel",
		TransactionName: "testTx",
		Parameters:      []interface{}{"hello", 100.99, true},
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
	assert.Equal(t, "MockCC called successfully", output.Message)
	result := output.Result.(map[string]interface{})
	assert.True(t, result["boolParam"].(bool))
}
