# Simple Contract

This sample is a step-by-step instruction for developing smart contract by using the Dovetail zero-code tools.

## Prerequisite

Follow instructions in [README.md](../../README.md) to setup dev environment and Flogo Web-UI or Flogo Enterprise.

## Define contract in JSON

Use a JSON editor such as [Visual Studio Code](https://code.visualstudio.com/download) to define a contract, which is already done for this sample as shown in [contract.json](./contract.json).

This simple contract specifies only 2 transactions, i.e., `createMarble` and `getMarble`, which demonstrate update and query operations on a ledger, respectively. The contract definition must be valid against the JSON schema as specified in [contract-schema.json](../../contract/contract-schema.json). A more advanced contract definition can be found in [sample-contract.json](../../contract/sample-contract.json), which demonstrates more advanced features such as composite key, rich query, private data collection, and ABAC, etc.

## Generate Flogo model for the contract

The following command generates a Flogo model from the contract definition, and the resulting Flogo model `simple.json` can be editted visually in Flogo Web UI.

```bash
flogo contract2flow -e -c contract.json -o simple.json
```

The flag `-e` in the above command means to generate a model for Flogo Enterprise, which has a more user-friendly model editor than the open-source Flogo Web UI. If you do not have license for Flogo Enterprise, you can remove the `-e` flag from the above command to generate a Flogo model for the open-source Flogo Web UI.

## Edit Flogo model in Web UI

In the contract defintion [contract.json](./contract.json), we deliberately omitted the input data mapping in the `#put` activity of the transaction `createMarble`. Thus, the generated Flogo model `simple.json` is not fully functional yet. It is, in fact, a common scenario that data mappings in a contract definition may be complex and thus can be better mapped with the help of the utilities in Flogo Web UI.

Assume that we use Flogo Enterprise to edit the model (the open-source Flogo Web UI works similarly). Open Flogo Enterprise UI, and create a new model and name it as `simple_fe`, then import the generated model file `simple.json`.

Drill down to the flow `createMarble` and activity `put_1`. Map the input data of the activity to match the following;

```json
{
  "key": "=$flow.parameters.name",
  "value": {
    "color": "=$flow.parameters.color",
    "docType": "marble",
    "name": "=$flow.parameters.name",
    "owner": "=$flow.parameters.owner",
    "size": "=$flow.parameters.size"
  }
}
```

Export the updated Flogo App, which will download the Flogo model as a file `simple_fe.json`.

## Build chaincode package

The following command builds the Flogo model `simple_fe.json` into a chaincode package.

```bash
../../scripts/build.sh simple_fe.json simple_cc
```

It builds a chaincode package named `simple_cc_1.0.tar.gz`, which can be installed in a Hyperledger Fabric network.

## Start Hyperledger Fabric test network

Start the `test-network` that was downloaded during the dev setup, and deploy the chaincode package to the test-network. (If your dev environment is not the same as the default, you may need to change the location of the `test-network` accordingly.)

```bash
cd ../../../hyperledger/fabric-samples/test-network && ./network.sh up createChannel &

cp simple_cc_1.0.tar.gz ../../../hyperledger/fabric-samples/chaincode
cp cc-init.sh ../../../hyperledger/fabric-samples/test-network/cc-init-simple.sh
cp cc-test.sh ../../../hyperledger/fabric-samples/test-network/cc-test-simple.sh
```

The above commands copied 2 test scripts to the `test-network` and they are used to verify the chaincode in the next step.

## Install and verify chaincode

The following command invokes the script [cc-init.sh](./cc-init.sh) from the `cli` docker container to install, approve, and commit the chaincode `simple_cc_1.0`.

```bash
docker exec cli bash -c './cc-init-simple.sh'
```

Run test script [cc-test.sh](./cc-test.sh) from the `cli` docker container to verify the functions of the chaincode:

```bash
docker exec cli bash -c './cc-test-simple.sh'
```

## Cleanup

Shutdown and cleanup the `test-network`.

```bash
cd ../../../hyperledger/fabric-samples/test-network && ./network.sh down
```
