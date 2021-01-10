# Flogo extension for Hyperledger Fabric chaincode

This Flogo extension is designed to allow developers to design and implement Hyperledger Fabric chaincode in the Flogo visual programming environment. This extension supports the following release versions:

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

- Download and setup Golang from [here](https://golang.org/dl/).
- Clone this repo into an empty working directory: `git clone https://github.com/open-dovetail/fabric-chaincode.git`
- Setup development environment by executing the script: `scripts/setup.sh`
- Build and run a sample Flogo model [marble](./samples/marble) as described in [README.md](./samples/marble/README.md)

## View and edit Flogo model

You can view and edit the chaincode implementation in a web-browser. First, start the **Flogo Web UI**:

```bash
docker run -it -p 3303:3303 flogo/flogo-docker eula-accept
```

Open the **Flogo Web UI** in a web-browser by using the URL: `http://localhost:3303`.

Install the following Dovetail contributions, i.e., click the link `Install contribution` at the top-right corner of the UI, and then enter the following URL to install.  If the installation fails, you can follow the `Troubleshoot` steps to patch the Flogo, and then retry installation.

- github.com/open-dovetail/fabric-chaincode/trigger/transaction
- github.com/open-dovetail/fabric-chaincode/activity/put
- github.com/open-dovetail/fabric-chaincode/activity/get
- github.com/open-dovetail/fabric-chaincode/activity/delete
- github.com/project-flogo/contrib/activity/noop


Then import the app by selecting the model file [marble.json](./marble.json).

## Troubleshoot

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
replace github.com/project-flogo/core => github.com/yxuco/core v1.2.1
```

This will make the Flogo Web UI to use the Flogo fork containing the patches required by the Dovetail contributions.
