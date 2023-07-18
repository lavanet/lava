"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ProjectData = exports.ProtoDeveloperData = exports.ProjectKey = exports.Project = exports.projectKey_TypeToJSON = exports.projectKey_TypeFromJSON = exports.ProjectKey_Type = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const plan_1 = require("../plans/plan");
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
        if (message.enabled === true) {
            writer.uint32(32).bool(message.enabled);
        }
        for (const v of message.projectKeys) {
            exports.ProjectKey.encode(v, writer.uint32(42).fork()).ldelim();
        }
        if (message.adminPolicy !== undefined) {
            plan_1.Policy.encode(message.adminPolicy, writer.uint32(50).fork()).ldelim();
        }
        if (!message.usedCu.isZero()) {
            writer.uint32(56).uint64(message.usedCu);
        }
        if (message.subscriptionPolicy !== undefined) {
            plan_1.Policy.encode(message.subscriptionPolicy, writer.uint32(66).fork()).ldelim();
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
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.subscription = reader.string();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.projectKeys.push(exports.ProjectKey.decode(reader, reader.uint32()));
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.adminPolicy = plan_1.Policy.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag != 56) {
                        break;
                    }
                    message.usedCu = reader.uint64();
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.subscriptionPolicy = plan_1.Policy.decode(reader, reader.uint32());
                    continue;
                case 9:
                    if (tag != 72) {
                        break;
                    }
                    message.snapshot = reader.uint64();
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
            subscription: isSet(object.subscription) ? String(object.subscription) : "",
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            projectKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.projectKeys) ? object.projectKeys.map((e) => exports.ProjectKey.fromJSON(e)) : [],
            adminPolicy: isSet(object.adminPolicy) ? plan_1.Policy.fromJSON(object.adminPolicy) : undefined,
            usedCu: isSet(object.usedCu) ? long_1.default.fromValue(object.usedCu) : long_1.default.UZERO,
            subscriptionPolicy: isSet(object.subscriptionPolicy) ? plan_1.Policy.fromJSON(object.subscriptionPolicy) : undefined,
            snapshot: isSet(object.snapshot) ? long_1.default.fromValue(object.snapshot) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        message.subscription !== undefined && (obj.subscription = message.subscription);
        message.enabled !== undefined && (obj.enabled = message.enabled);
        if (message.projectKeys) {
            obj.projectKeys = message.projectKeys.map((e) => e ? exports.ProjectKey.toJSON(e) : undefined);
        }
        else {
            obj.projectKeys = [];
        }
        message.adminPolicy !== undefined &&
            (obj.adminPolicy = message.adminPolicy ? plan_1.Policy.toJSON(message.adminPolicy) : undefined);
        message.usedCu !== undefined && (obj.usedCu = (message.usedCu || long_1.default.UZERO).toString());
        message.subscriptionPolicy !== undefined &&
            (obj.subscriptionPolicy = message.subscriptionPolicy ? plan_1.Policy.toJSON(message.subscriptionPolicy) : undefined);
        message.snapshot !== undefined && (obj.snapshot = (message.snapshot || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Project.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseProject();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        message.subscription = (_b = object.subscription) !== null && _b !== void 0 ? _b : "";
        message.enabled = (_c = object.enabled) !== null && _c !== void 0 ? _c : false;
        message.projectKeys = ((_d = object.projectKeys) === null || _d === void 0 ? void 0 : _d.map((e) => exports.ProjectKey.fromPartial(e))) || [];
        message.adminPolicy = (object.adminPolicy !== undefined && object.adminPolicy !== null)
            ? plan_1.Policy.fromPartial(object.adminPolicy)
            : undefined;
        message.usedCu = (object.usedCu !== undefined && object.usedCu !== null)
            ? long_1.default.fromValue(object.usedCu)
            : long_1.default.UZERO;
        message.subscriptionPolicy = (object.subscriptionPolicy !== undefined && object.subscriptionPolicy !== null)
            ? plan_1.Policy.fromPartial(object.subscriptionPolicy)
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
                    if (tag != 10) {
                        break;
                    }
                    message.key = reader.string();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.kinds = reader.uint32();
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
                    if (tag != 10) {
                        break;
                    }
                    message.projectID = reader.string();
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
    return { name: "", enabled: false, projectKeys: [], policy: undefined };
}
exports.ProjectData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.enabled === true) {
            writer.uint32(24).bool(message.enabled);
        }
        for (const v of message.projectKeys) {
            exports.ProjectKey.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (message.policy !== undefined) {
            plan_1.Policy.encode(message.policy, writer.uint32(42).fork()).ldelim();
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
                    if (tag != 10) {
                        break;
                    }
                    message.name = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.projectKeys.push(exports.ProjectKey.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 42) {
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
            name: isSet(object.name) ? String(object.name) : "",
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            projectKeys: Array.isArray(object === null || object === void 0 ? void 0 : object.projectKeys) ? object.projectKeys.map((e) => exports.ProjectKey.fromJSON(e)) : [],
            policy: isSet(object.policy) ? plan_1.Policy.fromJSON(object.policy) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.enabled !== undefined && (obj.enabled = message.enabled);
        if (message.projectKeys) {
            obj.projectKeys = message.projectKeys.map((e) => e ? exports.ProjectKey.toJSON(e) : undefined);
        }
        else {
            obj.projectKeys = [];
        }
        message.policy !== undefined && (obj.policy = message.policy ? plan_1.Policy.toJSON(message.policy) : undefined);
        return obj;
    },
    create(base) {
        return exports.ProjectData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseProjectData();
        message.name = (_a = object.name) !== null && _a !== void 0 ? _a : "";
        message.enabled = (_b = object.enabled) !== null && _b !== void 0 ? _b : false;
        message.projectKeys = ((_c = object.projectKeys) === null || _c === void 0 ? void 0 : _c.map((e) => exports.ProjectKey.fromPartial(e))) || [];
        message.policy = (object.policy !== undefined && object.policy !== null)
            ? plan_1.Policy.fromPartial(object.policy)
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
