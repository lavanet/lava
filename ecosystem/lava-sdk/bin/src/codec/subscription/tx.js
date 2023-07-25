"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.MsgClientImpl = exports.MsgDelProjectResponse = exports.MsgDelProject = exports.MsgAddProjectResponse = exports.MsgAddProject = exports.MsgBuyResponse = exports.MsgBuy = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const project_1 = require("../projects/project");
exports.protobufPackage = "lavanet.lava.subscription";
function createBaseMsgBuy() {
    return { creator: "", consumer: "", index: "", duration: long_1.default.UZERO };
}
exports.MsgBuy = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.consumer !== "") {
            writer.uint32(18).string(message.consumer);
        }
        if (message.index !== "") {
            writer.uint32(26).string(message.index);
        }
        if (!message.duration.isZero()) {
            writer.uint32(32).uint64(message.duration);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgBuy();
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
                    message.consumer = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.index = reader.string();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.duration = reader.uint64();
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
            consumer: isSet(object.consumer) ? String(object.consumer) : "",
            index: isSet(object.index) ? String(object.index) : "",
            duration: isSet(object.duration) ? long_1.default.fromValue(object.duration) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.consumer !== undefined && (obj.consumer = message.consumer);
        message.index !== undefined && (obj.index = message.index);
        message.duration !== undefined && (obj.duration = (message.duration || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.MsgBuy.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgBuy();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.consumer = (_b = object.consumer) !== null && _b !== void 0 ? _b : "";
        message.index = (_c = object.index) !== null && _c !== void 0 ? _c : "";
        message.duration = (object.duration !== undefined && object.duration !== null)
            ? long_1.default.fromValue(object.duration)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseMsgBuyResponse() {
    return {};
}
exports.MsgBuyResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgBuyResponse();
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
        return exports.MsgBuyResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgBuyResponse();
        return message;
    },
};
function createBaseMsgAddProject() {
    return { creator: "", projectData: undefined };
}
exports.MsgAddProject = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.projectData !== undefined) {
            project_1.ProjectData.encode(message.projectData, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgAddProject();
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
                    message.projectData = project_1.ProjectData.decode(reader, reader.uint32());
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
            projectData: isSet(object.projectData) ? project_1.ProjectData.fromJSON(object.projectData) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.projectData !== undefined &&
            (obj.projectData = message.projectData ? project_1.ProjectData.toJSON(message.projectData) : undefined);
        return obj;
    },
    create(base) {
        return exports.MsgAddProject.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseMsgAddProject();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.projectData = (object.projectData !== undefined && object.projectData !== null)
            ? project_1.ProjectData.fromPartial(object.projectData)
            : undefined;
        return message;
    },
};
function createBaseMsgAddProjectResponse() {
    return {};
}
exports.MsgAddProjectResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgAddProjectResponse();
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
        return exports.MsgAddProjectResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgAddProjectResponse();
        return message;
    },
};
function createBaseMsgDelProject() {
    return { creator: "", name: "" };
}
exports.MsgDelProject = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.name !== "") {
            writer.uint32(18).string(message.name);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgDelProject();
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
                    message.name = reader.string();
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
            name: isSet(object.name) ? String(object.name) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.name !== undefined && (obj.name = message.name);
        return obj;
    },
    create(base) {
        return exports.MsgDelProject.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMsgDelProject();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.name = (_b = object.name) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseMsgDelProjectResponse() {
    return {};
}
exports.MsgDelProjectResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgDelProjectResponse();
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
        return exports.MsgDelProjectResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgDelProjectResponse();
        return message;
    },
};
class MsgClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.subscription.Msg";
        this.rpc = rpc;
        this.Buy = this.Buy.bind(this);
        this.AddProject = this.AddProject.bind(this);
        this.DelProject = this.DelProject.bind(this);
    }
    Buy(request) {
        const data = exports.MsgBuy.encode(request).finish();
        const promise = this.rpc.request(this.service, "Buy", data);
        return promise.then((data) => exports.MsgBuyResponse.decode(minimal_1.default.Reader.create(data)));
    }
    AddProject(request) {
        const data = exports.MsgAddProject.encode(request).finish();
        const promise = this.rpc.request(this.service, "AddProject", data);
        return promise.then((data) => exports.MsgAddProjectResponse.decode(minimal_1.default.Reader.create(data)));
    }
    DelProject(request) {
        const data = exports.MsgDelProject.encode(request).finish();
        const promise = this.rpc.request(this.service, "DelProject", data);
        return promise.then((data) => exports.MsgDelProjectResponse.decode(minimal_1.default.Reader.create(data)));
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
