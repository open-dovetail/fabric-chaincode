package transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/support"
	"github.com/project-flogo/core/support/test"
	"github.com/project-flogo/core/trigger"
	"github.com/stretchr/testify/assert"
)

func TestTrigger_Register(t *testing.T) {

	ref := support.GetRef(&Trigger{})
	f := trigger.GetFactory(ref)
	assert.NotNil(t, f)
}

func TestHandlerSettings(t *testing.T) {
	config := `{
		"name": "myTransaction",
		"arguments": [{
			"name": "color"
		},
		{
			"name": "size",
			"type": "integer"
		}]
	}`
	var configMap map[string]interface{}
	err := json.Unmarshal([]byte(config), &configMap)
	assert.Nil(t, err)

	setting := &HandlerSettings{}
	err = setting.FromMap(configMap)
	assert.Nil(t, err)

	assert.Equal(t, "myTransaction", setting.Name)
	assert.Equal(t, "color", setting.Arguments[0].Name)
	assert.Equal(t, "string", setting.Arguments[0].Type)
	assert.Equal(t, "size", setting.Arguments[1].Name)
	assert.Equal(t, "integer", setting.Arguments[1].Type)
	assert.Equal(t, "(color:string)", fmt.Sprint(setting.Arguments[0]))
}

type noOpAction struct {
}

func (a *noOpAction) IOMetadata() *metadata.IOMetadata {
	return nil
}

// Metadata get the Action's metadata
func (a *noOpAction) Metadata() *action.Metadata {
	return nil
}

// Run implementation of action.SyncAction.Run to test expected trigger inputs
func (a *noOpAction) Run(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	// replace stub description for JSON serialization
	if stub, ok := inputs[common.FabricStub]; ok && stub != nil {
		inputs[common.FabricStub] = fmt.Sprintf("%v", stub)
	}
	result := make(map[string]interface{})
	data, err := json.Marshal(inputs)
	if err != nil {
		result["status"] = 500
		result["message"] = err.Error()
		return result, err
	}
	result["status"] = 200
	result["message"] = ""
	result["returns"] = string(data)
	return result, nil
}

func TestTriggerEval(t *testing.T) {
	config := `{
      "id": "fabric_transaction",
      "ref": "#transaction",
      "settings": {
        "cidattrs": null
      },
      "handlers": [
        {
          "settings": {
            "name": "initMarble",
            "arguments": [
              {
                "name": "name"
              },
              {
                "name": "color"
              },
              {
                "name": "size",
                "type": "integer"
              },
              {
                "name": "owner"
              }
            ]
          },
          "action": {
			"id": "test",
            "ref": "#flow",
            "settings": {
              "flowURI": "res://flow:init_marble"
            },
            "input": {
              "parameters": "=$.parameters"
            },
            "output": {
              "message": "=$.message",
              "returns": "=$.returns",
              "status": "=$.status"
            }
          }
        }
      ]
	}`

	var trigConfig trigger.Config
	err := json.Unmarshal([]byte(config), &trigConfig)
	assert.Nil(t, err, "unmarshal of trigger config should not throw error")

	fac := new(Factory)
	act := new(noOpAction)
	acts := map[string]action.Action{"test": act}
	trig, err := test.InitTrigger(fac, &trigConfig, acts)
	assert.Nil(t, err, "initialize trigger should not throw error")

	trans, ok := trig.(*Trigger)
	assert.True(t, ok, "initialized trigger should of type *Trigger")

	_, ok = trans.handlers["initMarble"]
	assert.True(t, ok, "trigger should contain a handler for 'initMarble'")

	args, ok := trans.arguments["initMarble"]
	assert.True(t, ok, "trigger should contain arguments for 'initMarble'")
	assert.Equal(t, 4, len(args), "initMarble transaction should accept 4 arguments")

	stub := shimtest.NewMockStub("mock", nil)
	status, returns, err := Invoke(stub, "initMarble", []string{"marble1", "blue", "50", "tom"})
	assert.Nil(t, err, "trigger invocation should not throw error")
	assert.Equal(t, 200, status, "trigger status should be 200")

	output := make(map[string]interface{})
	err = json.Unmarshal([]byte(returns), &output)
	assert.Nil(t, err, "trigger execution result should return a map")
	assert.True(t, len(output[common.FabricStub].(string)) > 0, "stub should not be nil")
	cid, ok := output[common.FabricCID].(map[string]interface{})
	assert.True(t, ok, "client ID should be a map")
	assert.Equal(t, "unknown", cid["cn"].(string), "client common name should be unknown")
	params, ok := output["parameters"].(map[string]interface{})
	assert.True(t, ok, "input parameters should be a map")
	assert.Equal(t, "blue", params["color"].(string), "color should be blue")
}
