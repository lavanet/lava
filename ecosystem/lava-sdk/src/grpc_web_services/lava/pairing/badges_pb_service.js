// package: lava.pairing
// file: lava/pairing/badges.proto

var lava_pairing_badges_pb = require("../../lava/pairing/badges_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var BadgeGenerator = (function () {
  function BadgeGenerator() {}
  BadgeGenerator.serviceName = "lava.pairing.BadgeGenerator";
  return BadgeGenerator;
}());

BadgeGenerator.GenerateBadge = {
  methodName: "GenerateBadge",
  service: BadgeGenerator,
  requestStream: false,
  responseStream: false,
  requestType: lava_pairing_badges_pb.GenerateBadgeRequest,
  responseType: lava_pairing_badges_pb.GenerateBadgeResponse
};

exports.BadgeGenerator = BadgeGenerator;

function BadgeGeneratorClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

BadgeGeneratorClient.prototype.generateBadge = function generateBadge(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(BadgeGenerator.GenerateBadge, {
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

exports.BadgeGeneratorClient = BadgeGeneratorClient;

