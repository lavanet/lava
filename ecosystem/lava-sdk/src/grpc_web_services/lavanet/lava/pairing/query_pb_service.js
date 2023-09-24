// package: lavanet.lava.pairing
// file: lavanet/lava/pairing/query.proto

var lavanet_lava_pairing_query_pb = require("../../../lavanet/lava/pairing/query_pb");
var grpc = require("@improbable-eng/grpc-web").grpc;

var Query = (function () {
  function Query() {}
  Query.serviceName = "lavanet.lava.pairing.Query";
  return Query;
}());

Query.Params = {
  methodName: "Params",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryParamsRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryParamsResponse
};

Query.Providers = {
  methodName: "Providers",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryProvidersRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryProvidersResponse
};

Query.GetPairing = {
  methodName: "GetPairing",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryGetPairingRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryGetPairingResponse
};

Query.VerifyPairing = {
  methodName: "VerifyPairing",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryVerifyPairingRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryVerifyPairingResponse
};

Query.UniquePaymentStorageClientProvider = {
  methodName: "UniquePaymentStorageClientProvider",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryGetUniquePaymentStorageClientProviderResponse
};

Query.UniquePaymentStorageClientProviderAll = {
  methodName: "UniquePaymentStorageClientProviderAll",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryAllUniquePaymentStorageClientProviderResponse
};

Query.ProviderPaymentStorage = {
  methodName: "ProviderPaymentStorage",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryGetProviderPaymentStorageResponse
};

Query.ProviderPaymentStorageAll = {
  methodName: "ProviderPaymentStorageAll",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryAllProviderPaymentStorageResponse
};

Query.EpochPayments = {
  methodName: "EpochPayments",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryGetEpochPaymentsResponse
};

Query.EpochPaymentsAll = {
  methodName: "EpochPaymentsAll",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryAllEpochPaymentsResponse
};

Query.UserEntry = {
  methodName: "UserEntry",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryUserEntryRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryUserEntryResponse
};

Query.StaticProvidersList = {
  methodName: "StaticProvidersList",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryStaticProvidersListRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryStaticProvidersListResponse
};

Query.EffectivePolicy = {
  methodName: "EffectivePolicy",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryEffectivePolicyRequest,
  responseType: lavanet_lava_pairing_query_pb.QueryEffectivePolicyResponse
};

Query.SdkPairing = {
  methodName: "SdkPairing",
  service: Query,
  requestStream: false,
  responseStream: false,
  requestType: lavanet_lava_pairing_query_pb.QueryGetPairingRequest,
  responseType: lavanet_lava_pairing_query_pb.QuerySdkPairingResponse
};

exports.Query = Query;

function QueryClient(serviceHost, options) {
  this.serviceHost = serviceHost;
  this.options = options || {};
}

QueryClient.prototype.params = function params(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.Params, {
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

QueryClient.prototype.providers = function providers(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.Providers, {
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

QueryClient.prototype.getPairing = function getPairing(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.GetPairing, {
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

QueryClient.prototype.verifyPairing = function verifyPairing(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.VerifyPairing, {
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

QueryClient.prototype.uniquePaymentStorageClientProvider = function uniquePaymentStorageClientProvider(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.UniquePaymentStorageClientProvider, {
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

QueryClient.prototype.uniquePaymentStorageClientProviderAll = function uniquePaymentStorageClientProviderAll(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.UniquePaymentStorageClientProviderAll, {
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

QueryClient.prototype.providerPaymentStorage = function providerPaymentStorage(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.ProviderPaymentStorage, {
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

QueryClient.prototype.providerPaymentStorageAll = function providerPaymentStorageAll(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.ProviderPaymentStorageAll, {
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

QueryClient.prototype.epochPayments = function epochPayments(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.EpochPayments, {
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

QueryClient.prototype.epochPaymentsAll = function epochPaymentsAll(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.EpochPaymentsAll, {
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

QueryClient.prototype.userEntry = function userEntry(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.UserEntry, {
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

QueryClient.prototype.staticProvidersList = function staticProvidersList(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.StaticProvidersList, {
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

QueryClient.prototype.effectivePolicy = function effectivePolicy(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.EffectivePolicy, {
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

QueryClient.prototype.sdkPairing = function sdkPairing(requestMessage, metadata, callback) {
  if (arguments.length === 2) {
    callback = arguments[1];
  }
  var client = grpc.unary(Query.SdkPairing, {
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

