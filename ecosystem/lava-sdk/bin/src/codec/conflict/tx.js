"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.MsgClientImpl = exports.MsgConflictVoteRevealResponse = exports.MsgConflictVoteReveal = exports.MsgConflictVoteCommitResponse = exports.MsgConflictVoteCommit = exports.MsgDetectionResponse = exports.MsgDetection = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const conflict_data_1 = require("./conflict_data");
exports.protobufPackage = "lavanet.lava.conflict";
function createBaseMsgDetection() {
    return { creator: "", finalizationConflict: undefined, responseConflict: undefined, sameProviderConflict: undefined };
}
exports.MsgDetection = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.finalizationConflict !== undefined) {
            conflict_data_1.FinalizationConflict.encode(message.finalizationConflict, writer.uint32(18).fork()).ldelim();
        }
        if (message.responseConflict !== undefined) {
            conflict_data_1.ResponseConflict.encode(message.responseConflict, writer.uint32(26).fork()).ldelim();
        }
        if (message.sameProviderConflict !== undefined) {
            conflict_data_1.FinalizationConflict.encode(message.sameProviderConflict, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgDetection();
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
                    message.finalizationConflict = conflict_data_1.FinalizationConflict.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.responseConflict = conflict_data_1.ResponseConflict.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.sameProviderConflict = conflict_data_1.FinalizationConflict.decode(reader, reader.uint32());
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
            finalizationConflict: isSet(object.finalizationConflict)
                ? conflict_data_1.FinalizationConflict.fromJSON(object.finalizationConflict)
                : undefined,
            responseConflict: isSet(object.responseConflict) ? conflict_data_1.ResponseConflict.fromJSON(object.responseConflict) : undefined,
            sameProviderConflict: isSet(object.sameProviderConflict)
                ? conflict_data_1.FinalizationConflict.fromJSON(object.sameProviderConflict)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.finalizationConflict !== undefined && (obj.finalizationConflict = message.finalizationConflict
            ? conflict_data_1.FinalizationConflict.toJSON(message.finalizationConflict)
            : undefined);
        message.responseConflict !== undefined &&
            (obj.responseConflict = message.responseConflict ? conflict_data_1.ResponseConflict.toJSON(message.responseConflict) : undefined);
        message.sameProviderConflict !== undefined && (obj.sameProviderConflict = message.sameProviderConflict
            ? conflict_data_1.FinalizationConflict.toJSON(message.sameProviderConflict)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.MsgDetection.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseMsgDetection();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.finalizationConflict = (object.finalizationConflict !== undefined && object.finalizationConflict !== null)
            ? conflict_data_1.FinalizationConflict.fromPartial(object.finalizationConflict)
            : undefined;
        message.responseConflict = (object.responseConflict !== undefined && object.responseConflict !== null)
            ? conflict_data_1.ResponseConflict.fromPartial(object.responseConflict)
            : undefined;
        message.sameProviderConflict = (object.sameProviderConflict !== undefined && object.sameProviderConflict !== null)
            ? conflict_data_1.FinalizationConflict.fromPartial(object.sameProviderConflict)
            : undefined;
        return message;
    },
};
function createBaseMsgDetectionResponse() {
    return {};
}
exports.MsgDetectionResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgDetectionResponse();
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
        return exports.MsgDetectionResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgDetectionResponse();
        return message;
    },
};
function createBaseMsgConflictVoteCommit() {
    return { creator: "", voteID: "", hash: new Uint8Array() };
}
exports.MsgConflictVoteCommit = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.voteID !== "") {
            writer.uint32(18).string(message.voteID);
        }
        if (message.hash.length !== 0) {
            writer.uint32(26).bytes(message.hash);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgConflictVoteCommit();
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
                    message.voteID = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.hash = reader.bytes();
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
            voteID: isSet(object.voteID) ? String(object.voteID) : "",
            hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.voteID !== undefined && (obj.voteID = message.voteID);
        message.hash !== undefined &&
            (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.MsgConflictVoteCommit.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgConflictVoteCommit();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.voteID = (_b = object.voteID) !== null && _b !== void 0 ? _b : "";
        message.hash = (_c = object.hash) !== null && _c !== void 0 ? _c : new Uint8Array();
        return message;
    },
};
function createBaseMsgConflictVoteCommitResponse() {
    return {};
}
exports.MsgConflictVoteCommitResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgConflictVoteCommitResponse();
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
        return exports.MsgConflictVoteCommitResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgConflictVoteCommitResponse();
        return message;
    },
};
function createBaseMsgConflictVoteReveal() {
    return { creator: "", voteID: "", nonce: long_1.default.ZERO, hash: new Uint8Array() };
}
exports.MsgConflictVoteReveal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.voteID !== "") {
            writer.uint32(18).string(message.voteID);
        }
        if (!message.nonce.isZero()) {
            writer.uint32(24).int64(message.nonce);
        }
        if (message.hash.length !== 0) {
            writer.uint32(34).bytes(message.hash);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgConflictVoteReveal();
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
                    message.voteID = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.nonce = reader.int64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.hash = reader.bytes();
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
            voteID: isSet(object.voteID) ? String(object.voteID) : "",
            nonce: isSet(object.nonce) ? long_1.default.fromValue(object.nonce) : long_1.default.ZERO,
            hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.voteID !== undefined && (obj.voteID = message.voteID);
        message.nonce !== undefined && (obj.nonce = (message.nonce || long_1.default.ZERO).toString());
        message.hash !== undefined &&
            (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.MsgConflictVoteReveal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseMsgConflictVoteReveal();
        message.creator = (_a = object.creator) !== null && _a !== void 0 ? _a : "";
        message.voteID = (_b = object.voteID) !== null && _b !== void 0 ? _b : "";
        message.nonce = (object.nonce !== undefined && object.nonce !== null) ? long_1.default.fromValue(object.nonce) : long_1.default.ZERO;
        message.hash = (_c = object.hash) !== null && _c !== void 0 ? _c : new Uint8Array();
        return message;
    },
};
function createBaseMsgConflictVoteRevealResponse() {
    return {};
}
exports.MsgConflictVoteRevealResponse = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMsgConflictVoteRevealResponse();
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
        return exports.MsgConflictVoteRevealResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseMsgConflictVoteRevealResponse();
        return message;
    },
};
class MsgClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.conflict.Msg";
        this.rpc = rpc;
        this.Detection = this.Detection.bind(this);
        this.ConflictVoteCommit = this.ConflictVoteCommit.bind(this);
        this.ConflictVoteReveal = this.ConflictVoteReveal.bind(this);
    }
    Detection(request) {
        const data = exports.MsgDetection.encode(request).finish();
        const promise = this.rpc.request(this.service, "Detection", data);
        return promise.then((data) => exports.MsgDetectionResponse.decode(minimal_1.default.Reader.create(data)));
    }
    ConflictVoteCommit(request) {
        const data = exports.MsgConflictVoteCommit.encode(request).finish();
        const promise = this.rpc.request(this.service, "ConflictVoteCommit", data);
        return promise.then((data) => exports.MsgConflictVoteCommitResponse.decode(minimal_1.default.Reader.create(data)));
    }
    ConflictVoteReveal(request) {
        const data = exports.MsgConflictVoteReveal.encode(request).finish();
        const promise = this.rpc.request(this.service, "ConflictVoteReveal", data);
        return promise.then((data) => exports.MsgConflictVoteRevealResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.MsgClientImpl = MsgClientImpl;
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
