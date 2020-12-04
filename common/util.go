package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/expression"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/data/schema"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/flow/instance"
	jschema "github.com/xeipuuv/gojsonschema"
)

const (
	// KeyField attribute used in query response of key-value pairs
	KeyField = "key"
	// ValueField attribute used in query response of key-value pairs
	ValueField = "value"
	// FabricStub is the name of flow property for passing chaincode stub to activities
	FabricStub = "_chaincode_stub"
	// FabricTxID is the name of flow property for passing auto-generated transaction ID to activities
	FabricTxID = "txID"
	// FabricTxTime is the name of flow property for passing auto-generated transaction Time to activities
	FabricTxTime = "txTime"
)

var logger = log.ChildLogger(log.RootLogger(), "fabric-chaincode-common")

// GetChaincodeStub returns Fabric chaincode stub from the activity context
func GetChaincodeStub(ctx activity.Context) (shim.ChaincodeStubInterface, error) {
	// get chaincode stub
	if taskInst, ok := ctx.(*instance.TaskInst); ok {
		flowInst := taskInst.ActivityHost().(*instance.Instance)
		scope := flowInst.GetMasterScope()
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
	logger.Errorf("ctx is not a TaskInst: %+v", ctx)
	return nil, errors.Errorf("ctx is not a TaskInst: %+v", ctx)
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

// ParameterIndex stores transaction parameters and its location in raw JSON schema string
// start and end location is used to sort the parameter list to match the parameter order in schema
type ParameterIndex struct {
	Name     string
	JSONType string
	start    int
	end      int
}

// addIndex adds a new parameter position to the index, ignore or merge index if index region overlaps.
func addIndex(parameters []ParameterIndex, param ParameterIndex) []ParameterIndex {
	for i, v := range parameters {
		if param.start > v.start && param.start < v.end {
			// ignore if new param's start postion falls in region covered by a known parameter
			return parameters
		} else if v.start > param.start && v.start < param.end {
			// replace old parameter region if its start position falls in the region covered by the new parameter
			updated := append(parameters[:i], param)
			if len(parameters) > i+1 {
				// check the remaining knonw parameters
				for _, p := range parameters[i+1:] {
					if !(p.start > param.start && p.start < param.end) {
						updated = append(updated, p)
					}
				}
			}
			return updated
		}
	}
	// append new parameter
	return append(parameters, param)
}

// OrderedParameters returns parameters of a JSON schema object sorted by their position in schema definition
// This is necessary because Golang JSON parser does not maintain the sequence of object parameters.
func OrderedParameters(schemaData []byte) ([]ParameterIndex, error) {
	if schemaData == nil || len(schemaData) == 0 {
		logger.Debug("schema data is empty")
		return nil, nil
	}
	// extract root object properties from JSON schema
	var rawProperties struct {
		Data json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schemaData, &rawProperties); err != nil {
		logger.Errorf("failed to extract properties from metadata: %+v", err)
		return nil, err
	}

	// extract parameter names from raw object properties
	var params map[string]json.RawMessage
	if err := json.Unmarshal(rawProperties.Data, &params); err != nil {
		logger.Errorf("failed to extract parameters from object schema: %+v", err)
		return nil, err
	}

	// collect parameter locations in the raw object schema
	var paramIndex []ParameterIndex
	for p, v := range params {
		// encode parameter name with quotes
		key, _ := json.Marshal(p)
		// key may exist in raw schema multiple times,
		// so check each occurence to determine its correct location in the raw schema
		items := bytes.Split(rawProperties.Data, key)
		pos := 0
		for _, seg := range items {
			if pos == 0 {
				// first segment should not be the key definition
				pos += len(seg)
				continue
			}
			vpos := bytes.Index(seg, v)
			if vpos >= 0 {
				// the segment contains the key definition, so collect its position in raw schema
				endPos := pos + len(key) + vpos + len(v)
				// extract JSON type of the parameter
				var paramDef struct {
					RawType string `json:"type"`
				}
				if err := json.Unmarshal(v, &paramDef); err != nil {
					logger.Errorf("failed to extract JSON type of parameter %s: %+v", p, err)
				}
				paramType := jschema.TYPE_OBJECT
				if paramDef.RawType != "" {
					paramType = paramDef.RawType
				}
				logger.Debugf("add index parameter '%s' type '%s'\n", p, paramType)
				paramIndex = addIndex(paramIndex, ParameterIndex{Name: p, JSONType: paramType, start: pos, end: endPos})
			}
			pos += len(key) + len(seg)
		}
	}

	// sort parameter index by start location in raw schema
	if len(paramIndex) > 1 {
		sort.Slice(paramIndex, func(i, j int) bool {
			return paramIndex[i].start < paramIndex[j].start
		})
	}
	return paramIndex, nil
}

// ConstructQueryResponse iterate through query result to create array of key-value pairs, i.e.
// JSON string of format [{"key":"mykey", "value":{}}, ...]
func ConstructQueryResponse(resultsIterator shim.StateQueryIteratorInterface, collection string, isCompositeKey bool, stub shim.ChaincodeStubInterface) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("[")

	isEmpty := true
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		key := queryResponse.Key
		value := queryResponse.Value
		if isCompositeKey {
			// get state from composite key
			_, compositeParts, err := stub.SplitCompositeKey(key)
			if err != nil {
				return nil, err
			}
			// the last composite attribute must be the state key
			key = compositeParts[len(compositeParts)-1]
			if collection == "" {
				if value, err = stub.GetState(key); err != nil {
					// ignore key if value does not exist
					logger.Errorf("failed to retrieve state for key %s: %+v", key, err)
					continue
				}
			} else {
				if value, err = stub.GetPrivateData(collection, key); err != nil {
					// ignore key if value does not exist
					logger.Errorf("failed to retrieve state for key %s: %+v", key, err)
					continue
				}
			}
			if value == nil {
				// ignore nil state
				logger.Warnf("nil state for key %s", key)
				continue
			}
		}
		// Add a comma before array members, suppress it for the first array member
		if !isEmpty {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"" + KeyField + "\":")
		buffer.WriteString("\"")
		buffer.WriteString(key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"" + ValueField + "\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(value))
		buffer.WriteString("}")
		isEmpty = false
	}
	if isEmpty {
		return nil, nil
	}
	buffer.WriteString("]")
	return buffer.Bytes(), nil
}

// ExtractCompositeKeys collects all valid composite-keys matching composite-key definitions using fields of a value object
func ExtractCompositeKeys(stub shim.ChaincodeStubInterface, compositeKeyDefs string, keyValue string, value interface{}) []string {
	// verify that value is a map
	obj, ok := value.(map[string]interface{})
	if !ok {
		logger.Debugf("No composite keys because state value is not a map\n")
		return nil
	}

	// check composite keys
	if len(compositeKeyDefs) > 0 {
		keyMap, err := ParseCompositeKeyDefs(compositeKeyDefs)
		if err != nil {
			logger.Errorf("failed to parse composite key definition: %+v", err)
			return nil
		}
		var compositeKeys []string
		for keyName, attributes := range keyMap {
			if ck := makeCompositeKey(stub, keyName, attributes, keyValue, obj); ck != "" {
				compositeKeys = append(compositeKeys, ck)
			}
		}
		return compositeKeys
	}
	logger.Debugf("No composite key is defined")
	return nil
}

// ParseCompositeKeyDefs parses composite-key definition of format 'key1=field1,field2;key2=field3,field4'
func ParseCompositeKeyDefs(def string) (map[string][]string, error) {
	m := make(map[string][]string)
	keys := strings.Split(def, ";")
	for _, key := range keys {
		tokens := strings.Split(key, "=")
		if len(tokens) < 2 || len(tokens[0]) <= 0 || len(tokens[1]) <= 0 {
			return nil, errors.Errorf("invalid composite-key definition %s", def)
		}
		fields := strings.Split(tokens[1], ",")
		m[tokens[0]] = fields
	}
	return m, nil
}

// constructs composite key if all specified attributes exist in the value object
// returns "" if failed to extract any attribute from the value object
func makeCompositeKey(stub shim.ChaincodeStubInterface, keyName string, attributes []string, keyValue string, value map[string]interface{}) string {
	if keyName == "" || attributes == nil || len(attributes) == 0 {
		logger.Debugf("invalid composite key definition: name %s attributes %+v\n", keyName, attributes)
		return ""
	}
	var attrValues []string
	for _, k := range attributes {
		if v, ok := value[k]; ok {
			attrValues = append(attrValues, fmt.Sprintf("%v", v))
		} else {
			logger.Debugf("composite key attribute %s is not found in state value\n", k)
			return ""
		}
	}
	if attrValues == nil || len(attrValues) == 0 {
		logger.Debug("No composite key attribute found in state value\n")
		return ""
	}

	// the last element of composite key must be the keyValue itself
	if keyValue != attrValues[len(attrValues)-1] {
		attrValues = append(attrValues, keyValue)
	}
	compositeKey, err := stub.CreateCompositeKey(keyName, attrValues)
	if err != nil {
		logger.Errorf("failed to create composite key %s with values %+v\n", keyName, attrValues)
		return ""
	}
	return compositeKey
}
