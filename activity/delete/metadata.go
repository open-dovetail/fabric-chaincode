/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package delete

import (
	"strings"

	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/project-flogo/core/data/coerce"
)

// Settings of the activity
type Settings struct {
	CompositeKeys map[string][]string `md:"compositeKeys"`
	KeysOnly      bool                `md:"keysOnly"`
}

// Input of the activity
type Input struct {
	Data              interface{} `md:"data"`
	PrivateCollection string      `md:"privateCollection"`
}

// Output of the activity
type Output struct {
	Code    int           `md:"code"`
	Message string        `md:"message"`
	Result  []interface{} `md:"result"`
}

// FromMap sets settings from a map
// construct composite key definition of format {"index": ["field1, "field2"]}
func (h *Settings) FromMap(values map[string]interface{}) error {
	var err error
	if h.KeysOnly, err = coerce.ToBool(values["keysOnly"]); err != nil {
		return err
	}

	keys, err := common.MapToObject(values["compositeKeys"])
	if err != nil || len(keys) == 0 {
		logger.Debugf("No composite key is defined. error: %+v", err)
		return err
	}
	h.CompositeKeys = make(map[string][]string)
	for k, v := range keys {
		var fields []string
		values, err := coerce.ToArray(v)
		if err != nil || len(values) == 0 {
			logger.Warnf("ignored composite key config for key %s. error: %+v", k, err)
			continue
		}
		for _, n := range values {
			if f, ok := n.(string); ok && len(f) > 0 {
				path := f
				if !strings.HasPrefix(f, "$.") {
					// make it valid JsonPath expression
					path = "$." + f
				}
				fields = append(fields, path)
			}
		}
		if len(fields) > 0 {
			h.CompositeKeys[k] = fields
			logger.Infof("configured composite key %s with fields %+v", k, fields)
		}
	}
	return nil
}

// ToMap converts activity input to a map
func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"data":              i.Data,
		"privateCollection": i.PrivateCollection,
	}
}

// FromMap sets activity input values from a map
func (i *Input) FromMap(values map[string]interface{}) error {

	var err error
	if i.Data, err = coerce.ToAny(values["data"]); err != nil {
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
	if o.Result, err = coerce.ToArray(values["result"]); err != nil {
		return err
	}

	return nil
}
