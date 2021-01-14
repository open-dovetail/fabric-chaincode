/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package invokechaincode

import (
	"github.com/project-flogo/core/data/coerce"
)

// Settings of the activity
type Settings struct {
}

// Input of the activity
type Input struct {
	ChaincodeName   string        `md:"chaincodeName,required"`
	ChannelID       string        `md:"channelID"`
	TransactionName string        `md:"transactionName,required"`
	Parameters      []interface{} `md:"parameters"`
}

// Output of the activity
type Output struct {
	Code    int         `md:"code"`
	Message string      `md:"message"`
	Result  interface{} `md:"result"`
}

// ToMap converts activity input to a map
func (i *Input) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"chaincodeName":   i.ChaincodeName,
		"channelID":       i.ChannelID,
		"transactionName": i.TransactionName,
		"parameters":      i.Parameters,
	}
}

// FromMap sets activity input values from a map
func (i *Input) FromMap(values map[string]interface{}) error {

	var err error
	if i.ChaincodeName, err = coerce.ToString(values["chaincodeName"]); err != nil {
		return err
	}
	if i.ChannelID, err = coerce.ToString(values["channelID"]); err != nil {
		return err
	}
	if i.TransactionName, err = coerce.ToString(values["transactionName"]); err != nil {
		return err
	}
	if i.Parameters, err = coerce.ToArray(values["parameters"]); err != nil {
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
	if o.Result, err = coerce.ToAny(values["result"]); err != nil {
		return err
	}

	return nil
}
