# Fabric Put activity

This Flogo activity contribution can be configured to perform write operations on Hyperledger Fabric distributed ledger and private data collections. Most of the write operations are demonstrated in the [contract example](../../contract).

## Insert or update one or more ledger states

This operation requires input data of one or an array of key-value pairs, and optionally defintions of one or more composite-keys. For `insert-only` operations, you can turn on the `createOnly` flag, e.g.,

```json
    "activity": {
        "ref": "#put",
        "settings": {
            "compositeKeys": {
                "mapping": {
                    "color~name": ["docType", "color", "name"],
                    "owner~name": ["docType", "owner", "name"]
                }
            },
            "createOnly": true
        },
        "input": {
            "data": {
                "mapping": {
                    "key": "=$flow.parameters.name",
                    "value": {
                        "color": "=$flow.parameters.color",
                        "docType": "marble",
                        "name": "=$flow.parameters.name",
                        "owner": "=$flow.parameters.owner",
                        "size": "=$flow.parameters.size"
                    }
                }
            }
        }
    }
```

This example creates a new record on the ledger using the specified state key and value. It also creates 2 composite keys of `color~name` and `owner~name` for indexed search. The request will be rejected if the specified state key already exists because `createOnly` is set to `true`.

The following example will, however, create or update multiple ledger records and the associated composite key `owner~name`.

```json
    "activity": {
        "ref": "#put",
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
                        "key": "=$loop.key",
                        "value": {
                            "color": "=$loop.value.color",
                            "docType": "=$loop.value.docType",
                            "name": "=$loop.value.name",
                            "owner": "=$flow.parameters.newOwner",
                            "size": "=$loop.value.size"
                        }
                    }
                }
            }
        }
    }
```

## Create or update one or more composite keys

This operation requires one or more composite-key definition, and input data used to construct composite-keys, e.g.,

```json
    "activity": {
        "ref": "#put",
        "settings": {
            "keysOnly": true,
            "compositeKeys": {
                "mapping": {
                    "color~name": ["docType", "color", "name"]
                }
            }
        },
        "input": {
            "data": {
                "mapping": {
                    "@foreach($activity[get_1].result)": {
                        "color": "=$loop.value.color",
                        "docType": "=$loop.value.docType",
                        "name": "=$loop.value.name"
                    }
                }
            }
        }
    }
```

This example will create the composite-key `color~name` for each of the input data records. It will not create any ledger records. This operation may be used to add search capability for existing ledger states, or used to store temperary data as composite-keys, which can be aggregated later in batches.

## Create or update records on private data collection

When a private data collection is specified in the input, data will be created/updated in the specified private data collection, e.g.,

```json
    "activity": {
        "ref": "#put",
        "input": {
            "data": {
                "mapping": {
                    "key": "=$activity[get_1].result[0].value.name",
                    "value": {
                        "docType": "marblePrivate",
                        "name": "=$activity[get_1].result[0].value.name",
                        "price": "=$flow.transient.marble.price"
                    }
                }
            },
            "privateCollection": "_implicit"
        }
    }
```

This example will create/update data in the client's implicit private collection, i.e., `_implicit_org_<mspid>`.
