/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/project-flogo/core/action"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/app"
	"github.com/project-flogo/core/app/resource"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/trigger"
	"github.com/project-flogo/flow/definition"
)

// ReadAppConfig reads a Flogo app json file and returns app.Config
func ReadAppConfig(configFile string) (*app.Config, map[string]*definition.DefinitionRep, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, nil, err
	}
	config := &app.Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, nil, err
	}

	resources := make(map[string]*definition.DefinitionRep)
	for _, r := range config.Resources {
		var def = &definition.DefinitionRep{}
		err := json.Unmarshal(r.Data, def)
		if err != nil {
			return nil, nil, err
		}
		resources[r.ID] = def
	}
	return config, resources, nil
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
func WriteAppConfig(config *app.Config, outFile string) error {
	// change trigger handler to use 'action' instead of 'actions'
	// -- this is required to import for Flogo Web UI
	fixedConfig, err := fixTriggerConfig(config)
	if err != nil {
		return err
	}

	// generate activity metadata for complex settings used by Flogo Enterprise
	fixActivitySchema(fixedConfig)

	result, err := marshalNoEscape(fixedConfig)
	if err != nil {
		return err
	}
	result = bytes.ReplaceAll(result, []byte("int64"), []byte("integer"))

	return ioutil.WriteFile(outFile, result, 0644)
}

// make app.Config serialized with `action` in trigger handler (instead of `actions`),
// so the exported model can be imported to OSS Web UI
func fixTriggerConfig(config *app.Config) (interface{}, error) {
	jsonbytes, err := marshalNoEscape(config)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(jsonbytes, &result); err != nil {
		return nil, err
	}
	handlers := lookupJSONPath(result, "$.triggers.handlers.")
	for _, h := range handlers {
		handler := h.(map[string]interface{})
		actions := handler["actions"].([]interface{})
		if len(actions) > 0 {
			act := actions[0]
			delete(handler, "actions")
			handler["action"] = act
		}
	}
	return result, nil
}

// fill in schema for complex activity settings, which is required by Flogo Enterprise
func fixActivitySchema(appjson interface{}) {

	activities := lookupJSONPath(appjson, "$.resources.data.tasks.activity")
	for _, act := range activities {
		a, ok := act.(map[string]interface{})
		if !ok || len(a) == 0 {
			continue
		}
		matched, err := regexp.Match(a["ref"].(string), []byte("#put|#get|#delete"))
		if !matched || err != nil {
			continue
		}
		if _, ok := a["schemas"]; !ok {
			// no schema definition, so not a model for Flogo Enterprise
			continue
		}
		s, ok := a["settings"].(map[string]interface{})
		if !ok || len(s) == 0 {
			continue
		}
		for k, v := range s {
			if reflect.ValueOf(v).Kind() == reflect.Map {
				// create fe_metadata here
				md := v
				if d, ok := v.(map[string]interface{})["mapping"]; ok {
					md = d
				}
				femd, _ := json.Marshal(md)
				// fmt.Printf("fe_metadata for %s: %s => %s\n", a["ref"], k, string(femd))
				addActivityFEMetadata(a, k, string(femd))
			}
		}
	}
}

func addActivityFEMetadata(activity map[string]interface{}, key, feMetadata string) {
	schm, ok := activity["schemas"].(map[string]interface{})
	if !ok {
		schm = make(map[string]interface{})
		activity["schemas"] = schm
	}
	settings, ok := schm["settings"].(map[string]interface{})
	if !ok {
		settings = make(map[string]interface{})
		schm["settings"] = settings
	}
	settings[key] = map[string]interface{}{
		"type":        "json",
		"fe_metadata": feMetadata,
	}
}

// ToAppConfig converts the first contract in a contract spec to a Flogo App Config
func (s *Spec) ToAppConfig(fe bool) (*app.Config, error) {
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
		AppModel:    "1.1.1",
		Imports:     s.Imports,
	}
	if fe {
		// convert and cache app schemas for Flogo Enterprise
		if err := s.ConvertAppSchemas(); err != nil {
			fmt.Printf("failed to convert app schema: %v\n", err)
		}
	}

	// create one trigger with one handler per transaction
	trig, err := con.ToTrigger(fe)
	if err != nil {
		return nil, err
	}
	ac.Triggers = []*trigger.Config{trig}

	// create a flow resource per transaction
	resources := make(map[string]*definition.DefinitionRep)
	for _, tx := range con.Transactions {
		var schm *trigger.SchemaConfig
		if fe {
			schm = handlerSchema(trig, tx.Name)
		}
		id, res, err := tx.ToResource(schm, con.CID)
		if err != nil {
			return nil, err
		}
		resources[id] = res
	}

	if fe {
		// collect app schema for Flogo Enterprise
		if schm, err := GetAppSchemas(); err == nil {
			ac.Schemas = schm
		} else {
			fmt.Printf("failed to collect app schemas: %v\n", err)
		}
	}

	// serializes resources
	SetAppResources(ac, resources)
	return ac, nil
}

// SetAppResources serialize flow resources and add them to app config
func SetAppResources(config *app.Config, resources map[string]*definition.DefinitionRep) {
	config.Resources = make([]*resource.Config, 0)
	for k, v := range resources {
		if data, err := marshalNoEscape(v); err == nil {
			res := &resource.Config{
				ID:   k,
				Data: data,
			}
			config.Resources = append(config.Resources, res)
		}
	}
}

// search trigger config for a schema config of a specified transaction name
func handlerSchema(trigConfig *trigger.Config, txName string) *trigger.SchemaConfig {
	for _, h := range trigConfig.Handlers {
		if h.Name == txName {
			return h.Schemas
		}
	}
	return nil
}

// ToTrigger converts contract transactions to trigger handlers
func (c *Contract) ToTrigger(fe bool) (*trigger.Config, error) {
	trig := &trigger.Config{
		Id:       "fabric_transaction",
		Ref:      "#transaction",
		Settings: make(map[string]interface{}),
	}
	if len(c.CID) > 0 {
		trig.Settings["cid"] = c.CID
	}
	for _, tx := range c.Transactions {
		handler, err := tx.ToHandler(fe)
		if err != nil {
			return nil, err
		}
		trig.Handlers = append(trig.Handlers, handler)
	}
	return trig, nil
}

// ToHandler converts a contract transaction to trigger handler config
func (tx *Transaction) ToHandler(fe bool) (*trigger.HandlerConfig, error) {
	handler := &trigger.HandlerConfig{
		Name: tx.Name,
	}

	param, err := tx.ParameterDef()
	if err != nil {
		return nil, err
	}
	handler.Settings = map[string]interface{}{
		"name":       tx.Name,
		"parameters": param,
	}

	// generate flow action
	res := "res://flow:" + ToSnakeCase(tx.Name)
	// map all parameters as a single object
	input := map[string]interface{}{
		"parameters": "=$.parameters",
		"transient":  "=$.transient",
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
	if fe {
		// set handler schema for Flogo enterprise
		if schm, err := tx.ToHandlerSchema(); err == nil {
			handler.Schemas = schm
		} else {
			fmt.Printf("failed to convert handler schema for transaction %s: %v\n", tx.Name, err)
		}
	}
	return handler, nil
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

// noops caches activity followed by noop activity for branching
type actNoop struct {
	action *Action
	noop   string
	expr   string
	links  []*actNoop
}

var noops map[string]*actNoop
var firstNode *actNoop

// actSeq caches current sequence number of an activity type
var actSeq map[string]int

const (
	startNoop = "_"
	noopRef   = "#noop"
	returnRef = "#actreturn"
)

func addAction(resource *definition.DefinitionRep, act *actNoop, includeSchema bool) {
	tsk := act.action.toResourceTask(includeSchema)
	resource.Tasks = append(resource.Tasks, tsk)
	for _, a := range act.links {
		link := &definition.LinkRep{
			FromID: act.action.Name,
			ToID:   a.action.Name,
		}
		if len(a.expr) > 0 {
			link.Type = "expression"
			link.Value = a.expr
		}
		resource.Links = append(resource.Links, link)
		addAction(resource, a, includeSchema)
	}
}

// convert action to a task resource
func (a *Action) toResourceTask(includeSchema bool) *definition.TaskRep {
	actCfg := &activity.Config{
		Ref: a.Activity,
	}
	if a.Activity == returnRef {
		// #actreturn maps input onto settings
		actCfg.Settings = map[string]interface{}{"mappings": a.Input.Mapping}
	} else {
		actCfg.Settings = a.toActivitySetting()
		actCfg.Input = a.toActivityInput()
		if includeSchema {
			actCfg.Schemas = a.toActivitySchemas()
		}
	}
	return &definition.TaskRep{
		ID:             a.Name,
		Name:           a.Name,
		ActivityCfgRep: actCfg,
	}
}

func (a *Action) toActivitySetting() map[string]interface{} {
	if len(a.Config) == 0 {
		return nil
	}

	settings := make(map[string]interface{})
	// for OSS Flogo add "mapping" nesting for objects and arrays
	for k, v := range a.Config {
		switch v.(type) {
		case map[string]interface{}:
			settings[k] = map[string]interface{}{"mapping": v}
		case []interface{}:
			settings[k] = map[string]interface{}{"mapping": v}
		default:
			settings[k] = v
		}
	}

	return settings
}

func (a *Action) toActivityInput() map[string]interface{} {
	if a.Input == nil || len(a.Input.Mapping) == 0 {
		return nil
	}

	input := make(map[string]interface{})
	// add mapping nesting for object mapper
	for k, v := range a.Input.Mapping {
		switch v.(type) {
		case map[string]interface{}:
			input[k] = map[string]interface{}{"mapping": v}
		default:
			input[k] = v
		}
	}

	return input
}

func (tx *Transaction) initTxnResource() (err error) {
	noops = make(map[string]*actNoop)
	actSeq = make(map[string]int)
	// collect used activity seq numbers
	for _, r := range tx.Rules {
		for _, a := range r.Actions {
			a.initActivitySeq()
		}
	}
	// register activity and branching noops
	for _, r := range tx.Rules {
		var prev *actNoop
		var expr string
		if r.Condition != nil {
			p := r.Condition.Prerequisite
			if len(p) == 0 {
				p = startNoop
			}
			if prev, err = addNoop(p); err != nil {
				return err
			}
			expr = r.Condition.Expr
			if len(expr) == 0 {
				// include description for user to edit concrete condition expr
				expr = fmt.Sprintf("\"changeme\" == \"%s\"", r.Condition.Description)
			}
		}
		for _, a := range r.Actions {
			prev = a.linkAction(prev, expr)
			expr = "" // use condition expre only for the first action
		}
	}
	return nil
}

// add noop for branching if action is not #log or #noop
func addNoop(name string) (*actNoop, error) {
	n, ok := noops[name]
	if name == startNoop {
		// create noop for branch from flow startup
		if !ok {
			n = &actNoop{
				action: &Action{
					Activity: noopRef,
					Name:     nextActivityID(noopRef),
				},
			}
			noops[name] = n
			firstNode = n
		}
		return n, nil
	}

	if !ok || n.action == nil {
		// prerequisite action not defined
		return nil, errors.Errorf("prerequisite action %s is not defined in contract spec", name)
	}

	if n.action.Activity == "#log" || n.action.Activity == "#noop" {
		// do not add noop for branching from #log or #noop
		return n, nil
	}

	if len(n.noop) > 0 {
		// return the noop previously set already
		return noops[n.noop], nil
	}

	// create noop action
	n.noop = nextActivityID(noopRef)
	an := &actNoop{
		action: &Action{
			Activity: noopRef,
			Name:     n.noop,
		},
	}

	n.links = append(n.links, an)
	noops[n.noop] = an
	return an, nil
}

// register named activity and collect max sequence number used by named activities
func (a *Action) initActivitySeq() {
	if len(a.Activity) == 0 || len(a.Name) == 0 {
		return
	}

	// register named activity
	noops[a.Name] = &actNoop{action: a}

	if !strings.HasPrefix(a.Name, a.Activity[1:]+"_") {
		// not a pattern for activity sequence
		return
	}
	// update actSeq to keep max used seq
	seq := a.Name[len(a.Activity):]
	if s, err := strconv.Atoi(seq); err == nil {
		if c, ok := actSeq[a.Activity]; !ok || c < s {
			actSeq[a.Activity] = s
		}
	}
}

// create unique name for an action and register its link to previous action
func (a *Action) linkAction(prev *actNoop, expr string) *actNoop {
	var an *actNoop
	if len(a.Name) == 0 {
		// register the activity with a new unique name
		a.Name = nextActivityID(a.Activity)
		an = &actNoop{action: a}
		noops[a.Name] = an
	} else {
		// should have been registered, so add branching condition
		an = noops[a.Name]
	}
	an.expr = expr
	if prev != nil {
		// add it to links from prev action
		prev.links = append(prev.links, an)
	} else {
		firstNode = an
	}
	return an
}

// returns next id for an activity type ref, e.g., #get
func nextActivityID(ref string) string {
	seq, ok := actSeq[ref]
	if !ok {
		seq = 0
	}
	actSeq[ref] = seq + 1
	return fmt.Sprintf("%s_%d", ref[1:], seq+1)
}

// ToResource converts a contract transaction to flow resource definition
func (tx *Transaction) ToResource(schm *trigger.SchemaConfig, cid string) (string, *definition.DefinitionRep, error) {
	id := "flow:" + ToSnakeCase(tx.Name)

	input := make(map[string]data.TypedValue)
	if len(tx.Parameters) > 0 {
		input["parameters"] = data.NewAttribute("parameters", data.TypeObject, nil)
	}
	if len(tx.Transient) > 0 {
		input["transient"] = data.NewAttribute("transient", data.TypeObject, nil)
	}
	rAttr := data.NewAttribute("returns", data.TypeAny, nil)
	includeSchema := false
	if schm != nil {
		// add schema info for Flogo Enterprise
		includeSchema = true
		if len(tx.Parameters) > 0 {
			if sc := ExtractFlowSchema(schm.Output["parameters"]); sc != nil {
				input["parameters"] = data.NewAttributeWithSchema("parameters", data.TypeObject, nil, sc)
			}
		}
		if len(tx.Transient) > 0 {
			if sc := ExtractFlowSchema(schm.Output["transient"]); sc != nil {
				input["transient"] = data.NewAttributeWithSchema("transient", data.TypeObject, nil, sc)
			}
		}
		input["cid"] = data.NewAttributeWithSchema("cid", data.TypeObject, nil, cidSchema(cid))
		input["txID"] = data.NewAttribute("txID", data.TypeString, "")
		input["txTime"] = data.NewAttribute("txTime", data.TypeString, "")
		if sc := ExtractFlowSchema(schm.Reply["returns"]); sc != nil {
			rAttr = data.NewAttributeWithSchema("returns", data.TypeAny, nil, sc)
		}
	}

	md := &metadata.IOMetadata{
		Input: input,
		Output: map[string]data.TypedValue{
			"status":  data.NewAttribute("status", data.TypeInt64, 0),
			"message": data.NewAttribute("message", data.TypeString, ""),
			"returns": rAttr,
		},
	}

	// initialize for tasks and links
	if err := tx.initTxnResource(); err != nil {
		return "", nil, err
	}

	res := &definition.DefinitionRep{
		Name:     tx.Name,
		Metadata: md,
	}

	// add tasks and links using depth first search from first action node
	addAction(res, firstNode, includeSchema)

	return id, res, nil
}
