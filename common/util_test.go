package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCompositKey(t *testing.T) {

	key := "key1=field1,field2;key2=field3,field4"
	keyMap, err := ParseCompositeKeyDefs(key)
	assert.NoError(t, err, "parse composite key should not raise error")
	assert.Equal(t, 2, len(keyMap["key1"]), "key1 should contain 2 attributes")
	assert.Equal(t, "field4", keyMap["key2"][1], "second attribute of key2 should be 'field4'")
}
