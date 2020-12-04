package transaction

import (
	"github.com/project-flogo/core/data/coerce"
)

// Settings for the trigger
type Settings struct {
}

// HandlerSettings for the trigger
// arguments are of fomat "name:type", e.g., ["owner:string", "price:number"]
// type is any valid JSON type, i.e., string (default), number, integer, boolean, array, object.
type HandlerSettings struct {
	Name      string   `md:"name,required"`
	Arguments []string `md:"arguments"`
}

// Output of the trigger
type Output struct {
	Parameters map[string]interface{} `md:"parameters"`
	Transient  map[string]interface{} `md:"transient"`
	TxID       string                 `md:"txID"`
	TxTime     string                 `md:"txTime"`
}

// Reply from the trigger
type Reply struct {
	Status  int    `md:"status"`
	Message string `md:"message"`
	Returns string `md:"returns"`
}

// FromMap sets handling settings from a map
func (h *HandlerSettings) FromMap(values map[string]interface{}) error {
	var err error
	if h.Name, err = coerce.ToString(values["name"]); err != nil {
		return err
	}
	args, err := coerce.ToArray(values["arguments"])
	if err != nil {
		return err
	}
	if args != nil && len(args) > 0 {
		h.Arguments = make([]string, len(args))
		for i, v := range args {
			h.Arguments[i] = v.(string)
		}
	}
	return nil
}

// FromMap sets trigger output values from a map
func (o *Output) FromMap(values map[string]interface{}) error {
	var err error
	if o.Parameters, err = coerce.ToObject(values["parameters"]); err != nil {
		return err
	}
	if o.Transient, err = coerce.ToObject(values["transient"]); err != nil {
		return err
	}
	if o.TxID, err = coerce.ToString(values["txID"]); err != nil {
		return err
	}
	if o.TxTime, err = coerce.ToString(values["txTime"]); err != nil {
		return err
	}

	return nil
}

// ToMap converts trigger output to a map
func (o *Output) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"parameters": o.Parameters,
		"transient":  o.Transient,
		"txID":       o.TxID,
		"txTime":     o.TxTime,
	}
}

// FromMap sets trigger reply values from a map
func (r *Reply) FromMap(values map[string]interface{}) error {
	var err error
	if r.Status, err = coerce.ToInt(values["status"]); err != nil {
		return err
	}
	if r.Message, err = coerce.ToString(values["message"]); err != nil {
		r.Message = ""
	}
	if r.Returns, err = coerce.ToString(values["returns"]); err != nil {
		return err
	}
	return nil
}

// ToMap converts trigger reply to a map
func (r *Reply) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"status":  r.Status,
		"message": r.Message,
		"returns": r.Returns,
	}
}
