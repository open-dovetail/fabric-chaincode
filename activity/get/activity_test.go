package get

import (
	"testing"

	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/mapper"
	"github.com/project-flogo/core/data/resolve"
	"github.com/project-flogo/core/support/test"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {

	ref := activity.GetRef(&Activity{})
	act := activity.Get(ref)

	assert.NotNil(t, act)
}

func TestCreate(t *testing.T) {

	mf := mapper.NewFactory(resolve.GetBasicResolver())
	iCtx := test.NewActivityInitContext(Settings{}, mf)
	act, err := New(iCtx)
	assert.Nil(t, err)
	assert.NotNil(t, act, "activity should not be nil")
}

func TestEval(t *testing.T) {
	act := &Activity{}
	tc := test.NewActivityContext(act.Metadata())
	input := &Input{StateKey: "test"}
	err := tc.SetInputObject(input)
	assert.Nil(t, err)

	done, err := act.Eval(tc)
	assert.False(t, done)
	assert.NotNil(t, err)

	output := &Output{}
	err = tc.GetOutputObject(output)
	assert.Nil(t, err)
	assert.Equal(t, 500, output.Code)
}
