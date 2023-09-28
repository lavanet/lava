// package: lavanet.lava.downtime.v1
// file: lavanet/lava/downtime/v1/query.proto

var lavanet_lava_downtime_v1_query_pb = require("../../../../lavanet/lava/downtime/v1/query_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var Query = (function () {
  function Query() {}
  Query.serviceName = "lavanet.lava.downtime.v1.Query";
  return Query;
}());

Query.QueryParams = {
  methodName: "QueryParams",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_downtime_v1_query_pb.QueryParamsRequest,
  responseType: lavanet_lava_downtime_v1_query_pb.QueryParamsResponse
};

Query.QueryDowntime = {
  methodName: "QueryDowntime",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_downtime_v1_query_pb.QueryDowntimeRequest,
  responseType: lavanet_lava_downtime_v1_query_pb.QueryDowntimeResponse
};

exports.Query = Query;

function QueryClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

QueryClient.prototype.queryParams = function queryParams(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.QueryParams, {
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

QueryClient.prototype.queryDowntime = function queryDowntime(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.QueryDowntime, {
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

exports.QueryClient = QueryClient;

