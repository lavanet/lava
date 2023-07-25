"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Endpoint = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseEndpoint() {
    return { iPPORT: "", geolocation: long_1.default.UZERO, addons: [], apiInterfaces: [] };
}
exports.Endpoint = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.iPPORT !== "") {
            writer.uint32(10).string(message.iPPORT);
        }
        if (!message.geolocation.isZero()) {
            writer.uint32(24).uint64(message.geolocation);
        }
        for (const v of message.addons) {
            writer.uint32(34).string(v);
        }
        for (const v of message.apiInterfaces) {
            writer.uint32(42).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEndpoint();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.iPPORT = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.geolocation = reader.uint64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.addons.push(reader.string());
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.apiInterfaces.push(reader.string());
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
            iPPORT: isSet(object.iPPORT) ? String(object.iPPORT) : "",
            geolocation: isSet(object.geolocation) ? long_1.default.fromValue(object.geolocation) : long_1.default.UZERO,
            addons: Array.isArray(object === null || object === void 0 ? void 0 : object.addons) ? object.addons.map((e) => String(e)) : [],
            apiInterfaces: Array.isArray(object === null || object === void 0 ? void 0 : object.apiInterfaces) ? object.apiInterfaces.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.iPPORT !== undefined && (obj.iPPORT = message.iPPORT);
        message.geolocation !== undefined && (obj.geolocation = (message.geolocation || long_1.default.UZERO).toString());
        if (message.addons) {
            obj.addons = message.addons.map((e) => e);
        }
        else {
            obj.addons = [];
        }
        if (message.apiInterfaces) {
            obj.apiInterfaces = message.apiInterfaces.map((e) => e);
        }
        else {
            obj.apiInterfaces = [];
        }
        return obj;
    },
    create(base) {
        return exports.Endpoint.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseEndpoint();
        message.iPPORT = (_a = object.iPPORT) !== null && _a !== void 0 ? _a : "";
        message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
            ? long_1.default.fromValue(object.geolocation)
            : long_1.default.UZERO;
        message.addons = ((_b = object.addons) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        message.apiInterfaces = ((_c = object.apiInterfaces) === null || _c === void 0 ? void 0 : _c.map((e) => e)) || [];
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
