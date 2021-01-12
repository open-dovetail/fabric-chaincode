# contract sample

The [contract schema](./contract-schema.json) defines smart contracts that are independent of programming languages and development platforms. It is based on the [fabric-chaincode-node](https://github.com/hyperledger/fabric-chaincode-node/blob/master/apis/fabric-contract-api/schema/contract-schema.json) schema definition, but extended with additional execution rules to make the contract specification executable.

This [contract sample](./sample-contract.json) demonstrates how you can use **open-dovetail** to view and edit the contract specification, and to build and deploy the contract to the Fabric `test-network`, and then verify the functions of the specified contract, all with zero code.

## Prerequisite

Set up development environment by following the **Getting Started** instructions in [README.md](../README.md).

## Build and deploy chaincode to Hyperledger Fabric

In a terminal console, change to this directory, and type the command `make`, which will perform the following steps:

- Use `flogo contract2flow` CLI extension to convert the [sample-contract.json](./sample-contract.json) to an executable Flogo model `sample.json`;
- Build the Flogo model, `sample.json`, into a deployable chaincode package `sample_cc_1.0.tar.gz`;
- Deploy the chaincode package and test scripts to the **Fabric test-network** that was installed during the prerequisite setup.

## Start test-network and test chaincode

Execute following steps to start the **Fabric test-network** and invoke the **sample_cc** chaincode:

```bash
# start Fabric test-network
make start

# install sample_cc
make cc-init

# invoke transactions of sample_cc
make cc-test
```

## Shutdown test-network

After successful test, you may shutdown the **Fabric test-network**:

```bash
make shutdown
```

## View and edit the Flogo model

You can view and edit the chaincode implementation in a web-browser. First, start the **Flogo Web UI**:

```bash
docker run -it -p 3303:3303 flogo/flogo-docker eula-accept
```

Open the **Flogo Web UI** in a web-browser by using the URL: `http://localhost:3303`. Then install the required Flogo extensions as listed in [README](https://github.com/open-dovetail/fabric-chaincode#view-and-edit-flogo-model), and import the app by selecting the generated model file `sample.json`.

If you have license to [Flogo Enterprise](https://edelivery.tibco.com/storefront/eval/tibco-flogo-enterprise/prod11810.html), you can uncomment the line for `FE` in the [Makefile](./Makefile), and then execute `make` to generate model file for Flogo Enterprise. You can then start the Flogo Enterprise with `/path/to/flogo/2.10/bin/start-webui.sh`. Import the generated model, `sample.json`, you can then use the Web UI to edit the model, which is quite a bit more user-friendly than the open-source version of the Flogo Web UI.
