"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.QueryAllFixatedParamsResponse = exports.QueryAllFixatedParamsRequest = exports.QueryGetFixatedParamsResponse = exports.QueryGetFixatedParamsRequest = exports.QueryGetEpochDetailsResponse = exports.QueryGetEpochDetailsRequest = exports.QueryAllStakeStorageResponse = exports.QueryAllStakeStorageRequest = exports.QueryGetStakeStorageResponse = exports.QueryGetStakeStorageRequest = exports.QueryParamsResponse = exports.QueryParamsRequest = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const pagination_1 = require("../cosmos/base/query/v1beta1/pagination");
const epoch_details_1 = require("./epoch_details");
const fixated_params_1 = require("./fixated_params");
const params_1 = require("./params");
const stake_storage_1 = require("./stake_storage");
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseQueryParamsRequest() {
    return {};
}
exports.QueryParamsRequest = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsRequest();
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
        return exports.QueryParamsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseQueryParamsRequest();
        return message;
    },
};
function createBaseQueryParamsResponse() {
    return { params: undefined };
}
exports.QueryParamsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.params = params_1.Params.decode(reader, reader.uint32());
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
        return { params: isSet(object.params) ? params_1.Params.fromJSON(object.params) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryParamsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryParamsResponse();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        return message;
    },
};
function createBaseQueryGetStakeStorageRequest() {
    return { index: "" };
}
exports.QueryGetStakeStorageRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetStakeStorageRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
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
        return { index: isSet(object.index) ? String(object.index) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    create(base) {
        return exports.QueryGetStakeStorageRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryGetStakeStorageRequest();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryGetStakeStorageResponse() {
    return { stakeStorage: undefined };
}
exports.QueryGetStakeStorageResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.stakeStorage !== undefined) {
            stake_storage_1.StakeStorage.encode(message.stakeStorage, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetStakeStorageResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.stakeStorage = stake_storage_1.StakeStorage.decode(reader, reader.uint32());
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
        return { stakeStorage: isSet(object.stakeStorage) ? stake_storage_1.StakeStorage.fromJSON(object.stakeStorage) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.stakeStorage !== undefined &&
            (obj.stakeStorage = message.stakeStorage ? stake_storage_1.StakeStorage.toJSON(message.stakeStorage) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryGetStakeStorageResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryGetStakeStorageResponse();
        message.stakeStorage = (object.stakeStorage !== undefined && object.stakeStorage !== null)
            ? stake_storage_1.StakeStorage.fromPartial(object.stakeStorage)
            : undefined;
        return message;
    },
};
function createBaseQueryAllStakeStorageRequest() {
    return { pagination: undefined };
}
exports.QueryAllStakeStorageRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllStakeStorageRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
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
        return { pagination: isSet(object.pagination) ? pagination_1.PageRequest.fromJSON(object.pagination) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageRequest.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllStakeStorageRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryAllStakeStorageRequest();
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageRequest.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryAllStakeStorageResponse() {
    return { stakeStorage: [], pagination: undefined };
}
exports.QueryAllStakeStorageResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.stakeStorage) {
            stake_storage_1.StakeStorage.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllStakeStorageResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.stakeStorage.push(stake_storage_1.StakeStorage.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
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
            stakeStorage: Array.isArray(object === null || object === void 0 ? void 0 : object.stakeStorage)
                ? object.stakeStorage.map((e) => stake_storage_1.StakeStorage.fromJSON(e))
                : [],
            pagination: isSet(object.pagination) ? pagination_1.PageResponse.fromJSON(object.pagination) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.stakeStorage) {
            obj.stakeStorage = message.stakeStorage.map((e) => e ? stake_storage_1.StakeStorage.toJSON(e) : undefined);
        }
        else {
            obj.stakeStorage = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageResponse.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllStakeStorageResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryAllStakeStorageResponse();
        message.stakeStorage = ((_a = object.stakeStorage) === null || _a === void 0 ? void 0 : _a.map((e) => stake_storage_1.StakeStorage.fromPartial(e))) || [];
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageResponse.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryGetEpochDetailsRequest() {
    return {};
}
exports.QueryGetEpochDetailsRequest = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetEpochDetailsRequest();
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
        return exports.QueryGetEpochDetailsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseQueryGetEpochDetailsRequest();
        return message;
    },
};
function createBaseQueryGetEpochDetailsResponse() {
    return { EpochDetails: undefined };
}
exports.QueryGetEpochDetailsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.EpochDetails !== undefined) {
            epoch_details_1.EpochDetails.encode(message.EpochDetails, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetEpochDetailsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.EpochDetails = epoch_details_1.EpochDetails.decode(reader, reader.uint32());
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
        return { EpochDetails: isSet(object.EpochDetails) ? epoch_details_1.EpochDetails.fromJSON(object.EpochDetails) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.EpochDetails !== undefined &&
            (obj.EpochDetails = message.EpochDetails ? epoch_details_1.EpochDetails.toJSON(message.EpochDetails) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryGetEpochDetailsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryGetEpochDetailsResponse();
        message.EpochDetails = (object.EpochDetails !== undefined && object.EpochDetails !== null)
            ? epoch_details_1.EpochDetails.fromPartial(object.EpochDetails)
            : undefined;
        return message;
    },
};
function createBaseQueryGetFixatedParamsRequest() {
    return { index: "" };
}
exports.QueryGetFixatedParamsRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetFixatedParamsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.index = reader.string();
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
        return { index: isSet(object.index) ? String(object.index) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    create(base) {
        return exports.QueryGetFixatedParamsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryGetFixatedParamsRequest();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryGetFixatedParamsResponse() {
    return { fixatedParams: undefined };
}
exports.QueryGetFixatedParamsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.fixatedParams !== undefined) {
            fixated_params_1.FixatedParams.encode(message.fixatedParams, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryGetFixatedParamsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.fixatedParams = fixated_params_1.FixatedParams.decode(reader, reader.uint32());
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
        return { fixatedParams: isSet(object.fixatedParams) ? fixated_params_1.FixatedParams.fromJSON(object.fixatedParams) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.fixatedParams !== undefined &&
            (obj.fixatedParams = message.fixatedParams ? fixated_params_1.FixatedParams.toJSON(message.fixatedParams) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryGetFixatedParamsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryGetFixatedParamsResponse();
        message.fixatedParams = (object.fixatedParams !== undefined && object.fixatedParams !== null)
            ? fixated_params_1.FixatedParams.fromPartial(object.fixatedParams)
            : undefined;
        return message;
    },
};
function createBaseQueryAllFixatedParamsRequest() {
    return { pagination: undefined };
}
exports.QueryAllFixatedParamsRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllFixatedParamsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
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
        return { pagination: isSet(object.pagination) ? pagination_1.PageRequest.fromJSON(object.pagination) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageRequest.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllFixatedParamsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryAllFixatedParamsRequest();
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageRequest.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
function createBaseQueryAllFixatedParamsResponse() {
    return { fixatedParams: [], pagination: undefined };
}
exports.QueryAllFixatedParamsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.fixatedParams) {
            fixated_params_1.FixatedParams.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryAllFixatedParamsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.fixatedParams.push(fixated_params_1.FixatedParams.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
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
            fixatedParams: Array.isArray(object === null || object === void 0 ? void 0 : object.fixatedParams)
                ? object.fixatedParams.map((e) => fixated_params_1.FixatedParams.fromJSON(e))
                : [],
            pagination: isSet(object.pagination) ? pagination_1.PageResponse.fromJSON(object.pagination) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.fixatedParams) {
            obj.fixatedParams = message.fixatedParams.map((e) => e ? fixated_params_1.FixatedParams.toJSON(e) : undefined);
        }
        else {
            obj.fixatedParams = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination ? pagination_1.PageResponse.toJSON(message.pagination) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryAllFixatedParamsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryAllFixatedParamsResponse();
        message.fixatedParams = ((_a = object.fixatedParams) === null || _a === void 0 ? void 0 : _a.map((e) => fixated_params_1.FixatedParams.fromPartial(e))) || [];
        message.pagination = (object.pagination !== undefined && object.pagination !== null)
            ? pagination_1.PageResponse.fromPartial(object.pagination)
            : undefined;
        return message;
    },
};
class QueryClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.epochstorage.Query";
        this.rpc = rpc;
        this.Params = this.Params.bind(this);
        this.StakeStorage = this.StakeStorage.bind(this);
        this.StakeStorageAll = this.StakeStorageAll.bind(this);
        this.EpochDetails = this.EpochDetails.bind(this);
        this.FixatedParams = this.FixatedParams.bind(this);
        this.FixatedParamsAll = this.FixatedParamsAll.bind(this);
    }
    Params(request) {
        const data = exports.QueryParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Params", data);
        return promise.then((data) => exports.QueryParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    StakeStorage(request) {
        const data = exports.QueryGetStakeStorageRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "StakeStorage", data);
        return promise.then((data) => exports.QueryGetStakeStorageResponse.decode(minimal_1.default.Reader.create(data)));
    }
    StakeStorageAll(request) {
        const data = exports.QueryAllStakeStorageRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "StakeStorageAll", data);
        return promise.then((data) => exports.QueryAllStakeStorageResponse.decode(minimal_1.default.Reader.create(data)));
    }
    EpochDetails(request) {
        const data = exports.QueryGetEpochDetailsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "EpochDetails", data);
        return promise.then((data) => exports.QueryGetEpochDetailsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    FixatedParams(request) {
        const data = exports.QueryGetFixatedParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "FixatedParams", data);
        return promise.then((data) => exports.QueryGetFixatedParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    FixatedParamsAll(request) {
        const data = exports.QueryAllFixatedParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "FixatedParamsAll", data);
        return promise.then((data) => exports.QueryAllFixatedParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.QueryClientImpl = QueryClientImpl;
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
