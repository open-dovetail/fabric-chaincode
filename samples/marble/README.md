# marble

This example uses the [Open-source Flogo](http://www.flogo.io/) to implement a [Hyperledger Fabric](https://www.hyperledger.org/projects/fabric) sample chaincode [marbles02](https://github.com/hyperledger/fabric-samples/tree/master/chaincode/marbles02/go). The sample demonstrates basic features of the Hyperledger Fabric, including creeation and update of states and composite-keys, as well as different types of queries for state and history with pagination. The chaincode is extended to illustrate Attribute-Based Access Control (ABAC), and it is implemented visually by using [Flogo Web UI](https://github.com/project-flogo/flogo-web).

## Prerequisite

Set up development environment by following the **Getting Started** instructions in [README.md](../../README.md).

## Build and deploy chaincode to Hyperledger Fabric

The Flogo model [marble.json](./marble.json) is the chaincode implementation. In a terminal console, type the command `make`, which will perform the following steps:

- Build a deployable chaincode package [marble_cc_1.0.tar.gz](./marble_cc_1.0.tar.gz) for the model file [marble.json](marble.json).
- Deploy the package and test scripts to the **Fabric test-network** that was installed during the prerequisite setup.

## Start test-network and test chaincode

Execute following steps to start the **Fabric test-network** and invoke the **marble_cc** chaincode:

```bash
# start Fabric test-network
make start

# install marble_cc
make cc-init

# invoke transactions of marble_cc
make cc-test
```

## Shutdown test-network

After successful test, you may shutdown the **Fabric test-network**:

```bash
make shutdown
```

## View and edit Flogo model

You can view and edit the chaincode implementation in a web-browser. First, start the **Flogo Web UI**:

```bash
docker run -it -p 3303:3303 yxuco/flogo-ui eula-accept
```

Open the **Flogo Web UI** in a web-browser by using the URL: `http://localhost:3303`. Then import the app by selecting the model file [marble.json](./marble.json).

For problems of importing the model, refer the troubleshoot instructions [here](../../README.md).
