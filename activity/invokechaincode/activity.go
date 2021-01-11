package invokechaincode

import (
	"encoding/json"
	"fmt"

	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-invokechaincode")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// Activity is a stub for executing Hyperledger Fabric get operations
type Activity struct {
}

// New creates a new Activity
func New(ctx activity.InitContext) (activity.Activity, error) {
	return &Activity{}, nil
}

// Metadata implements activity.Activity.Metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements activity.Activity.Eval
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	input := &Input{}
	if err = ctx.GetInputObject(input); err != nil {
		return false, err
	}

	if input.ChaincodeName == "" {
		msg := "chaincode name is not specified"
		logger.Error(msg)
		output := &Output{Code: 400, Message: msg}
		ctx.SetOutputObject(output)
		return false, errors.New(msg)
	}
	logger.Debugf("chaincode name: %s", input.ChaincodeName)
	logger.Debugf("channel ID: %s", input.ChannelID)

	// extract transaction name and parameters
	args, err := constructChaincodeArgs(ctx, input)
	if err != nil {
		output := &Output{Code: 400, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		msg := fmt.Sprintf("failed to retrieve fabric stub: %+v", err)
		logger.Errorf("%s", msg)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	// invoke chaincode
	response := stub.InvokeChaincode(input.ChaincodeName, args, input.ChannelID)
	output := &Output{Code: int(response.GetStatus()), Message: response.GetMessage()}
	jsonBytes := response.GetPayload()
	if jsonBytes == nil {
		logger.Debugf("no data returned by chaincode %s", input.ChaincodeName)
		ctx.SetOutputObject(output)
		return true, nil
	}

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		logger.Errorf("failed to unmarshal chaincode response %+v, error: %+v\n", string(jsonBytes), err)
		output.Result = string(jsonBytes)
		ctx.SetOutputObject(output)
		return true, nil
	}
	output.Result = value
	ctx.SetOutputObject(output)
	return true, nil
}

func constructChaincodeArgs(ctx activity.Context, input *Input) ([][]byte, error) {
	var result [][]byte
	// transaction name from input
	if input.TransactionName == "" {
		msg := "transaction name is not specified"
		logger.Error(msg)
		return nil, errors.New(msg)
	}
	logger.Debugf("transaction name: %s", input.TransactionName)
	result = append(result, []byte(input.TransactionName))

	if len(input.Parameters) == 0 {
		logger.Debug("no parameter is specified")
		return result, nil
	}

	// add transaction parameters
	for _, p := range input.Parameters {
		param := fmt.Sprintf("%v", p)
		logger.Debugf("add chaincode parameter: %s", p)
		result = append(result, []byte(param))
	}
	return result, nil
}
