package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/project-flogo/core/trigger"

	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/project-flogo/core/support/log"
	jschema "github.com/xeipuuv/gojsonschema"
)

var triggerMd = trigger.NewMetadata(&Settings{}, &HandlerSettings{}, &Output{}, &Reply{})
var singleton *Trigger

var logger = log.ChildLogger(log.RootLogger(), "trigger-fabric-transaction")

func init() {
	_ = trigger.Register(&Trigger{}, &Factory{})
}

// Factory for trigger
type Factory struct {
}

// New implements trigger.Factory.New
func (t *Factory) New(config *trigger.Config) (trigger.Trigger, error) {
	if singleton == nil {
		setting := &Settings{}
		if err := setting.FromMap(config.Settings); err != nil {
			logger.Warnf("Failed to extract trigger setting config: %+v", err)
		}

		singleton = &Trigger{
			id:        config.Id,
			cidAttrs:  setting.CIDAttrs,
			handlers:  map[string]trigger.Handler{},
			arguments: map[string][]*Attribute{},
		}
		return singleton, nil
	}
	logger.Warn("Only one transaction trigger can be instantiated")
	return nil, nil
}

// Metadata implements trigger.Factory.Metadata
func (*Factory) Metadata() *trigger.Metadata {
	return triggerMd
}

// Trigger is the Fabric transaction Trigger implementation
type Trigger struct {
	id        string
	cidAttrs  []string
	handlers  map[string]trigger.Handler
	arguments map[string][]*Attribute
}

// Initialize implements trigger.Init.Initialize
func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	for _, handler := range ctx.GetHandlers() {
		setting := &HandlerSettings{}
		if err := setting.FromMap(handler.Settings()); err != nil {
			return err
		}

		if len(setting.Name) == 0 {
			logger.Warn("Ignore transaction with empty name")
			continue
		}
		t.handlers[setting.Name] = handler
		t.arguments[setting.Name] = setting.Arguments
		logger.Infof("transaction %s accepts arguments %v", setting.Name, setting.Arguments)
	}
	return nil
}

// Start implements trigger.Trigger.Start
func (t *Trigger) Start() error {
	return nil
}

// Stop implements trigger.Trigger.Start
func (t *Trigger) Stop() error {
	// stop the trigger
	return nil
}

// Invoke starts the singleton trigger and invokes the action registered in the handler,
// and returns status code and result as JSON string
func Invoke(stub shim.ChaincodeStubInterface, fn string, args []string) (int, string, error) {
	logger.Debugf("fabric.Trigger invokes fn %s with args %+v", fn, args)

	handler, ok := singleton.handlers[fn]
	if !ok {
		msg := fmt.Sprintf("Handler not defined for transaction %s", fn)
		return 400, msg, errors.New(msg)
	}

	// extract client ID
	triggerData := &Output{}
	if client, err := singleton.extractCID(stub); err == nil {
		triggerData.CID = client
	}

	// construct transaction parameters
	paramData, err := prepareParameters(singleton.arguments[fn], args)
	if err != nil {
		return 400, err.Error(), err
	}
	if logger.DebugEnabled() {
		// debug flow data
		paramBytes, _ := json.Marshal(paramData)
		logger.Debugf("trigger parameters: %s", string(paramBytes))
	}
	triggerData.Parameters = paramData

	// construct transient attributes
	transData, err := prepareTransient(stub)
	if err != nil {
		return 400, err.Error(), err
	}
	if logger.DebugEnabled() {
		// debug flow data
		transBytes, _ := json.Marshal(transData)
		logger.Debugf("trigger transient attributes: %s", string(transBytes))
	}
	triggerData.Transient = transData

	triggerData.TxID = stub.GetTxID()
	if ts, err := stub.GetTxTimestamp(); err == nil {
		triggerData.TxTime = time.Unix(ts.Seconds, int64(ts.Nanos)).UTC().Format(time.RFC3339Nano)
	}

	// execute flogo flow, and pass stub, txID and txTime to the flow context
	logger.Debugf("flogo flow started transaction %s with timestamp %s", triggerData.TxID, triggerData.TxTime)
	ctxValues := map[string]interface{}{
		common.FabricStub:   stub,
		common.FabricTxID:   triggerData.TxID,
		common.FabricTxTime: triggerData.TxTime,
		common.FabricCID:    triggerData.CID,
	}
	ctx := trigger.NewContextWithValues(context.Background(), ctxValues)
	results, err := handler.Handle(ctx, triggerData.ToMap())
	if err != nil {
		logger.Errorf("flogo flow returned error: %+v", err)
		return 500, err.Error(), err
	}

	// processing reply
	reply := &Reply{}
	if err := reply.FromMap(results); err != nil {
		return 500, err.Error(), err
	}

	if reply.Status != 200 {
		logger.Infof("flogo flow returned status %d with message %s", reply.Status, reply.Message)
		return reply.Status, reply.Message, nil
	}
	if reply.Returns == "" {
		logger.Info("Flogo flow did not return any data")
		if reply.Message == "" {
			reply.Message = "No data returned"
		}
		return 300, reply.Message, nil
	}

	logger.Debugf("Flogo flow returned data: %s", reply.Returns)
	return 200, reply.Returns, nil
}

func (t *Trigger) extractCID(stub shim.ChaincodeStubInterface) (map[string]string, error) {
	// get client identity
	c, err := cid.New(stub)
	if err != nil {
		logger.Errorf("failed to extract client identity from stub: %+v\n", err)
		return nil, errors.Wrap(err, "failed to extract client identity from stub")
	}

	// retrieve data from client identity
	var client map[string]string
	if id, err := c.GetID(); err == nil {
		client["id"] = id
	}
	if mspid, err := c.GetMSPID(); err == nil {
		client["mspid"] = mspid
	}

	if cert, err := c.GetX509Certificate(); err == nil {
		client["cn"] = cert.Subject.CommonName
	}

	// retrieve custom attributes from client identity
	if t.cidAttrs != nil {
		for _, k := range t.cidAttrs {
			if v, ok, err := c.GetAttributeValue(k); err == nil && ok {
				client[k] = v
			}
		}
	}
	return client, nil
}

// construct trigger output transient attributes
func prepareTransient(stub shim.ChaincodeStubInterface) (map[string]interface{}, error) {
	transient := make(map[string]interface{})
	transMap, err := stub.GetTransient()
	if err != nil {
		// cannot find transient attributes
		logger.Warnf("no transient map: %+v", err)
		return transient, nil
	}
	if len(transMap) == 0 {
		return transient, nil
	}
	for k, v := range transMap {
		var obj interface{}
		if err := json.Unmarshal(v, &obj); err != nil {
			return nil, errors.Wrapf(err, "failed to extract unmarshal data %s", k)
		}
		logger.Debugf("received transient data, name: %s, value: %+v", k, obj)
		transient[k] = obj
	}
	return transient, nil
}

// construct trigger output parameters for specified parameter index, and values of the parameters
func prepareParameters(attrs []*Attribute, values []string) (map[string]interface{}, error) {
	logger.Debugf("prepare parameters %+v values %+v", attrs, values)
	if attrs == nil || len(values) != len(attrs) {
		return nil, errors.New("transaction paramters do not match required argument list")
	}

	// convert string array to name-values as defined by transaction arguments
	result := make(map[string]interface{})
	if values != nil && len(values) > 0 {
		// populate input args
		for i, v := range values {
			name := strings.TrimSpace(attrs[i].Name)
			jsonType := strings.TrimSpace(attrs[i].Type)
			if obj := unmarshalString(v, jsonType, name); obj != nil {
				result[name] = obj
			}
		}
	}
	return result, nil
}

// unmarshalString returns unmarshaled object if input is a valid JSON object or array,
// or returns the input string if it is not a valid JSON format
func unmarshalString(data, jsonType, name string) interface{} {
	s := strings.TrimSpace(data)
	switch jsonType {
	case jschema.TYPE_ARRAY:
		var result []interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			logger.Warnf("failed to parse parameter %s as JSON array: data '%s' error %+v", name, data, err)
		}
		return result
	case jschema.TYPE_BOOLEAN:
		b, err := strconv.ParseBool(s)
		if err != nil {
			logger.Warnf("failed to convert parameter %s to boolean: data '%s' error %+v", name, data, err)
			return false
		}
		return b
	case jschema.TYPE_INTEGER:
		i, err := strconv.Atoi(s)
		if err != nil {
			logger.Warnf("failed to convert parameter %s to integer: data '%s' error %+v", name, data, err)
			return 0
		}
		return i
	case jschema.TYPE_NUMBER:
		if !strings.Contains(s, ".") {
			i, err := strconv.Atoi(s)
			if err != nil {
				logger.Warnf("failed to convert parameter %s to integer: data '%s' error %+v", name, data, err)
				return 0
			}
			return i
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			logger.Warnf("failed to convert parameter %s to float: data '%s' error %+v", name, data, err)
			return 0.0
		}
		return n
	case jschema.TYPE_OBJECT:
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			logger.Warnf("failed to convert parameter %s to object: data '%s' error: %+v", name, data, err)
		}
		return result
	default:
		return s
	}
}
