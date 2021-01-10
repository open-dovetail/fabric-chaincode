# Fabric Delete activity

This Flogo activity contribution can be configured to perform any delete operations on Hyperledger Fabric distributed ledger and private data collections. Some delete operations are demonstrated in the [contract example](../../contract).

## Delete one or more ledger states by state keys

The operation requires input data of one or an array of key-value pairs, and optionally specify one or more composite-keys. For `insert-only` operations, you can turn on the `createOnly` flag, e.g.,

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

This example delete the specified state from the ledger, as well as the 2 composite keys of `color~name` and `owner~name` for this record.

## Delete multiple ledger states by composite keys

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

This example will collect the ledger states for the specified composite-keys, and delete the matching ledger records and the associated composite keys.

## Delete composite keys only

This operation requires to turn on the `keysOnly` flag besides a composite-key definition, and input data used to construct composite-keys, e.g.,

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

This example will delete data from the client's implicit private collection, i.e., `_implicit_org_<mspid>`.
