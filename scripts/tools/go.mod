module github.com/open-dovetail/tools

go 1.14

replace github.com/project-flogo/cli => github.com/yxuco/cli v0.10.1-0.20201211003232-196e588c1452

replace github.com/open-dovetail/fabric-chaincode/plugin => /Users/yxu/work/open-dovetail/fabric-chaincode/plugin

require (
	github.com/open-dovetail/fabric-chaincode/plugin v0.0.2
	github.com/project-flogo/cli v0.10.0
	github.com/spf13/cobra v1.1.1 // indirect
)
