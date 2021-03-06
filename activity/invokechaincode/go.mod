module github.com/open-dovetail/fabric-chaincode/activity/invokechaincode

go 1.14

replace github.com/project-flogo/flow => github.com/yxuco/flow v1.1.1

replace github.com/project-flogo/core => github.com/yxuco/core v1.2.2

replace go.uber.org/multierr => go.uber.org/multierr v1.6.0

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20201119163726-f8ef75b17719
	github.com/hyperledger/fabric-protos-go v0.0.0-20190919234611-2a87503ac7c9
	github.com/open-dovetail/fabric-chaincode/common v0.1.1
	github.com/pkg/errors v0.9.1
	github.com/project-flogo/core v1.2.0
	github.com/stretchr/testify v1.6.1
)
