"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.FinalizationConflict = exports.ConflictRelayData = exports.ResponseConflict = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const relay_1 = require("../pairing/relay");
exports.protobufPackage = "lavanet.lava.conflict";
function createBaseResponseConflict() {
    return { conflictRelayData0: undefined, conflictRelayData1: undefined };
}
exports.ResponseConflict = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.conflictRelayData0 !== undefined) {
            exports.ConflictRelayData.encode(message.conflictRelayData0, writer.uint32(10).fork()).ldelim();
        }
        if (message.conflictRelayData1 !== undefined) {
            exports.ConflictRelayData.encode(message.conflictRelayData1, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseConflict();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.conflictRelayData0 = exports.ConflictRelayData.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.conflictRelayData1 = exports.ConflictRelayData.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            conflictRelayData0: isSet(object.conflictRelayData0)
                ? exports.ConflictRelayData.fromJSON(object.conflictRelayData0)
                : undefined,
            conflictRelayData1: isSet(object.conflictRelayData1)
                ? exports.ConflictRelayData.fromJSON(object.conflictRelayData1)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.conflictRelayData0 !== undefined && (obj.conflictRelayData0 = message.conflictRelayData0
            ? exports.ConflictRelayData.toJSON(message.conflictRelayData0)
            : undefined);
        message.conflictRelayData1 !== undefined && (obj.conflictRelayData1 = message.conflictRelayData1
            ? exports.ConflictRelayData.toJSON(message.conflictRelayData1)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.ResponseConflict.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseResponseConflict();
        message.conflictRelayData0 = (object.conflictRelayData0 !== undefined && object.conflictRelayData0 !== null)
            ? exports.ConflictRelayData.fromPartial(object.conflictRelayData0)
            : undefined;
        message.conflictRelayData1 = (object.conflictRelayData1 !== undefined && object.conflictRelayData1 !== null)
            ? exports.ConflictRelayData.fromPartial(object.conflictRelayData1)
            : undefined;
        return message;
    },
};
function createBaseConflictRelayData() {
    return { request: undefined, reply: undefined };
}
exports.ConflictRelayData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.request !== undefined) {
            relay_1.RelayRequest.encode(message.request, writer.uint32(10).fork()).ldelim();
        }
        if (message.reply !== undefined) {
            relay_1.RelayReply.encode(message.reply, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseConflictRelayData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.request = relay_1.RelayRequest.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.reply = relay_1.RelayReply.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            request: isSet(object.request) ? relay_1.RelayRequest.fromJSON(object.request) : undefined,
            reply: isSet(object.reply) ? relay_1.RelayReply.fromJSON(object.reply) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.request !== undefined && (obj.request = message.request ? relay_1.RelayRequest.toJSON(message.request) : undefined);
        message.reply !== undefined && (obj.reply = message.reply ? relay_1.RelayReply.toJSON(message.reply) : undefined);
        return obj;
    },
    create(base) {
        return exports.ConflictRelayData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseConflictRelayData();
        message.request = (object.request !== undefined && object.request !== null)
            ? relay_1.RelayRequest.fromPartial(object.request)
            : undefined;
        message.reply = (object.reply !== undefined && object.reply !== null)
            ? relay_1.RelayReply.fromPartial(object.reply)
            : undefined;
        return message;
    },
};
function createBaseFinalizationConflict() {
    return { relayReply0: undefined, relayReply1: undefined };
}
exports.FinalizationConflict = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.relayReply0 !== undefined) {
            relay_1.RelayReply.encode(message.relayReply0, writer.uint32(10).fork()).ldelim();
        }
        if (message.relayReply1 !== undefined) {
            relay_1.RelayReply.encode(message.relayReply1, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseFinalizationConflict();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.relayReply0 = relay_1.RelayReply.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.relayReply1 = relay_1.RelayReply.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) === 4 || tag === 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return {
            relayReply0: isSet(object.relayReply0) ? relay_1.RelayReply.fromJSON(object.relayReply0) : undefined,
            relayReply1: isSet(object.relayReply1) ? relay_1.RelayReply.fromJSON(object.relayReply1) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.relayReply0 !== undefined &&
            (obj.relayReply0 = message.relayReply0 ? relay_1.RelayReply.toJSON(message.relayReply0) : undefined);
        message.relayReply1 !== undefined &&
            (obj.relayReply1 = message.relayReply1 ? relay_1.RelayReply.toJSON(message.relayReply1) : undefined);
        return obj;
    },
    create(base) {
        return exports.FinalizationConflict.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseFinalizationConflict();
        message.relayReply0 = (object.relayReply0 !== undefined && object.relayReply0 !== null)
            ? relay_1.RelayReply.fromPartial(object.relayReply0)
            : undefined;
        message.relayReply1 = (object.relayReply1 !== undefined && object.relayReply1 !== null)
            ? relay_1.RelayReply.fromPartial(object.relayReply1)
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
