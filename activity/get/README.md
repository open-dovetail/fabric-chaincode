# Fabric Get activity

This Flogo activity contribution can be configured to perform any read operations on Hyperledger Fabric distributed ledger and private data collections. Most of the read operations are demonstrated in the [contract example](../../contract).

## Retrieve one or more ledger states by state keys

The operation does not need to change any default configurations, you can simply map one or an array of state keys to the input, e.g.,

```json
    "activity": {
        "ref": "#get",
        "input": {
            "data": "=$flow.parameters.name"
        }
    }
```

## Retrieve multiple ledger states by partial composite keys

This operation requires a composite-key configuration, and input data for the attributes of the composite key, e.g.,

```json
    "activity": {
        "ref": "#get",
        "settings": {
            "compositeKeys": {
                "mapping": {
                    "color~name": ["docType", "color", "name"]
                }
            }
        },
        "input": {
            "data": {
                "mapping": {
                    "docType": "marble",
                    "color": "=$flow.parameters.color"
                }
            },
            "pageSize": 0,
            "bookmark": ""
        }
    }
```

This example retrieves all ledger states that matches the first 2 fields of the composite key `color~name`, in other words, it retrieves all marbles of a specified color from the Fabric ledger.

`pageSize` and `bookmark` are optional, and can be specified when result pagination is required.

## Retrieve multiple ledger states by CouchDB query

This operation requires a configuration of the CouchDB query statement, and input data for the query parameters, e.g.,

```json
    "activity": {
        "ref": "#get",
        "settings": {
            "query": {
                "mapping": {
                    "selector": {
                        "docType": "marble",
                        "owner": "$owner"
                    }
                }
            }
        },
        "input": {
            "data": {
                "mapping": {
                    "owner": "=$flow.parameters.owner"
                }
            },
            "pageSize": 0,
            "bookmark": ""
        }
    }
```

This example executes a CouchDB query that takes `$owner` as a parameter that is provided by the input data. In other words, this query retrieves all marbles in the Fabric ledger by a specified owner.

`pageSize` and `bookmark` are optional, and can be specified when result pagination is required.

## Retrieve the history of one or more state keys

The operation requires turning on the `history` flag, and input of one or an array of state keys, e.g.,

```json
    "activity": {
        "ref": "#get",
        "settings": {
            "history": true
        },
        "input": {
            "data": "=$flow.parameters.name"
        }
    }
```

## Retrieve composite-keys by partial composite key

This operation is normally not used, but it is supported, and requires turning on the `keysOnly` flag besides a composite-key configuration, and input data for the attributes of the composite key, e.g.,

```json
    "activity": {
        "ref": "#get",
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
                    "docType": "marble",
                    "color": "=$flow.parameters.color"
                }
            }
        }
    }
```

This example will return only the matching composite-keys, but not the corresponding ledger states.

## Retrieve ledger states by key range

This operation requires input data that specifies the `start` and `end` of the range of state keys, e.g.,

```json
    "activity": {
        "ref": "#get",
        "input": {
            "data": {
                "mapping": {
                    "start": "=$flow.parameters.startKey",
                    "end": "=$flow.parameters.endKey"
                }
            },
            "pageSize": 0,
            "bookmark": ""
        }
    }
```

This example will return all ledger states between a start state key (inclusive) and an end state key (exclusive).

`pageSize` and `bookmark` are optional, and can be specified when the result pagination is required.

## Retrieve data from private data collection

When a private data collection is specified in the input, data will be fetched from the specified private data collection, e.g.,

```json
    "activity": {
        "ref": "#get",
        "input": {
            "data": "=$flow.parameters.name",
            "privateCollection": "_implicit"
        }
    }
```

This example will retrieve data from the client's implicit private collection, i.e., `_implicit_org_<mspid>`.

Most of the above read operations can be executed on private data collections, except for the `history` query, which is not supported by private collections. Besides, pagination is mostly ignored for read operations on private data collections.

## Retrieve private data hash

To retrieve the public hash of a private data record, you can turn on the `privateHash` flag, and specify the state key and name of the private data collection, e.g.,

```json
    "activity": {
        "ref": "#get",
        "settings": {
            "privateHash": true
        },
        "input": {
            "data": "=$flow.parameters.name",
            "privateCollection": "_implicit"
        }
    }
```

This example will return the public hash of the specified private data on the implicit private data collection. The input data can specify one or an array of multiple state keys.
