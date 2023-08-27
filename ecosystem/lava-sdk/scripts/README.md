# after protoc_grpc_relay generates
protoc grpc egenrates incompatible names to the proto field names
so we need to change objects form camel casing to snake casing that fits their field names
for the objects RelaySession RelayRequest RelayPrivateData, Badge, RelayReply
example:
## before 
```
proto.lavanet.lava.pairing.RelaySession.toObject = function(includeInstance, msg) {
  var f, obj = {
    specId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    contentHash: msg.getContentHash_asB64(),
    sessionId: jspb.Message.getFieldWithDefault(msg, 3, 0),
    cuSum: jspb.Message.getFieldWithDefault(msg, 4, 0),
    provider: jspb.Message.getFieldWithDefault(msg, 5, ""),
    relayNum: jspb.Message.getFieldWithDefault(msg, 6, 0),
    qosReport: (f = msg.getQosReport()) && proto.lavanet.lava.pairing.QualityOfServiceReport.toObject(includeInstance, f),
    epoch: jspb.Message.getFieldWithDefault(msg, 8, 0),
    unresponsiveProviders: msg.getUnresponsiveProviders_asB64(),
    lavaChainId: jspb.Message.getFieldWithDefault(msg, 10, ""),
    sig: msg.getSig_asB64(),
    badge: (f = msg.getBadge()) && proto.lavanet.lava.pairing.Badge.toObject(includeInstance, f),
    qosExcellenceReport: (f = msg.getQosExcellenceReport()) && proto.lavanet.lava.pairing.QualityOfServiceReport.toObject(includeInstance, f)
  };
};
```
## after
```
proto.lavanet.lava.pairing.RelaySession.toObject = function(includeInstance, msg) {
  var f, obj = {
    spec_id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    content_hash: msg.getContentHash_asB64(),
    session_id: jspb.Message.getFieldWithDefault(msg, 3, 0),
    cu_sum: jspb.Message.getFieldWithDefault(msg, 4, 0),
    provider: jspb.Message.getFieldWithDefault(msg, 5, ""),
    relay_num: jspb.Message.getFieldWithDefault(msg, 6, 0),
    qos_report: (f = msg.getQosReport()) && proto.lavanet.lava.pairing.QualityOfServiceReport.toObject(includeInstance, f),
    epoch: jspb.Message.getFieldWithDefault(msg, 8, 0),
    unresponsive_providers: msg.getUnresponsiveProviders_asB64(),
    lava_chain_id: jspb.Message.getFieldWithDefault(msg, 10, ""),
    sig: msg.getSig_asB64(),
    badge: (f = msg.getBadge()) && proto.lavanet.lava.pairing.Badge.toObject(includeInstance, f),
    qos_excellence_report: (f = msg.getQosExcellenceReport()) && proto.lavanet.lava.pairing.QualityOfServiceReport.toObject(includeInstance, f)
  };
```

we now added a script to automatically change the names see fix_grpc_web_camel_case.py