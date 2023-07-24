"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.PlansDelProposal = exports.PlansAddProposal = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const plan_1 = require("./plan");
exports.protobufPackage = "lavanet.lava.plans";
function createBasePlansAddProposal() {
    return { title: "", description: "", plans: [] };
}
exports.PlansAddProposal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.title !== "") {
            writer.uint32(10).string(message.title);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        for (const v of message.plans) {
            plan_1.Plan.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePlansAddProposal();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.title = reader.string();
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
                    message.plans.push(plan_1.Plan.decode(reader, reader.uint32()));
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
            title: isSet(object.title) ? String(object.title) : "",
            description: isSet(object.description) ? String(object.description) : "",
            plans: Array.isArray(object === null || object === void 0 ? void 0 : object.plans) ? object.plans.map((e) => plan_1.Plan.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.title !== undefined && (obj.title = message.title);
        message.description !== undefined && (obj.description = message.description);
        if (message.plans) {
            obj.plans = message.plans.map((e) => e ? plan_1.Plan.toJSON(e) : undefined);
        }
        else {
            obj.plans = [];
        }
        return obj;
    },
    create(base) {
        return exports.PlansAddProposal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBasePlansAddProposal();
        message.title = (_a = object.title) !== null && _a !== void 0 ? _a : "";
        message.description = (_b = object.description) !== null && _b !== void 0 ? _b : "";
        message.plans = ((_c = object.plans) === null || _c === void 0 ? void 0 : _c.map((e) => plan_1.Plan.fromPartial(e))) || [];
        return message;
    },
};
function createBasePlansDelProposal() {
    return { title: "", description: "", plans: [] };
}
exports.PlansDelProposal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.title !== "") {
            writer.uint32(10).string(message.title);
        }
        if (message.description !== "") {
            writer.uint32(18).string(message.description);
        }
        for (const v of message.plans) {
            writer.uint32(26).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBasePlansDelProposal();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.title = reader.string();
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
                    message.plans.push(reader.string());
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
            title: isSet(object.title) ? String(object.title) : "",
            description: isSet(object.description) ? String(object.description) : "",
            plans: Array.isArray(object === null || object === void 0 ? void 0 : object.plans) ? object.plans.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.title !== undefined && (obj.title = message.title);
        message.description !== undefined && (obj.description = message.description);
        if (message.plans) {
            obj.plans = message.plans.map((e) => e);
        }
        else {
            obj.plans = [];
        }
        return obj;
    },
    create(base) {
        return exports.PlansDelProposal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBasePlansDelProposal();
        message.title = (_a = object.title) !== null && _a !== void 0 ? _a : "";
        message.description = (_b = object.description) !== null && _b !== void 0 ? _b : "";
        message.plans = ((_c = object.plans) === null || _c === void 0 ? void 0 : _c.map((e) => e)) || [];
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
