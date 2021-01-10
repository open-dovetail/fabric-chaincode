package contract

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"reflect"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data/schema"
	"github.com/project-flogo/core/trigger"
	jschema "github.com/xeipuuv/gojsonschema"
)

var appSchemas map[string]interface{}

// ConvertAppSchemas converts schemas in contract spec to Flogo schema def
func (s *Spec) ConvertAppSchemas() error {
	if s.Components == nil || len(s.Components.Schemas) == 0 {
		return nil
	}

	// collect all app schemas
	appSchemas = make(map[string]interface{})
	for k, v := range s.Components.Schemas {
		obj := map[string]interface{}{
			"type":       jschema.TYPE_OBJECT,
			"properties": v.Properties,
		}
		appSchemas["#/components/schemas/"+k] = obj
	}

	// expand refs until all references are removed
	for i := 0; i < 10; i++ {
		count := 0
		for _, v := range appSchemas {
			replaced, err := expandRef(v)
			if err != nil {
				return err
			}
			if replaced {
				count++
			}
		}
		fmt.Printf("replaced %d schema refs\n", count)
		if count == 0 {
			break
		}
	}

	return nil
}

func getAppSchemas() (map[string]*schema.Def, error) {
	// construct Flogo schema defs
	result := make(map[string]*schema.Def)
	for k, v := range appSchemas {
		jsonbytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		key := k[strings.LastIndex(k, "/")+1:]
		result[key] = &schema.Def{
			Type:  "json",
			Value: string(jsonbytes),
		}
	}
	return result, nil
}

// expand schema ref, returns true if some refs are replaced
func expandRef(def interface{}) (bool, error) {
	s, ok := def.(map[string]interface{})
	if !ok {
		// not a JSON object, no ref to expand
		return false, nil
	}

	replaced := false
	for k, v := range s {
		c, ok := v.(map[string]interface{})
		if !ok {
			// value is not JSON object, e.g., when k="$id" or "type"
			continue
		}
		r, ok := c["$ref"]
		if !ok {
			// object is not a ref, so call expand recursively
			done, err := expandRef(v)
			if err != nil {
				return replaced, err
			}
			if done {
				replaced = done
			}
			continue
		}
		// replace ref
		a, ok := appSchemas[r.(string)]
		if !ok {
			return replaced, errors.Errorf("schema ref %s is not found", r)
		}
		s[k] = a
		replaced = true
	}
	return replaced, nil
}

// ToHandlerSchema extracts schema config from  a contract transaction
func (tx *Transaction) ToHandlerSchema() (*trigger.SchemaConfig, error) {
	result := &trigger.SchemaConfig{}

	if len(tx.Returns) > 0 {
		if v, ok := tx.Returns["$ref"].(string); ok {
			// copy schema ref
			ref := "schema://" + v[strings.LastIndex(v, "/")+1:]
			result.Reply = map[string]interface{}{
				"returns": ref,
			}
		} else {
			// convert returns schema
			rs := map[string]interface{}{
				"returns": tx.Returns,
			}
			if _, err := expandRef(rs); err == nil {
				if rbytes, err := json.Marshal(rs["returns"]); err == nil {
					result.Reply = map[string]interface{}{
						"returns": &schema.Def{
							Type:  "json",
							Value: string(rbytes),
						},
					}
				}
			}
		}
	}

	// convert parameters schema
	result.Output = make(map[string]interface{})
	if len(tx.Parameters) > 0 {
		ps := parametersToSchema(tx.Parameters)
		if pbytes, err := json.Marshal(ps); err == nil {
			result.Output["parameters"] = &schema.Def{
				Type:  "json",
				Value: string(pbytes),
			}
		}
	}

	// convert transient schema
	if len(tx.Transient) > 0 {
		if _, err := expandRef(tx.Transient); err == nil {
			ts := map[string]interface{}{
				"type":       jschema.TYPE_OBJECT,
				"properties": tx.Transient,
			}
			if tbytes, err := json.Marshal(ts); err == nil {
				result.Output["transient"] = &schema.Def{
					Type:  "json",
					Value: string(tbytes),
				}
			}
		}
	}
	return result, nil
}

// convert transaction parameters to schema def
func parametersToSchema(params []*Parameter) map[string]interface{} {
	props := make(map[string]interface{})
	for _, p := range params {
		props[p.Name] = p.Schema
	}
	return map[string]interface{}{
		"type":       jschema.TYPE_OBJECT,
		"properties": props,
	}
}

// flowSchema implements schema.Schema, used for flow metadata
type flowSchema struct {
	SchemaType  string `json:"type"`
	SchemaValue string `json:"value"`
}

func (f *flowSchema) Type() string {
	return f.SchemaType
}
func (f *flowSchema) Value() string {
	return f.SchemaValue
}
func (f *flowSchema) Validate(data interface{}) error {
	return nil
}
func (f *flowSchema) MarshalJSON() ([]byte, error) {
	if strings.HasPrefix(f.SchemaValue, "schema://") {
		return []byte("\"" + f.SchemaValue + "\""), nil
	}
	return json.Marshal(&struct {
		SchemaType  string `json:"type"`
		SchemaValue string `json:"value"`
	}{
		SchemaType:  f.SchemaType,
		SchemaValue: f.SchemaValue,
	})
}

// create serializable schema for a flow for a given schema def
// to work around FE import issue, the schema def is changed as follows:
//   for object, export only properties of the object
//   for array, create app schema, and export a ref
func extractFlowSchema(schemadef interface{}) schema.Schema {
	var def *schema.Def
	switch d := schemadef.(type) {
	case string:
		if strings.HasPrefix(d, "schema://") {
			return &flowSchema{
				SchemaType:  "json",
				SchemaValue: d,
			}
		}
		fmt.Printf("invalid schema def %s\n", d)
		return nil
	case *schema.Def:
		def = d
	default:
		fmt.Printf("schema is not a *Def: %T - %v\n", schemadef, schemadef)
		return nil
	}

	var data map[string]interface{}
	err := json.Unmarshal([]byte(def.Value), &data)
	if err != nil {
		fmt.Printf("failed to unmarshal schema value %s: %v\n", def.Value, err)
		return nil
	}

	value := def.Value
	if data["type"].(string) == jschema.TYPE_ARRAY {
		// add schema to app schema and return the ref
		key := fnvHash(def.Value)
		appSchemas["/"+key] = data
		value = "schema://" + key
	} else if data["type"].(string) == jschema.TYPE_OBJECT {
		// return object properties
		jsonbytes, err := json.Marshal(data["properties"])
		if err != nil {
			fmt.Printf("failed to marshal object properties %s: %v\n", data["properties"], err)
			return nil
		}
		value = string(jsonbytes)
	}

	return &flowSchema{
		SchemaType:  def.Type,
		SchemaValue: value,
	}
}

// JSON schema for CID attributes include standard id, mspid, cn, and extra attributes in comma-delimited cid config
func cidSchema(cid string) schema.Schema {
	attrs := []string{"id", "mspid", "cn"}
	if len(cid) > 0 {
		extra := strings.Split(cid, ",")
		attrs = append(attrs, extra...)
	}
	var buff strings.Builder
	buff.WriteString("{")
	delimiter := ""
	for _, v := range attrs {
		fmt.Fprintf(&buff, `%s"%s":{"type":"%s"}`, delimiter, v, jschema.TYPE_STRING)
		delimiter = ","
	}
	buff.WriteString("}")
	return &flowSchema{
		SchemaType:  "json",
		SchemaValue: buff.String(),
	}
}

func fnvHash(text string) string {
	h := fnv.New32()
	h.Write([]byte(text))
	c := h.Sum32()
	return fmt.Sprintf("%x", c)
}

// convert a sample JSON doc or mapper into JSON schema
func json2schema(doc string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(doc), &data); err != nil {
		return "", err
	}
	return genSchema(data), nil
}

// depth first generation of json schema from map
func genSchema(obj interface{}) string {
	var buff strings.Builder
	switch reflect.ValueOf(obj).Kind() {
	case reflect.Slice:
		fmt.Fprintf(&buff, `{"type":"%s","items":`, jschema.TYPE_ARRAY)
		d, _ := obj.([]interface{})
		buff.WriteString(genSchema(d[0]))
		buff.WriteString("}")
	case reflect.Map:
		d, _ := obj.(map[string]interface{})
		if len(d) == 1 {
			for k, v := range d {
				if strings.HasPrefix(k, "@foreach(") {
					// this is an array of objects
					fmt.Fprintf(&buff, `{"type":"%s","items":`, jschema.TYPE_ARRAY)
					buff.WriteString(genSchema(v))
					buff.WriteString("}")
					return buff.String()
				}
			}
		}
		fmt.Fprintf(&buff, `{"type":"%s","properties":{`, jschema.TYPE_OBJECT)
		delimiter := ""
		for k, v := range d {
			fmt.Fprintf(&buff, `%s"%s":`, delimiter, k)
			buff.WriteString(genSchema(v))
			delimiter = ","
		}
		buff.WriteString("}}")
	case reflect.Float64:
		fmt.Fprintf(&buff, `{"type":"%s"}`, jschema.TYPE_NUMBER)
	case reflect.Bool:
		fmt.Fprintf(&buff, `{"type":"%s"}`, jschema.TYPE_BOOLEAN)
	default:
		fmt.Fprintf(&buff, `{"type":"%s"}`, jschema.TYPE_STRING)
	}
	return buff.String()
}

// returns activity schema for Flogo Enterprise
// it does not include special schema for setting objects
func (a *Action) toActivitySchemas() *activity.SchemaConfig {
	result := &activity.SchemaConfig{}
	if input := a.activityInputSchema(); input != nil {
		result.Input = input
	}
	if output := a.ledgerOutputSchema(); output != nil {
		result.Output = output
	}
	return result
}

// return schemas of activity input objects
func (a *Action) activityInputSchema() map[string]interface{} {
	input := make(map[string]interface{})
	if a.Input == nil {
		return nil
	}

	// add specified schema first
	if len(a.Input.Schema) > 0 {
		if _, err := expandRef(a.Input.Schema); err != nil {
			// ignore schema ref errors
			fmt.Printf("failed to resolve ref in activity schema: %v\n", err)
		}
		for k, v := range a.Input.Schema {
			scbytes, err := json.Marshal(v)
			if err != nil {
				fmt.Printf("failed to serialize activity schema %s: %v\n", k, err)
				continue
			}
			input[k] = &flowSchema{
				SchemaType:  "json",
				SchemaValue: string(scbytes),
			}
		}
	}

	// add schema defined by sample JSON only if no schema is already specified
	if len(a.Input.Sample) > 0 {
		for k, v := range a.Input.Sample {
			if _, ok := input[k]; ok {
				continue
			}
			sbytes, err := json.Marshal(v)
			if err != nil {
				fmt.Printf("failed to serialize activity input sample %s: %v\n", k, err)
				continue
			}
			sc, err := json2schema(string(sbytes))
			if err != nil {
				fmt.Printf("failed to convert schema from input sample %s: %v\n", k, err)
				continue
			}
			input[k] = &flowSchema{
				SchemaType:  "json",
				SchemaValue: sc,
			}
		}
	}

	// generate schema from input mapping only if no schema nor sample is already specified
	// Note: in this result, all primitive type will be resented as string
	if len(a.Input.Mapping) > 0 {
		for k, v := range a.Input.Mapping {
			if _, ok := input[k]; ok {
				continue
			}
			if reflect.ValueOf(v).Kind() == reflect.String {
				// do not generate schema for simple
				continue
			}
			sbytes, err := json.Marshal(v)
			if err != nil {
				fmt.Printf("failed to serialize activity input mapping %s: %v\n", k, err)
				continue
			}
			sc, err := json2schema(string(sbytes))
			if err != nil {
				fmt.Printf("failed to convert schema from input mapping %s: %v\n", k, err)
				continue
			}
			input[k] = &flowSchema{
				SchemaType:  "json",
				SchemaValue: sc,
			}
		}
	}
	return input
}

// return result schema for ledger activities
func (a *Action) ledgerOutputSchema() map[string]interface{} {
	useLedger, _ := regexp.Match(a.Activity, []byte("#put|#get|#delete"))
	if !useLedger {
		// not a ledger activity
		return nil
	}

	if len(a.Config) > 0 && a.Config["keysOnly"] != nil {
		if v, ok := a.Config["keysOnly"].(bool); ok && v {
			// return schema for composite keys
			return map[string]interface{}{
				"result": compositeKeySchema(),
			}
		}
	}
	if len(a.Config) > 0 && a.Config["privateHash"] != nil {
		if v, ok := a.Config["privateHash"].(bool); ok && v {
			schm := `{"type":"array","items":{"type":"object","properties":{"key":{"type":"string"},"value":{"type":"string"}}}}`
			return map[string]interface{}{
				"result": &flowSchema{
					SchemaType:  "json",
					SchemaValue: schm,
				},
			}
		}
	}

	if len(a.Ledger) > 0 {
		// return schema for ledger array
		ledger, err := json.Marshal(a.Ledger)
		if err != nil {
			fmt.Printf("failed to marshal ledger schema spec %v: %v\n", a.Ledger, err)
			return nil
		}
		scfmt := `{
    		"type": "array",
    		"items": {
        		"type": "object",
        		"properties": {
            		"key": {
                		"type": "string"
            		},
            		"value": %s
        		}
			}
		}`
		if len(a.Config) > 0 && a.Config["history"] != nil {
			if v, ok := a.Config["history"].(bool); ok && v {
				scfmt = `{
    				"type": "array",
    				"items": {
        				"type": "object",
        				"properties": {
            				"key": {
                				"type": "string"
            				},
            				"value": {
                				"type": "array",
                				"items": {
                    				"type": "object",
                    				"properties": {
                        				"txID": {
                            				"type": "string"
                        				},
                        				"txTime": {
                            				"type": "string"
                        				},
                        				"isDeleted": {
                            				"type": "boolean"
                        				},
                        				"value": %s
                    				}
                				}
            				}
        				}
    				}
				}`
			}
		}
		s := fmt.Sprintf(scfmt, string(ledger))
		var schm map[string]interface{}
		if err := json.Unmarshal([]byte(s), &schm); err != nil {
			fmt.Printf("failed to construct ledger result schema: %v\n", err)
			return nil
		}
		// expand component refs in ledger schema spec
		_, err = expandRef(schm)
		if err != nil {
			fmt.Printf("failed to resolve ref in ledger result schema: %v\n", err)
			return nil
		}
		scbytes, err := json.Marshal(schm)
		if err != nil {
			fmt.Printf("failed to serialize ledger result schema: %v\n", err)
			return nil
		}
		return map[string]interface{}{
			"result": &flowSchema{
				SchemaType:  "json",
				SchemaValue: string(scbytes),
			},
		}
	}
	return nil
}

// returns schema for composite keys
func compositeKeySchema() schema.Schema {
	cks := `[{"name":"","attributes":[""],"keys":[{"name":"","fields":[""],"key":""}]}]`
	s, _ := json2schema(cks)
	return &flowSchema{
		SchemaType:  "json",
		SchemaValue: s,
	}
}

// return all objects matching the JSON path in specified JSON document
func lookupJSONPath(doc interface{}, path string) []interface{} {
	result := []interface{}{doc}
	tokens := strings.Split(path, ".")
	for _, p := range tokens[1:] {
		result = getJSONElement(result, p)
	}
	return result
}

func getJSONElement(doc []interface{}, key string) []interface{} {
	var result []interface{}
	for _, v := range doc {
		switch reflect.ValueOf(v).Kind() {
		case reflect.Slice:
			data := getJSONElement(v.([]interface{}), key)
			if len(data) > 0 {
				result = append(result, data...)
			}
		case reflect.Map:
			if len(key) == 0 {
				result = append(result, v)
			} else if elem, ok := v.(map[string]interface{})[key]; ok {
				result = append(result, elem)
			}
		default:
			if len(key) == 0 {
				result = append(result, v)
			}
		}
	}
	return result
}
