"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.GenesisState = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const fixationEntry_1 = require("../common/fixationEntry");
const params_1 = require("./params");
exports.protobufPackage = "lavanet.lava.projects";
function createBaseGenesisState() {
    return { params: undefined, projectsFS: [], developerFS: [] };
}
exports.GenesisState = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.projectsFS) {
            fixationEntry_1.RawMessage.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.developerFS) {
            fixationEntry_1.RawMessage.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseGenesisState();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.params = params_1.Params.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.projectsFS.push(fixationEntry_1.RawMessage.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.developerFS.push(fixationEntry_1.RawMessage.decode(reader, reader.uint32()));
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
            params: isSet(object.params) ? params_1.Params.fromJSON(object.params) : undefined,
            projectsFS: Array.isArray(object === null || object === void 0 ? void 0 : object.projectsFS) ? object.projectsFS.map((e) => fixationEntry_1.RawMessage.fromJSON(e)) : [],
            developerFS: Array.isArray(object === null || object === void 0 ? void 0 : object.developerFS) ? object.developerFS.map((e) => fixationEntry_1.RawMessage.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        if (message.projectsFS) {
            obj.projectsFS = message.projectsFS.map((e) => e ? fixationEntry_1.RawMessage.toJSON(e) : undefined);
        }
        else {
            obj.projectsFS = [];
        }
        if (message.developerFS) {
            obj.developerFS = message.developerFS.map((e) => e ? fixationEntry_1.RawMessage.toJSON(e) : undefined);
        }
        else {
            obj.developerFS = [];
        }
        return obj;
    },
    create(base) {
        return exports.GenesisState.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseGenesisState();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        message.projectsFS = ((_a = object.projectsFS) === null || _a === void 0 ? void 0 : _a.map((e) => fixationEntry_1.RawMessage.fromPartial(e))) || [];
        message.developerFS = ((_b = object.developerFS) === null || _b === void 0 ? void 0 : _b.map((e) => fixationEntry_1.RawMessage.fromPartial(e))) || [];
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
