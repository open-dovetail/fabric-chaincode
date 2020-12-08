package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoCompositeKey(t *testing.T) {
	keys := ExtractCompositeKeys(nil, nil, "", "")
	assert.Nil(t, keys, "non-object value should return no composite keys")
}
