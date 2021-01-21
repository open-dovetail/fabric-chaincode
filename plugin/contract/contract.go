/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package contract

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
	jschema "github.com/xeipuuv/gojsonschema"
)

// Spec specifies one or more smart contracts
type Spec struct {
	Info       *Info                `json:"info"`
	Imports    []string             `json:"imports"`
	Contracts  map[string]*Contract `json:"contracts"`
	Components *Components          `json:"components"`
}

// Info defines general metadata of a contract spec
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Contract defines a smart contract
type Contract struct {
	Name         string         `json:"name"`
	CID          string         `json:"cid"`
	Transactions []*Transaction `json:"transactions"`
	Info         *Info          `json:"info,omitempty"`
}

// Transaction defines a transaction in a contract
type Transaction struct {
	Name       string                 `json:"name"`
	Tag        []string               `json:"tag,omitempty"`
	Parameters []*Parameter           `json:"parameters"`
	Transient  map[string]interface{} `json:"transient"`
	Returns    map[string]interface{} `json:"returns"`
	Rules      []*Rule                `json:"rules"`
}

// Parameter defines a parameter of transaction
type Parameter struct {
	Name        string                 `json:"name"`
	Schema      map[string]interface{} `json:"schema"`
	Description string                 `json:"description,omitempty"`
	Required    bool                   `json:"required,omitempty"`
}

// Rule defines condition and actions for processing a transaction
type Rule struct {
	Description string     `json:"description,omitempty"`
	Condition   *Condition `json:"condition,omitempty"`
	Actions     []*Action  `json:"actions"`
}

// Condition defines condition for executing list of actions for a transaction
type Condition struct {
	Name         string `json:"name,omitempty"`
	Description  string `json:"description"`
	Prerequisite string `json:"prerequisite,omitempty"`
	Expr         string `json:"expr,omitempty"`
}

// Action defines an activity for processing a transaction
type Action struct {
	Activity    string                 `json:"activity"`
	Description string                 `json:"description,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Ledger      map[string]interface{} `json:"ledger,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Input       *Input                 `json:"input,omitempty"`
}

// Input defines schema and mapping of an activity input
type Input struct {
	Schema  map[string]interface{} `json:"schema,omitempty"`
	Sample  map[string]interface{} `json:"sample,omitempty"`
	Mapping map[string]interface{} `json:"mapping"`
}

// Components contains reusable schema definitions
type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

// Schema defines reusable JSON schema
type Schema struct {
	ID         string                 `json:"$id"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// ReadContract reads a contract file and returns the contract spec
func ReadContract(contractFile string) (*Spec, error) {
	data, err := ioutil.ReadFile(contractFile)
	if err != nil {
		return nil, err
	}
	spec := &Spec{}
	err = json.Unmarshal(data, spec)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

// ParameterDef returns comma-delimited string of transaction parameters
func (tx *Transaction) ParameterDef() (string, error) {
	var args bytes.Buffer
	delimiter := ""
	for _, p := range tx.Parameters {
		attr, err := parameterToAttribute(p)
		if err != nil {
			return "", err
		}
		args.WriteString(delimiter + attr)
		delimiter = ","
	}
	return args.String(), nil
}

// ContainsParameter returns true if a parameter matches the specified name
func (tx *Transaction) ContainsParameter(name string) bool {
	for _, p := range tx.Parameters {
		if p.Name == name {
			return true
		}
	}
	return false
}

// contract schema accepts transaction parameters of any JSON schema types, but
// Flogo model simplify it to support primitive types only, which practically covers all use-cases.
// so consider only primitive schema types here
func parameterToAttribute(param *Parameter) (string, error) {
	if len(param.Name) == 0 {
		return "", errors.New("name not specified for a transaction parameter")
	}
	jsontype, ok := param.Schema["type"].(string)
	if !ok || len(jsontype) == 0 {
		return param.Name, nil
	}
	switch jsontype {
	case jschema.TYPE_BOOLEAN:
		return param.Name + ":false", nil
	case jschema.TYPE_INTEGER:
		return param.Name + ":0", nil
	case jschema.TYPE_NUMBER:
		return param.Name + ":0.0", nil
	default:
		return param.Name, nil
	}
}
