package endorsement

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/pkg/statebased"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	cb "github.com/hyperledger/fabric-protos-go/common"
	cm "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/common/policydsl"
	"github.com/open-dovetail/fabric-chaincode/common"
	"github.com/pkg/errors"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/support/log"
)

// Create a new logger
var logger = log.ChildLogger(log.RootLogger(), "activity-fabric-endorsement")

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

func init() {
	_ = activity.Register(&Activity{}, New)
}

// Activity is a stub for executing Hyperledger Fabric get operations
type Activity struct {
	operation string
	role      string
}

// New creates a new Activity
func New(ctx activity.InitContext) (activity.Activity, error) {
	s := &Settings{}
	logger.Infof("Create Endorsement activity with InitContxt settings %v", ctx.Settings())
	if err := s.FromMap(ctx.Settings()); err != nil {
		logger.Errorf("failed to configure Endorsement activity %v", err)
		return nil, err
	}

	return &Activity{
		operation: s.Operation,
		role:      s.Role,
	}, nil
}

// Metadata implements activity.Activity.Metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements activity.Activity.Eval
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	logger.Debugf("%v", a)

	// check input args
	input := &Input{}
	if err = ctx.GetInputObject(input); err != nil {
		return false, err
	}

	if strings.HasPrefix(input.PrivateCollection, "_implicit") {
		// override implicit collection using client's org
		mspid, err := common.ResolveFlowData("$.cid.mspid", ctx)
		if err != nil {
			logger.Debugf("failed to fetch client mspid: %v\n", err)
		} else {
			if msp, ok := mspid.(string); ok && len(msp) > 0 {
				input.PrivateCollection = "_implicit_org_" + msp
				logger.Debugf("set implicit PDC to %s\n", input.PrivateCollection)
			}
		}
	}

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		msg := fmt.Sprintf("failed to retrieve fabric stub: %v", err)
		logger.Errorf("%s", msg)
		output := &Output{Code: 500, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	if len(input.StateKeys) == 0 {
		msg := "state key is not specified"
		logger.Errorf("%s", msg)
		output := &Output{Code: 400, Message: msg}
		ctx.SetOutputObject(output)
		return false, err
	}

	var code int
	var value []interface{}
	for _, key := range input.StateKeys {
		c, v, e := a.handlePolicy(stub, input, key)
		if e != nil {
			err = e
		}
		if c > code {
			code = c
		}
		if v != nil {
			value = append(value, v)
		}
	}

	// set partial success code
	if len(value) > 0 && code >= 300 {
		code = 206
		err = nil
	}

	if err != nil {
		// error response
		output := &Output{Code: code, Message: err.Error()}
		ctx.SetOutputObject(output)
		return false, err
	}

	// successful response
	data, _ := json.Marshal(value)
	output := &Output{
		Code:    code,
		Message: string(data),
		Result:  value,
	}
	ctx.SetOutputObject(output)
	return true, nil
}

// handlePolicy performs an operation on endorsement policy of a state key
// returns status, operation result or error
func (a *Activity) handlePolicy(stub shim.ChaincodeStubInterface, input *Input, key string) (int, interface{}, error) {
	ep, err := getEndorsementPolicy(stub, input.PrivateCollection, key)
	if err != nil {
		return 500, nil, err
	}

	var stateEP statebased.KeyEndorsementPolicy
	switch a.operation {
	case "ADD":
		stateEP, err = a.addOrgsToPolicy(ep, input.Organizations)
	case "DELETE":
		stateEP, err = a.deleteOrgsFromPolicy(ep, input.Organizations)
	case "LIST":
		stateEP, err = statebased.NewStateEP(ep)
	case "SET":
		stateEP, err = createNewPolicy(input.Policy)
	default:
		msg := fmt.Sprintf("operation %s is not supported", a.operation)
		logger.Error(msg)
		err = errors.New(msg)
	}
	if err != nil {
		return 500, nil, err
	}

	if a.operation != "LIST" {
		if ep, err = stateEP.Policy(); err != nil {
			return 500, nil, err
		}
		// update endorsement policy for key
		if err := setEndorsementPolicy(stub, input.PrivateCollection, key, ep); err != nil {
			msg := fmt.Sprintf("failed to set policy for %s @ %s", key, input.PrivateCollection)
			logger.Errorf("%s: %+v", msg, err)
			return 500, nil, errors.Wrapf(err, msg)
		}
	}
	orgs := stateEP.ListOrgs()
	policy, _ := unmarshalPolicy(ep)

	return 200, map[string]interface{}{
		"key":           key,
		"organizations": orgs,
		"policy":        policy,
	}, nil
}

func unmarshalPolicy(policy []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	envl := &cb.SignaturePolicyEnvelope{}
	if err := proto.Unmarshal(policy, envl); err != nil {
		return nil, err
	}
	if rule := envl.GetRule(); rule != nil {
		result["rule"] = ruleToMap(rule)
	}

	var ids []interface{}
	for _, p := range envl.GetIdentities() {
		mr := &cm.MSPRole{}
		if err := proto.Unmarshal(p.Principal, mr); err == nil {
			p := fmt.Sprintf("%s.%s", mr.GetMspIdentifier(), mr.GetRole().String())
			ids = append(ids, p)
		}
	}
	if len(ids) > 0 {
		result["identies"] = ids
	}
	return result, nil
}

func ruleToMap(rule *cb.SignaturePolicy) map[string]interface{} {
	result := make(map[string]interface{})
	outOf := rule.GetNOutOf()
	if outOf == nil {
		// this is a leaf node of sign-by
		result["signedBy"] = rule.GetSignedBy()
		return result
	}
	result["outOf"] = outOf.N
	var subs []interface{}
	for _, r := range outOf.GetRules() {
		subs = append(subs, ruleToMap(r))
	}
	result["rules"] = subs
	return result
}

func getEndorsementPolicy(stub shim.ChaincodeStubInterface, store string, key string) ([]byte, error) {
	if len(store) > 0 {
		return stub.GetPrivateDataValidationParameter(store, key)
	}
	return stub.GetStateValidationParameter(key)
}

func setEndorsementPolicy(stub shim.ChaincodeStubInterface, store string, key string, ep []byte) error {
	if len(store) > 0 {
		return stub.SetPrivateDataValidationParameter(store, key, ep)
	}
	return stub.SetStateValidationParameter(key, ep)
}

func createNewPolicy(policy string) (statebased.KeyEndorsementPolicy, error) {
	// create new policy from policy string
	if len(policy) == 0 {
		msg := "policy is not specified for SET operation"
		logger.Error(msg)
		return nil, errors.New(msg)
	}
	envelope, err := policydsl.FromString(policy)
	if err != nil {
		msg := fmt.Sprintf("failed to parse policy string %s", policy)
		logger.Errorf("%s: %v", msg, err)
		return nil, errors.Wrapf(err, "%s", msg)
	}
	ep, err := proto.Marshal(envelope)
	if err != nil {
		msg := "failed to marshal signature policy"
		logger.Errorf("%s: %+v", msg, err)
		return nil, errors.Wrapf(err, msg)
	}
	return statebased.NewStateEP(ep)
}

func (a *Activity) deleteOrgsFromPolicy(ep []byte, orgs []string) (statebased.KeyEndorsementPolicy, error) {
	stateEP, err := statebased.NewStateEP(ep)
	if err != nil {
		logger.Errorf("failed to construct policy from channel default: %+v", err)
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, errors.New("No organization is specified")
	}
	stateEP.DelOrgs(orgs...)
	return stateEP, nil
}

func (a *Activity) addOrgsToPolicy(ep []byte, orgs []string) (statebased.KeyEndorsementPolicy, error) {
	stateEP, err := statebased.NewStateEP(ep)
	if err != nil {
		logger.Errorf("failed to construct policy from channel default: %+v", err)
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, errors.New("No organization is specified")
	}
	err = stateEP.AddOrgs(statebased.RoleType(a.role), orgs...)
	return stateEP, err
}
