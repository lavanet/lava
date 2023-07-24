"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.QueryInfoResponse = exports.QueryInfoRequest = exports.ListInfoStruct = exports.QueryListResponse = exports.QueryListRequest = exports.QueryParamsResponse = exports.QueryParamsRequest = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const coin_1 = require("../cosmos/base/v1beta1/coin");
const params_1 = require("./params");
const plan_1 = require("./plan");
exports.protobufPackage = "lavanet.lava.plans";
function createBaseQueryParamsRequest() {
    return {};
}
exports.QueryParamsRequest = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(_) {
        return {};
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    create(base) {
        return exports.QueryParamsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseQueryParamsRequest();
        return message;
    },
};
function createBaseQueryParamsResponse() {
    return { params: undefined };
}
exports.QueryParamsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.params = params_1.Params.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { params: isSet(object.params) ? params_1.Params.fromJSON(object.params) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryParamsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryParamsResponse();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        return message;
    },
};
function createBaseQueryListRequest() {
    return {};
}
exports.QueryListRequest = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryListRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(_) {
        return {};
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    create(base) {
        return exports.QueryListRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseQueryListRequest();
        return message;
    },
};
function createBaseQueryListResponse() {
    return { plansInfo: [] };
}
exports.QueryListResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.plansInfo) {
            exports.ListInfoStruct.encode(v, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryListResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.plansInfo.push(exports.ListInfoStruct.decode(reader, reader.uint32()));
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            plansInfo: Array.isArray(object === null || object === void 0 ? void 0 : object.plansInfo) ? object.plansInfo.map((e) => exports.ListInfoStruct.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.plansInfo) {
            obj.plansInfo = message.plansInfo.map((e) => e ? exports.ListInfoStruct.toJSON(e) : undefined);
        }
        else {
            obj.plansInfo = [];
        }
        return obj;
    },
    create(base) {
        return exports.QueryListResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryListResponse();
        message.plansInfo = ((_a = object.plansInfo) === null || _a === void 0 ? void 0 : _a.map((e) => exports.ListInfoStruct.fromPartial(e))) || [];
        return message;
    },
};
function createBaseListInfoStruct() {
    return { index: "", description: "", price: undefined };
}
exports.ListInfoStruct = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        if (message.price !== undefined) {
            coin_1.Coin.encode(message.price, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseListInfoStruct();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.description = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.price = coin_1.Coin.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            index: isSet(object.index) ? String(object.index) : "",
            description: isSet(object.description) ? String(object.description) : "",
            price: isSet(object.price) ? coin_1.Coin.fromJSON(object.price) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.description !== undefined && (obj.description = message.description);
        message.price !== undefined && (obj.price = message.price ? coin_1.Coin.toJSON(message.price) : undefined);
        return obj;
    },
    create(base) {
        return exports.ListInfoStruct.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseListInfoStruct();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.description = (_b = object.description) !== null && _b !== void 0 ? _b : "";
        message.price = (object.price !== undefined && object.price !== null) ? coin_1.Coin.fromPartial(object.price) : undefined;
        return message;
    },
};
function createBaseQueryInfoRequest() {
    return { planIndex: "" };
}
exports.QueryInfoRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.planIndex !== "") {
            writer.uint32(10).string(message.planIndex);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryInfoRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.planIndex = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { planIndex: isSet(object.planIndex) ? String(object.planIndex) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.planIndex !== undefined && (obj.planIndex = message.planIndex);
        return obj;
    },
    create(base) {
        return exports.QueryInfoRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryInfoRequest();
        message.planIndex = (_a = object.planIndex) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryInfoResponse() {
    return { planInfo: undefined };
}
exports.QueryInfoResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.planInfo !== undefined) {
            plan_1.Plan.encode(message.planInfo, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryInfoResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.planInfo = plan_1.Plan.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { planInfo: isSet(object.planInfo) ? plan_1.Plan.fromJSON(object.planInfo) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.planInfo !== undefined && (obj.planInfo = message.planInfo ? plan_1.Plan.toJSON(message.planInfo) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryInfoResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryInfoResponse();
        message.planInfo = (object.planInfo !== undefined && object.planInfo !== null)
            ? plan_1.Plan.fromPartial(object.planInfo)
            : undefined;
        return message;
    },
};
class QueryClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.plans.Query";
        this.rpc = rpc;
        this.Params = this.Params.bind(this);
        this.List = this.List.bind(this);
        this.Info = this.Info.bind(this);
    }
    Params(request) {
        const data = exports.QueryParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Params", data);
        return promise.then((data) => exports.QueryParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    List(request) {
        const data = exports.QueryListRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "List", data);
        return promise.then((data) => exports.QueryListResponse.decode(minimal_1.default.Reader.create(data)));
    }
    Info(request) {
        const data = exports.QueryInfoRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Info", data);
        return promise.then((data) => exports.QueryInfoResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.QueryClientImpl = QueryClientImpl;
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
