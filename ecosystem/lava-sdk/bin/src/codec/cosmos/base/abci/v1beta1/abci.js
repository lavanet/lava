"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SearchTxsResult = exports.TxMsgData = exports.MsgData = exports.SimulationResponse = exports.Result = exports.GasInfo = exports.Attribute = exports.StringEvent = exports.ABCIMessageLog = exports.TxResponse = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const any_1 = require("../../../../google/protobuf/any");
const types_1 = require("../../../../tendermint/abci/types");
exports.protobufPackage = "cosmos.base.abci.v1beta1";
function createBaseTxResponse() {
    return {
        height: long_1.default.ZERO,
        txhash: "",
        codespace: "",
        code: 0,
        data: "",
        rawLog: "",
        logs: [],
        info: "",
        gasWanted: long_1.default.ZERO,
        gasUsed: long_1.default.ZERO,
        tx: undefined,
        timestamp: "",
        events: [],
    };
}
exports.TxResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.height.isZero()) {
            writer.uint32(8).int64(message.height);
        }
        if (message.txhash !== "") {
            writer.uint32(18).string(message.txhash);
        }
        if (message.codespace !== "") {
            writer.uint32(26).string(message.codespace);
        }
        if (message.code !== 0) {
            writer.uint32(32).uint32(message.code);
        }
        if (message.data !== "") {
            writer.uint32(42).string(message.data);
        }
        if (message.rawLog !== "") {
            writer.uint32(50).string(message.rawLog);
        }
        for (const v of message.logs) {
            exports.ABCIMessageLog.encode(v, writer.uint32(58).fork()).ldelim();
        }
        if (message.info !== "") {
            writer.uint32(66).string(message.info);
        }
        if (!message.gasWanted.isZero()) {
            writer.uint32(72).int64(message.gasWanted);
        }
        if (!message.gasUsed.isZero()) {
            writer.uint32(80).int64(message.gasUsed);
        }
        if (message.tx !== undefined) {
            any_1.Any.encode(message.tx, writer.uint32(90).fork()).ldelim();
        }
        if (message.timestamp !== "") {
            writer.uint32(98).string(message.timestamp);
        }
        for (const v of message.events) {
            types_1.Event.encode(v, writer.uint32(106).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseTxResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.txhash = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.codespace = reader.string();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.code = reader.uint32();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.data = reader.string();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.rawLog = reader.string();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.logs.push(exports.ABCIMessageLog.decode(reader, reader.uint32()));
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.info = reader.string();
                    continue;
                case 9:
                    if (tag != 72) {
                        break;
                    }
                    message.gasWanted = reader.int64();
                    continue;
                case 10:
                    if (tag != 80) {
                        break;
                    }
                    message.gasUsed = reader.int64();
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.tx = any_1.Any.decode(reader, reader.uint32());
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.timestamp = reader.string();
                    continue;
                case 13:
                    if (tag != 106) {
                        break;
                    }
                    message.events.push(types_1.Event.decode(reader, reader.uint32()));
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
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            txhash: isSet(object.txhash) ? String(object.txhash) : "",
            codespace: isSet(object.codespace) ? String(object.codespace) : "",
            code: isSet(object.code) ? Number(object.code) : 0,
            data: isSet(object.data) ? String(object.data) : "",
            rawLog: isSet(object.rawLog) ? String(object.rawLog) : "",
            logs: Array.isArray(object === null || object === void 0 ? void 0 : object.logs) ? object.logs.map((e) => exports.ABCIMessageLog.fromJSON(e)) : [],
            info: isSet(object.info) ? String(object.info) : "",
            gasWanted: isSet(object.gasWanted) ? long_1.default.fromValue(object.gasWanted) : long_1.default.ZERO,
            gasUsed: isSet(object.gasUsed) ? long_1.default.fromValue(object.gasUsed) : long_1.default.ZERO,
            tx: isSet(object.tx) ? any_1.Any.fromJSON(object.tx) : undefined,
            timestamp: isSet(object.timestamp) ? String(object.timestamp) : "",
            events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => types_1.Event.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.txhash !== undefined && (obj.txhash = message.txhash);
        message.codespace !== undefined && (obj.codespace = message.codespace);
        message.code !== undefined && (obj.code = Math.round(message.code));
        message.data !== undefined && (obj.data = message.data);
        message.rawLog !== undefined && (obj.rawLog = message.rawLog);
        if (message.logs) {
            obj.logs = message.logs.map((e) => e ? exports.ABCIMessageLog.toJSON(e) : undefined);
        }
        else {
            obj.logs = [];
        }
        message.info !== undefined && (obj.info = message.info);
        message.gasWanted !== undefined && (obj.gasWanted = (message.gasWanted || long_1.default.ZERO).toString());
        message.gasUsed !== undefined && (obj.gasUsed = (message.gasUsed || long_1.default.ZERO).toString());
        message.tx !== undefined && (obj.tx = message.tx ? any_1.Any.toJSON(message.tx) : undefined);
        message.timestamp !== undefined && (obj.timestamp = message.timestamp);
        if (message.events) {
            obj.events = message.events.map((e) => e ? types_1.Event.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        return obj;
    },
    create(base) {
        return exports.TxResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f, _g, _h, _j;
        const message = createBaseTxResponse();
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.txhash = (_a = object.txhash) !== null && _a !== void 0 ? _a : "";
        message.codespace = (_b = object.codespace) !== null && _b !== void 0 ? _b : "";
        message.code = (_c = object.code) !== null && _c !== void 0 ? _c : 0;
        message.data = (_d = object.data) !== null && _d !== void 0 ? _d : "";
        message.rawLog = (_e = object.rawLog) !== null && _e !== void 0 ? _e : "";
        message.logs = ((_f = object.logs) === null || _f === void 0 ? void 0 : _f.map((e) => exports.ABCIMessageLog.fromPartial(e))) || [];
        message.info = (_g = object.info) !== null && _g !== void 0 ? _g : "";
        message.gasWanted = (object.gasWanted !== undefined && object.gasWanted !== null)
            ? long_1.default.fromValue(object.gasWanted)
            : long_1.default.ZERO;
        message.gasUsed = (object.gasUsed !== undefined && object.gasUsed !== null)
            ? long_1.default.fromValue(object.gasUsed)
            : long_1.default.ZERO;
        message.tx = (object.tx !== undefined && object.tx !== null) ? any_1.Any.fromPartial(object.tx) : undefined;
        message.timestamp = (_h = object.timestamp) !== null && _h !== void 0 ? _h : "";
        message.events = ((_j = object.events) === null || _j === void 0 ? void 0 : _j.map((e) => types_1.Event.fromPartial(e))) || [];
        return message;
    },
};
function createBaseABCIMessageLog() {
    return { msgIndex: 0, log: "", events: [] };
}
exports.ABCIMessageLog = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.msgIndex !== 0) {
            writer.uint32(8).uint32(message.msgIndex);
        }
        if (message.log !== "") {
            writer.uint32(18).string(message.log);
        }
        for (const v of message.events) {
            exports.StringEvent.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseABCIMessageLog();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.msgIndex = reader.uint32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.log = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.events.push(exports.StringEvent.decode(reader, reader.uint32()));
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
            msgIndex: isSet(object.msgIndex) ? Number(object.msgIndex) : 0,
            log: isSet(object.log) ? String(object.log) : "",
            events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => exports.StringEvent.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.msgIndex !== undefined && (obj.msgIndex = Math.round(message.msgIndex));
        message.log !== undefined && (obj.log = message.log);
        if (message.events) {
            obj.events = message.events.map((e) => e ? exports.StringEvent.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        return obj;
    },
    create(base) {
        return exports.ABCIMessageLog.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseABCIMessageLog();
        message.msgIndex = (_a = object.msgIndex) !== null && _a !== void 0 ? _a : 0;
        message.log = (_b = object.log) !== null && _b !== void 0 ? _b : "";
        message.events = ((_c = object.events) === null || _c === void 0 ? void 0 : _c.map((e) => exports.StringEvent.fromPartial(e))) || [];
        return message;
    },
};
function createBaseStringEvent() {
    return { type: "", attributes: [] };
}
exports.StringEvent = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.type !== "") {
            writer.uint32(10).string(message.type);
        }
        for (const v of message.attributes) {
            exports.Attribute.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseStringEvent();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.type = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.attributes.push(exports.Attribute.decode(reader, reader.uint32()));
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
            type: isSet(object.type) ? String(object.type) : "",
            attributes: Array.isArray(object === null || object === void 0 ? void 0 : object.attributes) ? object.attributes.map((e) => exports.Attribute.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.type !== undefined && (obj.type = message.type);
        if (message.attributes) {
            obj.attributes = message.attributes.map((e) => e ? exports.Attribute.toJSON(e) : undefined);
        }
        else {
            obj.attributes = [];
        }
        return obj;
    },
    create(base) {
        return exports.StringEvent.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseStringEvent();
        message.type = (_a = object.type) !== null && _a !== void 0 ? _a : "";
        message.attributes = ((_b = object.attributes) === null || _b === void 0 ? void 0 : _b.map((e) => exports.Attribute.fromPartial(e))) || [];
        return message;
    },
};
function createBaseAttribute() {
    return { key: "", value: "" };
}
exports.Attribute = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.key !== "") {
            writer.uint32(10).string(message.key);
        }
        if (message.value !== "") {
            writer.uint32(18).string(message.value);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseAttribute();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.key = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.value = reader.string();
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
        return { key: isSet(object.key) ? String(object.key) : "", value: isSet(object.value) ? String(object.value) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.key !== undefined && (obj.key = message.key);
        message.value !== undefined && (obj.value = message.value);
        return obj;
    },
    create(base) {
        return exports.Attribute.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseAttribute();
        message.key = (_a = object.key) !== null && _a !== void 0 ? _a : "";
        message.value = (_b = object.value) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseGasInfo() {
    return { gasWanted: long_1.default.UZERO, gasUsed: long_1.default.UZERO };
}
exports.GasInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.gasWanted.isZero()) {
            writer.uint32(8).uint64(message.gasWanted);
        }
        if (!message.gasUsed.isZero()) {
            writer.uint32(16).uint64(message.gasUsed);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseGasInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.gasWanted = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.gasUsed = reader.uint64();
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
            gasWanted: isSet(object.gasWanted) ? long_1.default.fromValue(object.gasWanted) : long_1.default.UZERO,
            gasUsed: isSet(object.gasUsed) ? long_1.default.fromValue(object.gasUsed) : long_1.default.UZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.gasWanted !== undefined && (obj.gasWanted = (message.gasWanted || long_1.default.UZERO).toString());
        message.gasUsed !== undefined && (obj.gasUsed = (message.gasUsed || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.GasInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseGasInfo();
        message.gasWanted = (object.gasWanted !== undefined && object.gasWanted !== null)
            ? long_1.default.fromValue(object.gasWanted)
            : long_1.default.UZERO;
        message.gasUsed = (object.gasUsed !== undefined && object.gasUsed !== null)
            ? long_1.default.fromValue(object.gasUsed)
            : long_1.default.UZERO;
        return message;
    },
};
function createBaseResult() {
    return { data: new Uint8Array(), log: "", events: [], msgResponses: [] };
}
exports.Result = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.data.length !== 0) {
            writer.uint32(10).bytes(message.data);
        }
        if (message.log !== "") {
            writer.uint32(18).string(message.log);
        }
        for (const v of message.events) {
            types_1.Event.encode(v, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.msgResponses) {
            any_1.Any.encode(v, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResult();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.log = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.events.push(types_1.Event.decode(reader, reader.uint32()));
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.msgResponses.push(any_1.Any.decode(reader, reader.uint32()));
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
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            log: isSet(object.log) ? String(object.log) : "",
            events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => types_1.Event.fromJSON(e)) : [],
            msgResponses: Array.isArray(object === null || object === void 0 ? void 0 : object.msgResponses) ? object.msgResponses.map((e) => any_1.Any.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.log !== undefined && (obj.log = message.log);
        if (message.events) {
            obj.events = message.events.map((e) => e ? types_1.Event.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        if (message.msgResponses) {
            obj.msgResponses = message.msgResponses.map((e) => e ? any_1.Any.toJSON(e) : undefined);
        }
        else {
            obj.msgResponses = [];
        }
        return obj;
    },
    create(base) {
        return exports.Result.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseResult();
        message.data = (_a = object.data) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.log = (_b = object.log) !== null && _b !== void 0 ? _b : "";
        message.events = ((_c = object.events) === null || _c === void 0 ? void 0 : _c.map((e) => types_1.Event.fromPartial(e))) || [];
        message.msgResponses = ((_d = object.msgResponses) === null || _d === void 0 ? void 0 : _d.map((e) => any_1.Any.fromPartial(e))) || [];
        return message;
    },
};
function createBaseSimulationResponse() {
    return { gasInfo: undefined, result: undefined };
}
exports.SimulationResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.gasInfo !== undefined) {
            exports.GasInfo.encode(message.gasInfo, writer.uint32(10).fork()).ldelim();
        }
        if (message.result !== undefined) {
            exports.Result.encode(message.result, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseSimulationResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.gasInfo = exports.GasInfo.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.result = exports.Result.decode(reader, reader.uint32());
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
            gasInfo: isSet(object.gasInfo) ? exports.GasInfo.fromJSON(object.gasInfo) : undefined,
            result: isSet(object.result) ? exports.Result.fromJSON(object.result) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.gasInfo !== undefined && (obj.gasInfo = message.gasInfo ? exports.GasInfo.toJSON(message.gasInfo) : undefined);
        message.result !== undefined && (obj.result = message.result ? exports.Result.toJSON(message.result) : undefined);
        return obj;
    },
    create(base) {
        return exports.SimulationResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseSimulationResponse();
        message.gasInfo = (object.gasInfo !== undefined && object.gasInfo !== null)
            ? exports.GasInfo.fromPartial(object.gasInfo)
            : undefined;
        message.result = (object.result !== undefined && object.result !== null)
            ? exports.Result.fromPartial(object.result)
            : undefined;
        return message;
    },
};
function createBaseMsgData() {
    return { msgType: "", data: new Uint8Array() };
}
exports.MsgData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.msgType !== "") {
            writer.uint32(10).string(message.msgType);
        }
        if (message.data.length !== 0) {
            writer.uint32(18).bytes(message.data);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.msgType = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.data = reader.bytes();
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
            msgType: isSet(object.msgType) ? String(object.msgType) : "",
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.msgType !== undefined && (obj.msgType = message.msgType);
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.MsgData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMsgData();
        message.msgType = (_a = object.msgType) !== null && _a !== void 0 ? _a : "";
        message.data = (_b = object.data) !== null && _b !== void 0 ? _b : new Uint8Array();
        return message;
    },
};
function createBaseTxMsgData() {
    return { data: [], msgResponses: [] };
}
exports.TxMsgData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.data) {
            exports.MsgData.encode(v, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.msgResponses) {
            any_1.Any.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseTxMsgData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.data.push(exports.MsgData.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.msgResponses.push(any_1.Any.decode(reader, reader.uint32()));
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
            data: Array.isArray(object === null || object === void 0 ? void 0 : object.data) ? object.data.map((e) => exports.MsgData.fromJSON(e)) : [],
            msgResponses: Array.isArray(object === null || object === void 0 ? void 0 : object.msgResponses) ? object.msgResponses.map((e) => any_1.Any.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.data) {
            obj.data = message.data.map((e) => e ? exports.MsgData.toJSON(e) : undefined);
        }
        else {
            obj.data = [];
        }
        if (message.msgResponses) {
            obj.msgResponses = message.msgResponses.map((e) => e ? any_1.Any.toJSON(e) : undefined);
        }
        else {
            obj.msgResponses = [];
        }
        return obj;
    },
    create(base) {
        return exports.TxMsgData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseTxMsgData();
        message.data = ((_a = object.data) === null || _a === void 0 ? void 0 : _a.map((e) => exports.MsgData.fromPartial(e))) || [];
        message.msgResponses = ((_b = object.msgResponses) === null || _b === void 0 ? void 0 : _b.map((e) => any_1.Any.fromPartial(e))) || [];
        return message;
    },
};
function createBaseSearchTxsResult() {
    return {
        totalCount: long_1.default.UZERO,
        count: long_1.default.UZERO,
        pageNumber: long_1.default.UZERO,
        pageTotal: long_1.default.UZERO,
        limit: long_1.default.UZERO,
        txs: [],
    };
}
exports.SearchTxsResult = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.totalCount.isZero()) {
            writer.uint32(8).uint64(message.totalCount);
        }
        if (!message.count.isZero()) {
            writer.uint32(16).uint64(message.count);
        }
        if (!message.pageNumber.isZero()) {
            writer.uint32(24).uint64(message.pageNumber);
        }
        if (!message.pageTotal.isZero()) {
            writer.uint32(32).uint64(message.pageTotal);
        }
        if (!message.limit.isZero()) {
            writer.uint32(40).uint64(message.limit);
        }
        for (const v of message.txs) {
            exports.TxResponse.encode(v, writer.uint32(50).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseSearchTxsResult();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.totalCount = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.count = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.pageNumber = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.pageTotal = reader.uint64();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.limit = reader.uint64();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.txs.push(exports.TxResponse.decode(reader, reader.uint32()));
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
            totalCount: isSet(object.totalCount) ? long_1.default.fromValue(object.totalCount) : long_1.default.UZERO,
            count: isSet(object.count) ? long_1.default.fromValue(object.count) : long_1.default.UZERO,
            pageNumber: isSet(object.pageNumber) ? long_1.default.fromValue(object.pageNumber) : long_1.default.UZERO,
            pageTotal: isSet(object.pageTotal) ? long_1.default.fromValue(object.pageTotal) : long_1.default.UZERO,
            limit: isSet(object.limit) ? long_1.default.fromValue(object.limit) : long_1.default.UZERO,
            txs: Array.isArray(object === null || object === void 0 ? void 0 : object.txs) ? object.txs.map((e) => exports.TxResponse.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.totalCount !== undefined && (obj.totalCount = (message.totalCount || long_1.default.UZERO).toString());
        message.count !== undefined && (obj.count = (message.count || long_1.default.UZERO).toString());
        message.pageNumber !== undefined && (obj.pageNumber = (message.pageNumber || long_1.default.UZERO).toString());
        message.pageTotal !== undefined && (obj.pageTotal = (message.pageTotal || long_1.default.UZERO).toString());
        message.limit !== undefined && (obj.limit = (message.limit || long_1.default.UZERO).toString());
        if (message.txs) {
            obj.txs = message.txs.map((e) => e ? exports.TxResponse.toJSON(e) : undefined);
        }
        else {
            obj.txs = [];
        }
        return obj;
    },
    create(base) {
        return exports.SearchTxsResult.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseSearchTxsResult();
        message.totalCount = (object.totalCount !== undefined && object.totalCount !== null)
            ? long_1.default.fromValue(object.totalCount)
            : long_1.default.UZERO;
        message.count = (object.count !== undefined && object.count !== null) ? long_1.default.fromValue(object.count) : long_1.default.UZERO;
        message.pageNumber = (object.pageNumber !== undefined && object.pageNumber !== null)
            ? long_1.default.fromValue(object.pageNumber)
            : long_1.default.UZERO;
        message.pageTotal = (object.pageTotal !== undefined && object.pageTotal !== null)
            ? long_1.default.fromValue(object.pageTotal)
            : long_1.default.UZERO;
        message.limit = (object.limit !== undefined && object.limit !== null) ? long_1.default.fromValue(object.limit) : long_1.default.UZERO;
        message.txs = ((_a = object.txs) === null || _a === void 0 ? void 0 : _a.map((e) => exports.TxResponse.fromPartial(e))) || [];
        return message;
    },
};
var tsProtoGlobalThis = (() => {
    if (typeof globalThis !== "undefined") {
        return globalThis;
    }
    if (typeof self !== "undefined") {
        return self;
    }
    if (typeof window !== "undefined") {
        return window;
    }
    if (typeof global !== "undefined") {
        return global;
    }
    throw "Unable to locate global object";
})();
function bytesFromBase64(b64) {
    if (tsProtoGlobalThis.Buffer) {
        return Uint8Array.from(tsProtoGlobalThis.Buffer.from(b64, "base64"));
    }
    else {
        const bin = tsProtoGlobalThis.atob(b64);
        const arr = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; ++i) {
            arr[i] = bin.charCodeAt(i);
        }
        return arr;
    }
}
function base64FromBytes(arr) {
    if (tsProtoGlobalThis.Buffer) {
        return tsProtoGlobalThis.Buffer.from(arr).toString("base64");
    }
    else {
        const bin = [];
        arr.forEach((byte) => {
            bin.push(String.fromCharCode(byte));
        });
        return tsProtoGlobalThis.btoa(bin.join(""));
    }
}
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
