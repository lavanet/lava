"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.ListInfoStruct = exports.QueryListResponse = exports.QueryListRequest = exports.QueryListProjectsResponse = exports.QueryListProjectsRequest = exports.QueryCurrentResponse = exports.QueryCurrentRequest = exports.QueryParamsResponse = exports.QueryParamsRequest = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const params_1 = require("./params");
const subscription_1 = require("./subscription");
exports.protobufPackage = "lavanet.lava.subscription";
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
function createBaseQueryCurrentRequest() {
    return { consumer: "" };
}
exports.QueryCurrentRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.consumer !== "") {
            writer.uint32(10).string(message.consumer);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryCurrentRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.consumer = reader.string();
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
        return { consumer: isSet(object.consumer) ? String(object.consumer) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.consumer !== undefined && (obj.consumer = message.consumer);
        return obj;
    },
    create(base) {
        return exports.QueryCurrentRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryCurrentRequest();
        message.consumer = (_a = object.consumer) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryCurrentResponse() {
    return { sub: undefined };
}
exports.QueryCurrentResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.sub !== undefined) {
            subscription_1.Subscription.encode(message.sub, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryCurrentResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.sub = subscription_1.Subscription.decode(reader, reader.uint32());
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
        return { sub: isSet(object.sub) ? subscription_1.Subscription.fromJSON(object.sub) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.sub !== undefined && (obj.sub = message.sub ? subscription_1.Subscription.toJSON(message.sub) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryCurrentResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryCurrentResponse();
        message.sub = (object.sub !== undefined && object.sub !== null) ? subscription_1.Subscription.fromPartial(object.sub) : undefined;
        return message;
    },
};
function createBaseQueryListProjectsRequest() {
    return { subscription: "" };
}
exports.QueryListProjectsRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.subscription !== "") {
            writer.uint32(10).string(message.subscription);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryListProjectsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.subscription = reader.string();
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
        return { subscription: isSet(object.subscription) ? String(object.subscription) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.subscription !== undefined && (obj.subscription = message.subscription);
        return obj;
    },
    create(base) {
        return exports.QueryListProjectsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryListProjectsRequest();
        message.subscription = (_a = object.subscription) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryListProjectsResponse() {
    return { projects: [] };
}
exports.QueryListProjectsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.projects) {
            writer.uint32(10).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryListProjectsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.projects.push(reader.string());
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
        return { projects: Array.isArray(object === null || object === void 0 ? void 0 : object.projects) ? object.projects.map((e) => String(e)) : [] };
    },
    toJSON(message) {
        const obj = {};
        if (message.projects) {
            obj.projects = message.projects.map((e) => e);
        }
        else {
            obj.projects = [];
        }
        return obj;
    },
    create(base) {
        return exports.QueryListProjectsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryListProjectsResponse();
        message.projects = ((_a = object.projects) === null || _a === void 0 ? void 0 : _a.map((e) => e)) || [];
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
    return { subsInfo: [] };
}
exports.QueryListResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.subsInfo) {
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
                    message.subsInfo.push(exports.ListInfoStruct.decode(reader, reader.uint32()));
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
            subsInfo: Array.isArray(object === null || object === void 0 ? void 0 : object.subsInfo) ? object.subsInfo.map((e) => exports.ListInfoStruct.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.subsInfo) {
            obj.subsInfo = message.subsInfo.map((e) => e ? exports.ListInfoStruct.toJSON(e) : undefined);
        }
        else {
            obj.subsInfo = [];
        }
        return obj;
    },
    create(base) {
        return exports.QueryListResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryListResponse();
        message.subsInfo = ((_a = object.subsInfo) === null || _a === void 0 ? void 0 : _a.map((e) => exports.ListInfoStruct.fromPartial(e))) || [];
        return message;
    },
};
function createBaseListInfoStruct() {
    return {
        consumer: "",
        plan: "",
        durationTotal: long_1.default.UZERO,
        durationLeft: long_1.default.UZERO,
        monthExpiry: long_1.default.UZERO,
        monthCuTotal: long_1.default.UZERO,
        monthCuLeft: long_1.default.UZERO,
    };
}
exports.ListInfoStruct = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.consumer !== "") {
            writer.uint32(10).string(message.consumer);
        }
        if (message.plan !== "") {
            writer.uint32(18).string(message.plan);
        }
        if (!message.durationTotal.isZero()) {
            writer.uint32(24).uint64(message.durationTotal);
        }
        if (!message.durationLeft.isZero()) {
            writer.uint32(32).uint64(message.durationLeft);
        }
        if (!message.monthExpiry.isZero()) {
            writer.uint32(40).uint64(message.monthExpiry);
        }
        if (!message.monthCuTotal.isZero()) {
            writer.uint32(48).uint64(message.monthCuTotal);
        }
        if (!message.monthCuLeft.isZero()) {
            writer.uint32(56).uint64(message.monthCuLeft);
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
                    message.consumer = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.plan = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.durationTotal = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.durationLeft = reader.uint64();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.monthExpiry = reader.uint64();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.monthCuTotal = reader.uint64();
                    continue;
                case 7:
                    if (tag != 56) {
                        break;
                    }
                    message.monthCuLeft = reader.uint64();
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
            consumer: isSet(object.consumer) ? String(object.consumer) : "",
            plan: isSet(object.plan) ? String(object.plan) : "",
            durationTotal: isSet(object.durationTotal) ? long_1.default.fromValue(object.durationTotal) : long_1.default.UZERO,
            durationLeft: isSet(object.durationLeft) ? long_1.default.fromValue(object.durationLeft) : long_1.default.UZERO,
            monthExpiry: isSet(object.monthExpiry) ? long_1.default.fromValue(object.monthExpiry) : long_1.default.UZERO,
            monthCuTotal: isSet(object.monthCuTotal) ? long_1.default.fromValue(object.monthCuTotal) : long_1.default.UZERO,
            monthCuLeft: isSet(object.monthCuLeft) ? long_1.default.fromValue(object.monthCuLeft) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.consumer !== undefined && (obj.consumer = message.consumer);
        message.plan !== undefined && (obj.plan = message.plan);
        message.durationTotal !== undefined && (obj.durationTotal = (message.durationTotal || long_1.default.UZERO).toString());
        message.durationLeft !== undefined && (obj.durationLeft = (message.durationLeft || long_1.default.UZERO).toString());
        message.monthExpiry !== undefined && (obj.monthExpiry = (message.monthExpiry || long_1.default.UZERO).toString());
        message.monthCuTotal !== undefined && (obj.monthCuTotal = (message.monthCuTotal || long_1.default.UZERO).toString());
        message.monthCuLeft !== undefined && (obj.monthCuLeft = (message.monthCuLeft || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.ListInfoStruct.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseListInfoStruct();
        message.consumer = (_a = object.consumer) !== null && _a !== void 0 ? _a : "";
        message.plan = (_b = object.plan) !== null && _b !== void 0 ? _b : "";
        message.durationTotal = (object.durationTotal !== undefined && object.durationTotal !== null)
            ? long_1.default.fromValue(object.durationTotal)
            : long_1.default.UZERO;
        message.durationLeft = (object.durationLeft !== undefined && object.durationLeft !== null)
            ? long_1.default.fromValue(object.durationLeft)
            : long_1.default.UZERO;
        message.monthExpiry = (object.monthExpiry !== undefined && object.monthExpiry !== null)
            ? long_1.default.fromValue(object.monthExpiry)
            : long_1.default.UZERO;
        message.monthCuTotal = (object.monthCuTotal !== undefined && object.monthCuTotal !== null)
            ? long_1.default.fromValue(object.monthCuTotal)
            : long_1.default.UZERO;
        message.monthCuLeft = (object.monthCuLeft !== undefined && object.monthCuLeft !== null)
            ? long_1.default.fromValue(object.monthCuLeft)
            : long_1.default.UZERO;
        return message;
    },
};
class QueryClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.subscription.Query";
        this.rpc = rpc;
        this.Params = this.Params.bind(this);
        this.Current = this.Current.bind(this);
        this.ListProjects = this.ListProjects.bind(this);
        this.List = this.List.bind(this);
    }
    Params(request) {
        const data = exports.QueryParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Params", data);
        return promise.then((data) => exports.QueryParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    Current(request) {
        const data = exports.QueryCurrentRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Current", data);
        return promise.then((data) => exports.QueryCurrentResponse.decode(minimal_1.default.Reader.create(data)));
    }
    ListProjects(request) {
        const data = exports.QueryListProjectsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "ListProjects", data);
        return promise.then((data) => exports.QueryListProjectsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    List(request) {
        const data = exports.QueryListRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "List", data);
        return promise.then((data) => exports.QueryListResponse.decode(minimal_1.default.Reader.create(data)));
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
