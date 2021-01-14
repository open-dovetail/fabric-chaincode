/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package common

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/oliveagle/jsonpath"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/data/expression"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/data/schema"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/flow/instance"
)

const (
	// KeyField attribute used in query response of key-value pairs
	KeyField = "key"
	// ValueField attribute used in query response of key-value pairs
	ValueField = "value"
	// ValueDeleted marks a deleted Fabric state
	ValueDeleted = "isDeleted"
	// FabricStub is the name of flow property for passing chaincode stub to activities
	FabricStub = "_chaincode_stub"
	// FabricTxID is the name of flow property for passing auto-generated transaction ID to activities
	FabricTxID = "txID"
	// FabricTxTime is the name of flow property for passing auto-generated transaction Time to activities
	FabricTxTime = "txTime"
	// FabricCID is the name of flow property for passing client ID to activities
	FabricCID = "cid"

	compositeKeyNamespace = "\x00"
)

var logger = log.ChildLogger(log.RootLogger(), "fabric-chaincode-common")

// CompositeKey holds components in a composite key
type CompositeKey struct {
	Name   string   `json:"name,omitempty"`
	Fields []string `json:"fields"`
	Key    string   `json:"key"`
}

// CompositeKeyBag holds components in composite keys of the same name
type CompositeKeyBag struct {
	Name       string          `json:"name"`
	Attributes []string        `json:"attributes"`
	Keys       []*CompositeKey `json:"keys"`
}

// MapToObject strips extra nesting of mapping in config setting of activities exported by OSS Web UI
func MapToObject(data interface{}) (map[string]interface{}, error) {
	val, err := coerce.ToObject(data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal JSON data: %v", data)
	}

	// strip nested mapping for OSS Web UI
	if stripped, ok := val["mapping"]; ok && len(val) == 1 {
		// mapping is the only element of nesting, strip it
		result, ok := stripped.(map[string]interface{})
		if !ok {
			// stripped is not JSON object, so return the original data
			return val, nil
		}
		return result, nil
	}
	return val, nil
}

// GetChaincodeStub returns Fabric chaincode stub from the activity context
func GetChaincodeStub(ctx activity.Context) (shim.ChaincodeStubInterface, error) {
	// get chaincode stub - protect call to GetMasterScope to support unit test with mock
	scope := ctx.ActivityHost().Scope()
	if inst, ok := scope.(*instance.Instance); ok {
		scope = inst.GetMasterScope()
	}
	logger.Debugf("flow scope: %+v", scope)

	if stub, exists := scope.GetValue(FabricStub); exists && stub != nil {
		ccshim, found := stub.(shim.ChaincodeStubInterface)
		if !found {
			logger.Errorf("stub type %T is not a ChaincodeStubInterface\n", stub)
			return nil, errors.Errorf("stub type %T is not a ChaincodeStubInterface", stub)
		}
		return ccshim, nil
	}
	logger.Error("no stub found in flow scope")
	return nil, errors.New("no stub found in flow scope")
}

// GetActivityInputSchema returns schema of an activity input attribute
func GetActivityInputSchema(ctx activity.Context, name string) (string, error) {
	if sIO, ok := ctx.(schema.HasSchemaIO); ok {
		s := sIO.GetInputSchema(name)
		if s != nil {
			logger.Debugf("schema for attribute '%s': %T, %s\n", name, s, s.Value())
			return s.Value(), nil
		}
	}
	return "", errors.Errorf("schema not found for attribute %s", name)
}

// GetActivityOutputSchema returns schema of an activity output attribute
func GetActivityOutputSchema(ctx activity.Context, name string) (string, error) {
	if sIO, ok := ctx.(schema.HasSchemaIO); ok {
		s := sIO.GetOutputSchema(name)
		if s != nil {
			logger.Debugf("schema for attribute '%s': %T, %s\n", name, s, s.Value())
			return s.Value(), nil
		}
	}
	return "", errors.Errorf("schema not found for attribute %s", name)
}

// ResolveFlowData resolves and returns data from the flow's context, unmarshals JSON string to map[string]interface{}.
// The name to Resolve is a valid output attribute of a flogo activity, e.g., `activity[app_16].value` or `$.content`,
// which is shown in normal flogo mapper as, e.g., "$.content"
func ResolveFlowData(toResolve string, context activity.Context) (value interface{}, err error) {
	actionCtx := context.ActivityHost()
	logger.Debugf("Resolving flow data %s; context data: %+v", toResolve, actionCtx.Scope())
	factory := expression.NewFactory(resolve.GetBasicResolver())
	expr, err := factory.NewExpr(toResolve)
	if err != nil {
		logger.Errorf("failed to construct resolver expression: %+v", err)
	}
	actValue, err := expr.Eval(actionCtx.Scope())
	if err != nil {
		logger.Errorf("failed to resolve expression %+v", err)
	}
	logger.Debugf("Resolved value for %s: %T - %+v", toResolve, actValue, actValue)
	return actValue, err
}

// ExtractCompositeKeys collects all valid composite-keys matching composite-key definitions using fields of a value object
func ExtractCompositeKeys(stub shim.ChaincodeStubInterface, compositeKeyDefs map[string][]string, keyValue string, value interface{}) []string {
	// check arguments
	if len(compositeKeyDefs) == 0 || value == nil {
		logger.Debugf("No composite keys because state value is not a non-zero map\n")
		return nil
	}

	// construct composite keys
	var compositeKeys []string
	for keyName, attributes := range compositeKeyDefs {
		if ck, _ := MakeCompositeKey(stub, keyName, attributes, keyValue, value); len(ck) > 0 {
			compositeKeys = append(compositeKeys, ck)
		}
	}
	return compositeKeys
}

// MakeCompositeKey constructs composite key if all specified attributes exist in the value object
//   attributes contain JsonPath for fields in value objects
// returns key, false if key does not include all fields defined in the attributes
func MakeCompositeKey(stub shim.ChaincodeStubInterface, keyName string, attributes []string, keyValue string, value interface{}) (string, bool) {
	if len(keyName) == 0 || len(attributes) == 0 {
		logger.Debugf("invalid composite key definition: name %s attributes %+v\n", keyName, attributes)
		return "", false
	}
	attrValues := ExtractDataAttributes(attributes, value)
	if len(attrValues) == 0 {
		logger.Infof("no field specified for composite key %s in data %+v\n", keyName, value)
		return "", false
	}

	if len(attrValues) >= len(attributes)-1 {
		//  append the key value if at most 1 attribute missing and key value is not included
		if len(keyValue) > 0 && keyValue != attrValues[len(attrValues)-1] {
			attrValues = append(attrValues, keyValue)
		}
	}

	compositeKey, err := stub.CreateCompositeKey(keyName, attrValues)
	if err != nil {
		logger.Warnf("failed to create composite key %s with values %+v\n", keyName, attrValues)
		return "", false
	}
	return compositeKey, len(attrValues) >= len(attributes)
}

// ExtractDataAttributes collects values of specified attributes from a data object for constructing a partial composite key
func ExtractDataAttributes(attrs []string, data interface{}) []string {
	if len(attrs) == 0 || data == nil {
		return nil
	}

	var values []string
	for _, f := range attrs {
		v, err := jsonpath.JsonPathLookup(data, f)
		if err != nil {
			logger.Debugf("composite key attribute %s is not found in input data\n", f)
			break
		}
		values = append(values, fmt.Sprintf("%v", v))
	}
	return values
}

// PutData writes key and value to the ledger if 'store' is not specified, or a private data collection specified by 'store'
func PutData(stub shim.ChaincodeStubInterface, store string, key string, value []byte) error {
	if len(key) == 0 {
		return errors.New("key is not specified for Put")
	}

	// set nil value for composite key
	var v []byte
	if IsCompositeKey(key) {
		v = []byte{0x00}
	} else if len(value) > 0 {
		v = value
	} else {
		logger.Warn("value of ledger record is nil")
	}

	// write data to ledger or private data collection
	if len(store) == 0 {
		return stub.PutState(key, v)
	}
	return stub.PutPrivateData(store, key, v)
}

// GetData retrieves data by state key from the ledger if 'store' is not specified, or a private data collection specified by 'store'
func GetData(stub shim.ChaincodeStubInterface, store string, key string, privateHash bool) (string, []byte, error) {
	if len(key) == 0 {
		return key, nil, errors.New("key is not specified for Get")
	}

	k := key
	if IsCompositeKey(key) {
		// this is a composite key, so extract state key from it
		if ck, err := SplitCompositeKey(stub, key); err == nil {
			k = ck.Key
		}
	}

	// retrieve data from ledger or private data collection
	if len(store) == 0 {
		data, err := stub.GetState(k)
		return k, data, err
	}
	if privateHash {
		data, err := stub.GetPrivateDataHash(store, k)
		return k, data, err
	}
	data, err := stub.GetPrivateData(store, k)
	return k, data, err
}

// DeleteData deletes a state or a composite key from the ledger if 'store' is not specified, or a private data collection specified by 'store'
func DeleteData(stub shim.ChaincodeStubInterface, store string, key string) error {
	if len(key) == 0 {
		return errors.New("key is not specified for Delete")
	}

	// delete data from ledger or private data collection
	if len(store) == 0 {
		return stub.DelState(key)
	}
	return stub.DelPrivateData(store, key)
}

// GetCompositeKeys retrieves iterator for composite keys from from the ledger if 'store' is not specified, or a private data collection specified by 'store'
func GetCompositeKeys(stub shim.ChaincodeStubInterface, store string, name string, values []string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {
	if len(name) == 0 || len(values) == 0 {
		return nil, nil, errors.New("name and attributes are not specified for composite key")
	}

	// retrieve iterator of composite keys from ledger
	if len(store) == 0 {
		if pageSize > 0 {
			return stub.GetStateByPartialCompositeKeyWithPagination(name, values, pageSize, bookmark)
		}
		iter, err := stub.GetStateByPartialCompositeKey(name, values)
		return iter, nil, err
	}
	// ignore pagesize for private data collection
	iter, err := stub.GetPrivateDataByPartialCompositeKey(store, name, values)
	return iter, nil, err
}

// IsCompositeKey returns true if a key belongs to composite key namespace
func IsCompositeKey(key string) bool {
	return key[0] == compositeKeyNamespace[0]
}

// SplitCompositeKey returns components of a composite key
func SplitCompositeKey(stub shim.ChaincodeStubInterface, key string) (*CompositeKey, error) {
	if !IsCompositeKey(key) {
		return nil, errors.New("key value is not a composite key")
	}

	name, attrs, err := stub.SplitCompositeKey(key)
	if err != nil {
		return nil, err
	}
	if len(attrs) == 0 {
		return nil, errors.New("key value contains no attributes")
	}

	// the last composite attribute must be the state key
	stateKey := attrs[len(attrs)-1]
	return &CompositeKey{
		Name:   name,
		Fields: attrs,
		Key:    stateKey,
	}, nil
}

// AddCompositeKey adds a new key to the bag
func (b *CompositeKeyBag) AddCompositeKey(k *CompositeKey) (*CompositeKeyBag, error) {
	if b.Name != k.Name {
		return b, errors.Errorf("key name %s does not match bag name %s", k.Name, b.Name)
	}
	// save memory by removing redundant data
	k.Name = ""
	k.Fields = k.Fields[:len(k.Fields)-1]
	b.Keys = append(b.Keys, k)
	return b, nil
}

// ToMap converts CompositeKeyBag to map[string]interface{}
func (b *CompositeKeyBag) ToMap() (map[string]interface{}, error) {
	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}

// GetDataByRange retrieves iterator for range of state keys from from the ledger if 'store' is not specified, or a private data collection specified by 'store'
func GetDataByRange(stub shim.ChaincodeStubInterface, store, startKey, endKey string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	// retrieve iterator of state range from ledger
	if len(store) == 0 {
		if pageSize > 0 {
			return stub.GetStateByRangeWithPagination(startKey, endKey, pageSize, bookmark)
		}
		iter, err := stub.GetStateByRange(startKey, endKey)
		return iter, nil, err
	}
	// ignore pagesize for private data collection
	iter, err := stub.GetPrivateDataByRange(store, startKey, endKey)
	return iter, nil, err
}

// GetDataByQuery retrieves iterator for rich query from from the ledger if 'store' is not specified, or a private data collection specified by 'store'
func GetDataByQuery(stub shim.ChaincodeStubInterface, store, query string, pageSize int32, bookmark string) (shim.StateQueryIteratorInterface, *pb.QueryResponseMetadata, error) {

	// retrieve iterator of state range from ledger
	if len(store) == 0 {
		if pageSize > 0 {
			return stub.GetQueryResultWithPagination(query, pageSize, bookmark)
		}
		iter, err := stub.GetQueryResult(query)
		return iter, nil, err
	}
	// ignore pagesize for private data collection
	iter, err := stub.GetPrivateDataQueryResult(store, query)
	return iter, nil, err
}
