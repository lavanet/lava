"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.GenesisState = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const epoch_details_1 = require("./epoch_details");
const fixated_params_1 = require("./fixated_params");
const params_1 = require("./params");
const stake_storage_1 = require("./stake_storage");
exports.protobufPackage = "lavanet.lava.epochstorage";
function createBaseGenesisState() {
    return { params: undefined, stakeStorageList: [], epochDetails: undefined, fixatedParamsList: [] };
}
exports.GenesisState = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.stakeStorageList) {
            stake_storage_1.StakeStorage.encode(v, writer.uint32(18).fork()).ldelim();
        }
        if (message.epochDetails !== undefined) {
            epoch_details_1.EpochDetails.encode(message.epochDetails, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.fixatedParamsList) {
            fixated_params_1.FixatedParams.encode(v, writer.uint32(34).fork()).ldelim();
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
                    message.stakeStorageList.push(stake_storage_1.StakeStorage.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.epochDetails = epoch_details_1.EpochDetails.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.fixatedParamsList.push(fixated_params_1.FixatedParams.decode(reader, reader.uint32()));
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
            stakeStorageList: Array.isArray(object === null || object === void 0 ? void 0 : object.stakeStorageList)
                ? object.stakeStorageList.map((e) => stake_storage_1.StakeStorage.fromJSON(e))
                : [],
            epochDetails: isSet(object.epochDetails) ? epoch_details_1.EpochDetails.fromJSON(object.epochDetails) : undefined,
            fixatedParamsList: Array.isArray(object === null || object === void 0 ? void 0 : object.fixatedParamsList)
                ? object.fixatedParamsList.map((e) => fixated_params_1.FixatedParams.fromJSON(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        if (message.stakeStorageList) {
            obj.stakeStorageList = message.stakeStorageList.map((e) => e ? stake_storage_1.StakeStorage.toJSON(e) : undefined);
        }
        else {
            obj.stakeStorageList = [];
        }
        message.epochDetails !== undefined &&
            (obj.epochDetails = message.epochDetails ? epoch_details_1.EpochDetails.toJSON(message.epochDetails) : undefined);
        if (message.fixatedParamsList) {
            obj.fixatedParamsList = message.fixatedParamsList.map((e) => e ? fixated_params_1.FixatedParams.toJSON(e) : undefined);
        }
        else {
            obj.fixatedParamsList = [];
        }
        return obj;
    },
    create(base) {
        return exports.GenesisState.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseGenesisState();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        message.stakeStorageList = ((_a = object.stakeStorageList) === null || _a === void 0 ? void 0 : _a.map((e) => stake_storage_1.StakeStorage.fromPartial(e))) || [];
        message.epochDetails = (object.epochDetails !== undefined && object.epochDetails !== null)
            ? epoch_details_1.EpochDetails.fromPartial(object.epochDetails)
            : undefined;
        message.fixatedParamsList = ((_b = object.fixatedParamsList) === null || _b === void 0 ? void 0 : _b.map((e) => fixated_params_1.FixatedParams.fromPartial(e))) || [];
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
