# Flogo extension for Hyperledger Fabric chaincode

This [Flogo](http://www.flogo.io/) extension is designed to allow developers to design and implement Hyperledger Fabric chaincode in the Flogo visual programming environment. This extension supports the following release versions:

- [Flogo Web UI](http://www.flogo.io/)
- [Hyperledger Fabric 2.2](https://www.hyperledger.org/projects/fabric)

The [Transaction Trigger](trigger/transaction) starts chaincode transactions for requests containing preconfigured input parameters and/or transient data.

The Flogo extension supports the following activities for storing and querying data in the distributed ledger and/or in private data collections.

- [**Put**](activity/put): Insert or update one or more records in the distributed ledger or a private data collection, and optionally update associated compsite keys.
- [**Get**](activity/get): Retrieve one or more records corresponds to state keys or composite keys in the distributed ledger or a private data collection, including execution of range query or couchdb rich query, as well as fetching history of states of specified state keys.
- [**Delete**](activity/delete): Mark the state as deleted for one or more state keys in the distributed ledger or a private data collection, and delete its associated composite keys. Optionally, it can delete only the state, or only a composite key.
- [**Set Event**](activity/setevent): Set a specified event and payload for a blockchain transaction.
- [**Set Endorsement Policy**](activity/endorsement): Set state-based endorsement policy by adding or deleting an endorsement organization, or by specifying a new endorsement policy.
- [**Invoke Chaincode**](activity/invokechaincode): Invoke a local chaincode, and returns response data from the called transaction.

With these Flogo extensions, Hyperledger Fabric chaincode can be designed and implemented by using the **Flogo Web UI** with zero code.

## Getting Started

For Golang developers on Mac or Linux, you can setup the full development environment locally as follows:

- Download and setup Golang from [here](https://golang.org/dl/).
- Clone this repo into an empty working directory: `git clone https://github.com/open-dovetail/fabric-chaincode.git`
- Setup development environment by executing the script: `scripts/setup.sh`
- Build and run a sample Flogo model [marble](./samples/marble) as described in [README.md](./samples/marble/README.md)

Other developers can follow the instructions in the [demo](https://github.com/open-dovetail/demo/tree/master/blockchain/docker) that uses prebuilt Docker images to build Dovetail applications. This approach requires installation of only Docker and docker-compose, and should work in any platform that supports Docker.

## Write smart contract in JSON

For smart contract developers who do not want to code, you can define chaincode transactions in a JSON file, and then use the Flogo CLI command to generate a Flogo app and build it into a chaincode package that can be deployed and run in a Fabric network.

All you need to do is to provided a JSON specification of a contract, and then optionally edit the data mapping via drag-and-drop in either the open-source Flogo Web-UI or the more advanced TIBCO Flogo Enterprise Web-UI. Thus, you can implement a Fabric chaincode without any programming in Java, Go, nor JavaScript.

The [contract example](./contract) shows the JSON schema for smart contract and a sample contract that you can build and test using the Fabric test-network.

The [demo contract](https://github.com/open-dovetail/demo/tree/master/blockchain) shows the same build process and [steps](https://github.com/open-dovetail/demo/blob/master/blockchain/docker/README.md) to build artifacts by using preconfigured Docker containers.

## View and edit Flogo model

To view and edit the chaincode implementation in a web-browser, you can start a **Flogo Web UI** that is preconfigured with Dovetail extensions:

```bash
docker run -it -p 3303:3303 yxuco/flogo-ui eula-accept
```

or, you can start the most recent release of **Flogo Web UI** and then install required Dovetail extensions:

```bash
docker run -it -p 3303:3303 flogo/flogo-docker eula-accept
```

Open the **Flogo Web UI** in a web-browser by using the URL: `http://localhost:3303`.

Install the following Dovetail contributions, i.e., click the link `Install contribution` at the top-right corner of the UI, and then enter the following URL to install. If the installation fails, you can follow the `Troubleshoot` steps below to patch Flogo libs, and then retry the installation.

- github.com/open-dovetail/fabric-chaincode/trigger/transaction
- github.com/open-dovetail/fabric-chaincode/activity/put
- github.com/open-dovetail/fabric-chaincode/activity/get
- github.com/open-dovetail/fabric-chaincode/activity/delete
- github.com/project-flogo/contrib/activity/noop

You can then import a sample app by selecting the model file [samples/marble/marble.json](./samples/marble/marble.json).

## Status code

The Dovetail contributions will all return a status code similar to HTTP spec as follows:

- **200** OK
- **201** Created
- **206** Partial Content (e.g., paged result, or not all PUT succeeded)
- **400** Bad Request
- **403** Forbidden (e.g., user authenticated but not authorized)
- **404** Not Found (e.g., query result is empty)
- **409** Conflict (e.g., cannot create record for existing key when `createOnly=true`)
- **500** Internal Server Error (e.g., unexpected exception in chaincode)
- **501** Not Implemented (e.g., transaction name is not configured by trigger)

## Troubleshoot

### Failed to import Flogo model

Make sure that you have installed required Flogo contributions listed above.

### Failed to install dovetail contributions in Web UI

The Dovetail contributions require a couple of Flogo patches that have not yet been merged to the Flogo core/flow projects, and so to install Dovetail contributions in the Web UI, you can make the following changes after you start the Web UI docker container:

First, start a shell in the Web UI docker container:

```bash
docker exec -it $(docker ps --filter ancestor=flogo/flogo-docker --format "{{.ID}}") bash
```

Then, in the docker container shell, change directory to the `flogo-web` source directory:

```bash
cd /flogo-web/local/engines/flogo-web/src
```

If you need `vi` editor to edit files in the docker container, you can install it as follows:

```bash
apt update
apt install vim
```

Then, you can add the following lines to the end of the file `go.mod`:

```script
replace github.com/project-flogo/flow => github.com/yxuco/flow v1.1.1
replace github.com/project-flogo/core => github.com/yxuco/core v1.2.2
```

This will make the Flogo Web UI to use the Flogo fork containing the patches required by the Dovetail contributions.
