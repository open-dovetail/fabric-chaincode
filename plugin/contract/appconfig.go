package contract

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/app"
	"github.com/project-flogo/core/app/resource"
	"github.com/project-flogo/core/trigger"
	"github.com/project-flogo/flow/definition"
)

// AppConfig contains config of a Flogo app and its unmarshalled resource configs
type AppConfig struct {
	Config    *app.Config
	Resources map[string]*definition.DefinitionRep
}

// ReadAppConfig reads a Flogo app json file and returns app.Config
func ReadAppConfig(configFile string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	config := &app.Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	result := &AppConfig{Config: config}
	result.Resources = make(map[string]*definition.DefinitionRep)
	for _, r := range config.Resources {
		var def = &definition.DefinitionRep{}
		err := json.Unmarshal(r.Data, def)
		if err != nil {
			return nil, err
		}
		result.Resources[r.ID] = def
	}
	return result, nil
}

// marshal JSON w/o escaping condition chars &, <, >
func marshalNoEscape(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "   ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteAppConfig serializes an app config and its resources
func (c *AppConfig) WriteAppConfig(outFile string) error {
	// serializes resources
	c.Config.Resources = make([]*resource.Config, 0)
	for k, v := range c.Resources {
		data, err := marshalNoEscape(v)
		if err != nil {
			return err
		}
		res := &resource.Config{
			ID:   k,
			Data: data,
		}
		c.Config.Resources = append(c.Config.Resources, res)
	}

	result, err := marshalNoEscape(c.Config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outFile, result, 0644)
}

// ToAppConfig converts the first contract in a contract spec to a Flogo AppConfig
func (s *Spec) ToAppConfig() (*AppConfig, error) {
	if len(s.Contracts) == 0 {
		return nil, errors.New("No contract is defined in the spec")
	}

	var name string
	var con *Contract
	for k, v := range s.Contracts {
		name = k
		con = v
		break
	}
	ac := &app.Config{
		Name:        name,
		Type:        "flogo:app",
		Version:     s.Info.Version,
		Description: con.Name,
		AppModel:    "1.1.0",
		Imports:     s.Imports,
	}
	c := &AppConfig{Config: ac}
	trig, err := con.ToTrigger()
	if err != nil {
		return nil, err
	}
	ac.Triggers = []*trigger.Config{trig}
	return c, nil
}

// ToTrigger converts contract transactions to trigger handlers
func (c *Contract) ToTrigger() (*trigger.Config, error) {
	trig := &trigger.Config{
		Id:       "fabric_transaction",
		Ref:      "#transaction",
		Settings: make(map[string]interface{}),
	}
	if len(c.CID) > 0 {
		trig.Settings["cidattrs"] = c.CID
	}
	for _, tx := range c.Transactions {
		handler, err := tx.ToHandler()
		if err != nil {
			return nil, err
		}
		trig.Handlers = append(trig.Handlers, handler)
	}
	return trig, nil
}

// ToHandler converts a contract transaction to trigger handler config
func (tx *Transaction) ToHandler() (*trigger.HandlerConfig, error) {
	handler := &trigger.HandlerConfig{}

	// convert tranaction parameters
	var args []interface{}
	for _, p := range tx.Parameters {
		attr, err := parameterToAttribute(p)
		if err != nil {
			return nil, err
		}
		args = append(args, attr)
	}
	handler.Settings = map[string]interface{}{
		"name":      tx.Name,
		"arguments": args,
	}

	// generate flow action
	res := "res://flow:" + ToSnakeCase(tx.Name)
	// map all parameters as a single object
	// TODO: support transient params
	input := map[string]interface{}{
		"parameters": "=$.parameters",
	}
	output := map[string]interface{}{
		"message": "=$.message",
		"returns": "=$.returns",
		"status":  "=$.status",
	}
	action := &trigger.ActionConfig{
		Config: &action.Config{
			Ref:      "#flow",
			Settings: map[string]interface{}{"flowURI": res}},
		Input:  input,
		Output: output,
	}
	handler.Actions = []*trigger.ActionConfig{action}
	return handler, nil
}

// contract schema accepts transaction parameters of any schema types, but
// Flogo model simplify it to support primitive types only, which practically covers all use-cases.
// so convert these 2 expressions by extract primitive schema types
func parameterToAttribute(param *Parameter) (map[string]interface{}, error) {
	if len(param.Name) == 0 {
		return nil, errors.New("missing name of transaction parameter")
	}
	atype, ok := param.Schema["type"].(string)
	if !ok || len(atype) == 0 {
		atype = "string"
	}
	return map[string]interface{}{
		"name": param.Name,
		"type": atype,
	}, nil
}

var matchFirstCap = regexp.MustCompile("([A-Z])([A-Z][a-z])")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// ToSnakeCase converts camel case string to snake case
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake = strings.ReplaceAll(snake, "-", "_")
	return strings.ToLower(snake)
}
