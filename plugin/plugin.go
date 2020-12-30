package plugin

import (
	"fmt"
	"os"

	"github.com/open-dovetail/fabric-chaincode/plugin/contract"
	"github.com/project-flogo/cli/common" // Flogo CLI support code
	"github.com/spf13/cobra"
)

var contractFile string

func init() {
	contract2flow.Flags().StringVarP(&contractFile, "contract", "c", "contract.json", "specify a contract.json to create Flogo app from")
	common.RegisterPlugin(contract2flow)
}

var contract2flow = &cobra.Command{
	Use:              "contract2flow",
	Short:            "convert fabric contract to flogo flow",
	Long:             "This plugin read a Hyperledger Fabric smart contract, as defined by https://github.com/open-dovetail/fabric-chaincode/blob/master/contract/contract-schema.json, and generate a Flogo app that can be built and run as Fabric chaincode.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Create Flogo app from", contractFile)
		spec, err := contract.ReadContract(contractFile)
		if err != nil {
			fmt.Printf("Failed read and parse contract file %s: %+v\n", contractFile, err)
			os.Exit(1)
		}
		fmt.Printf("parsed contract: %v\n", spec)
	},
}
