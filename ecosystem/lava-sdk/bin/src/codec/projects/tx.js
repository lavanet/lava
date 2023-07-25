"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.MsgClientImpl = exports.MsgSetSubscriptionPolicyResponse = exports.MsgSetSubscriptionPolicy = exports.MsgSetPolicyResponse = exports.MsgSetPolicy = exports.MsgDelKeysResponse = exports.MsgDelKeys = exports.MsgAddKeysResponse = exports.MsgAddKeys = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const plan_1 = require("../plans/plan");
const project_1 = require("./project");
exports.protobufPackage = "lavanet.lava.projects";
function createBaseMsgAddKeys() {
    return { creator: "", project: "", projectKeys: [] };
}
exports.MsgAddKeys = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.project !== "") {
            writer.uint32(18).string(message.project);
        }
        for (const v of message.projectKeys) {
            project_1.ProjectKey.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgAddKeys();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.project = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.projectKeys.push(project_1.ProjectKey.decode(reader, reader.uint32()));
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            project: isSet(object.project) ? String(object.project) : "",
            projectKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.projectKeys) ? object.projectKeys.map((e) => project_1.ProjectKey.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.project !== undefined && (obj.project = message.project);
        if (message.projectKeys) {
            obj.projectKeys = message.projectKeys.map((e) => e ? project_1.ProjectKey.toJSON(e) : undefined);
        }
        else {
            obj.projectKeys = [];
        }
        return obj;
    },
    create(base) {
        return exports.MsgAddKeys.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgAddKeys();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.project = (_b = object.project) !== null && _b !== void 0 ? _b : "";
        message.projectKeys = ((_c = object.projectKeys) === null || _c === void 0 ? void 0 : _c.map((e) => project_1.ProjectKey.fromPartial(e))) || [];
        return message;
    },
};
function createBaseMsgAddKeysResponse() {
    return {};
}
exports.MsgAddKeysResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgAddKeysResponse();
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
        return exports.MsgAddKeysResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgAddKeysResponse();
        return message;
    },
};
function createBaseMsgDelKeys() {
    return { creator: "", project: "", projectKeys: [] };
}
exports.MsgDelKeys = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.project !== "") {
            writer.uint32(18).string(message.project);
        }
        for (const v of message.projectKeys) {
            project_1.ProjectKey.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgDelKeys();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.project = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.projectKeys.push(project_1.ProjectKey.decode(reader, reader.uint32()));
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            project: isSet(object.project) ? String(object.project) : "",
            projectKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.projectKeys) ? object.projectKeys.map((e) => project_1.ProjectKey.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.project !== undefined && (obj.project = message.project);
        if (message.projectKeys) {
            obj.projectKeys = message.projectKeys.map((e) => e ? project_1.ProjectKey.toJSON(e) : undefined);
        }
        else {
            obj.projectKeys = [];
        }
        return obj;
    },
    create(base) {
        return exports.MsgDelKeys.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgDelKeys();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.project = (_b = object.project) !== null && _b !== void 0 ? _b : "";
        message.projectKeys = ((_c = object.projectKeys) === null || _c === void 0 ? void 0 : _c.map((e) => project_1.ProjectKey.fromPartial(e))) || [];
        return message;
    },
};
function createBaseMsgDelKeysResponse() {
    return {};
}
exports.MsgDelKeysResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgDelKeysResponse();
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
        return exports.MsgDelKeysResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgDelKeysResponse();
        return message;
    },
};
function createBaseMsgSetPolicy() {
    return { creator: "", project: "", policy: undefined };
}
exports.MsgSetPolicy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.project !== "") {
            writer.uint32(18).string(message.project);
        }
        if (message.policy !== undefined) {
            plan_1.Policy.encode(message.policy, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgSetPolicy();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.project = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.policy = plan_1.Policy.decode(reader, reader.uint32());
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            project: isSet(object.project) ? String(object.project) : "",
            policy: isSet(object.policy) ? plan_1.Policy.fromJSON(object.policy) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.project !== undefined && (obj.project = message.project);
        message.policy !== undefined && (obj.policy = message.policy ? plan_1.Policy.toJSON(message.policy) : undefined);
        return obj;
    },
    create(base) {
        return exports.MsgSetPolicy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMsgSetPolicy();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.project = (_b = object.project) !== null && _b !== void 0 ? _b : "";
        message.policy = (object.policy !== undefined && object.policy !== null)
            ? plan_1.Policy.fromPartial(object.policy)
            : undefined;
        return message;
    },
};
function createBaseMsgSetPolicyResponse() {
    return {};
}
exports.MsgSetPolicyResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgSetPolicyResponse();
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
        return exports.MsgSetPolicyResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgSetPolicyResponse();
        return message;
    },
};
function createBaseMsgSetSubscriptionPolicy() {
    return { creator: "", projects: [], policy: undefined };
}
exports.MsgSetSubscriptionPolicy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        for (const v of message.projects) {
            writer.uint32(18).string(v);
        }
        if (message.policy !== undefined) {
            plan_1.Policy.encode(message.policy, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgSetSubscriptionPolicy();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.creator = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.projects.push(reader.string());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.policy = plan_1.Policy.decode(reader, reader.uint32());
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
            creator: isSet(object.creator) ? String(object.creator) : "",
            projects: Array.isArray(object === null || object === void 0 ? void 0 : object.projects) ? object.projects.map((e) => String(e)) : [],
            policy: isSet(object.policy) ? plan_1.Policy.fromJSON(object.policy) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        if (message.projects) {
            obj.projects = message.projects.map((e) => e);
        }
        else {
            obj.projects = [];
        }
        message.policy !== undefined && (obj.policy = message.policy ? plan_1.Policy.toJSON(message.policy) : undefined);
        return obj;
    },
    create(base) {
        return exports.MsgSetSubscriptionPolicy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMsgSetSubscriptionPolicy();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.projects = ((_b = object.projects) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        message.policy = (object.policy !== undefined && object.policy !== null)
            ? plan_1.Policy.fromPartial(object.policy)
            : undefined;
        return message;
    },
};
function createBaseMsgSetSubscriptionPolicyResponse() {
    return {};
}
exports.MsgSetSubscriptionPolicyResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgSetSubscriptionPolicyResponse();
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
        return exports.MsgSetSubscriptionPolicyResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgSetSubscriptionPolicyResponse();
        return message;
    },
};
class MsgClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.projects.Msg";
        this.rpc = rpc;
        this.AddKeys = this.AddKeys.bind(this);
        this.DelKeys = this.DelKeys.bind(this);
        this.SetPolicy = this.SetPolicy.bind(this);
        this.SetSubscriptionPolicy = this.SetSubscriptionPolicy.bind(this);
    }
    AddKeys(request) {
        const data = exports.MsgAddKeys.encode(request).finish();
        const promise = this.rpc.request(this.service, "AddKeys", data);
        return promise.then((data) => exports.MsgAddKeysResponse.decode(minimal_1.default.Reader.create(data)));
    }
    DelKeys(request) {
        const data = exports.MsgDelKeys.encode(request).finish();
        const promise = this.rpc.request(this.service, "DelKeys", data);
        return promise.then((data) => exports.MsgDelKeysResponse.decode(minimal_1.default.Reader.create(data)));
    }
    SetPolicy(request) {
        const data = exports.MsgSetPolicy.encode(request).finish();
        const promise = this.rpc.request(this.service, "SetPolicy", data);
        return promise.then((data) => exports.MsgSetPolicyResponse.decode(minimal_1.default.Reader.create(data)));
    }
    SetSubscriptionPolicy(request) {
        const data = exports.MsgSetSubscriptionPolicy.encode(request).finish();
        const promise = this.rpc.request(this.service, "SetSubscriptionPolicy", data);
        return promise.then((data) => exports.MsgSetSubscriptionPolicyResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.MsgClientImpl = MsgClientImpl;
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
