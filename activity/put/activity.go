package put

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/coerce"
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
	keysOnly      bool
	createOnly    bool
}

func (a *Activity) String() string {
	return fmt.Sprintf("PutActivity(keys:%v, keyOnly:%t, createOnly:%t)", a.compositeKeys, a.keysOnly, a.createOnly)
}

// New creates a new Activity
func New(ctx activity.InitContext) (activity.Activity, error) {
	s := &Settings{}
	logger.Infof("Create Put activity with InitContxt settings %v", ctx.Settings())
	if err := s.FromMap(ctx.Settings()); err != nil {
		logger.Errorf("failed to configure Put activity %v", err)
		return nil, err
	}

	return &Activity{
		compositeKeys: s.CompositeKeys,
		keysOnly:      s.KeysOnly,
		createOnly:    s.CreateOnly,
	}, nil
}

// Metadata implements activity.Activity.Metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements activity.Activity.Eval
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	logger.Debugf("%v", a)

	// check input args
	input := &Input{}
	if err = ctx.GetInputObject(input); err != nil {
		return false, err
	}

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		msg := fmt.Sprintf("failed to retrieve fabric stub: %v", err)
		logger.Errorf("%s", msg)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	var code int
	var value []interface{}

	switch t := reflect.TypeOf(input.Data).Kind(); t {
	case reflect.Slice:
		data := input.Data.([]interface{})
		for _, item := range data {
			d, ok := item.(map[string]interface{})
			if !ok {
				logger.Warnf("ignore bad input data %v", item)
				continue
			}
			c, v, e := a.storeData(stub, input.PrivateCollection, d)
			if e != nil {
				err = e
			}
			if c > code {
				code = c
			}
			if len(v) > 0 {
				value = append(value, v...)
			}
		}
	case reflect.Map:
		// update single data object
		data := input.Data.(map[string]interface{})
		code, value, err = a.storeData(stub, input.PrivateCollection, data)
	default:
		msg := fmt.Sprintf("invalid input data type %T", input.Data)
		logger.Errorf("%s", msg)
		output := &Output{Code: 400, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	// set partial success code
	if len(value) > 0 && code >= 300 {
		code = 206
		err = nil
	}
	if code == 404 {
		// no data response
		output := &Output{Code: 404, Message: "no data updated"}
		ctx.SetOutputObject(output)
		return true, nil
	}

	if err != nil {
		// error response
		output := &Output{Code: code, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}

	if a.keysOnly && len(value) > 0 {
		// merge key map
		keyMap := make(map[string]*common.CompositeKeyBag)
		for _, v := range value {
			if reflect.TypeOf(v).Elem().Name() == "CompositeKeyBag" {
				b := v.(*common.CompositeKeyBag)
				bag, ok := keyMap[b.Name]
				if ok {
					bag.Keys = append(bag.Keys, b.Keys...)
				} else {
					keyMap[b.Name] = b
				}
			}
		}
		// transform merged bag to map
		var keys []interface{}
		for _, v := range keyMap {
			if key, e := v.ToMap(); e == nil {
				logger.Debugf("merged composite key %v", key)
				keys = append(keys, key)
			}
		}
		value = keys
	}

	// successful response
	data, _ := json.Marshal(value)
	output := &Output{
		Code:    code,
		Message: string(data),
		Result:  value,
	}
	ctx.SetOutputObject(output)
	return true, nil
}

// process one input data object
//   - data must be either a key-value map for state, or a data object containing composite-key attributes
//   - composite-key data object is ignored if KeysOnly is false
// returns status code, updated states or composite keys, or error
//   - if input data is key-value, return the key-value object for updated states
//   - if input data is not key-value, return list of created composite-keys
func (a *Activity) storeData(stub shim.ChaincodeStubInterface, collection string, data map[string]interface{}) (int, []interface{}, error) {
	key := data[common.KeyField]
	value := data[common.ValueField]
	if len(data) == 2 && key != nil && value != nil {
		// this is key-value for state update
		if a.keysOnly {
			logger.Warnf("update state key %s although activity is configured to write keys only", key)
		}
		stateKey, err := coerce.ToString(key)
		if err != nil {
			return 400, nil, errors.Errorf("invalid state key: %v", key)
		}
		code, err := a.putData(stub, collection, stateKey, value)
		if err != nil {
			return code, nil, err
		}
		return code, []interface{}{data}, nil
	}

	if !a.keysOnly {
		// keyOnly is not set, it must not create composite key w/o state update
		return 400, nil, errors.New("key is not specified for state update")
	}

	// store composite keys
	code, keys, err := a.putCompositeKey(stub, collection, data)
	if err != nil {
		return code, nil, err
	}
	var result []interface{}
	for _, k := range keys {
		// construct key objects from composite key strings
		if c, err := common.SplitCompositeKey(stub, k); err == nil {
			bag := &common.CompositeKeyBag{
				Name:       c.Name,
				Attributes: a.compositeKeys[c.Name],
			}
			if bag, err = bag.AddCompositeKey(c); err == nil {
				result = append(result, bag)
			}
		}
	}
	return code, result, nil
}

// update specified key-value on ledger or private data collection, and create associated composite keys
// if createOnly setting is true, do not update it, instead return 409 if already exist
// returns status code, updated state object, or error
func (a *Activity) putData(stub shim.ChaincodeStubInterface, collection string, key string, data interface{}) (int, error) {
	if len(key) == 0 {
		return 400, errors.New("state key is not specified")
	}
	if a.createOnly {
		// check if key already exist
		if _, v, err := common.GetData(stub, collection, key); err == nil && v != nil {
			return 409, errors.New("state key already exists")
		}
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		msg := fmt.Sprintf("failed to marshal data: %+v", data)
		logger.Errorf("%s: %+v", msg, err)
		return 400, errors.Wrapf(err, msg)
	}

	// store data on ledger or private data collection
	if err := common.PutData(stub, collection, key, jsonBytes); err != nil {
		msg := fmt.Sprintf("failed to store data %s @ %s", key, collection)
		logger.Errorf("%s: %+v", msg, err)
		return 500, errors.Wrapf(err, msg)
	}
	logger.Debugf("stored data %s @ %s, data: %s", key, collection, string(jsonBytes))

	// store composite keys if required
	compKeys := common.ExtractCompositeKeys(stub, a.compositeKeys, key, data)
	if len(compKeys) > 0 {
		for _, k := range compKeys {
			if err := common.PutData(stub, collection, k, nil); err != nil {
				logger.Warnf("failed to store composite key %s @ %s: %+v", k, collection, err)
			} else {
				logger.Debugf("stored composite key %s @ %s", k, collection)
			}
		}
	}

	return 200, nil
}

// create composite keys on ledger or private collection
// returns status code, list of composite keys, or error
func (a *Activity) putCompositeKey(stub shim.ChaincodeStubInterface, collection string, data map[string]interface{}) (int, []string, error) {
	if len(data) == 0 {
		return 400, nil, errors.New("attributes for composite keys are not specified")
	}
	if len(a.compositeKeys) == 0 {
		return 404, nil, errors.New("no composite key is defined")
	}
	// put only complete keys that contain all attributes
	var result []string
	for name, attrs := range a.compositeKeys {
		if key, isComplete := common.MakeCompositeKey(stub, name, attrs, "", data); isComplete {
			if err := common.PutData(stub, collection, key, nil); err == nil {
				result = append(result, key)
			}
		}
	}

	if len(result) > 0 {
		return 200, result, nil
	}
	return 404, nil, errors.New("data not complete for any composite key")
}
