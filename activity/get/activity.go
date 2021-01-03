package get

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-get")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// StateData contains a state key and its associated data
type StateData struct {
	Key   string
	Value []byte
}

// Activity is a stub for executing Hyperledger Fabric get operations
type Activity struct {
	keyName    string
	attributes []string
	query      string
	keysOnly   bool
	history    bool
}

func (a *Activity) String() string {
	return fmt.Sprintf("GetActivity(key:%s, attrs:%v, query:%s, keyOnly:%t, history:%t)", a.keyName, a.attributes, a.query, a.keysOnly, a.history)
}

// New creates a new Activity
func New(ctx activity.InitContext) (activity.Activity, error) {
	s := &Settings{}
	logger.Infof("Create Get activity with InitContxt settings %v", ctx.Settings())
	if err := s.FromMap(ctx.Settings()); err != nil {
		logger.Errorf("failed to configure Get activity %v", err)
		return nil, err
	}

	return &Activity{
		keyName:    s.KeyName,
		attributes: s.Attributes,
		query:      s.QueryStmt,
		keysOnly:   s.KeysOnly,
		history:    s.History,
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
	var bookmark string

	switch t := reflect.TypeOf(input.Data).Kind(); t {
	case reflect.Slice:
		data := input.Data.([]interface{})
		for _, item := range data {
			// Note: ignore pagination if multiple get operations are specified
			c, v, _, e := a.retrieveData(stub, input.PrivateCollection, item, 0, "")
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
	case reflect.Map, reflect.String:
		// update single data object
		code, value, bookmark, err = a.retrieveData(stub, input.PrivateCollection, input.Data, input.PageSize, input.Bookmark)
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
		output := &Output{Code: 404, Message: "no data found"}
		ctx.SetOutputObject(output)
		return true, nil
	}

	if err != nil {
		// error response
		output := &Output{Code: code, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}

	if len(value) > 0 {
		if a.keysOnly {
			// add composite key metadata
			bag := &common.CompositeKeyBag{
				Name:       a.keyName,
				Attributes: a.attributes,
			}
			for _, v := range value {
				if reflect.TypeOf(v).Kind() == reflect.String {
					if k, err := common.SplitCompositeKey(stub, v.(string)); err == nil {
						bag.AddCompositeKey(k)
					}
				}
			}
			// transform merged bag to map
			keys, _ := bag.ToMap()
			value = []interface{}{keys}
		} else {
			// expand ledger state value
			var result []interface{}
			for _, v := range value {
				if reflect.TypeOf(v).Elem().Name() == "StateData" {
					state := v.(*StateData)
					var d interface{}
					if err := json.Unmarshal(state.Value, &d); err == nil {
						rec := map[string]interface{}{
							common.KeyField:   state.Key,
							common.ValueField: d,
						}
						result = append(result, rec)
					}
				}
			}
			value = result
		}
	}

	// successful response
	data, _ := json.Marshal(value)
	output := &Output{
		Code:     code,
		Message:  string(data),
		Bookmark: bookmark,
		Result:   value,
	}
	ctx.SetOutputObject(output)
	return true, nil
}

// execute read or query operation to fetech data from ledger or private data collections
// return code, result, bookmark, or error
//   if keysOnly is true, result contains list of composite keys as []string
//   if keysOnly is false, result contains list of state key-value as []*StateData
func (a *Activity) retrieveData(stub shim.ChaincodeStubInterface, collection string, data interface{}, pageSize int32, bookmark string) (int, []interface{}, string, error) {
	switch t := reflect.TypeOf(data).Kind(); t {
	case reflect.String:
		// retrieve state by a key
		if a.keysOnly {
			msg := fmt.Sprintf("cannot retrieve state for key %v while keysOnly is true", data)
			logger.Errorf("%s", msg)
			return 400, nil, "", errors.New(msg)
		}
		code, value, err := a.retrieveDataByKey(stub, collection, data.(string))
		if err != nil {
			return code, nil, "", err
		}
		return code, []interface{}{value}, "", nil
	case reflect.Map:
		request := data.(map[string]interface{})
		if len(a.query) > 0 {
			// execute rich query if query statement is defined
			return a.retrieveDataByQuery(stub, collection, request, pageSize, bookmark)
		}
		rangeStart, okStart := request["start"]
		rangeEnd, okEnd := request["end"]
		if ((okStart || okEnd) && len(request) == 1) || (okStart && okEnd && len(request) == 2) {
			// execute range query for state keys
			return a.retrieveDataByRange(stub, collection, rangeStart, rangeEnd, pageSize, bookmark)
		}
		// fetch data by partial key
		return a.retrieveDataByPartialKey(stub, collection, request, pageSize, bookmark)
	default:
		msg := fmt.Sprintf("invalid input data type %T", data)
		logger.Errorf("%s", msg)
		return 400, nil, "", errors.New(msg)
	}
}

// retrieve data for a specified state key or composite key from the ledger or a private data collection
// return code, state or error
func (a *Activity) retrieveDataByKey(stub shim.ChaincodeStubInterface, collection string, key string) (int, *StateData, error) {
	if common.IsCompositeKey(key) {
		return 400, nil, errors.Errorf("Cannot get state for composite key %s", key)
	}

	var jsonBytes []byte
	var err error
	if a.history && len(collection) == 0 {
		jsonBytes, err = retrieveHistory(stub, key)
	} else {
		_, jsonBytes, err = common.GetData(stub, collection, key)
	}
	if err != nil {
		msg := fmt.Sprintf("failed to get data '%s @ %s'", key, collection)
		logger.Errorf("%s: %+v", msg, err)
		return 500, nil, errors.Wrapf(err, msg)
	}
	if jsonBytes == nil {
		msg := fmt.Sprintf("no data found for '%s @ %s'", key, collection)
		logger.Debugf("%s'", msg)
		return 404, nil, errors.New(msg)
	}
	logger.Debugf("retrieved data %s @ %s, data: %s", key, collection, string(jsonBytes))

	return 200, &StateData{Key: key, Value: jsonBytes}, nil
}

// execute rich query for ledger states
// returns code, result, bookmark or error
//   rich query does not apply to composite keys, so if keysOnly is set to true, this will return error
func (a *Activity) retrieveDataByQuery(stub shim.ChaincodeStubInterface, collection string, parameters interface{}, pageSize int32, bookmark string) (int, []interface{}, string, error) {
	if len(a.query) == 0 {
		msg := "rich query is not defined"
		logger.Errorf("%s", msg)
		return 400, nil, "", errors.New(msg)
	}
	if a.keysOnly {
		msg := "rich query cannot be executed for composite keys"
		logger.Errorf("%s", msg)
		return 400, nil, "", errors.New(msg)
	}
	qrystmt := a.query
	if params, ok := parameters.(map[string]interface{}); ok && len(params) > 0 {
		qrystmt = prepareQueryStatement(qrystmt, params)
	}

	// run rich query
	iter, queryMd, err := common.GetDataByQuery(stub, collection, qrystmt, pageSize, bookmark)
	if err != nil {
		msg := fmt.Sprintf("failed rich query '%s'; error: %v", qrystmt, err)
		logger.Errorf("%s", msg)
		return 500, nil, "", errors.New(msg)
	}
	defer iter.Close()

	// return the response values
	var values []interface{}
	for iter.HasNext() {
		resp, err := iter.Next()
		if err != nil {
			logger.Warnf("ignore query iterator error %v", err)
			continue
		}
		values = append(values, &StateData{
			Key:   resp.Key,
			Value: resp.Value,
		})
	}
	newBookmark := ""
	if queryMd != nil {
		newBookmark = queryMd.Bookmark
	}
	return 200, values, newBookmark, nil
}

func prepareQueryStatement(query string, params map[string]interface{}) string {
	if len(params) == 0 {
		logger.Debug("no parameter is defined for query")
		return query
	}

	// collect replacer args
	var args []string
	for k, v := range params {
		var value string
		ref := reflect.ValueOf(v)
		switch ref.Kind() {
		case reflect.Float64, reflect.Int32, reflect.Bool:
			value = fmt.Sprintf("%v", v)
		default:
			if jsonBytes, err := json.Marshal(v); err != nil {
				logger.Debugf("failed to marshal value %v: %+v", v, err)
				value = "null"
			} else {
				value = string(jsonBytes)
			}
		}
		args = append(args, fmt.Sprintf(`"$%s"`, k), value)
	}
	logger.Debugf("query replacer args %v", args)

	// replace query parameters with values
	r := strings.NewReplacer(args...)
	return r.Replace(query)
}

// execute range query for state key range
// returns code, result, bookmark or error
//   If keysOnly is true, error because range query works for state keys only
//   if keysOnly is false, result is a list of state data as []*StateData
func (a *Activity) retrieveDataByRange(stub shim.ChaincodeStubInterface, collection string, start interface{}, end interface{}, pageSize int32, bookmark string) (int, []interface{}, string, error) {
	if a.keysOnly {
		// when keysOnly is set, cannot run range query
		msg := "range query does not work for composite keys"
		logger.Errorf("%s", msg)
		return 400, nil, "", errors.New(msg)
	}

	rangeStart, ok := start.(string)
	if !ok {
		rangeStart = ""
	}
	rangeEnd, ok := end.(string)
	if !ok {
		rangeEnd = ""
	}

	// run range query
	iter, queryMd, err := common.GetDataByRange(stub, collection, rangeStart, rangeEnd, pageSize, bookmark)

	if err != nil {
		msg := fmt.Sprintf("range query error: %v", err)
		logger.Errorf("%s", msg)
		return 500, nil, "", errors.New(msg)
	}
	defer iter.Close()

	// collect state data
	var values []interface{}
	for iter.HasNext() {
		resp, err := iter.Next()
		if err != nil {
			logger.Warnf("ignore query iterator error %v", err)
			continue
		}
		values = append(values, &StateData{
			Key:   resp.Key,
			Value: resp.Value,
		})
	}
	newBookmark := ""
	if queryMd != nil {
		newBookmark = queryMd.Bookmark
	}
	return 200, values, newBookmark, nil
}

// retrieve composite keys or state data that matche the specified composite key data attributes.
// returns code, result, bookmark or error
//   If keysOnly is true, result is a list of composite keys as []string
//   if keysOnly is false, result is a list of state data as []*StateData
func (a *Activity) retrieveDataByPartialKey(stub shim.ChaincodeStubInterface, collection string, data map[string]interface{}, pageSize int32, bookmark string) (int, []interface{}, string, error) {
	if len(a.keyName) == 0 || len(data) == 0 {
		msg := fmt.Sprintf("composite key %s and data %v are not specified for partial key query", a.keyName, data)
		logger.Errorf("%s", msg)
		return 400, nil, "", errors.New(msg)
	}

	fields := common.ExtractDataAttributes(a.attributes, data)
	if len(fields) == 0 {
		msg := fmt.Sprintf("no field specified for composite key %s with attributes %v in data %+v", a.keyName, a.attributes, data)
		logger.Errorf("%s", msg)
		return 404, nil, "", errors.New(msg)
	}

	// run partial key query to get matching composite keys
	iter, queryMd, err := common.GetCompositeKeys(stub, collection, a.keyName, fields, pageSize, bookmark)
	if err != nil {
		msg := fmt.Sprintf("partial key query error: %v", err)
		logger.Errorf("%s", msg)
		return 500, nil, "", errors.New(msg)
	}
	defer iter.Close()

	var keys []interface{}
	for iter.HasNext() {
		resp, err := iter.Next()
		if err != nil {
			logger.Warnf("ignore key iterator error %v", err)
			continue
		}
		keys = append(keys, resp.Key)
	}
	newBookmark := ""
	if queryMd != nil {
		newBookmark = queryMd.Bookmark
	}
	if a.keysOnly {
		// returns keys
		return 200, keys, newBookmark, nil
	}

	// fetch corresponding state data
	var values []interface{}
	for _, ck := range keys {
		k, v, err := common.GetData(stub, collection, ck.(string))
		if err != nil {
			logger.Warnf("failed to data for composite key %s", ck)
			continue
		}
		values = append(values, &StateData{
			Key:   k,
			Value: v,
		})
	}
	return 200, values, newBookmark, nil
}

// retrieve history records of a specified state key
func retrieveHistory(stub shim.ChaincodeStubInterface, key string) ([]byte, error) {
	// retrieve data for the key
	resultsIterator, err := stub.GetHistoryForKey(key)
	if err != nil {
		msg := "error retrieving history"
		logger.Errorf("%s: %+v", msg, err)
		return nil, errors.Wrapf(err, msg)
	}
	defer resultsIterator.Close()

	jsonBytes, err := constructHistoryResponse(resultsIterator)
	if err != nil {
		msg := "history iterator error"
		logger.Errorf("%s: %+v", msg, err)
		return nil, errors.Wrapf(err, msg)
	}

	return jsonBytes, nil
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
