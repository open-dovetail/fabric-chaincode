package transaction

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/project-flogo/core/support"
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
