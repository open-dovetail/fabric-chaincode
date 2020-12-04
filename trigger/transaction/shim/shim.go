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
	"github.com/project-flogo/core/engine"
	"github.com/project-flogo/core/support/log"
)

var (
	cfgJson       string
	cfgEngine     string
	cfgCompressed bool
)

// Contract implements chaincode interface for invoking Flogo flows
type Contract struct {
}

var logger = log.ChildLogger(log.RootLogger(), "fabric-transaction-shim")

func init() {
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

	status, result, err := trigger.Invoke(stub, fn, args)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to execute transaction: %s, error: %+v", fn, err))
	} else if status == shim.OK {
		return shim.Success([]byte(result))
	} else {
		return pb.Response{
			Status:  int32(status),
			Payload: []byte(result),
		}
	}
}

// main function starts up chaincode in the container during instantiate
func main() {

	os.Setenv("FLOGO_RUNNER_TYPE", "DIRECT")
	os.Setenv("FLOGO_ENGINE_STOP_ON_ERROR", "false")

	// necessary to access schema of complex object attributes from activity context
	schema.Enable()
	schema.DisableValidation()

	cfg, err := engine.LoadAppConfig(cfgJson, cfgCompressed)
	if err != nil {
		logger.Errorf("Failed to load flogo config: %s", err.Error())
		os.Exit(1)
	}

	_, err = engine.New(cfg, engine.ConfigOption(cfgEngine, cfgCompressed))
	if err != nil {
		logger.Errorf("Failed to create flogo engine instance: %+v", err)
		os.Exit(1)
	}

	if err := shim.Start(new(Contract)); err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
