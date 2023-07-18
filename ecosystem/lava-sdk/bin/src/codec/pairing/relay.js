"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.RelayerClientImpl = exports.QualityOfServiceReport = exports.RelayReply = exports.RelayRequest = exports.Metadata = exports.RelayPrivateData = exports.Badge = exports.RelaySession = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const operators_1 = require("rxjs/operators");
const wrappers_1 = require("../google/protobuf/wrappers");
exports.protobufPackage = "lavanet.lava.pairing";
function createBaseRelaySession() {
    return {
        specId: "",
        contentHash: new Uint8Array(),
        sessionId: long_1.default.UZERO,
        cuSum: long_1.default.UZERO,
        provider: "",
        relayNum: long_1.default.UZERO,
        qosReport: undefined,
        epoch: long_1.default.ZERO,
        unresponsiveProviders: new Uint8Array(),
        lavaChainId: "",
        sig: new Uint8Array(),
        badge: undefined,
        qosExcellenceReport: undefined,
    };
}
exports.RelaySession = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.specId !== "") {
            writer.uint32(10).string(message.specId);
        }
        if (message.contentHash.length !== 0) {
            writer.uint32(18).bytes(message.contentHash);
        }
        if (!message.sessionId.isZero()) {
            writer.uint32(24).uint64(message.sessionId);
        }
        if (!message.cuSum.isZero()) {
            writer.uint32(32).uint64(message.cuSum);
        }
        if (message.provider !== "") {
            writer.uint32(42).string(message.provider);
        }
        if (!message.relayNum.isZero()) {
            writer.uint32(48).uint64(message.relayNum);
        }
        if (message.qosReport !== undefined) {
            exports.QualityOfServiceReport.encode(message.qosReport, writer.uint32(58).fork()).ldelim();
        }
        if (!message.epoch.isZero()) {
            writer.uint32(64).int64(message.epoch);
        }
        if (message.unresponsiveProviders.length !== 0) {
            writer.uint32(74).bytes(message.unresponsiveProviders);
        }
        if (message.lavaChainId !== "") {
            writer.uint32(82).string(message.lavaChainId);
        }
        if (message.sig.length !== 0) {
            writer.uint32(90).bytes(message.sig);
        }
        if (message.badge !== undefined) {
            exports.Badge.encode(message.badge, writer.uint32(98).fork()).ldelim();
        }
        if (message.qosExcellenceReport !== undefined) {
            exports.QualityOfServiceReport.encode(message.qosExcellenceReport, writer.uint32(106).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRelaySession();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.specId = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.contentHash = reader.bytes();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.sessionId = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.cuSum = reader.uint64();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.provider = reader.string();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.relayNum = reader.uint64();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.qosReport = exports.QualityOfServiceReport.decode(reader, reader.uint32());
                    continue;
                case 8:
                    if (tag != 64) {
                        break;
                    }
                    message.epoch = reader.int64();
                    continue;
                case 9:
                    if (tag != 74) {
                        break;
                    }
                    message.unresponsiveProviders = reader.bytes();
                    continue;
                case 10:
                    if (tag != 82) {
                        break;
                    }
                    message.lavaChainId = reader.string();
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.sig = reader.bytes();
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.badge = exports.Badge.decode(reader, reader.uint32());
                    continue;
                case 13:
                    if (tag != 106) {
                        break;
                    }
                    message.qosExcellenceReport = exports.QualityOfServiceReport.decode(reader, reader.uint32());
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
            specId: isSet(object.specId) ? String(object.specId) : "",
            contentHash: isSet(object.contentHash) ? bytesFromBase64(object.contentHash) : new Uint8Array(),
            sessionId: isSet(object.sessionId) ? long_1.default.fromValue(object.sessionId) : long_1.default.UZERO,
            cuSum: isSet(object.cuSum) ? long_1.default.fromValue(object.cuSum) : long_1.default.UZERO,
            provider: isSet(object.provider) ? String(object.provider) : "",
            relayNum: isSet(object.relayNum) ? long_1.default.fromValue(object.relayNum) : long_1.default.UZERO,
            qosReport: isSet(object.qosReport) ? exports.QualityOfServiceReport.fromJSON(object.qosReport) : undefined,
            epoch: isSet(object.epoch) ? long_1.default.fromValue(object.epoch) : long_1.default.ZERO,
            unresponsiveProviders: isSet(object.unresponsiveProviders)
                ? bytesFromBase64(object.unresponsiveProviders)
                : new Uint8Array(),
            lavaChainId: isSet(object.lavaChainId) ? String(object.lavaChainId) : "",
            sig: isSet(object.sig) ? bytesFromBase64(object.sig) : new Uint8Array(),
            badge: isSet(object.badge) ? exports.Badge.fromJSON(object.badge) : undefined,
            qosExcellenceReport: isSet(object.qosExcellenceReport)
                ? exports.QualityOfServiceReport.fromJSON(object.qosExcellenceReport)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.specId !== undefined && (obj.specId = message.specId);
        message.contentHash !== undefined &&
            (obj.contentHash = base64FromBytes(message.contentHash !== undefined ? message.contentHash : new Uint8Array()));
        message.sessionId !== undefined && (obj.sessionId = (message.sessionId || long_1.default.UZERO).toString());
        message.cuSum !== undefined && (obj.cuSum = (message.cuSum || long_1.default.UZERO).toString());
        message.provider !== undefined && (obj.provider = message.provider);
        message.relayNum !== undefined && (obj.relayNum = (message.relayNum || long_1.default.UZERO).toString());
        message.qosReport !== undefined &&
            (obj.qosReport = message.qosReport ? exports.QualityOfServiceReport.toJSON(message.qosReport) : undefined);
        message.epoch !== undefined && (obj.epoch = (message.epoch || long_1.default.ZERO).toString());
        message.unresponsiveProviders !== undefined &&
            (obj.unresponsiveProviders = base64FromBytes(message.unresponsiveProviders !== undefined ? message.unresponsiveProviders : new Uint8Array()));
        message.lavaChainId !== undefined && (obj.lavaChainId = message.lavaChainId);
        message.sig !== undefined &&
            (obj.sig = base64FromBytes(message.sig !== undefined ? message.sig : new Uint8Array()));
        message.badge !== undefined && (obj.badge = message.badge ? exports.Badge.toJSON(message.badge) : undefined);
        message.qosExcellenceReport !== undefined && (obj.qosExcellenceReport = message.qosExcellenceReport
            ? exports.QualityOfServiceReport.toJSON(message.qosExcellenceReport)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.RelaySession.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f;
        const message = createBaseRelaySession();
        message.specId = (_a = object.specId) !== null && _a !== void 0 ? _a : "";
        message.contentHash = (_b = object.contentHash) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.sessionId = (object.sessionId !== undefined && object.sessionId !== null)
            ? long_1.default.fromValue(object.sessionId)
            : long_1.default.UZERO;
        message.cuSum = (object.cuSum !== undefined && object.cuSum !== null) ? long_1.default.fromValue(object.cuSum) : long_1.default.UZERO;
        message.provider = (_c = object.provider) !== null && _c !== void 0 ? _c : "";
        message.relayNum = (object.relayNum !== undefined && object.relayNum !== null)
            ? long_1.default.fromValue(object.relayNum)
            : long_1.default.UZERO;
        message.qosReport = (object.qosReport !== undefined && object.qosReport !== null)
            ? exports.QualityOfServiceReport.fromPartial(object.qosReport)
            : undefined;
        message.epoch = (object.epoch !== undefined && object.epoch !== null) ? long_1.default.fromValue(object.epoch) : long_1.default.ZERO;
        message.unresponsiveProviders = (_d = object.unresponsiveProviders) !== null && _d !== void 0 ? _d : new Uint8Array();
        message.lavaChainId = (_e = object.lavaChainId) !== null && _e !== void 0 ? _e : "";
        message.sig = (_f = object.sig) !== null && _f !== void 0 ? _f : new Uint8Array();
        message.badge = (object.badge !== undefined && object.badge !== null) ? exports.Badge.fromPartial(object.badge) : undefined;
        message.qosExcellenceReport = (object.qosExcellenceReport !== undefined && object.qosExcellenceReport !== null)
            ? exports.QualityOfServiceReport.fromPartial(object.qosExcellenceReport)
            : undefined;
        return message;
    },
};
function createBaseBadge() {
    return { cuAllocation: long_1.default.UZERO, epoch: long_1.default.UZERO, address: "", lavaChainId: "", projectSig: new Uint8Array() };
}
exports.Badge = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.cuAllocation.isZero()) {
            writer.uint32(8).uint64(message.cuAllocation);
        }
        if (!message.epoch.isZero()) {
            writer.uint32(16).uint64(message.epoch);
        }
        if (message.address !== "") {
            writer.uint32(26).string(message.address);
        }
        if (message.lavaChainId !== "") {
            writer.uint32(34).string(message.lavaChainId);
        }
        if (message.projectSig.length !== 0) {
            writer.uint32(42).bytes(message.projectSig);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseBadge();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.cuAllocation = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.epoch = reader.uint64();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.address = reader.string();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.lavaChainId = reader.string();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.projectSig = reader.bytes();
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
            cuAllocation: isSet(object.cuAllocation) ? long_1.default.fromValue(object.cuAllocation) : long_1.default.UZERO,
            epoch: isSet(object.epoch) ? long_1.default.fromValue(object.epoch) : long_1.default.UZERO,
            address: isSet(object.address) ? String(object.address) : "",
            lavaChainId: isSet(object.lavaChainId) ? String(object.lavaChainId) : "",
            projectSig: isSet(object.projectSig) ? bytesFromBase64(object.projectSig) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.cuAllocation !== undefined && (obj.cuAllocation = (message.cuAllocation || long_1.default.UZERO).toString());
        message.epoch !== undefined && (obj.epoch = (message.epoch || long_1.default.UZERO).toString());
        message.address !== undefined && (obj.address = message.address);
        message.lavaChainId !== undefined && (obj.lavaChainId = message.lavaChainId);
        message.projectSig !== undefined &&
            (obj.projectSig = base64FromBytes(message.projectSig !== undefined ? message.projectSig : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.Badge.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseBadge();
        message.cuAllocation = (object.cuAllocation !== undefined && object.cuAllocation !== null)
            ? long_1.default.fromValue(object.cuAllocation)
            : long_1.default.UZERO;
        message.epoch = (object.epoch !== undefined && object.epoch !== null) ? long_1.default.fromValue(object.epoch) : long_1.default.UZERO;
        message.address = (_a = object.address) !== null && _a !== void 0 ? _a : "";
        message.lavaChainId = (_b = object.lavaChainId) !== null && _b !== void 0 ? _b : "";
        message.projectSig = (_c = object.projectSig) !== null && _c !== void 0 ? _c : new Uint8Array();
        return message;
    },
};
function createBaseRelayPrivateData() {
    return {
        connectionType: "",
        apiUrl: "",
        data: new Uint8Array(),
        requestBlock: long_1.default.ZERO,
        apiInterface: "",
        salt: new Uint8Array(),
        metadata: [],
        addon: [],
    };
}
exports.RelayPrivateData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.connectionType !== "") {
            writer.uint32(10).string(message.connectionType);
        }
        if (message.apiUrl !== "") {
            writer.uint32(18).string(message.apiUrl);
        }
        if (message.data.length !== 0) {
            writer.uint32(26).bytes(message.data);
        }
        if (!message.requestBlock.isZero()) {
            writer.uint32(32).int64(message.requestBlock);
        }
        if (message.apiInterface !== "") {
            writer.uint32(42).string(message.apiInterface);
        }
        if (message.salt.length !== 0) {
            writer.uint32(50).bytes(message.salt);
        }
        for (const v of message.metadata) {
            exports.Metadata.encode(v, writer.uint32(58).fork()).ldelim();
        }
        for (const v of message.addon) {
            writer.uint32(66).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRelayPrivateData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.connectionType = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.apiUrl = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.requestBlock = reader.int64();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.apiInterface = reader.string();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.salt = reader.bytes();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.metadata.push(exports.Metadata.decode(reader, reader.uint32()));
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.addon.push(reader.string());
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
            connectionType: isSet(object.connectionType) ? String(object.connectionType) : "",
            apiUrl: isSet(object.apiUrl) ? String(object.apiUrl) : "",
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            requestBlock: isSet(object.requestBlock) ? long_1.default.fromValue(object.requestBlock) : long_1.default.ZERO,
            apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
            salt: isSet(object.salt) ? bytesFromBase64(object.salt) : new Uint8Array(),
            metadata: Array.isArray(object === null || object === void 0 ? void 0 : object.metadata) ? object.metadata.map((e) => exports.Metadata.fromJSON(e)) : [],
            addon: Array.isArray(object === null || object === void 0 ? void 0 : object.addon) ? object.addon.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.connectionType !== undefined && (obj.connectionType = message.connectionType);
        message.apiUrl !== undefined && (obj.apiUrl = message.apiUrl);
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.requestBlock !== undefined && (obj.requestBlock = (message.requestBlock || long_1.default.ZERO).toString());
        message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
        message.salt !== undefined &&
            (obj.salt = base64FromBytes(message.salt !== undefined ? message.salt : new Uint8Array()));
        if (message.metadata) {
            obj.metadata = message.metadata.map((e) => e ? exports.Metadata.toJSON(e) : undefined);
        }
        else {
            obj.metadata = [];
        }
        if (message.addon) {
            obj.addon = message.addon.map((e) => e);
        }
        else {
            obj.addon = [];
        }
        return obj;
    },
    create(base) {
        return exports.RelayPrivateData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f, _g;
        const message = createBaseRelayPrivateData();
        message.connectionType = (_a = object.connectionType) !== null && _a !== void 0 ? _a : "";
        message.apiUrl = (_b = object.apiUrl) !== null && _b !== void 0 ? _b : "";
        message.data = (_c = object.data) !== null && _c !== void 0 ? _c : new Uint8Array();
        message.requestBlock = (object.requestBlock !== undefined && object.requestBlock !== null)
            ? long_1.default.fromValue(object.requestBlock)
            : long_1.default.ZERO;
        message.apiInterface = (_d = object.apiInterface) !== null && _d !== void 0 ? _d : "";
        message.salt = (_e = object.salt) !== null && _e !== void 0 ? _e : new Uint8Array();
        message.metadata = ((_f = object.metadata) === null || _f === void 0 ? void 0 : _f.map((e) => exports.Metadata.fromPartial(e))) || [];
        message.addon = ((_g = object.addon) === null || _g === void 0 ? void 0 : _g.map((e) => e)) || [];
        return message;
    },
};
function createBaseMetadata() {
    return { name: "", value: "" };
}
exports.Metadata = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.value !== "") {
            writer.uint32(18).string(message.value);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMetadata();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.name = reader.string();
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
        return {
            name: isSet(object.name) ? String(object.name) : "",
            value: isSet(object.value) ? String(object.value) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.value !== undefined && (obj.value = message.value);
        return obj;
    },
    create(base) {
        return exports.Metadata.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMetadata();
        message.name = (_a = object.name) !== null && _a !== void 0 ? _a : "";
        message.value = (_b = object.value) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseRelayRequest() {
    return { relaySession: undefined, relayData: undefined };
}
exports.RelayRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.relaySession !== undefined) {
            exports.RelaySession.encode(message.relaySession, writer.uint32(10).fork()).ldelim();
        }
        if (message.relayData !== undefined) {
            exports.RelayPrivateData.encode(message.relayData, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRelayRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.relaySession = exports.RelaySession.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.relayData = exports.RelayPrivateData.decode(reader, reader.uint32());
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
            relaySession: isSet(object.relaySession) ? exports.RelaySession.fromJSON(object.relaySession) : undefined,
            relayData: isSet(object.relayData) ? exports.RelayPrivateData.fromJSON(object.relayData) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.relaySession !== undefined &&
            (obj.relaySession = message.relaySession ? exports.RelaySession.toJSON(message.relaySession) : undefined);
        message.relayData !== undefined &&
            (obj.relayData = message.relayData ? exports.RelayPrivateData.toJSON(message.relayData) : undefined);
        return obj;
    },
    create(base) {
        return exports.RelayRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseRelayRequest();
        message.relaySession = (object.relaySession !== undefined && object.relaySession !== null)
            ? exports.RelaySession.fromPartial(object.relaySession)
            : undefined;
        message.relayData = (object.relayData !== undefined && object.relayData !== null)
            ? exports.RelayPrivateData.fromPartial(object.relayData)
            : undefined;
        return message;
    },
};
function createBaseRelayReply() {
    return {
        data: new Uint8Array(),
        sig: new Uint8Array(),
        latestBlock: long_1.default.ZERO,
        finalizedBlocksHashes: new Uint8Array(),
        sigBlocks: new Uint8Array(),
        metadata: [],
    };
}
exports.RelayReply = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.data.length !== 0) {
            writer.uint32(10).bytes(message.data);
        }
        if (message.sig.length !== 0) {
            writer.uint32(18).bytes(message.sig);
        }
        if (!message.latestBlock.isZero()) {
            writer.uint32(32).int64(message.latestBlock);
        }
        if (message.finalizedBlocksHashes.length !== 0) {
            writer.uint32(42).bytes(message.finalizedBlocksHashes);
        }
        if (message.sigBlocks.length !== 0) {
            writer.uint32(50).bytes(message.sigBlocks);
        }
        for (const v of message.metadata) {
            exports.Metadata.encode(v, writer.uint32(58).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRelayReply();
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
                    message.sig = reader.bytes();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.latestBlock = reader.int64();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.finalizedBlocksHashes = reader.bytes();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.sigBlocks = reader.bytes();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.metadata.push(exports.Metadata.decode(reader, reader.uint32()));
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
            sig: isSet(object.sig) ? bytesFromBase64(object.sig) : new Uint8Array(),
            latestBlock: isSet(object.latestBlock) ? long_1.default.fromValue(object.latestBlock) : long_1.default.ZERO,
            finalizedBlocksHashes: isSet(object.finalizedBlocksHashes)
                ? bytesFromBase64(object.finalizedBlocksHashes)
                : new Uint8Array(),
            sigBlocks: isSet(object.sigBlocks) ? bytesFromBase64(object.sigBlocks) : new Uint8Array(),
            metadata: Array.isArray(object === null || object === void 0 ? void 0 : object.metadata) ? object.metadata.map((e) => exports.Metadata.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.sig !== undefined &&
            (obj.sig = base64FromBytes(message.sig !== undefined ? message.sig : new Uint8Array()));
        message.latestBlock !== undefined && (obj.latestBlock = (message.latestBlock || long_1.default.ZERO).toString());
        message.finalizedBlocksHashes !== undefined &&
            (obj.finalizedBlocksHashes = base64FromBytes(message.finalizedBlocksHashes !== undefined ? message.finalizedBlocksHashes : new Uint8Array()));
        message.sigBlocks !== undefined &&
            (obj.sigBlocks = base64FromBytes(message.sigBlocks !== undefined ? message.sigBlocks : new Uint8Array()));
        if (message.metadata) {
            obj.metadata = message.metadata.map((e) => e ? exports.Metadata.toJSON(e) : undefined);
        }
        else {
            obj.metadata = [];
        }
        return obj;
    },
    create(base) {
        return exports.RelayReply.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseRelayReply();
        message.data = (_a = object.data) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.sig = (_b = object.sig) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.latestBlock = (object.latestBlock !== undefined && object.latestBlock !== null)
            ? long_1.default.fromValue(object.latestBlock)
            : long_1.default.ZERO;
        message.finalizedBlocksHashes = (_c = object.finalizedBlocksHashes) !== null && _c !== void 0 ? _c : new Uint8Array();
        message.sigBlocks = (_d = object.sigBlocks) !== null && _d !== void 0 ? _d : new Uint8Array();
        message.metadata = ((_e = object.metadata) === null || _e === void 0 ? void 0 : _e.map((e) => exports.Metadata.fromPartial(e))) || [];
        return message;
    },
};
function createBaseQualityOfServiceReport() {
    return { latency: "", availability: "", sync: "" };
}
exports.QualityOfServiceReport = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.latency !== "") {
            writer.uint32(10).string(message.latency);
        }
        if (message.availability !== "") {
            writer.uint32(18).string(message.availability);
        }
        if (message.sync !== "") {
            writer.uint32(26).string(message.sync);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQualityOfServiceReport();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.latency = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.availability = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.sync = reader.string();
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
            latency: isSet(object.latency) ? String(object.latency) : "",
            availability: isSet(object.availability) ? String(object.availability) : "",
            sync: isSet(object.sync) ? String(object.sync) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.latency !== undefined && (obj.latency = message.latency);
        message.availability !== undefined && (obj.availability = message.availability);
        message.sync !== undefined && (obj.sync = message.sync);
        return obj;
    },
    create(base) {
        return exports.QualityOfServiceReport.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseQualityOfServiceReport();
        message.latency = (_a = object.latency) !== null && _a !== void 0 ? _a : "";
        message.availability = (_b = object.availability) !== null && _b !== void 0 ? _b : "";
        message.sync = (_c = object.sync) !== null && _c !== void 0 ? _c : "";
        return message;
    },
};
class RelayerClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.pairing.Relayer";
        this.rpc = rpc;
        this.Relay = this.Relay.bind(this);
        this.RelaySubscribe = this.RelaySubscribe.bind(this);
        this.Probe = this.Probe.bind(this);
    }
    Relay(request) {
        const data = exports.RelayRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Relay", data);
        return promise.then((data) => exports.RelayReply.decode(minimal_1.default.Reader.create(data)));
    }
    RelaySubscribe(request) {
        const data = exports.RelayRequest.encode(request).finish();
        const result = this.rpc.serverStreamingRequest(this.service, "RelaySubscribe", data);
        return result.pipe((0, operators_1.map)((data) => exports.RelayReply.decode(minimal_1.default.Reader.create(data))));
    }
    Probe(request) {
        const data = wrappers_1.UInt64Value.encode(request).finish();
        const promise = this.rpc.request(this.service, "Probe", data);
        return promise.then((data) => wrappers_1.UInt64Value.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.RelayerClientImpl = RelayerClientImpl;
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
