package transaction

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/project-flogo/core/data/coerce"
	jschema "github.com/xeipuuv/gojsonschema"
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
func toAttribute(name, value string) *Attribute {
	jsonType := jschema.TYPE_STRING
	if strings.EqualFold(value, "true") || strings.EqualFold(value, "false") {
		jsonType = jschema.TYPE_BOOLEAN
	} else if matched, err := regexp.MatchString(`\d+\.\d*`, value); err == nil && matched {
		jsonType = jschema.TYPE_NUMBER
	} else if matched, err := regexp.MatchString(`\d+`, value); err == nil && matched {
		jsonType = jschema.TYPE_INTEGER
	}
	return &Attribute{
		Name: name,
		Type: jsonType,
	}
}

func (p *Attribute) String() string {
	return fmt.Sprintf("(%s:%s)", p.Name, p.Type)
}

// FromMap sets settings from a map
func (s *Settings) FromMap(values map[string]interface{}) error {
	cid, err := coerce.ToString(values["cid"])
	if err != nil {
		return err
	}
	if len(cid) == 0 {
		return nil
	}

	attrs := strings.Split(strings.TrimSpace(cid), ",")
	for _, v := range attrs {
		a := strings.TrimSpace(v)
		if len(a) > 0 {
			s.CIDAttrs = append(s.CIDAttrs, a)
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
	params, err := coerce.ToString(values["parameters"])
	if err != nil {
		return err
	}
	if len(params) == 0 {
		return nil
	}
	args := strings.Split(strings.TrimSpace(params), ",")
	for _, v := range args {
		pt := strings.Split(strings.TrimSpace(v), ":")
		if len(pt) == 0 || len(strings.TrimSpace(pt[0])) == 0 {
			continue
		}
		value := ""
		if len(pt) > 1 {
			value = strings.TrimSpace(pt[1])
		}
		if attr := toAttribute(strings.TrimSpace(pt[0]), value); attr != nil {
			h.Arguments = append(h.Arguments, attr)
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
