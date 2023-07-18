"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SpecCategory = exports.BlockParser = exports.ApiInterface = exports.Parsing = exports.ServiceApi = exports.parserFuncToJSON = exports.parserFuncFromJSON = exports.parserFunc = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.spec";
var parserFunc;
(function (parserFunc) {
    parserFunc[parserFunc["EMPTY"] = 0] = "EMPTY";
    /** PARSE_BY_ARG - means parameters are ordered and flat expected arguments are: [param index] (example: PARAMS: [<#BlockNum>,"banana"]) args: 0 */
    parserFunc[parserFunc["PARSE_BY_ARG"] = 1] = "PARSE_BY_ARG";
    /** PARSE_CANONICAL - means parameters are ordered and one of them has named properties, expected arguments are: [param index to object,prop_name in object] (example: PARAMS: ["banana",{prop_name:<#BlockNum>}]) need to configure args: 1,"prop_name" */
    parserFunc[parserFunc["PARSE_CANONICAL"] = 2] = "PARSE_CANONICAL";
    /** PARSE_DICTIONARY - means parameters are named, expected arguments are [prop_name,separator] (example: PARAMS: {prop_name:<#BlockNum>,prop2:"banana"}) args: "prop_name" */
    parserFunc[parserFunc["PARSE_DICTIONARY"] = 3] = "PARSE_DICTIONARY";
    /** PARSE_DICTIONARY_OR_ORDERED - means parameters are named expected arguments are [prop_name,separator,parameter order if not found] for input of: block=15&address=abc OR ?abc,15 we will do args: block,=,1 */
    parserFunc[parserFunc["PARSE_DICTIONARY_OR_ORDERED"] = 4] = "PARSE_DICTIONARY_OR_ORDERED";
    /** DEFAULT - reserved */
    parserFunc[parserFunc["DEFAULT"] = 6] = "DEFAULT";
    parserFunc[parserFunc["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(parserFunc = exports.parserFunc || (exports.parserFunc = {}));
function parserFuncFromJSON(object) {
    switch (object) {
        case 0:
        case "EMPTY":
            return parserFunc.EMPTY;
        case 1:
        case "PARSE_BY_ARG":
            return parserFunc.PARSE_BY_ARG;
        case 2:
        case "PARSE_CANONICAL":
            return parserFunc.PARSE_CANONICAL;
        case 3:
        case "PARSE_DICTIONARY":
            return parserFunc.PARSE_DICTIONARY;
        case 4:
        case "PARSE_DICTIONARY_OR_ORDERED":
            return parserFunc.PARSE_DICTIONARY_OR_ORDERED;
        case 6:
        case "DEFAULT":
            return parserFunc.DEFAULT;
        case -1:
        case "UNRECOGNIZED":
        default:
            return parserFunc.UNRECOGNIZED;
    }
}
exports.parserFuncFromJSON = parserFuncFromJSON;
function parserFuncToJSON(object) {
    switch (object) {
        case parserFunc.EMPTY:
            return "EMPTY";
        case parserFunc.PARSE_BY_ARG:
            return "PARSE_BY_ARG";
        case parserFunc.PARSE_CANONICAL:
            return "PARSE_CANONICAL";
        case parserFunc.PARSE_DICTIONARY:
            return "PARSE_DICTIONARY";
        case parserFunc.PARSE_DICTIONARY_OR_ORDERED:
            return "PARSE_DICTIONARY_OR_ORDERED";
        case parserFunc.DEFAULT:
            return "DEFAULT";
        case parserFunc.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.parserFuncToJSON = parserFuncToJSON;
function createBaseServiceApi() {
    return {
        name: "",
        blockParsing: undefined,
        computeUnits: long_1.default.UZERO,
        enabled: false,
        apiInterfaces: [],
        reserved: undefined,
        parsing: undefined,
        internalPath: "",
    };
}
exports.ServiceApi = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.blockParsing !== undefined) {
            exports.BlockParser.encode(message.blockParsing, writer.uint32(18).fork()).ldelim();
        }
        if (!message.computeUnits.isZero()) {
            writer.uint32(24).uint64(message.computeUnits);
        }
        if (message.enabled === true) {
            writer.uint32(32).bool(message.enabled);
        }
        for (const v of message.apiInterfaces) {
            exports.ApiInterface.encode(v, writer.uint32(42).fork()).ldelim();
        }
        if (message.reserved !== undefined) {
            exports.SpecCategory.encode(message.reserved, writer.uint32(50).fork()).ldelim();
        }
        if (message.parsing !== undefined) {
            exports.Parsing.encode(message.parsing, writer.uint32(58).fork()).ldelim();
        }
        if (message.internalPath !== "") {
            writer.uint32(66).string(message.internalPath);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseServiceApi();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.name = reader.string();
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.blockParsing = exports.BlockParser.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag !== 24) {
                        break;
                    }
                    message.computeUnits = reader.uint64();
                    continue;
                case 4:
                    if (tag !== 32) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 5:
                    if (tag !== 42) {
                        break;
                    }
                    message.apiInterfaces.push(exports.ApiInterface.decode(reader, reader.uint32()));
                    continue;
                case 6:
                    if (tag !== 50) {
                        break;
                    }
                    message.reserved = exports.SpecCategory.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag !== 58) {
                        break;
                    }
                    message.parsing = exports.Parsing.decode(reader, reader.uint32());
                    continue;
                case 8:
                    if (tag !== 66) {
                        break;
                    }
                    message.internalPath = reader.string();
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
            name: isSet(object.name) ? String(object.name) : "",
            blockParsing: isSet(object.blockParsing) ? exports.BlockParser.fromJSON(object.blockParsing) : undefined,
            computeUnits: isSet(object.computeUnits) ? long_1.default.fromValue(object.computeUnits) : long_1.default.UZERO,
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            apiInterfaces: Array.isArray(object === null || object === void 0 ? void 0 : object.apiInterfaces)
                ? object.apiInterfaces.map((e) => exports.ApiInterface.fromJSON(e))
                : [],
            reserved: isSet(object.reserved) ? exports.SpecCategory.fromJSON(object.reserved) : undefined,
            parsing: isSet(object.parsing) ? exports.Parsing.fromJSON(object.parsing) : undefined,
            internalPath: isSet(object.internalPath) ? String(object.internalPath) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.blockParsing !== undefined &&
            (obj.blockParsing = message.blockParsing ? exports.BlockParser.toJSON(message.blockParsing) : undefined);
        message.computeUnits !== undefined && (obj.computeUnits = (message.computeUnits || long_1.default.UZERO).toString());
        message.enabled !== undefined && (obj.enabled = message.enabled);
        if (message.apiInterfaces) {
            obj.apiInterfaces = message.apiInterfaces.map((e) => e ? exports.ApiInterface.toJSON(e) : undefined);
        }
        else {
            obj.apiInterfaces = [];
        }
        message.reserved !== undefined &&
            (obj.reserved = message.reserved ? exports.SpecCategory.toJSON(message.reserved) : undefined);
        message.parsing !== undefined && (obj.parsing = message.parsing ? exports.Parsing.toJSON(message.parsing) : undefined);
        message.internalPath !== undefined && (obj.internalPath = message.internalPath);
        return obj;
    },
    create(base) {
        return exports.ServiceApi.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseServiceApi();
        message.name = (_a = object.name) !== null && _a !== void 0 ? _a : "";
        message.blockParsing = (object.blockParsing !== undefined && object.blockParsing !== null)
            ? exports.BlockParser.fromPartial(object.blockParsing)
            : undefined;
        message.computeUnits = (object.computeUnits !== undefined && object.computeUnits !== null)
            ? long_1.default.fromValue(object.computeUnits)
            : long_1.default.UZERO;
        message.enabled = (_b = object.enabled) !== null && _b !== void 0 ? _b : false;
        message.apiInterfaces = ((_c = object.apiInterfaces) === null || _c === void 0 ? void 0 : _c.map((e) => exports.ApiInterface.fromPartial(e))) || [];
        message.reserved = (object.reserved !== undefined && object.reserved !== null)
            ? exports.SpecCategory.fromPartial(object.reserved)
            : undefined;
        message.parsing = (object.parsing !== undefined && object.parsing !== null)
            ? exports.Parsing.fromPartial(object.parsing)
            : undefined;
        message.internalPath = (_d = object.internalPath) !== null && _d !== void 0 ? _d : "";
        return message;
    },
};
function createBaseParsing() {
    return { functionTag: "", functionTemplate: "", resultParsing: undefined };
}
exports.Parsing = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.functionTag !== "") {
            writer.uint32(10).string(message.functionTag);
        }
        if (message.functionTemplate !== "") {
            writer.uint32(18).string(message.functionTemplate);
        }
        if (message.resultParsing !== undefined) {
            exports.BlockParser.encode(message.resultParsing, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseParsing();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.functionTag = reader.string();
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.functionTemplate = reader.string();
                    continue;
                case 3:
                    if (tag !== 26) {
                        break;
                    }
                    message.resultParsing = exports.BlockParser.decode(reader, reader.uint32());
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
            functionTag: isSet(object.functionTag) ? String(object.functionTag) : "",
            functionTemplate: isSet(object.functionTemplate) ? String(object.functionTemplate) : "",
            resultParsing: isSet(object.resultParsing) ? exports.BlockParser.fromJSON(object.resultParsing) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.functionTag !== undefined && (obj.functionTag = message.functionTag);
        message.functionTemplate !== undefined && (obj.functionTemplate = message.functionTemplate);
        message.resultParsing !== undefined &&
            (obj.resultParsing = message.resultParsing ? exports.BlockParser.toJSON(message.resultParsing) : undefined);
        return obj;
    },
    create(base) {
        return exports.Parsing.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseParsing();
        message.functionTag = (_a = object.functionTag) !== null && _a !== void 0 ? _a : "";
        message.functionTemplate = (_b = object.functionTemplate) !== null && _b !== void 0 ? _b : "";
        message.resultParsing = (object.resultParsing !== undefined && object.resultParsing !== null)
            ? exports.BlockParser.fromPartial(object.resultParsing)
            : undefined;
        return message;
    },
};
function createBaseApiInterface() {
    return {
        interface: "",
        type: "",
        extraComputeUnits: long_1.default.UZERO,
        category: undefined,
        overwriteBlockParsing: undefined,
    };
}
exports.ApiInterface = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.interface !== "") {
            writer.uint32(10).string(message.interface);
        }
        if (message.type !== "") {
            writer.uint32(18).string(message.type);
        }
        if (!message.extraComputeUnits.isZero()) {
            writer.uint32(24).uint64(message.extraComputeUnits);
        }
        if (message.category !== undefined) {
            exports.SpecCategory.encode(message.category, writer.uint32(34).fork()).ldelim();
        }
        if (message.overwriteBlockParsing !== undefined) {
            exports.BlockParser.encode(message.overwriteBlockParsing, writer.uint32(42).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseApiInterface();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.interface = reader.string();
                    continue;
                case 2:
                    if (tag !== 18) {
                        break;
                    }
                    message.type = reader.string();
                    continue;
                case 3:
                    if (tag !== 24) {
                        break;
                    }
                    message.extraComputeUnits = reader.uint64();
                    continue;
                case 4:
                    if (tag !== 34) {
                        break;
                    }
                    message.category = exports.SpecCategory.decode(reader, reader.uint32());
                    continue;
                case 5:
                    if (tag !== 42) {
                        break;
                    }
                    message.overwriteBlockParsing = exports.BlockParser.decode(reader, reader.uint32());
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
            interface: isSet(object.interface) ? String(object.interface) : "",
            type: isSet(object.type) ? String(object.type) : "",
            extraComputeUnits: isSet(object.extraComputeUnits) ? long_1.default.fromValue(object.extraComputeUnits) : long_1.default.UZERO,
            category: isSet(object.category) ? exports.SpecCategory.fromJSON(object.category) : undefined,
            overwriteBlockParsing: isSet(object.overwriteBlockParsing)
                ? exports.BlockParser.fromJSON(object.overwriteBlockParsing)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.interface !== undefined && (obj.interface = message.interface);
        message.type !== undefined && (obj.type = message.type);
        message.extraComputeUnits !== undefined &&
            (obj.extraComputeUnits = (message.extraComputeUnits || long_1.default.UZERO).toString());
        message.category !== undefined &&
            (obj.category = message.category ? exports.SpecCategory.toJSON(message.category) : undefined);
        message.overwriteBlockParsing !== undefined && (obj.overwriteBlockParsing = message.overwriteBlockParsing
            ? exports.BlockParser.toJSON(message.overwriteBlockParsing)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.ApiInterface.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseApiInterface();
        message.interface = (_a = object.interface) !== null && _a !== void 0 ? _a : "";
        message.type = (_b = object.type) !== null && _b !== void 0 ? _b : "";
        message.extraComputeUnits = (object.extraComputeUnits !== undefined && object.extraComputeUnits !== null)
            ? long_1.default.fromValue(object.extraComputeUnits)
            : long_1.default.UZERO;
        message.category = (object.category !== undefined && object.category !== null)
            ? exports.SpecCategory.fromPartial(object.category)
            : undefined;
        message.overwriteBlockParsing =
            (object.overwriteBlockParsing !== undefined && object.overwriteBlockParsing !== null)
                ? exports.BlockParser.fromPartial(object.overwriteBlockParsing)
                : undefined;
        return message;
    },
};
function createBaseBlockParser() {
    return { parserArg: [], parserFunc: 0, defaultValue: "", encoding: "" };
}
exports.BlockParser = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.parserArg) {
            writer.uint32(10).string(v);
        }
        if (message.parserFunc !== 0) {
            writer.uint32(16).int32(message.parserFunc);
        }
        if (message.defaultValue !== "") {
            writer.uint32(26).string(message.defaultValue);
        }
        if (message.encoding !== "") {
            writer.uint32(34).string(message.encoding);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseBlockParser();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 10) {
                        break;
                    }
                    message.parserArg.push(reader.string());
                    continue;
                case 2:
                    if (tag !== 16) {
                        break;
                    }
                    message.parserFunc = reader.int32();
                    continue;
                case 3:
                    if (tag !== 26) {
                        break;
                    }
                    message.defaultValue = reader.string();
                    continue;
                case 4:
                    if (tag !== 34) {
                        break;
                    }
                    message.encoding = reader.string();
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
            parserArg: Array.isArray(object === null || object === void 0 ? void 0 : object.parserArg) ? object.parserArg.map((e) => String(e)) : [],
            parserFunc: isSet(object.parserFunc) ? parserFuncFromJSON(object.parserFunc) : 0,
            defaultValue: isSet(object.defaultValue) ? String(object.defaultValue) : "",
            encoding: isSet(object.encoding) ? String(object.encoding) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.parserArg) {
            obj.parserArg = message.parserArg.map((e) => e);
        }
        else {
            obj.parserArg = [];
        }
        message.parserFunc !== undefined && (obj.parserFunc = parserFuncToJSON(message.parserFunc));
        message.defaultValue !== undefined && (obj.defaultValue = message.defaultValue);
        message.encoding !== undefined && (obj.encoding = message.encoding);
        return obj;
    },
    create(base) {
        return exports.BlockParser.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseBlockParser();
        message.parserArg = ((_a = object.parserArg) === null || _a === void 0 ? void 0 : _a.map((e) => e)) || [];
        message.parserFunc = (_b = object.parserFunc) !== null && _b !== void 0 ? _b : 0;
        message.defaultValue = (_c = object.defaultValue) !== null && _c !== void 0 ? _c : "";
        message.encoding = (_d = object.encoding) !== null && _d !== void 0 ? _d : "";
        return message;
    },
};
function createBaseSpecCategory() {
    return { deterministic: false, local: false, subscription: false, stateful: 0, hangingApi: false };
}
exports.SpecCategory = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.deterministic === true) {
            writer.uint32(8).bool(message.deterministic);
        }
        if (message.local === true) {
            writer.uint32(16).bool(message.local);
        }
        if (message.subscription === true) {
            writer.uint32(24).bool(message.subscription);
        }
        if (message.stateful !== 0) {
            writer.uint32(32).uint32(message.stateful);
        }
        if (message.hangingApi === true) {
            writer.uint32(40).bool(message.hangingApi);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseSpecCategory();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag !== 8) {
                        break;
                    }
                    message.deterministic = reader.bool();
                    continue;
                case 2:
                    if (tag !== 16) {
                        break;
                    }
                    message.local = reader.bool();
                    continue;
                case 3:
                    if (tag !== 24) {
                        break;
                    }
                    message.subscription = reader.bool();
                    continue;
                case 4:
                    if (tag !== 32) {
                        break;
                    }
                    message.stateful = reader.uint32();
                    continue;
                case 5:
                    if (tag !== 40) {
                        break;
                    }
                    message.hangingApi = reader.bool();
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
            deterministic: isSet(object.deterministic) ? Boolean(object.deterministic) : false,
            local: isSet(object.local) ? Boolean(object.local) : false,
            subscription: isSet(object.subscription) ? Boolean(object.subscription) : false,
            stateful: isSet(object.stateful) ? Number(object.stateful) : 0,
            hangingApi: isSet(object.hangingApi) ? Boolean(object.hangingApi) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.deterministic !== undefined && (obj.deterministic = message.deterministic);
        message.local !== undefined && (obj.local = message.local);
        message.subscription !== undefined && (obj.subscription = message.subscription);
        message.stateful !== undefined && (obj.stateful = Math.round(message.stateful));
        message.hangingApi !== undefined && (obj.hangingApi = message.hangingApi);
        return obj;
    },
    create(base) {
        return exports.SpecCategory.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseSpecCategory();
        message.deterministic = (_a = object.deterministic) !== null && _a !== void 0 ? _a : false;
        message.local = (_b = object.local) !== null && _b !== void 0 ? _b : false;
        message.subscription = (_c = object.subscription) !== null && _c !== void 0 ? _c : false;
        message.stateful = (_d = object.stateful) !== null && _d !== void 0 ? _d : 0;
        message.hangingApi = (_e = object.hangingApi) !== null && _e !== void 0 ? _e : false;
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
