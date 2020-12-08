package put

import (
	"github.com/project-flogo/core/data/coerce"
)

// Settings of the activity
type Settings struct {
	CompositeKeys map[string][]string `md:"compositeKeys"`
}

// Input of the activity
type Input struct {
	StateKey          string `md:"key,required"`
	StateData         string `md:"data,required"`
	PrivateCollection string `md:"privateCollection"`
}

// Output of the activity
type Output struct {
	Code     int                    `md:"code"`
	Message  string                 `md:"message"`
	StateKey string                 `md:"key"`
	Result   map[string]interface{} `md:"result"`
}

// FromMap sets settings from a map
// construct composite key definition of format {"index": ["field1, "field2"]}
func (h *Settings) FromMap(values map[string]interface{}) error {
	keys, err := coerce.ToObject(values["compositeKeys"])
	if err != nil {
		return err
	}
	if keys == nil || len(keys) == 0 {
		return nil
	}
	for k, v := range keys {
		var fields []string
		values, err := coerce.ToArray(v)
		if err != nil || values == nil || len(values) == 0 {
			logger.Warnf("ignored composite key setting for index %s. error: %+v", k, err)
			continue
		}
		for _, n := range values {
			if f, ok := n.(string); ok && len(f) > 0 {
				fields = append(fields, f)
			}
		}
		if len(fields) > 0 {
			h.CompositeKeys[k] = fields
			logger.Debugf("configured composite key %s with fields %+v", k, fields)
		}
	}
	return nil
}

// ToMap converts activity input to a map
func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"key":               i.StateKey,
		"data":              i.StateData,
		"privateCollection": i.PrivateCollection,
	}
}

// FromMap sets activity input values from a map
func (i *Input) FromMap(values map[string]interface{}) error {

	var err error
	if i.StateKey, err = coerce.ToString(values["key"]); err != nil {
		return err
	}
	if i.StateData, err = coerce.ToString(values["data"]); err != nil {
		return err
	}
	if i.PrivateCollection, err = coerce.ToString(values["privateCollection"]); err != nil {
		return err
	}

	return nil
}

// ToMap converts activity output to a map
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"code":    o.Code,
		"message": o.Message,
		"key":     o.StateKey,
		"result":  o.Result,
	}
}

// FromMap sets activity output values from a map
func (o *Output) FromMap(values map[string]interface{}) error {

	var err error
	if o.Code, err = coerce.ToInt(values["code"]); err != nil {
		return err
	}
	if o.Message, err = coerce.ToString(values["message"]); err != nil {
		o.Message = ""
	}
	if o.StateKey, err = coerce.ToString(values["key"]); err != nil {
		return err
	}
	if o.Result, err = coerce.ToObject(values["result"]); err != nil {
		return err
	}

	return nil
}
