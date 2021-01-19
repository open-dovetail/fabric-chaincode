# Fabric Invokechaincode activity

This Flogo activity contribution can be used to invoke a specified chaincode from the working chaincode, for example,

```json
    "activity": {
        "ref": "#invokechaincode",
        "input": {
            "chaincodeName": "marble_cc",
            "channelID": "mychannel",
            "transactionName": "createMarble",
            "parameters": "=array.create(\"marble1\", \"blue\", \"50\", \"tom\")"
        }
    }
```

If the `channelID` is not specified, the channel of the calling chaincode will be used.
