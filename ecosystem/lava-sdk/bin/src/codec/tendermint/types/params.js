"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.HashedParams = exports.VersionParams = exports.ValidatorParams = exports.EvidenceParams = exports.BlockParams = exports.ConsensusParams = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const duration_1 = require("../../google/protobuf/duration");
exports.protobufPackage = "tendermint.types";
function createBaseConsensusParams() {
    return { block: undefined, evidence: undefined, validator: undefined, version: undefined };
}
exports.ConsensusParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.block !== undefined) {
            exports.BlockParams.encode(message.block, writer.uint32(10).fork()).ldelim();
        }
        if (message.evidence !== undefined) {
            exports.EvidenceParams.encode(message.evidence, writer.uint32(18).fork()).ldelim();
        }
        if (message.validator !== undefined) {
            exports.ValidatorParams.encode(message.validator, writer.uint32(26).fork()).ldelim();
        }
        if (message.version !== undefined) {
            exports.VersionParams.encode(message.version, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseConsensusParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.block = exports.BlockParams.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.evidence = exports.EvidenceParams.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.validator = exports.ValidatorParams.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.version = exports.VersionParams.decode(reader, reader.uint32());
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
            block: isSet(object.block) ? exports.BlockParams.fromJSON(object.block) : undefined,
            evidence: isSet(object.evidence) ? exports.EvidenceParams.fromJSON(object.evidence) : undefined,
            validator: isSet(object.validator) ? exports.ValidatorParams.fromJSON(object.validator) : undefined,
            version: isSet(object.version) ? exports.VersionParams.fromJSON(object.version) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.block !== undefined && (obj.block = message.block ? exports.BlockParams.toJSON(message.block) : undefined);
        message.evidence !== undefined &&
            (obj.evidence = message.evidence ? exports.EvidenceParams.toJSON(message.evidence) : undefined);
        message.validator !== undefined &&
            (obj.validator = message.validator ? exports.ValidatorParams.toJSON(message.validator) : undefined);
        message.version !== undefined &&
            (obj.version = message.version ? exports.VersionParams.toJSON(message.version) : undefined);
        return obj;
    },
    create(base) {
        return exports.ConsensusParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseConsensusParams();
        message.block = (object.block !== undefined && object.block !== null)
            ? exports.BlockParams.fromPartial(object.block)
            : undefined;
        message.evidence = (object.evidence !== undefined && object.evidence !== null)
            ? exports.EvidenceParams.fromPartial(object.evidence)
            : undefined;
        message.validator = (object.validator !== undefined && object.validator !== null)
            ? exports.ValidatorParams.fromPartial(object.validator)
            : undefined;
        message.version = (object.version !== undefined && object.version !== null)
            ? exports.VersionParams.fromPartial(object.version)
            : undefined;
        return message;
    },
};
function createBaseBlockParams() {
    return { maxBytes: long_1.default.ZERO, maxGas: long_1.default.ZERO };
}
exports.BlockParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.maxBytes.isZero()) {
            writer.uint32(8).int64(message.maxBytes);
        }
        if (!message.maxGas.isZero()) {
            writer.uint32(16).int64(message.maxGas);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseBlockParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.maxBytes = reader.int64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.maxGas = reader.int64();
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
            maxBytes: isSet(object.maxBytes) ? long_1.default.fromValue(object.maxBytes) : long_1.default.ZERO,
            maxGas: isSet(object.maxGas) ? long_1.default.fromValue(object.maxGas) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.maxBytes !== undefined && (obj.maxBytes = (message.maxBytes || long_1.default.ZERO).toString());
        message.maxGas !== undefined && (obj.maxGas = (message.maxGas || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.BlockParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseBlockParams();
        message.maxBytes = (object.maxBytes !== undefined && object.maxBytes !== null)
            ? long_1.default.fromValue(object.maxBytes)
            : long_1.default.ZERO;
        message.maxGas = (object.maxGas !== undefined && object.maxGas !== null)
            ? long_1.default.fromValue(object.maxGas)
            : long_1.default.ZERO;
        return message;
    },
};
function createBaseEvidenceParams() {
    return { maxAgeNumBlocks: long_1.default.ZERO, maxAgeDuration: undefined, maxBytes: long_1.default.ZERO };
}
exports.EvidenceParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.maxAgeNumBlocks.isZero()) {
            writer.uint32(8).int64(message.maxAgeNumBlocks);
        }
        if (message.maxAgeDuration !== undefined) {
            duration_1.Duration.encode(message.maxAgeDuration, writer.uint32(18).fork()).ldelim();
        }
        if (!message.maxBytes.isZero()) {
            writer.uint32(24).int64(message.maxBytes);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEvidenceParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.maxAgeNumBlocks = reader.int64();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.maxAgeDuration = duration_1.Duration.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.maxBytes = reader.int64();
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
            maxAgeNumBlocks: isSet(object.maxAgeNumBlocks) ? long_1.default.fromValue(object.maxAgeNumBlocks) : long_1.default.ZERO,
            maxAgeDuration: isSet(object.maxAgeDuration) ? duration_1.Duration.fromJSON(object.maxAgeDuration) : undefined,
            maxBytes: isSet(object.maxBytes) ? long_1.default.fromValue(object.maxBytes) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.maxAgeNumBlocks !== undefined && (obj.maxAgeNumBlocks = (message.maxAgeNumBlocks || long_1.default.ZERO).toString());
        message.maxAgeDuration !== undefined &&
            (obj.maxAgeDuration = message.maxAgeDuration ? duration_1.Duration.toJSON(message.maxAgeDuration) : undefined);
        message.maxBytes !== undefined && (obj.maxBytes = (message.maxBytes || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.EvidenceParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseEvidenceParams();
        message.maxAgeNumBlocks = (object.maxAgeNumBlocks !== undefined && object.maxAgeNumBlocks !== null)
            ? long_1.default.fromValue(object.maxAgeNumBlocks)
            : long_1.default.ZERO;
        message.maxAgeDuration = (object.maxAgeDuration !== undefined && object.maxAgeDuration !== null)
            ? duration_1.Duration.fromPartial(object.maxAgeDuration)
            : undefined;
        message.maxBytes = (object.maxBytes !== undefined && object.maxBytes !== null)
            ? long_1.default.fromValue(object.maxBytes)
            : long_1.default.ZERO;
        return message;
    },
};
function createBaseValidatorParams() {
    return { pubKeyTypes: [] };
}
exports.ValidatorParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.pubKeyTypes) {
            writer.uint32(10).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseValidatorParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pubKeyTypes.push(reader.string());
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
        return { pubKeyTypes: Array.isArray(object === null || object === void 0 ? void 0 : object.pubKeyTypes) ? object.pubKeyTypes.map((e) => String(e)) : [] };
    },
    toJSON(message) {
        const obj = {};
        if (message.pubKeyTypes) {
            obj.pubKeyTypes = message.pubKeyTypes.map((e) => e);
        }
        else {
            obj.pubKeyTypes = [];
        }
        return obj;
    },
    create(base) {
        return exports.ValidatorParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseValidatorParams();
        message.pubKeyTypes = ((_a = object.pubKeyTypes) === null || _a === void 0 ? void 0 : _a.map((e) => e)) || [];
        return message;
    },
};
function createBaseVersionParams() {
    return { app: long_1.default.UZERO };
}
exports.VersionParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.app.isZero()) {
            writer.uint32(8).uint64(message.app);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseVersionParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.app = reader.uint64();
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
        return { app: isSet(object.app) ? long_1.default.fromValue(object.app) : long_1.default.UZERO };
    },
    toJSON(message) {
        const obj = {};
        message.app !== undefined && (obj.app = (message.app || long_1.default.UZERO).toString());
        return obj;
    },
    create(base) {
        return exports.VersionParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseVersionParams();
        message.app = (object.app !== undefined && object.app !== null) ? long_1.default.fromValue(object.app) : long_1.default.UZERO;
        return message;
    },
};
function createBaseHashedParams() {
    return { blockMaxBytes: long_1.default.ZERO, blockMaxGas: long_1.default.ZERO };
}
exports.HashedParams = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.blockMaxBytes.isZero()) {
            writer.uint32(8).int64(message.blockMaxBytes);
        }
        if (!message.blockMaxGas.isZero()) {
            writer.uint32(16).int64(message.blockMaxGas);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseHashedParams();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.blockMaxBytes = reader.int64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.blockMaxGas = reader.int64();
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
            blockMaxBytes: isSet(object.blockMaxBytes) ? long_1.default.fromValue(object.blockMaxBytes) : long_1.default.ZERO,
            blockMaxGas: isSet(object.blockMaxGas) ? long_1.default.fromValue(object.blockMaxGas) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.blockMaxBytes !== undefined && (obj.blockMaxBytes = (message.blockMaxBytes || long_1.default.ZERO).toString());
        message.blockMaxGas !== undefined && (obj.blockMaxGas = (message.blockMaxGas || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.HashedParams.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseHashedParams();
        message.blockMaxBytes = (object.blockMaxBytes !== undefined && object.blockMaxBytes !== null)
            ? long_1.default.fromValue(object.blockMaxBytes)
            : long_1.default.ZERO;
        message.blockMaxGas = (object.blockMaxGas !== undefined && object.blockMaxGas !== null)
            ? long_1.default.fromValue(object.blockMaxGas)
            : long_1.default.ZERO;
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
