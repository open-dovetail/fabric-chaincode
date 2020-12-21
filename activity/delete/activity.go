package delete

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-delete")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// Activity is a stub for executing Hyperledger Fabric delete operations
type Activity struct {
	compositeKeys map[string][]string
	keysOnly      bool
}

// New creates a new Activity
func New(ctx activity.InitContext) (activity.Activity, error) {
	s := &Settings{}
	if err := s.FromMap(ctx.Settings()); err != nil {
		return nil, err
	}

	return &Activity{
		compositeKeys: s.CompositeKeys,
		keysOnly:      s.KeysOnly,
	}, nil

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

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		msg := fmt.Sprintf("failed to retrieve fabric stub: %v", err)
		logger.Errorf("%s\n", msg)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	var code int
	var compositeKeys []string
	stateMap := make(map[string]interface{})

	switch t := reflect.TypeOf(input.Data).Kind(); t {
	case reflect.Slice:
		data := input.Data.([]interface{})
		for _, item := range data {
			c, v, e := a.collectData(stub, input.PrivateCollection, item)
			if e != nil {
				err = e
			}
			if c > code {
				code = c
			}
			if a.keysOnly {
				if keys, ok := v.([]string); ok && len(keys) > 0 {
					compositeKeys = append(compositeKeys, keys...)
				}
			} else {
				if states, ok := v.(map[string]interface{}); ok && len(states) > 0 {
					for s := range states {
						if len(s) > 0 {
							stateMap[s] = nil
						}
					}
				}
			}
		}
	case reflect.Map, reflect.String:
		// process single data object
		var v interface{}
		code, v, err = a.collectData(stub, input.PrivateCollection, input.Data)
		if v != nil {
			if a.keysOnly {
				compositeKeys = v.([]string)
			} else {
				stateMap = v.(map[string]interface{})
			}
		}
	default:
		msg := fmt.Sprintf("invalid input data type %T", input.Data)
		logger.Errorf("%s\n", msg)
		output := &Output{Code: 400, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	var result []interface{}
	if len(stateMap) > 0 {
		// delete collected ledger states
		for s := range stateMap {
			c, v, e := a.deleteDataByKey(stub, input.PrivateCollection, s)
			if e != nil {
				err = e
			}
			if c > code {
				code = c
			}
			if v != nil {
				result = append(result, map[string]interface{}{
					common.KeyField:   s,
					common.ValueField: v,
				})
			}
		}
	} else if len(compositeKeys) > 0 {
		// merge and convert deleted composite keys
		keyMap := make(map[string]*common.CompositeKeyBag)
		for _, k := range compositeKeys {
			// construct key objects from composite key strings
			if c, err := common.SplitCompositeKey(stub, k); err == nil {
				bag, ok := keyMap[c.Name]
				if !ok {
					bag = &common.CompositeKeyBag{
						Name:       c.Name,
						Attributes: a.compositeKeys[c.Name],
					}
					keyMap[c.Name] = bag
				}
				bag.AddCompositeKey(c)
			}
		}
		// convert compositeKeyBag to map and add to result
		for _, v := range keyMap {
			if s, e := v.ToMap(); e == nil {
				result = append(result, s)
			}
		}
	}

	// set partial success code
	if len(result) > 0 && code >= 300 {
		code = 206
		err = nil
	}

	if code == 404 {
		// no data response
		output := &Output{Code: 404, Message: "no data deleted"}
		ctx.SetOutputObject(output)
		return true, nil
	}

	if err != nil {
		// error response
		output := &Output{Code: code, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}

	// successful response
	data, _ := json.Marshal(result)
	output := &Output{
		Code:    200,
		Message: string(data),
		Result:  result,
	}
	ctx.SetOutputObject(output)
	return true, nil
}

// delete ledger state and associated composite keys by a specified state key
// returns status code, deleted state object, or error
//   It should be called only if keysOnly is false
func (a *Activity) deleteDataByKey(stub shim.ChaincodeStubInterface, collection string, key string) (int, interface{}, error) {
	if len(key) == 0 {
		return 400, nil, errors.New("state key is not specified")
	}

	_, jsonBytes, err := common.GetData(stub, collection, key)
	if err != nil {
		msg := fmt.Sprintf("failed to get data '%s @ %s'", key, collection)
		logger.Errorf("%s: %+v\n", msg, err)
		return 500, nil, errors.Wrapf(err, msg)
	}
	if jsonBytes == nil {
		msg := fmt.Sprintf("no data found for '%s @ %s'\n", key, collection)
		logger.Debugf("%s'\n", msg)
		return 404, nil, errors.New(msg)
	}

	// delete data
	if err := common.DeleteData(stub, collection, key); err != nil {
		msg := fmt.Sprintf("failed to delete data %s @ %s", key, collection)
		logger.Errorf("%s: %+v\n", msg, err)
		return 500, nil, errors.Wrapf(err, msg)
	}
	logger.Debugf("deleted %s @ %s, data: %s\n", key, collection, string(jsonBytes))

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		msg := fmt.Sprintf("failed to parse JSON data - %s", string(jsonBytes))
		logger.Errorf("%s: %+v\n", msg, err)
		return 500, nil, errors.Wrapf(err, msg)
	}

	// delete composite keys if specified
	compKeys := common.ExtractCompositeKeys(stub, a.compositeKeys, key, value)
	if len(compKeys) > 0 {
		for _, k := range compKeys {
			if err := common.DeleteData(stub, collection, k); err != nil {
				logger.Warnf("failed to delete composite key %s @ %s: %+v\n", k, collection, err)
			} else {
				logger.Debugf("deleted composite key %s @ %s\n", k, collection)
			}
		}
	}

	return 200, value, nil
}

// if keysOnly = false, collect unique state key, and return it as map[string]nil
// if keysOnly = true, delete composite keys, and return them as []string
func (a *Activity) collectData(stub shim.ChaincodeStubInterface, collection string, data interface{}) (int, interface{}, error) {
	switch t := reflect.TypeOf(data).Kind(); t {
	case reflect.String:
		// evaluate a state key
		if a.keysOnly {
			msg := fmt.Sprintf("cannot delete state key %v while keysOnly is true", data)
			logger.Errorf("%s\n", msg)
			return 400, nil, errors.New(msg)
		}
		k := data.(string)
		if len(k) == 0 {
			return 400, nil, errors.New("cannot delete empty key")
		}
		return 200, map[string]interface{}{k: nil}, nil
	case reflect.Map:
		request := data.(map[string]interface{})
		return a.deleteDataByPartialKey(stub, collection, request)
	default:
		msg := fmt.Sprintf("invalid input data type %T", data)
		logger.Errorf("%s\n", msg)
		return 400, nil, errors.New(msg)
	}
}

// delete composite keys matching the partial key query result, or collect unique state keys for deletion
// returns status code, result, or error
//   If keysOnly is true, delete only associated composite keys, and return the array of deleted keys
//   if keysOnly is false, collect unique state keys, and return them as map[string]nil
func (a *Activity) deleteDataByPartialKey(stub shim.ChaincodeStubInterface, collection string, data map[string]interface{}) (int, interface{}, error) {
	if len(data) == 0 {
		return 400, nil, errors.New("partial composite key is not specified")
	}
	keys := common.ExtractCompositeKeys(stub, a.compositeKeys, "", data)
	if len(keys) == 0 {
		msg := fmt.Sprintf("no composite key found for '%v'\n", data)
		logger.Debugf("%s'\n", msg)
		return 404, nil, errors.New(msg)
	}

	if a.keysOnly {
		// delete composite keys
		var compKeys []string
		for _, k := range keys {
			cks, err := deleteCompositeKeys(stub, collection, k)
			if err != nil {
				continue
			}
			compKeys = append(compKeys, cks...)
		}
		if len(compKeys) > 0 {
			return 200, compKeys, nil
		}
		return 404, nil, errors.Errorf("no data found for partial key %v", data)
	}

	// collect unique state keys
	stateKeys := make(map[string]interface{})
	for _, k := range keys {
		stateMap, err := collectStatesByCompositeKey(stub, collection, k)
		if err != nil {
			continue
		}
		for s := range stateMap {
			stateKeys[s] = nil
		}
	}

	if len(stateKeys) > 0 {
		return 200, stateKeys, nil
	}
	return 404, nil, errors.Errorf("no data found for partial key %v", data)
}

// delete composite keys only, called when keysOnly == true
// return list of deleted composite keys
func deleteCompositeKeys(stub shim.ChaincodeStubInterface, collection string, key string) ([]string, error) {
	ck, err := common.SplitCompositeKey(stub, key)
	if err != nil {
		msg := fmt.Sprintf("invalid composite key %s", key)
		logger.Warnf("%s: %v\n", msg, err)
		return nil, errors.Wrapf(err, msg)
	}
	// query matching composite keys
	iter, _, err := common.GetCompositeKeys(stub, collection, ck.Name, ck.Fields, 0, "")
	if err != nil {
		msg := fmt.Sprintf("error executing partial key query for %s", key)
		logger.Warnf("%s: %v\n", msg, err)
		return nil, errors.Wrapf(err, msg)
	}
	defer iter.Close()

	var compKeys []string
	for iter.HasNext() {
		resp, err := iter.Next()
		if err != nil {
			logger.Warnf("ignore query iterator error %v\n", err)
			continue
		}
		// delete composite key
		if err := common.DeleteData(stub, collection, resp.Key); err == nil {
			// add key attributes to result array
			compKeys = append(compKeys, resp.Key)
		}
	}
	return compKeys, nil
}

// collect state keys to be deleted, called when keysOnly == false
// associated composite keys will be deleted later when states are deleted
func collectStatesByCompositeKey(stub shim.ChaincodeStubInterface, collection string, key string) (map[string]interface{}, error) {
	ck, err := common.SplitCompositeKey(stub, key)
	if err != nil {
		msg := fmt.Sprintf("invalid composite key %s", key)
		logger.Warnf("%s: %v\n", msg, err)
		return nil, errors.Wrapf(err, msg)
	}
	// query matching composite keys
	iter, _, err := common.GetCompositeKeys(stub, collection, ck.Name, ck.Fields, 0, "")
	if err != nil {
		msg := fmt.Sprintf("error executing partial key query for %s", key)
		logger.Warnf("%s: %v\n", msg, err)
		return nil, errors.Wrapf(err, msg)
	}
	defer iter.Close()

	stateKeys := make(map[string]interface{})
	for iter.HasNext() {
		resp, err := iter.Next()
		if err != nil {
			logger.Warnf("ignore key iterator error %v\n", err)
			continue
		}
		// add state key
		if c, err := common.SplitCompositeKey(stub, resp.Key); err != nil {
			logger.Warnf("ignore invalid composite key %s with parsing error %v\n", resp.Key, err)
		} else {
			// collect unique state keys
			stateKeys[c.Key] = nil
		}
	}
	return stateKeys, nil
}
