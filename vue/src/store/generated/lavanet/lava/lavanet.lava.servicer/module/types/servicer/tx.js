/* eslint-disable */
import { Reader, util, configure, Writer } from "protobufjs/minimal";
import * as Long from "long";
import { SpecName } from "../servicer/spec_name";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../servicer/block_num";
import { SessionID } from "../servicer/session_id";
import { ClientRequest } from "../servicer/client_request";
import { WorkProof } from "../servicer/work_proof";
export const protobufPackage = "lavanet.lava.servicer";
const baseMsgStakeServicer = { creator: "" };
export const MsgStakeServicer = {
    encode(message, writer = Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.spec !== undefined) {
            SpecName.encode(message.spec, writer.uint32(18).fork()).ldelim();
        }
        if (message.amount !== undefined) {
            Coin.encode(message.amount, writer.uint32(26).fork()).ldelim();
        }
        if (message.deadline !== undefined) {
            BlockNum.encode(message.deadline, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseMsgStakeServicer };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.creator = reader.string();
                    break;
                case 2:
                    message.spec = SpecName.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.amount = Coin.decode(reader, reader.uint32());
                    break;
                case 4:
                    message.deadline = BlockNum.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseMsgStakeServicer };
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = String(object.creator);
        }
        else {
            message.creator = "";
        }
        if (object.spec !== undefined && object.spec !== null) {
            message.spec = SpecName.fromJSON(object.spec);
        }
        else {
            message.spec = undefined;
        }
        if (object.amount !== undefined && object.amount !== null) {
            message.amount = Coin.fromJSON(object.amount);
        }
        else {
            message.amount = undefined;
        }
        if (object.deadline !== undefined && object.deadline !== null) {
            message.deadline = BlockNum.fromJSON(object.deadline);
        }
        else {
            message.deadline = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.spec !== undefined &&
            (obj.spec = message.spec ? SpecName.toJSON(message.spec) : undefined);
        message.amount !== undefined &&
            (obj.amount = message.amount ? Coin.toJSON(message.amount) : undefined);
        message.deadline !== undefined &&
            (obj.deadline = message.deadline
                ? BlockNum.toJSON(message.deadline)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseMsgStakeServicer };
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = object.creator;
        }
        else {
            message.creator = "";
        }
        if (object.spec !== undefined && object.spec !== null) {
            message.spec = SpecName.fromPartial(object.spec);
        }
        else {
            message.spec = undefined;
        }
        if (object.amount !== undefined && object.amount !== null) {
            message.amount = Coin.fromPartial(object.amount);
        }
        else {
            message.amount = undefined;
        }
        if (object.deadline !== undefined && object.deadline !== null) {
            message.deadline = BlockNum.fromPartial(object.deadline);
        }
        else {
            message.deadline = undefined;
        }
        return message;
    },
};
const baseMsgStakeServicerResponse = {};
export const MsgStakeServicerResponse = {
    encode(_, writer = Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseMsgStakeServicerResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = {
            ...baseMsgStakeServicerResponse,
        };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = {
            ...baseMsgStakeServicerResponse,
        };
        return message;
    },
};
const baseMsgUnstakeServicer = { creator: "" };
export const MsgUnstakeServicer = {
    encode(message, writer = Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.spec !== undefined) {
            SpecName.encode(message.spec, writer.uint32(18).fork()).ldelim();
        }
        if (message.deadline !== undefined) {
            BlockNum.encode(message.deadline, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseMsgUnstakeServicer };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.creator = reader.string();
                    break;
                case 2:
                    message.spec = SpecName.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.deadline = BlockNum.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseMsgUnstakeServicer };
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = String(object.creator);
        }
        else {
            message.creator = "";
        }
        if (object.spec !== undefined && object.spec !== null) {
            message.spec = SpecName.fromJSON(object.spec);
        }
        else {
            message.spec = undefined;
        }
        if (object.deadline !== undefined && object.deadline !== null) {
            message.deadline = BlockNum.fromJSON(object.deadline);
        }
        else {
            message.deadline = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.spec !== undefined &&
            (obj.spec = message.spec ? SpecName.toJSON(message.spec) : undefined);
        message.deadline !== undefined &&
            (obj.deadline = message.deadline
                ? BlockNum.toJSON(message.deadline)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseMsgUnstakeServicer };
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = object.creator;
        }
        else {
            message.creator = "";
        }
        if (object.spec !== undefined && object.spec !== null) {
            message.spec = SpecName.fromPartial(object.spec);
        }
        else {
            message.spec = undefined;
        }
        if (object.deadline !== undefined && object.deadline !== null) {
            message.deadline = BlockNum.fromPartial(object.deadline);
        }
        else {
            message.deadline = undefined;
        }
        return message;
    },
};
const baseMsgUnstakeServicerResponse = {};
export const MsgUnstakeServicerResponse = {
    encode(_, writer = Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseMsgUnstakeServicerResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = {
            ...baseMsgUnstakeServicerResponse,
        };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = {
            ...baseMsgUnstakeServicerResponse,
        };
        return message;
    },
};
const baseMsgProofOfWork = { creator: "", computeUnits: 0 };
export const MsgProofOfWork = {
    encode(message, writer = Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        if (message.spec !== undefined) {
            SpecName.encode(message.spec, writer.uint32(18).fork()).ldelim();
        }
        if (message.session !== undefined) {
            SessionID.encode(message.session, writer.uint32(26).fork()).ldelim();
        }
        if (message.clientRequest !== undefined) {
            ClientRequest.encode(message.clientRequest, writer.uint32(34).fork()).ldelim();
        }
        if (message.workProof !== undefined) {
            WorkProof.encode(message.workProof, writer.uint32(42).fork()).ldelim();
        }
        if (message.computeUnits !== 0) {
            writer.uint32(48).uint64(message.computeUnits);
        }
        if (message.blockOfWork !== undefined) {
            BlockNum.encode(message.blockOfWork, writer.uint32(58).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseMsgProofOfWork };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.creator = reader.string();
                    break;
                case 2:
                    message.spec = SpecName.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.session = SessionID.decode(reader, reader.uint32());
                    break;
                case 4:
                    message.clientRequest = ClientRequest.decode(reader, reader.uint32());
                    break;
                case 5:
                    message.workProof = WorkProof.decode(reader, reader.uint32());
                    break;
                case 6:
                    message.computeUnits = longToNumber(reader.uint64());
                    break;
                case 7:
                    message.blockOfWork = BlockNum.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseMsgProofOfWork };
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = String(object.creator);
        }
        else {
            message.creator = "";
        }
        if (object.spec !== undefined && object.spec !== null) {
            message.spec = SpecName.fromJSON(object.spec);
        }
        else {
            message.spec = undefined;
        }
        if (object.session !== undefined && object.session !== null) {
            message.session = SessionID.fromJSON(object.session);
        }
        else {
            message.session = undefined;
        }
        if (object.clientRequest !== undefined && object.clientRequest !== null) {
            message.clientRequest = ClientRequest.fromJSON(object.clientRequest);
        }
        else {
            message.clientRequest = undefined;
        }
        if (object.workProof !== undefined && object.workProof !== null) {
            message.workProof = WorkProof.fromJSON(object.workProof);
        }
        else {
            message.workProof = undefined;
        }
        if (object.computeUnits !== undefined && object.computeUnits !== null) {
            message.computeUnits = Number(object.computeUnits);
        }
        else {
            message.computeUnits = 0;
        }
        if (object.blockOfWork !== undefined && object.blockOfWork !== null) {
            message.blockOfWork = BlockNum.fromJSON(object.blockOfWork);
        }
        else {
            message.blockOfWork = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        message.spec !== undefined &&
            (obj.spec = message.spec ? SpecName.toJSON(message.spec) : undefined);
        message.session !== undefined &&
            (obj.session = message.session
                ? SessionID.toJSON(message.session)
                : undefined);
        message.clientRequest !== undefined &&
            (obj.clientRequest = message.clientRequest
                ? ClientRequest.toJSON(message.clientRequest)
                : undefined);
        message.workProof !== undefined &&
            (obj.workProof = message.workProof
                ? WorkProof.toJSON(message.workProof)
                : undefined);
        message.computeUnits !== undefined &&
            (obj.computeUnits = message.computeUnits);
        message.blockOfWork !== undefined &&
            (obj.blockOfWork = message.blockOfWork
                ? BlockNum.toJSON(message.blockOfWork)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseMsgProofOfWork };
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = object.creator;
        }
        else {
            message.creator = "";
        }
        if (object.spec !== undefined && object.spec !== null) {
            message.spec = SpecName.fromPartial(object.spec);
        }
        else {
            message.spec = undefined;
        }
        if (object.session !== undefined && object.session !== null) {
            message.session = SessionID.fromPartial(object.session);
        }
        else {
            message.session = undefined;
        }
        if (object.clientRequest !== undefined && object.clientRequest !== null) {
            message.clientRequest = ClientRequest.fromPartial(object.clientRequest);
        }
        else {
            message.clientRequest = undefined;
        }
        if (object.workProof !== undefined && object.workProof !== null) {
            message.workProof = WorkProof.fromPartial(object.workProof);
        }
        else {
            message.workProof = undefined;
        }
        if (object.computeUnits !== undefined && object.computeUnits !== null) {
            message.computeUnits = object.computeUnits;
        }
        else {
            message.computeUnits = 0;
        }
        if (object.blockOfWork !== undefined && object.blockOfWork !== null) {
            message.blockOfWork = BlockNum.fromPartial(object.blockOfWork);
        }
        else {
            message.blockOfWork = undefined;
        }
        return message;
    },
};
const baseMsgProofOfWorkResponse = {};
export const MsgProofOfWorkResponse = {
    encode(_, writer = Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseMsgProofOfWorkResponse };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = { ...baseMsgProofOfWorkResponse };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = { ...baseMsgProofOfWorkResponse };
        return message;
    },
};
export class MsgClientImpl {
    constructor(rpc) {
        this.rpc = rpc;
    }
    StakeServicer(request) {
        const data = MsgStakeServicer.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Msg", "StakeServicer", data);
        return promise.then((data) => MsgStakeServicerResponse.decode(new Reader(data)));
    }
    UnstakeServicer(request) {
        const data = MsgUnstakeServicer.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Msg", "UnstakeServicer", data);
        return promise.then((data) => MsgUnstakeServicerResponse.decode(new Reader(data)));
    }
    ProofOfWork(request) {
        const data = MsgProofOfWork.encode(request).finish();
        const promise = this.rpc.request("lavanet.lava.servicer.Msg", "ProofOfWork", data);
        return promise.then((data) => MsgProofOfWorkResponse.decode(new Reader(data)));
    }
}
var globalThis = (() => {
    if (typeof globalThis !== "undefined")
        return globalThis;
    if (typeof self !== "undefined")
        return self;
    if (typeof window !== "undefined")
        return window;
    if (typeof global !== "undefined")
        return global;
    throw "Unable to locate global object";
})();
function longToNumber(long) {
    if (long.gt(Number.MAX_SAFE_INTEGER)) {
        throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
    }
    return long.toNumber();
}
if (util.Long !== Long) {
    util.Long = Long;
    configure();
}
