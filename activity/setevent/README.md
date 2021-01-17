# Fabric Setevent activity

This Flogo activity contribution can be used to create a chaincode event, for example,

```json
    "activity": {
        "ref": "#setevent",
        "input": {
            "name": "alert",
            "payload": {
                "message": "over threshold",
                "temperature": 100,
                "threshold": 90
            }
        }
    }
```

The `payload` can be any simple or complex JSON document.
