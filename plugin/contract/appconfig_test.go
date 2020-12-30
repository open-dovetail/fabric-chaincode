package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testConfig = "../../samples/marble/marble.json"

func TestAppConfig(t *testing.T) {
	config, err := ReadAppConfig(testConfig)
	assert.NoError(t, err, "read sample contract should not throw error")
	assert.Equal(t, 1, len(config.Config.Triggers), "sample file should contain 1 trigger")
	assert.Equal(t, 10, len(config.Config.Triggers[0].Handlers), "sample trigger should have 10 handlers")
	assert.Equal(t, 10, len(config.Resources), "sample file should contain 10 resources")

	err = WriteAppConfig(config, "out.json")
	assert.NoError(t, err, "write app config should not throw error")
}
