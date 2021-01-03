package contract

import (
	"encoding/json"
	"io/ioutil"
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
	Config      map[string]interface{} `json:"config,omitempty"`
	Input       *Input                 `json:"input,omitempty"`
}

// Input defines schema and mapping of an activity input
type Input struct {
	Schema  map[string]interface{} `json:"schema,omitempty"`
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
