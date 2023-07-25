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
exports.protobufPackage = "lavanet.lava.subscription";
function createBaseGenesisState() {
    return { params: undefined, subsFS: [], subsTS: [] };
}
exports.GenesisState = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.subsFS) {
            fixationEntry_1.RawMessage.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.subsTS) {
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
                    message.subsFS.push(fixationEntry_1.RawMessage.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.subsTS.push(fixationEntry_1.RawMessage.decode(reader, reader.uint32()));
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
            subsFS: Array.isArray(object === null || object === void 0 ? void 0 : object.subsFS) ? object.subsFS.map((e) => fixationEntry_1.RawMessage.fromJSON(e)) : [],
            subsTS: Array.isArray(object === null || object === void 0 ? void 0 : object.subsTS) ? object.subsTS.map((e) => fixationEntry_1.RawMessage.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        if (message.subsFS) {
            obj.subsFS = message.subsFS.map((e) => e ? fixationEntry_1.RawMessage.toJSON(e) : undefined);
        }
        else {
            obj.subsFS = [];
        }
        if (message.subsTS) {
            obj.subsTS = message.subsTS.map((e) => e ? fixationEntry_1.RawMessage.toJSON(e) : undefined);
        }
        else {
            obj.subsTS = [];
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
        message.subsFS = ((_a = object.subsFS) === null || _a === void 0 ? void 0 : _a.map((e) => fixationEntry_1.RawMessage.fromPartial(e))) || [];
        message.subsTS = ((_b = object.subsTS) === null || _b === void 0 ? void 0 : _b.map((e) => fixationEntry_1.RawMessage.fromPartial(e))) || [];
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
