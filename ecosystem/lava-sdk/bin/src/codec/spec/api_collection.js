"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SpecCategory = exports.BlockParser = exports.ParseDirective = exports.Api = exports.Header = exports.CollectionData = exports.ApiCollection = exports.header_HeaderTypeToJSON = exports.header_HeaderTypeFromJSON = exports.Header_HeaderType = exports.parserFuncToJSON = exports.parserFuncFromJSON = exports.parserFunc = exports.functionTagToJSON = exports.functionTagFromJSON = exports.functionTag = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
exports.protobufPackage = "lavanet.lava.spec";
var functionTag;
(function (functionTag) {
    functionTag[functionTag["DISABLED"] = 0] = "DISABLED";
    functionTag[functionTag["GET_BLOCKNUM"] = 1] = "GET_BLOCKNUM";
    functionTag[functionTag["GET_BLOCK_BY_NUM"] = 2] = "GET_BLOCK_BY_NUM";
    functionTag[functionTag["SET_LATEST_IN_METADATA"] = 3] = "SET_LATEST_IN_METADATA";
    functionTag[functionTag["SET_LATEST_IN_BODY"] = 4] = "SET_LATEST_IN_BODY";
    functionTag[functionTag["VERIFICATION"] = 5] = "VERIFICATION";
    functionTag[functionTag["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(functionTag = exports.functionTag || (exports.functionTag = {}));
function functionTagFromJSON(object) {
    switch (object) {
        case 0:
        case "DISABLED":
            return functionTag.DISABLED;
        case 1:
        case "GET_BLOCKNUM":
            return functionTag.GET_BLOCKNUM;
        case 2:
        case "GET_BLOCK_BY_NUM":
            return functionTag.GET_BLOCK_BY_NUM;
        case 3:
        case "SET_LATEST_IN_METADATA":
            return functionTag.SET_LATEST_IN_METADATA;
        case 4:
        case "SET_LATEST_IN_BODY":
            return functionTag.SET_LATEST_IN_BODY;
        case 5:
        case "VERIFICATION":
            return functionTag.VERIFICATION;
        case -1:
        case "UNRECOGNIZED":
        default:
            return functionTag.UNRECOGNIZED;
    }
}
exports.functionTagFromJSON = functionTagFromJSON;
function functionTagToJSON(object) {
    switch (object) {
        case functionTag.DISABLED:
            return "DISABLED";
        case functionTag.GET_BLOCKNUM:
            return "GET_BLOCKNUM";
        case functionTag.GET_BLOCK_BY_NUM:
            return "GET_BLOCK_BY_NUM";
        case functionTag.SET_LATEST_IN_METADATA:
            return "SET_LATEST_IN_METADATA";
        case functionTag.SET_LATEST_IN_BODY:
            return "SET_LATEST_IN_BODY";
        case functionTag.VERIFICATION:
            return "VERIFICATION";
        case functionTag.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.functionTagToJSON = functionTagToJSON;
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
var Header_HeaderType;
(function (Header_HeaderType) {
    Header_HeaderType[Header_HeaderType["pass_send"] = 0] = "pass_send";
    Header_HeaderType[Header_HeaderType["pass_reply"] = 1] = "pass_reply";
    Header_HeaderType[Header_HeaderType["pass_both"] = 2] = "pass_both";
    /** pass_ignore - allows it to pass around but is not signed */
    Header_HeaderType[Header_HeaderType["pass_ignore"] = 3] = "pass_ignore";
    Header_HeaderType[Header_HeaderType["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(Header_HeaderType = exports.Header_HeaderType || (exports.Header_HeaderType = {}));
function header_HeaderTypeFromJSON(object) {
    switch (object) {
        case 0:
        case "pass_send":
            return Header_HeaderType.pass_send;
        case 1:
        case "pass_reply":
            return Header_HeaderType.pass_reply;
        case 2:
        case "pass_both":
            return Header_HeaderType.pass_both;
        case 3:
        case "pass_ignore":
            return Header_HeaderType.pass_ignore;
        case -1:
        case "UNRECOGNIZED":
        default:
            return Header_HeaderType.UNRECOGNIZED;
    }
}
exports.header_HeaderTypeFromJSON = header_HeaderTypeFromJSON;
function header_HeaderTypeToJSON(object) {
    switch (object) {
        case Header_HeaderType.pass_send:
            return "pass_send";
        case Header_HeaderType.pass_reply:
            return "pass_reply";
        case Header_HeaderType.pass_both:
            return "pass_both";
        case Header_HeaderType.pass_ignore:
            return "pass_ignore";
        case Header_HeaderType.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.header_HeaderTypeToJSON = header_HeaderTypeToJSON;
function createBaseApiCollection() {
    return { enabled: false, collectionData: undefined, apis: [], headers: [], inheritanceApis: [], parseDirectives: [] };
}
exports.ApiCollection = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.enabled === true) {
            writer.uint32(8).bool(message.enabled);
        }
        if (message.collectionData !== undefined) {
            exports.CollectionData.encode(message.collectionData, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.apis) {
            exports.Api.encode(v, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.headers) {
            exports.Header.encode(v, writer.uint32(34).fork()).ldelim();
        }
        for (const v of message.inheritanceApis) {
            exports.CollectionData.encode(v, writer.uint32(42).fork()).ldelim();
        }
        for (const v of message.parseDirectives) {
            exports.ParseDirective.encode(v, writer.uint32(50).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseApiCollection();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.collectionData = exports.CollectionData.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.apis.push(exports.Api.decode(reader, reader.uint32()));
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.headers.push(exports.Header.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.inheritanceApis.push(exports.CollectionData.decode(reader, reader.uint32()));
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.parseDirectives.push(exports.ParseDirective.decode(reader, reader.uint32()));
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
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            collectionData: isSet(object.collectionData) ? exports.CollectionData.fromJSON(object.collectionData) : undefined,
            apis: Array.isArray(object === null || object === void 0 ? void 0 : object.apis) ? object.apis.map((e) => exports.Api.fromJSON(e)) : [],
            headers: Array.isArray(object === null || object === void 0 ? void 0 : object.headers) ? object.headers.map((e) => exports.Header.fromJSON(e)) : [],
            inheritanceApis: Array.isArray(object === null || object === void 0 ? void 0 : object.inheritanceApis)
                ? object.inheritanceApis.map((e) => exports.CollectionData.fromJSON(e))
                : [],
            parseDirectives: Array.isArray(object === null || object === void 0 ? void 0 : object.parseDirectives)
                ? object.parseDirectives.map((e) => exports.ParseDirective.fromJSON(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.enabled !== undefined && (obj.enabled = message.enabled);
        message.collectionData !== undefined &&
            (obj.collectionData = message.collectionData ? exports.CollectionData.toJSON(message.collectionData) : undefined);
        if (message.apis) {
            obj.apis = message.apis.map((e) => e ? exports.Api.toJSON(e) : undefined);
        }
        else {
            obj.apis = [];
        }
        if (message.headers) {
            obj.headers = message.headers.map((e) => e ? exports.Header.toJSON(e) : undefined);
        }
        else {
            obj.headers = [];
        }
        if (message.inheritanceApis) {
            obj.inheritanceApis = message.inheritanceApis.map((e) => e ? exports.CollectionData.toJSON(e) : undefined);
        }
        else {
            obj.inheritanceApis = [];
        }
        if (message.parseDirectives) {
            obj.parseDirectives = message.parseDirectives.map((e) => e ? exports.ParseDirective.toJSON(e) : undefined);
        }
        else {
            obj.parseDirectives = [];
        }
        return obj;
    },
    create(base) {
        return exports.ApiCollection.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseApiCollection();
        message.enabled = (_a = object.enabled) !== null && _a !== void 0 ? _a : false;
        message.collectionData = (object.collectionData !== undefined && object.collectionData !== null)
            ? exports.CollectionData.fromPartial(object.collectionData)
            : undefined;
        message.apis = ((_b = object.apis) === null || _b === void 0 ? void 0 : _b.map((e) => exports.Api.fromPartial(e))) || [];
        message.headers = ((_c = object.headers) === null || _c === void 0 ? void 0 : _c.map((e) => exports.Header.fromPartial(e))) || [];
        message.inheritanceApis = ((_d = object.inheritanceApis) === null || _d === void 0 ? void 0 : _d.map((e) => exports.CollectionData.fromPartial(e))) || [];
        message.parseDirectives = ((_e = object.parseDirectives) === null || _e === void 0 ? void 0 : _e.map((e) => exports.ParseDirective.fromPartial(e))) || [];
        return message;
    },
};
function createBaseCollectionData() {
    return { apiInterface: "", internalPath: "", type: "", addOn: "" };
}
exports.CollectionData = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.apiInterface !== "") {
            writer.uint32(10).string(message.apiInterface);
        }
        if (message.internalPath !== "") {
            writer.uint32(18).string(message.internalPath);
        }
        if (message.type !== "") {
            writer.uint32(26).string(message.type);
        }
        if (message.addOn !== "") {
            writer.uint32(34).string(message.addOn);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseCollectionData();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.apiInterface = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.internalPath = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.type = reader.string();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.addOn = reader.string();
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
            apiInterface: isSet(object.apiInterface) ? String(object.apiInterface) : "",
            internalPath: isSet(object.internalPath) ? String(object.internalPath) : "",
            type: isSet(object.type) ? String(object.type) : "",
            addOn: isSet(object.addOn) ? String(object.addOn) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.apiInterface !== undefined && (obj.apiInterface = message.apiInterface);
        message.internalPath !== undefined && (obj.internalPath = message.internalPath);
        message.type !== undefined && (obj.type = message.type);
        message.addOn !== undefined && (obj.addOn = message.addOn);
        return obj;
    },
    create(base) {
        return exports.CollectionData.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseCollectionData();
        message.apiInterface = (_a = object.apiInterface) !== null && _a !== void 0 ? _a : "";
        message.internalPath = (_b = object.internalPath) !== null && _b !== void 0 ? _b : "";
        message.type = (_c = object.type) !== null && _c !== void 0 ? _c : "";
        message.addOn = (_d = object.addOn) !== null && _d !== void 0 ? _d : "";
        return message;
    },
};
function createBaseHeader() {
    return { name: "", kind: 0, functionTag: 0 };
}
exports.Header = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.kind !== 0) {
            writer.uint32(16).int32(message.kind);
        }
        if (message.functionTag !== 0) {
            writer.uint32(24).int32(message.functionTag);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseHeader();
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
                    if (tag != 16) {
                        break;
                    }
                    message.kind = reader.int32();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.functionTag = reader.int32();
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
            kind: isSet(object.kind) ? header_HeaderTypeFromJSON(object.kind) : 0,
            functionTag: isSet(object.functionTag) ? functionTagFromJSON(object.functionTag) : 0,
        };
    },
    toJSON(message) {
        const obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.kind !== undefined && (obj.kind = header_HeaderTypeToJSON(message.kind));
        message.functionTag !== undefined && (obj.functionTag = functionTagToJSON(message.functionTag));
        return obj;
    },
    create(base) {
        return exports.Header.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseHeader();
        message.name = (_a = object.name) !== null && _a !== void 0 ? _a : "";
        message.kind = (_b = object.kind) !== null && _b !== void 0 ? _b : 0;
        message.functionTag = (_c = object.functionTag) !== null && _c !== void 0 ? _c : 0;
        return message;
    },
};
function createBaseApi() {
    return {
        enabled: false,
        name: "",
        computeUnits: long_1.default.UZERO,
        extraComputeUnits: long_1.default.UZERO,
        category: undefined,
        blockParsing: undefined,
    };
}
exports.Api = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.enabled === true) {
            writer.uint32(8).bool(message.enabled);
        }
        if (message.name !== "") {
            writer.uint32(18).string(message.name);
        }
        if (!message.computeUnits.isZero()) {
            writer.uint32(24).uint64(message.computeUnits);
        }
        if (!message.extraComputeUnits.isZero()) {
            writer.uint32(32).uint64(message.extraComputeUnits);
        }
        if (message.category !== undefined) {
            exports.SpecCategory.encode(message.category, writer.uint32(50).fork()).ldelim();
        }
        if (message.blockParsing !== undefined) {
            exports.BlockParser.encode(message.blockParsing, writer.uint32(58).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseApi();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.enabled = reader.bool();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.name = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.computeUnits = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.extraComputeUnits = reader.uint64();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.category = exports.SpecCategory.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.blockParsing = exports.BlockParser.decode(reader, reader.uint32());
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
            enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
            name: isSet(object.name) ? String(object.name) : "",
            computeUnits: isSet(object.computeUnits) ? long_1.default.fromValue(object.computeUnits) : long_1.default.UZERO,
            extraComputeUnits: isSet(object.extraComputeUnits) ? long_1.default.fromValue(object.extraComputeUnits) : long_1.default.UZERO,
            category: isSet(object.category) ? exports.SpecCategory.fromJSON(object.category) : undefined,
            blockParsing: isSet(object.blockParsing) ? exports.BlockParser.fromJSON(object.blockParsing) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.enabled !== undefined && (obj.enabled = message.enabled);
        message.name !== undefined && (obj.name = message.name);
        message.computeUnits !== undefined && (obj.computeUnits = (message.computeUnits || long_1.default.UZERO).toString());
        message.extraComputeUnits !== undefined &&
            (obj.extraComputeUnits = (message.extraComputeUnits || long_1.default.UZERO).toString());
        message.category !== undefined &&
            (obj.category = message.category ? exports.SpecCategory.toJSON(message.category) : undefined);
        message.blockParsing !== undefined &&
            (obj.blockParsing = message.blockParsing ? exports.BlockParser.toJSON(message.blockParsing) : undefined);
        return obj;
    },
    create(base) {
        return exports.Api.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseApi();
        message.enabled = (_a = object.enabled) !== null && _a !== void 0 ? _a : false;
        message.name = (_b = object.name) !== null && _b !== void 0 ? _b : "";
        message.computeUnits = (object.computeUnits !== undefined && object.computeUnits !== null)
            ? long_1.default.fromValue(object.computeUnits)
            : long_1.default.UZERO;
        message.extraComputeUnits = (object.extraComputeUnits !== undefined && object.extraComputeUnits !== null)
            ? long_1.default.fromValue(object.extraComputeUnits)
            : long_1.default.UZERO;
        message.category = (object.category !== undefined && object.category !== null)
            ? exports.SpecCategory.fromPartial(object.category)
            : undefined;
        message.blockParsing = (object.blockParsing !== undefined && object.blockParsing !== null)
            ? exports.BlockParser.fromPartial(object.blockParsing)
            : undefined;
        return message;
    },
};
function createBaseParseDirective() {
    return { functionTag: 0, functionTemplate: "", resultParsing: undefined, apiName: "" };
}
exports.ParseDirective = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.functionTag !== 0) {
            writer.uint32(8).int32(message.functionTag);
        }
        if (message.functionTemplate !== "") {
            writer.uint32(18).string(message.functionTemplate);
        }
        if (message.resultParsing !== undefined) {
            exports.BlockParser.encode(message.resultParsing, writer.uint32(26).fork()).ldelim();
        }
        if (message.apiName !== "") {
            writer.uint32(34).string(message.apiName);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseParseDirective();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.functionTag = reader.int32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.functionTemplate = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.resultParsing = exports.BlockParser.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.apiName = reader.string();
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
            functionTag: isSet(object.functionTag) ? functionTagFromJSON(object.functionTag) : 0,
            functionTemplate: isSet(object.functionTemplate) ? String(object.functionTemplate) : "",
            resultParsing: isSet(object.resultParsing) ? exports.BlockParser.fromJSON(object.resultParsing) : undefined,
            apiName: isSet(object.apiName) ? String(object.apiName) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.functionTag !== undefined && (obj.functionTag = functionTagToJSON(message.functionTag));
        message.functionTemplate !== undefined && (obj.functionTemplate = message.functionTemplate);
        message.resultParsing !== undefined &&
            (obj.resultParsing = message.resultParsing ? exports.BlockParser.toJSON(message.resultParsing) : undefined);
        message.apiName !== undefined && (obj.apiName = message.apiName);
        return obj;
    },
    create(base) {
        return exports.ParseDirective.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseParseDirective();
        message.functionTag = (_a = object.functionTag) !== null && _a !== void 0 ? _a : 0;
        message.functionTemplate = (_b = object.functionTemplate) !== null && _b !== void 0 ? _b : "";
        message.resultParsing = (object.resultParsing !== undefined && object.resultParsing !== null)
            ? exports.BlockParser.fromPartial(object.resultParsing)
            : undefined;
        message.apiName = (_c = object.apiName) !== null && _c !== void 0 ? _c : "";
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
                    if (tag != 10) {
                        break;
                    }
                    message.parserArg.push(reader.string());
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.parserFunc = reader.int32();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.defaultValue = reader.string();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.encoding = reader.string();
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
                    if (tag != 8) {
                        break;
                    }
                    message.deterministic = reader.bool();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.local = reader.bool();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.subscription = reader.bool();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.stateful = reader.uint32();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.hangingApi = reader.bool();
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
