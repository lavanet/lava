"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProjectData = exports.ProtoDeveloperData = exports.ChainPolicy = exports.Policy = exports.ProjectKey = exports.Project = exports.projectKey_TypeToJSON = exports.projectKey_TypeFromJSON = exports.ProjectKey_Type = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.projects";
var ProjectKey_Type;
(function (ProjectKey_Type) {
    ProjectKey_Type[ProjectKey_Type["NONE"] = 0] = "NONE";
    ProjectKey_Type[ProjectKey_Type["ADMIN"] = 1] = "ADMIN";
    ProjectKey_Type[ProjectKey_Type["DEVELOPER"] = 2] = "DEVELOPER";
    ProjectKey_Type[ProjectKey_Type["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(ProjectKey_Type = exports.ProjectKey_Type || (exports.ProjectKey_Type = {}));
function projectKey_TypeFromJSON(object) {
    switch (object) {
        case 0:
        case "NONE":
            return ProjectKey_Type.NONE;
        case 1:
        case "ADMIN":
            return ProjectKey_Type.ADMIN;
        case 2:
        case "DEVELOPER":
            return ProjectKey_Type.DEVELOPER;
        case -1:
        case "UNRECOGNIZED":
        default:
            return ProjectKey_Type.UNRECOGNIZED;
    }
}
exports.projectKey_TypeFromJSON = projectKey_TypeFromJSON;
function projectKey_TypeToJSON(object) {
    switch (object) {
        case ProjectKey_Type.NONE:
            return "NONE";
        case ProjectKey_Type.ADMIN:
            return "ADMIN";
        case ProjectKey_Type.DEVELOPER:
            return "DEVELOPER";
        case ProjectKey_Type.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.projectKey_TypeToJSON = projectKey_TypeToJSON;
function createBaseProject() {
    return {
        index: "",
        subscription: "",
        description: "",
        enabled: false,
        projectKeys: [],
        adminPolicy: undefined,
        usedCu: long_1.default.UZERO,
        subscriptionPolicy: undefined,
        snapshot: long_1.default.UZERO,
    };
}
exports.Project = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        if (message.subscription !== "") {
            writer.uint32(18).string(message.subscription);
        }
        if (message.description !== "") {
            writer.uint32(26).string(message.description);
        }
        if (message.enabled === true) {
            writer.uint32(32).bool(message.enabled);
        }
        for (const v of message.projectKeys) {
            exports.ProjectKey.encode(v, writer.uint32(42).fork()).ldelim();
        }
        if (message.adminPolicy !== undefined) {
            exports.Policy.encode(message.adminPolicy, writer.uint32(50).fork()).ldelim();
        }
        if (!message.usedCu.isZero()) {
            writer.uint32(56).uint64(message.usedCu);
        }
        if (message.subscriptionPolicy !== undefined) {
            exports.Policy.encode(message.subscriptionPolicy, writer.uint32(66).fork()).ldelim();
        }
        if (!message.snapshot.isZero()) {
            writer.uint32(72).uint64(message.snapshot);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseProject();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.subscription = reader.string();
                    continue;
                case 3:
                    if (tag !== 26) {
                        break;
                    }
                    message.description = reader.string();
                    continue;
                case 4:
                    if (tag !== 32) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 5:
                    if (tag !== 42) {
                        break;
                    }
                    message.projectKeys.push(exports.ProjectKey.decode(reader, reader.uint32()));
                    continue;
                case 6:
                    if (tag !== 50) {
                        break;
                    }
                    message.adminPolicy = exports.Policy.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag !== 56) {
                        break;
                    }
                    message.usedCu = reader.uint64();
                    continue;
                case 8:
                    if (tag !== 66) {
                        break;
                    }
                    message.subscriptionPolicy = exports.Policy.decode(reader, reader.uint32());
                    continue;
                case 9:
                    if (tag !== 72) {
                        break;
                    }
                    message.snapshot = reader.uint64();
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            index: isSet(object.index) ? String(object.index) : "",
            subscription: isSet(object.subscription) ? String(object.subscription) : "",
            description: isSet(object.description) ? String(object.description) : "",
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            projectKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.projectKeys) ? object.projectKeys.map((e) => exports.ProjectKey.fromJSON(e)) : [],
            adminPolicy: isSet(object.adminPolicy) ? exports.Policy.fromJSON(object.adminPolicy) : undefined,
            usedCu: isSet(object.usedCu) ? long_1.default.fromValue(object.usedCu) : long_1.default.UZERO,
            subscriptionPolicy: isSet(object.subscriptionPolicy) ? exports.Policy.fromJSON(object.subscriptionPolicy) : undefined,
            snapshot: isSet(object.snapshot) ? long_1.default.fromValue(object.snapshot) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.subscription !== undefined && (obj.subscription = message.subscription);
        message.description !== undefined && (obj.description = message.description);
        message.enabled !== undefined && (obj.enabled = message.enabled);
        if (message.projectKeys) {
            obj.projectKeys = message.projectKeys.map((e) => e ? exports.ProjectKey.toJSON(e) : undefined);
        }
        else {
            obj.projectKeys = [];
        }
        message.adminPolicy !== undefined &&
            (obj.adminPolicy = message.adminPolicy ? exports.Policy.toJSON(message.adminPolicy) : undefined);
        message.usedCu !== undefined && (obj.usedCu = (message.usedCu || long_1.default.UZERO).toString());
        message.subscriptionPolicy !== undefined &&
            (obj.subscriptionPolicy = message.subscriptionPolicy ? exports.Policy.toJSON(message.subscriptionPolicy) : undefined);
        message.snapshot !== undefined && (obj.snapshot = (message.snapshot || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Project.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseProject();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.subscription = (_b = object.subscription) !== null && _b !== void 0 ? _b : "";
        message.description = (_c = object.description) !== null && _c !== void 0 ? _c : "";
        message.enabled = (_d = object.enabled) !== null && _d !== void 0 ? _d : false;
        message.projectKeys = ((_e = object.projectKeys) === null || _e === void 0 ? void 0 : _e.map((e) => exports.ProjectKey.fromPartial(e))) || [];
        message.adminPolicy = (object.adminPolicy !== undefined && object.adminPolicy !== null)
            ? exports.Policy.fromPartial(object.adminPolicy)
            : undefined;
        message.usedCu = (object.usedCu !== undefined && object.usedCu !== null)
            ? long_1.default.fromValue(object.usedCu)
            : long_1.default.UZERO;
        message.subscriptionPolicy = (object.subscriptionPolicy !== undefined && object.subscriptionPolicy !== null)
            ? exports.Policy.fromPartial(object.subscriptionPolicy)
            : undefined;
        message.snapshot = (object.snapshot !== undefined && object.snapshot !== null)
            ? long_1.default.fromValue(object.snapshot)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseProjectKey() {
    return { key: "", kinds: 0 };
}
exports.ProjectKey = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.key !== "") {
            writer.uint32(10).string(message.key);
        }
        if (message.kinds !== 0) {
            writer.uint32(32).uint32(message.kinds);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseProjectKey();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.key = reader.string();
                    continue;
                case 4:
                    if (tag !== 32) {
                        break;
                    }
                    message.kinds = reader.uint32();
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { key: isSet(object.key) ? String(object.key) : "", kinds: isSet(object.kinds) ? Number(object.kinds) : 0 };
    },
    toJSON(message) {
        const obj = {};
        message.key !== undefined && (obj.key = message.key);
        message.kinds !== undefined && (obj.kinds = Math.round(message.kinds));
        return obj;
    },
    create(base) {
        return exports.ProjectKey.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseProjectKey();
        message.key = (_a = object.key) !== null && _a !== void 0 ? _a : "";
        message.kinds = (_b = object.kinds) !== null && _b !== void 0 ? _b : 0;
        return message;
    },
};
function createBasePolicy() {
    return {
        chainPolicies: [],
        geolocationProfile: long_1.default.UZERO,
        totalCuLimit: long_1.default.UZERO,
        epochCuLimit: long_1.default.UZERO,
        maxProvidersToPair: long_1.default.UZERO,
    };
}
exports.Policy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.chainPolicies) {
            exports.ChainPolicy.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (!message.geolocationProfile.isZero()) {
            writer.uint32(16).uint64(message.geolocationProfile);
        }
        if (!message.totalCuLimit.isZero()) {
            writer.uint32(24).uint64(message.totalCuLimit);
        }
        if (!message.epochCuLimit.isZero()) {
            writer.uint32(32).uint64(message.epochCuLimit);
        }
        if (!message.maxProvidersToPair.isZero()) {
            writer.uint32(40).uint64(message.maxProvidersToPair);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePolicy();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.chainPolicies.push(exports.ChainPolicy.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag !== 16) {
                        break;
                    }
                    message.geolocationProfile = reader.uint64();
                    continue;
                case 3:
                    if (tag !== 24) {
                        break;
                    }
                    message.totalCuLimit = reader.uint64();
                    continue;
                case 4:
                    if (tag !== 32) {
                        break;
                    }
                    message.epochCuLimit = reader.uint64();
                    continue;
                case 5:
                    if (tag !== 40) {
                        break;
                    }
                    message.maxProvidersToPair = reader.uint64();
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            chainPolicies: Array.isArray(object === null || object === void 0 ? void 0 : object.chainPolicies)
                ? object.chainPolicies.map((e) => exports.ChainPolicy.fromJSON(e))
                : [],
            geolocationProfile: isSet(object.geolocationProfile) ? long_1.default.fromValue(object.geolocationProfile) : long_1.default.UZERO,
            totalCuLimit: isSet(object.totalCuLimit) ? long_1.default.fromValue(object.totalCuLimit) : long_1.default.UZERO,
            epochCuLimit: isSet(object.epochCuLimit) ? long_1.default.fromValue(object.epochCuLimit) : long_1.default.UZERO,
            maxProvidersToPair: isSet(object.maxProvidersToPair) ? long_1.default.fromValue(object.maxProvidersToPair) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.chainPolicies) {
            obj.chainPolicies = message.chainPolicies.map((e) => e ? exports.ChainPolicy.toJSON(e) : undefined);
        }
        else {
            obj.chainPolicies = [];
        }
        message.geolocationProfile !== undefined &&
            (obj.geolocationProfile = (message.geolocationProfile || long_1.default.UZERO).toString());
        message.totalCuLimit !== undefined && (obj.totalCuLimit = (message.totalCuLimit || long_1.default.UZERO).toString());
        message.epochCuLimit !== undefined && (obj.epochCuLimit = (message.epochCuLimit || long_1.default.UZERO).toString());
        message.maxProvidersToPair !== undefined &&
            (obj.maxProvidersToPair = (message.maxProvidersToPair || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Policy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBasePolicy();
        message.chainPolicies = ((_a = object.chainPolicies) === null || _a === void 0 ? void 0 : _a.map((e) => exports.ChainPolicy.fromPartial(e))) || [];
        message.geolocationProfile = (object.geolocationProfile !== undefined && object.geolocationProfile !== null)
            ? long_1.default.fromValue(object.geolocationProfile)
            : long_1.default.UZERO;
        message.totalCuLimit = (object.totalCuLimit !== undefined && object.totalCuLimit !== null)
            ? long_1.default.fromValue(object.totalCuLimit)
            : long_1.default.UZERO;
        message.epochCuLimit = (object.epochCuLimit !== undefined && object.epochCuLimit !== null)
            ? long_1.default.fromValue(object.epochCuLimit)
            : long_1.default.UZERO;
        message.maxProvidersToPair = (object.maxProvidersToPair !== undefined && object.maxProvidersToPair !== null)
            ? long_1.default.fromValue(object.maxProvidersToPair)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseChainPolicy() {
    return { chainId: "", apis: [] };
}
exports.ChainPolicy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chainId !== "") {
            writer.uint32(10).string(message.chainId);
        }
        for (const v of message.apis) {
            writer.uint32(18).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseChainPolicy();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.chainId = reader.string();
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.apis.push(reader.string());
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            chainId: isSet(object.chainId) ? String(object.chainId) : "",
            apis: Array.isArray(object === null || object === void 0 ? void 0 : object.apis) ? object.apis.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.chainId !== undefined && (obj.chainId = message.chainId);
        if (message.apis) {
            obj.apis = message.apis.map((e) => e);
        }
        else {
            obj.apis = [];
        }
        return obj;
    },
    create(base) {
        return exports.ChainPolicy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseChainPolicy();
        message.chainId = (_a = object.chainId) !== null && _a !== void 0 ? _a : "";
        message.apis = ((_b = object.apis) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        return message;
    },
};
function createBaseProtoDeveloperData() {
    return { projectID: "" };
}
exports.ProtoDeveloperData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.projectID !== "") {
            writer.uint32(10).string(message.projectID);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseProtoDeveloperData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.projectID = reader.string();
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { projectID: isSet(object.projectID) ? String(object.projectID) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.projectID !== undefined && (obj.projectID = message.projectID);
        return obj;
    },
    create(base) {
        return exports.ProtoDeveloperData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseProtoDeveloperData();
        message.projectID = (_a = object.projectID) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseProjectData() {
    return { name: "", description: "", enabled: false, projectKeys: [], policy: undefined };
}
exports.ProjectData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        if (message.enabled === true) {
            writer.uint32(24).bool(message.enabled);
        }
        for (const v of message.projectKeys) {
            exports.ProjectKey.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (message.policy !== undefined) {
            exports.Policy.encode(message.policy, writer.uint32(42).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseProjectData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.name = reader.string();
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.description = reader.string();
                    continue;
                case 3:
                    if (tag !== 24) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 4:
                    if (tag !== 34) {
                        break;
                    }
                    message.projectKeys.push(exports.ProjectKey.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag !== 42) {
                        break;
                    }
                    message.policy = exports.Policy.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            name: isSet(object.name) ? String(object.name) : "",
            description: isSet(object.description) ? String(object.description) : "",
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            projectKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.projectKeys) ? object.projectKeys.map((e) => exports.ProjectKey.fromJSON(e)) : [],
            policy: isSet(object.policy) ? exports.Policy.fromJSON(object.policy) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.description !== undefined && (obj.description = message.description);
        message.enabled !== undefined && (obj.enabled = message.enabled);
        if (message.projectKeys) {
            obj.projectKeys = message.projectKeys.map((e) => e ? exports.ProjectKey.toJSON(e) : undefined);
        }
        else {
            obj.projectKeys = [];
        }
        message.policy !== undefined && (obj.policy = message.policy ? exports.Policy.toJSON(message.policy) : undefined);
        return obj;
    },
    create(base) {
        return exports.ProjectData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseProjectData();
        message.name = (_a = object.name) !== null && _a !== void 0 ? _a : "";
        message.description = (_b = object.description) !== null && _b !== void 0 ? _b : "";
        message.enabled = (_c = object.enabled) !== null && _c !== void 0 ? _c : false;
        message.projectKeys = ((_d = object.projectKeys) === null || _d === void 0 ? void 0 : _d.map((e) => exports.ProjectKey.fromPartial(e))) || [];
        message.policy = (object.policy !== undefined && object.policy !== null)
            ? exports.Policy.fromPartial(object.policy)
            : undefined;
        return message;
    },
};
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
