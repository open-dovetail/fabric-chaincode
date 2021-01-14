/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package setevent

import (
	"encoding/json"
	"fmt"

	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-setevent")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// Activity is a stub for executing Hyperledger Fabric setevent operations
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

	if input.Name == "" {
		msg := "event name is not specified"
		logger.Error(msg)
		output := &Output{Code: 400, Message: msg}
		ctx.SetOutputObject(output)
		return false, errors.New(msg)
	}
	logger.Debugf("event name: %s", input.Name)

	var jsonBytes []byte
	if input.Payload != nil {
		jsonBytes, err = json.Marshal(input.Payload)
		if err != nil {
			logger.Warnf("failed to marshal payload '%+v', error: %+v\n", input.Payload, err)
			pl := fmt.Sprintf("%v", input.Payload)
			jsonBytes = []byte(pl)
		}
	}
	logger.Debugf("event payload: %s", string(jsonBytes))

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		msg := fmt.Sprintf("failed to retrieve fabric stub: %v", err)
		logger.Errorf("%s", msg)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	// set fabric event
	if err := stub.SetEvent(input.Name, jsonBytes); err != nil {
		msg := fmt.Sprintf("failed to set event %s, error: %v", input.Name, err)
		logger.Errorf("%s", msg)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	result := map[string]interface{}{
		"name":    input.Name,
		"payload": input.Payload,
	}
	msgbytes, _ := json.Marshal(result)
	logger.Debugf("set activity output result: %v", result)
	output := &Output{Code: 200,
		Message: string(msgbytes),
		Result:  result,
	}
	ctx.SetOutputObject(output)
	return true, nil
}
