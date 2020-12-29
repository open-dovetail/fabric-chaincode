package plugin

import (
	"fmt"

	"github.com/project-flogo/cli/common" // Flogo CLI support code
	"github.com/spf13/cobra"
)

var contract string

func init() {
	contract2flow.Flags().StringVarP(&contract, "contract", "c", "contract.json", "specify a contract.json to create Flogo app from")
	common.RegisterPlugin(contract2flow)
}

var contract2flow = &cobra.Command{
	Use:   "contract2flow",
	Short: "convert fabric contract to flogo flow",
	Long:  "This plugin read a Hyperledger Fabric smart contract, as defined by https://github.com/open-dovetail/fabric-chaincode/blob/master/contract/contract-schema.json, and generate a Flogo app that can be built and run as Fabric chaincode.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Create Flogo app from", contract)
	},
}
