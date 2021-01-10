# Fabric Delete activity

This Flogo activity contribution can be configured to perform delete operations on Hyperledger Fabric distributed ledger and private data collections. Some delete operations are demonstrated in the [contract example](../../contract).

## Delete one or more ledger states by state keys

This operation requires input data of one or an array of state keys, and optionally the definitions of one or more composite-keys, e.g.,

```json
    "activity": {
        "ref": "#delete",
        "settings": {
            "compositeKeys": {
                "mapping": {
                    "color~name": ["docType", "color", "name"],
                    "owner~name": ["docType", "owner", "name"]
                }
            }
        },
        "input": "=$flow.parameters.name"
    }
```

This example will delete the specified state from the ledger, as well as the 2 composite keys of `color~name` and `owner~name` for the record.

## Delete multiple ledger states by partial composite keys

This operation requires a composite-key definition, and input data used to construct composite-keys, e.g.,

```json
    "activity": {
        "ref": "#delete",
        "settings": {
            "compositeKeys": {
                "mapping": {
                    "owner~name": ["docType", "owner", "name"]
                }
            }
        },
        "input": {
            "data": {
                "mapping": {
                    "@foreach($activity[get_1].result)": {
                        "docType": "=$loop.value.docType",
                        "name": "=$loop.value.name",
                        "owner": "=$loop.value.owner"
                    }
                }
            }
        }
    }
```

This example will collect the ledger states matching the specified composite-keys, and delete the resulting records and the associated composite keys.

## Delete composite keys only

This operation requires to turn on the `keysOnly` flag besides a composite-key definition, and the input data used to construct composite-keys, e.g.,

```json
    "activity": {
        "ref": "#delete",
        "settings": {
            "compositeKeys": {
                "mapping": {
                    "owner~name": ["docType", "owner", "name"]
                }
            },
            "keysOnly": true
        },
        "input": {
            "data": {
                "mapping": {
                    "@foreach($activity[get_1].result)": {
                        "docType": "=$loop.value.docType",
                        "name": "=$loop.value.name",
                        "owner": "=$loop.value.owner"
                    }
                }
            }
        }
    }
```

This example will delete only the matching composite keys, not the associated ledger records.

## Delete data from private data collection

When a private data collection is specified in the input, data will be deleted from the specified private data collection, e.g.,

```json
    "activity": {
        "ref": "#delete",
        "input": {
            "data": "=$flow.parameters.name",
            "privateCollection": "_implicit"
        }
    }
```

This example will delete a record from the client's implicit private collection, i.e., `_implicit_org_<mspid>`.
