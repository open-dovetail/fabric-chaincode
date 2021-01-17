package endorsement

import (
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/pkg/statebased"
	"github.com/project-flogo/core/data/coerce"
)

// Settings of the activity
type Settings struct {
	Operation string `md:"operation,required,allowed(ADD,DELETE,LIST,SET)"`
	Role      string `md:"role,allowed(MEMBER,ADMIN,CLIENT,PEER)"`
}

// Input of the activity
type Input struct {
	StateKeys         []string `md:"keys,required"`
	Organizations     []string `md:"organizations"`
	Policy            string   `md:"policy"`
	PrivateCollection string   `md:"privateCollection"`
}

// Output of the activity
type Output struct {
	Code    int           `md:"code"`
	Message string        `md:"message"`
	Result  []interface{} `md:"result"`
}

// FromMap sets settings from a map
func (h *Settings) FromMap(values map[string]interface{}) error {
	var err error
	if h.Operation, err = coerce.ToString(values["operation"]); err != nil {
		return err
	}
	if len(h.Operation) == 0 {
		h.Operation = "LIST"
	}
	if h.Role, err = coerce.ToString(values["role"]); err != nil {
		return err
	}
	if len(h.Role) == 0 {
		h.Role = string(statebased.RoleTypeMember)
	}
	return nil
}

// ToMap converts activity input to a map
func (i *Input) ToMap() map[string]interface{} {
	var keys []interface{}
	for _, k := range i.StateKeys {
		keys = append(keys, k)
	}
	var orgs []interface{}
	for _, org := range i.Organizations {
		orgs = append(orgs, org)
	}
	return map[string]interface{}{
		"keys":              keys,
		"organizations":     orgs,
		"policy":            i.Policy,
		"privateCollection": i.PrivateCollection,
	}
}

// FromMap sets activity input values from a map
func (i *Input) FromMap(values map[string]interface{}) error {
	var err error
	var keys interface{}
	if keys, err = coerce.ToAny(values["keys"]); err != nil {
		return err
	}
	switch v := keys.(type) {
	case []interface{}:
		for _, d := range v {
			k := strings.TrimSpace(d.(string))
			if len(k) > 0 {
				i.StateKeys = append(i.StateKeys, k)
			}
		}
	case string:
		i.StateKeys = []string{strings.TrimSpace(v)}
	}

	var orgs interface{}
	if orgs, err = coerce.ToAny(values["organizations"]); err != nil {
		return err
	}
	switch v := orgs.(type) {
	case []interface{}:
		for _, d := range v {
			k := strings.TrimSpace(d.(string))
			if len(k) > 0 {
				i.Organizations = append(i.Organizations, k)
			}
		}
	case string:
		i.Organizations = []string{strings.TrimSpace(v)}
	}

	if i.Policy, err = coerce.ToString(values["policy"]); err != nil {
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
