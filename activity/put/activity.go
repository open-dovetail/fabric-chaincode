package put

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-put")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// Activity is a stub for executing Hyperledger Fabric put operations
type Activity struct {
	compositeKeys map[string][]string
}

// New creates a new Activity
func New(ctx activity.InitContext) (activity.Activity, error) {
	s := &Settings{}
	if err := s.FromMap(ctx.Settings()); err == nil {
		return &Activity{compositeKeys: s.CompositeKeys}, nil
	}

	return &Activity{compositeKeys: nil}, nil
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
	if input.StateKey == "" {
		logger.Error("state key is not specified\n")
		output := &Output{Code: 400, Message: "state key is not specified"}
		ctx.SetOutputObject(output)
		return false, errors.New(output.Message)
	}
	logger.Debugf("state key: %s\n", input.StateKey)

	if len(input.StateData) == 0 {
		logger.Errorf("input data is empty\n")
		output := &Output{Code: 400, Message: "input data is empty"}
		ctx.SetOutputObject(output)
		return false, errors.New(output.Message)
	}
	logger.Debugf("input data: %+v\n", input.StateData)

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		logger.Errorf("failed to retrieve fabric stub: %+v\n", err)
		output := &Output{Code: 500, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}

	if input.PrivateCollection != "" {
		// store data on a private collection
		return a.storePrivateData(ctx, stub, input)
	}

	// store data on the ledger
	return a.storeData(ctx, stub, input)
}

func (a *Activity) storePrivateData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, input *Input) (bool, error) {
	jsonBytes := []byte(input.StateData)
	var dataObj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &dataObj); err != nil {
		logger.Errorf("input data is not a JSON object: '%s', error: %+v\n", input.StateData, err)
		output := &Output{Code: 400, Message: fmt.Sprintf("input data is not a JSON object: %s", input.StateData)}
		ctx.SetOutputObject(output)
		return false, errors.Wrapf(err, output.Message)
	}

	// store data on a private collection
	if err := ccshim.PutPrivateData(input.PrivateCollection, input.StateKey, jsonBytes); err != nil {
		logger.Errorf("failed to store data in private collection %s: %+v\n", input.PrivateCollection, err)
		output := &Output{Code: 500, Message: fmt.Sprintf("failed to store data in private collection %s", input.PrivateCollection)}
		ctx.SetOutputObject(output)
		return false, errors.Wrapf(err, output.Message)
	}
	logger.Debugf("stored in private collection %s, data: %s\n", input.PrivateCollection, input.StateData)

	// store composite keys if required
	compKeys := common.ExtractCompositeKeys(ccshim, a.compositeKeys, input.StateKey, dataObj)
	if compKeys != nil && len(compKeys) > 0 {
		for _, k := range compKeys {
			cv := []byte{0x00}
			if err := ccshim.PutPrivateData(input.PrivateCollection, k, cv); err != nil {
				logger.Errorf("failed to store composite key %s on collection %s: %+v\n", k, input.PrivateCollection, err)
			} else {
				logger.Debugf("stored composite key %s on collection %s\n", k, input.PrivateCollection)
			}
		}
	}

	output := &Output{
		Code:     200,
		Message:  input.StateData,
		StateKey: input.StateKey,
		Result:   dataObj,
	}
	ctx.SetOutputObject(output)
	return true, nil
}

func (a *Activity) storeData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, input *Input) (bool, error) {
	jsonBytes := []byte(input.StateData)
	var dataObj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &dataObj); err != nil {
		logger.Errorf("input data is not a JSON object: '%s', error: %+v\n", input.StateData, err)
		output := &Output{Code: 400, Message: fmt.Sprintf("input data is not a JSON object: %s", input.StateData)}
		ctx.SetOutputObject(output)
		return false, errors.Wrapf(err, output.Message)
	}

	// store data on the ledger
	if err := ccshim.PutState(input.StateKey, jsonBytes); err != nil {
		logger.Errorf("failed to store data on ledger: %+v\n", err)
		output := &Output{Code: 500, Message: "failed to store data on ledger"}
		ctx.SetOutputObject(output)
		return false, errors.Wrapf(err, output.Message)
	}
	logger.Debugf("stored data on ledger: %s\n", input.StateData)

	// store composite keys if required
	compKeys := common.ExtractCompositeKeys(ccshim, a.compositeKeys, input.StateKey, dataObj)
	if compKeys != nil && len(compKeys) > 0 {
		for _, k := range compKeys {
			cv := []byte{0x00}
			if err := ccshim.PutState(k, cv); err != nil {
				logger.Errorf("failed to store composite key %s: %+v\n", k, err)
			} else {
				logger.Debugf("stored composite key %s\n", k)
			}
		}
	}

	output := &Output{
		Code:     200,
		Message:  input.StateData,
		StateKey: input.StateKey,
		Result:   dataObj,
	}
	ctx.SetOutputObject(output)
	return true, nil
}
