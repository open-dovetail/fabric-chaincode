# Fabric Transaction trigger

This Flogo trigger contribution is the shim for Hyperledger Fabric chaincode. Its use is demonstrated in the [contract example](../../contract).

Each Fabric transaction in the chaincode is configured as a `handler` of this `Transaction trigger`, e.g.,

```json
    "ref": "#transaction",
    "settings": {
        "cid": "alias,role,email"
    },
    "handlers": [{
        "settings": {
            "name": "createMarble",
            "parameters": "name,color,size:0,owner"
        },
        "action: { ... }
    }]
```

It must specify a transaction name, e.g., `createMarble` in the above example, and a list of parameters for the transaction. The parameter names are configuted as a comma-delimited list, and the parameter data type is `string` by default, but other data types can be specified by a suffix of number or boolean value after a delimiter `:`. The supported JSON types are `0` for `integer`, `0.0` for `number`, and `false` for `boolean`.

The above example defines a Fabric transaction of name `createMarble` that accepts 4 parameters of names `name`, `color`, `size`, and `owner`, where the `size` is an integer, while other parameters are strings.

The `Transaction trigger` also extracts user info from the requestor's CA certificates, which includes the attributes of `id`, `mspid`, and `cn`. If the user certificates contain more custom attributes for the application, you can list the custom attrinute names in the `cid` configuration, and so they can be used by the chaincode for authorization purposes. In the above example, it lists 3 custom attribute names from the CA, i.e., `alias`, `role`, and `email`, which can be verified by the chainode to control the access of some operations.
