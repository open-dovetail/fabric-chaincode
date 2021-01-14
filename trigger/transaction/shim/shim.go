/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package main

import (
	"fmt"
	"os"
	"strings"

	shim "github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	trigger "github.com/open-dovetail/fabric-chaincode/trigger/transaction"
	_ "github.com/project-flogo/core/data/expression/script"
	"github.com/project-flogo/core/data/schema"
	"github.com/project-flogo/core/support/log"
)

// Contract implements chaincode interface for invoking Flogo flows
type Contract struct {
}

var logger = log.ChildLogger(log.RootLogger(), "fabric-transaction-shim")

func init() {
	os.Setenv("FLOGO_RUNNER_TYPE", "DIRECT")
	os.Setenv("FLOGO_ENGINE_STOP_ON_ERROR", "false")
	os.Setenv("FLOGO_MAPPING_IGNORE_ERRORS", "true")

	// necessary to access schema of complex object attributes from activity context
	schema.Enable()
	schema.DisableValidation()
}

func setLogLevel() {
	//  get log level from env FLOGO_LOG_LEVEL or CORE_CHAINCODE_LOGGING_LEVEL
	logLevel := "DEBUG"
	if l, ok := os.LookupEnv("FLOGO_LOG_LEVEL"); ok {
		logLevel = strings.ToUpper(l)
	} else if l, ok := os.LookupEnv("CORE_CHAINCODE_LOGGING_LEVEL"); ok {
		logLevel = strings.ToUpper(l)
	}
	switch logLevel {
	case "FATAL", "PANIC", "ERROR":
		log.SetLogLevel(log.RootLogger(), log.ErrorLevel)
	case "WARN", "WARNING":
		log.SetLogLevel(log.RootLogger(), log.WarnLevel)
	case "INFO":
		log.SetLogLevel(log.RootLogger(), log.InfoLevel)
	case "DEBUG", "TRACE":
		log.SetLogLevel(log.RootLogger(), log.DebugLevel)
	default:
		log.SetLogLevel(log.RootLogger(), log.DefaultLogLevel)
	}
}

// Init is called during chaincode instantiation to initialize any data,
// and also calls this function to reset or to migrate data.
func (t *Contract) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode.
func (t *Contract) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	fn, args := stub.GetFunctionAndParameters()
	logger.Debugf("invoke transaction fn=%s, args=%+v", fn, args)

	status, payload := trigger.Invoke(stub, fn, args)
	return pb.Response{
		Status:  int32(status),
		Payload: payload,
	}
}

// main function starts up chaincode in the container during instantiate
func main() {
	setLogLevel()

	if err := shim.Start(new(Contract)); err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
