"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.GenesisState = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const params_1 = require("./params");
const spec_1 = require("./spec");
exports.protobufPackage = "lavanet.lava.spec";
function createBaseGenesisState() {
    return { params: undefined, specList: [], specCount: long_1.default.UZERO };
}
exports.GenesisState = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.specList) {
            spec_1.Spec.encode(v, writer.uint32(18).fork()).ldelim();
        }
        if (!message.specCount.isZero()) {
            writer.uint32(24).uint64(message.specCount);
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
                    message.specList.push(spec_1.Spec.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.specCount = reader.uint64();
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
            specList: Array.isArray(object === null || object === void 0 ? void 0 : object.specList) ? object.specList.map((e) => spec_1.Spec.fromJSON(e)) : [],
            specCount: isSet(object.specCount) ? long_1.default.fromValue(object.specCount) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        if (message.specList) {
            obj.specList = message.specList.map((e) => e ? spec_1.Spec.toJSON(e) : undefined);
        }
        else {
            obj.specList = [];
        }
        message.specCount !== undefined && (obj.specCount = (message.specCount || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.GenesisState.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseGenesisState();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        message.specList = ((_a = object.specList) === null || _a === void 0 ? void 0 : _a.map((e) => spec_1.Spec.fromPartial(e))) || [];
        message.specCount = (object.specCount !== undefined && object.specCount !== null)
            ? long_1.default.fromValue(object.specCount)
            : long_1.default.UZERO;
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
