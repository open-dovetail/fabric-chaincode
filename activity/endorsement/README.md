# Fabric Endorsement activity

This Flogo activity contribution can be configured to update or list state-based endorsement policies.

## List the state-based endorsement policy for one or more ledger state

This operation requires to configure the operation name as `LIST`, and specify one or more state keys in the input data, e.g.,

```json
    "activity": {
        "ref": "#endorsement",
        "settings": {
            "operation": "LIST"
        },
        "input": {
            "keys": ["key1", "key2"]
        }
    }
```

This example will return the state-based endorsement policies for the specified state keys, e.g.,

```json
[
  {
    "key": "key1",
    "policy": {
      "orgs": ["org1.MEMBER", "org2.MEMBER"],
      "rule": {
        "outOf": 2,
        "rules": [
          {
            "signedBy": 0
          },
          {
            "signedBy": 1
          }
        ]
      }
    }
  },
  {
    "key": "key2",
    "policy": {
      "orgs": ["org1.MEMBER", "org2.MEMBER"],
      "rule": {
        "outOf": 2,
        "rules": [
          {
            "signedBy": 0
          },
          {
            "signedBy": 1
          }
        ]
      }
    }
  }
]
```

## Set state-based endorsement policy for one or more ledger states

This operation requires to configure the operation name as `SET`, and specify one or more state keys in the input data, as well as the endorsement policy, e.g.,

```json
    "activity": {
        "ref": "#endorsement",
        "settings": {
            "operation": "SET"
        },
        "input": {
            "keys": "key1",
            "policy": "OutOf(2, 'org1.peer', 'org2.peer', 'org3.peer')"
        }
    }
```

This sample will set the state key `key1` to require `2` of the 3 organizations to endorse the transaction.

## Add one or more organizations to the endorsement policy of one or more ledger states

This operation requires to configure the operation name as `ADD`, and specify one or more state keys in the input data, as well as a list of organizations to add, e.g.,

```json
    "activity": {
        "ref": "#endorsement",
        "settings": {
            "operation": "ADD",
            "role": "PEER"
        },
        "input": {
            "keys": "key1",
            "organizations": ["org1", "org2"]
        }
    }
```

This sample will add `org1.PEER` and `org2.PEER` to the endorsement policy for state key `key1`. Note that it also makes all the original organizations required for endorsement. Even if originally, it requires only some participants to endorse the transaction, after the `ADD` operation, all old and new organizations are required to endorse the transaction.

## Remove one or more organizations from the endorsement policy of one or more ledger states

This operation requires to configure the operation name as `DELETE`, and specify one or more state keys in the input data, as well as a list of organizations to remove, e.g.,

```json
    "activity": {
        "ref": "#endorsement",
        "settings": {
            "operation": "DELETE"
        },
        "input": {
            "keys": "key1",
            "organizations": "org1"
        }
    }
```

This sample will remove `org1` from the endorsement policy for state key `key1`. Note that it also makes all the remaining organizations required for endorsement. Even if originally, it requires only some participants to endorse the transaction, after the `DELETE` operation, all the remaining organizations are required to endorse the transaction.

## Set endorsement policy for one or more keys of a private data collection

This operation requires to specify the name of a private data collection, e.g.,

```json
    "activity": {
        "ref": "#endorsement",
        "settings": {
            "operation": "SET"
        },
        "input": {
            "keys": "key1",
            "policy": "OutOf(2, 'org1.peer', 'org2.peer', 'org3.peer')",
            "privateCollection": "_implicit"
        }
    }
```

This sample will set the endorsement policy for a key in th client's implicit private collection, i.e., `_implicit_org_<mspid>`. All the above examples apply to private data collections, too.
