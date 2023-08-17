// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/relay.proto

var lavanet_lava_pairing_relay_pb = require("../../../lavanet/lava/pairing/relay_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var Relayer = (function () {
  function Relayer() {}
  Relayer.serviceName = "lavanet.lava.pairing.Relayer";
  return Relayer;
}());

Relayer.Relay = {
  methodName: "Relay",
  service: Relayer,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_relay_pb.RelayRequest,
  responseType: lavanet_lava_pairing_relay_pb.RelayReply
};

Relayer.RelaySubscribe = {
  methodName: "RelaySubscribe",
  service: Relayer,
  requestStream: false,
  responseStream: true,
  requestType: lavanet_lava_pairing_relay_pb.RelayRequest,
  responseType: lavanet_lava_pairing_relay_pb.RelayReply
};

Relayer.Probe = {
  methodName: "Probe",
  service: Relayer,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_relay_pb.ProbeRequest,
  responseType: lavanet_lava_pairing_relay_pb.ProbeReply
};

exports.Relayer = Relayer;

function RelayerClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

RelayerClient.prototype.relay = function relay(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Relayer.Relay, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

RelayerClient.prototype.relaySubscribe = function relaySubscribe(requestMessage, metadata) {
  var listeners = {
    data: [],
    end: [],
    status: []
  };
  var client = grpc.invoke(Relayer.RelaySubscribe, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onMessage: function (responseMessage) {
      listeners.data.forEach(function (handler) {
        handler(responseMessage);
      });
    },
    onEnd: function (status, statusMessage, trailers) {
      listeners.status.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners.end.forEach(function (handler) {
        handler({ code: status, details: statusMessage, metadata: trailers });
      });
      listeners = null;
    }
  });
  return {
    on: function (type, handler) {
      listeners[type].push(handler);
      return this;
    },
    cancel: function () {
      listeners = null;
      client.close();
    }
  };
};

RelayerClient.prototype.probe = function probe(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Relayer.Probe, {
    request: requestMessage,
    host: this.serviceHost,
    metadata: metadata,
    transport: this.options.transport,
    debug: this.options.debug,
    onEnd: function (response) {
      if (callback) {
        if (response.status !== grpc.Code.OK) {
          var err = new Error(response.statusMessage);
          err.code = response.status;
          err.metadata = response.trailers;
          callback(err, null);
        } else {
          callback(null, response.message);
        }
      }
    }
  });
  return {
    cancel: function () {
      callback = null;
      client.close();
    }
  };
};

exports.RelayerClient = RelayerClient;

