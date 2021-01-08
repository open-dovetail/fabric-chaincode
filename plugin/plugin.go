package plugin

import (
	"fmt"
	"os"

	"github.com/open-dovetail/fabric-chaincode/plugin/contract"
	"github.com/project-flogo/cli/common" // Flogo CLI support code
	"github.com/spf13/cobra"
)

var enterprise bool
var contractFile string
var appFile string

func init() {
	contract2flow.Flags().StringVarP(&contractFile, "contract", "c", "contract.json", "specify a contract.json to create Flogo app from")
	contract2flow.Flags().StringVarP(&appFile, "app", "o", "app.json", "specify the output file app.json")
	contract2flow.Flags().BoolVarP(&enterprise, "fe", "e", false, "user Flogo Enterprise")
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
			fmt.Printf("Failed to read and parse contract file %s: %+v\n", contractFile, err)
			os.Exit(1)
		}
		app, err := spec.ToAppConfig(enterprise)
		if err != nil {
			fmt.Printf("Failed to convert contract file %s: %+v\n", contractFile, err)
			os.Exit(1)
		}
		if err = contract.WriteAppConfig(app, appFile); err != nil {
			fmt.Printf("Failed to write app config file %s: %+v\n", appFile, err)
			os.Exit(1)
		}
		fmt.Printf("Successfully written app config file %s\n", appFile)
	},
}
