package gethistory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-gethistory")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// Activity is a stub for executing Hyperledger Fabric get-history operations
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

	if len(input.StateKey) == 0 {
		logger.Error("state key is not specified\n")
		output := &Output{Code: 400, Message: "state key is not specified"}
		ctx.SetOutputObject(output)
		return false, errors.New(output.Message)
	}
	logger.Debugf("state key: %s\n", input.StateKey)

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		logger.Errorf("failed to retrieve fabric stub: %+v\n", err)
		output := &Output{Code: 500, Message: fmt.Sprintf("failed to retrieve fabric stub: %v", err)}
		ctx.SetOutputObject(output)
		return false, err
	}

	// retrieve history records for the key
	code, jsonBytes, err := retrieveHistory(stub, input.StateKey)

	if err != nil {
		// error response
		output := &Output{Code: code, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}
	if code == 404 {
		// no data response
		output := &Output{Code: 404, Message: "no data found"}
		ctx.SetOutputObject(output)
		return true, nil
	}

	var value []interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		msg := fmt.Sprintf("failed to parse JSON data - %s", string(jsonBytes))
		logger.Errorf("%s: %+v\n", msg, err)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, errors.Wrapf(err, msg)
	}
	output := &Output{
		Code:     200,
		Message:  string(jsonBytes),
		StateKey: input.StateKey,
		Result:   value,
	}
	ctx.SetOutputObject(output)

	return true, nil
}

func retrieveHistory(stub shim.ChaincodeStubInterface, key string) (int, []byte, error) {
	// retrieve data for the key
	resultsIterator, err := stub.GetHistoryForKey(key)
	if err != nil {
		msg := fmt.Sprintf("failed to retrieve history for key %s", key)
		logger.Errorf("%s: %+v\n", msg, err)
		return 500, nil, errors.Wrapf(err, msg)
	}
	defer resultsIterator.Close()

	jsonBytes, err := constructHistoryResponse(resultsIterator)
	if err != nil {
		msg := "failed to collect history records from iterator"
		logger.Errorf("%s: %+v\n", msg, err)
		return 500, nil, errors.Wrapf(err, msg)
	}

	if jsonBytes == nil {
		msg := fmt.Sprintf("no history found for key %s\n", key)
		logger.Infof("%s\n", msg)
		return 404, nil, nil
	}
	logger.Debugf("retrieved history for key %s: %s\n", key, string(jsonBytes))

	return 200, jsonBytes, nil
}

func constructHistoryResponse(resultsIterator shim.HistoryQueryIteratorInterface) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("[")

	isEmpty := true
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if !isEmpty {
			buffer.WriteString(",")
		}

		buffer.WriteString("{\"" + common.FabricTxID + "\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"" + common.ValueField + "\":")
		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"" + common.FabricTxTime + "\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).UTC().Format(time.RFC3339Nano))
		buffer.WriteString("\"")

		buffer.WriteString(", \"" + common.ValueDeleted + "\":")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))

		buffer.WriteString("}")
		isEmpty = false
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}
