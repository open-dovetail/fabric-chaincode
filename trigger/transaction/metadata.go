package transaction

import (
	"fmt"

	"github.com/project-flogo/core/data/coerce"
)

// Attribute describes a name and data type
type Attribute struct {
	Name string `md:"name"`
	Type string `md:"type"`
}

// Settings for the trigger
type Settings struct {
	CIDAttrs []string `md:"cidattrs"`
}

// HandlerSettings for the trigger
// arguments are of parameter names and associated JSON data type
// type is any valid JSON type, i.e., string, number, integer, boolean, array, object.
type HandlerSettings struct {
	Name      string       `md:"name,required"`
	Arguments []*Attribute `md:"arguments"`
}

// Output of the trigger
type Output struct {
	Parameters map[string]interface{} `md:"parameters"`
	Transient  map[string]interface{} `md:"transient"`
	TxID       string                 `md:"txID"`
	TxTime     string                 `md:"txTime"`
	CID        map[string]string      `md:"cid"`
}

// Reply from the trigger
type Reply struct {
	Status  int         `md:"status"`
	Message string      `md:"message"`
	Returns interface{} `md:"returns"`
}

// construct Attribute from map of name and type
func toAttribute(values interface{}) *Attribute {
	var attr Attribute
	if m, ok := values.(map[string]interface{}); ok {
		if v, s := m["name"].(string); s {
			attr.Name = v
		}
		if v, s := m["type"].(string); s {
			attr.Type = v
		}
	}
	if len(attr.Name) == 0 {
		return nil
	}
	if len(attr.Type) == 0 {
		attr.Type = "string"
	}
	return &attr
}

func (p *Attribute) String() string {
	return fmt.Sprintf("(%s:%s)", p.Name, p.Type)
}

// FromMap sets settings from a map
func (h *Settings) FromMap(values map[string]interface{}) error {
	attrs, err := coerce.ToArray(values["cidattrs"])
	if err != nil {
		return err
	}
	if attrs != nil && len(attrs) > 0 {
		for _, v := range attrs {
			if s, ok := v.(string); ok && len(s) > 0 {
				h.CIDAttrs = append(h.CIDAttrs, s)
			}
		}
	}
	return nil
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
		for _, v := range args {
			if attr := toAttribute(v); attr != nil {
				h.Arguments = append(h.Arguments, attr)
			}
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
	if o.CID, err = coerce.ToParams(values["cid"]); err != nil {
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
		"cid":        o.CID,
	}
}

// FromMap sets trigger reply values from a map
func (r *Reply) FromMap(values map[string]interface{}) error {
	var err error
	if r.Status, err = coerce.ToInt(values["status"]); err != nil {
		logger.Errorf("Failed to map returned status: %+v", err)
		r.Status = 500
	}
	if r.Message, err = coerce.ToString(values["message"]); err != nil {
		logger.Infof("Failed to map returned status: %+v", err)
		r.Message = ""
	}
	if r.Returns, err = coerce.ToAny(values["returns"]); err != nil {
		logger.Infof("Failed to map returned value: %+v", err)
		r.Returns = nil
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
