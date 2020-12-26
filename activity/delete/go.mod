module github.com/open-dovetail/fabric-chaincode/activity/delete

go 1.14

replace github.com/project-flogo/flow => github.com/yxuco/flow v1.1.1

replace github.com/project-flogo/core => github.com/yxuco/core v1.2.2

require (
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200728190242-9b3ae92d8664
	github.com/open-dovetail/fabric-chaincode/common v0.0.6
	github.com/pkg/errors v0.9.1
	github.com/project-flogo/core v1.2.0
	github.com/stretchr/testify v1.6.1
)
