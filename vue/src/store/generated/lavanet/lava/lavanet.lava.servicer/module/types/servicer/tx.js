/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { SpecName } from "../servicer/spec_name";
import { Coin } from "../cosmos/base/v1beta1/coin";
import { BlockNum } from "../servicer/block_num";
import { RelayRequest } from "../servicer/relay";
export const protobufPackage = "lavanet.lava.servicer";
const baseMsgStakeServicer = { creator: "", operatorAddresses: "" };
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
        for (const v of message.operatorAddresses) {
            writer.uint32(42).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseMsgStakeServicer };
        message.operatorAddresses = [];
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
                case 5:
                    message.operatorAddresses.push(reader.string());
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
        message.operatorAddresses = [];
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
        if (object.operatorAddresses !== undefined &&
            object.operatorAddresses !== null) {
            for (const e of object.operatorAddresses) {
                message.operatorAddresses.push(String(e));
            }
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
        if (message.operatorAddresses) {
            obj.operatorAddresses = message.operatorAddresses.map((e) => e);
        }
        else {
            obj.operatorAddresses = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseMsgStakeServicer };
        message.operatorAddresses = [];
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
        if (object.operatorAddresses !== undefined &&
            object.operatorAddresses !== null) {
            for (const e of object.operatorAddresses) {
                message.operatorAddresses.push(e);
            }
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
const baseMsgProofOfWork = { creator: "" };
export const MsgProofOfWork = {
    encode(message, writer = Writer.create()) {
        if (message.creator !== "") {
            writer.uint32(10).string(message.creator);
        }
        for (const v of message.relays) {
            RelayRequest.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseMsgProofOfWork };
        message.relays = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.creator = reader.string();
                    break;
                case 2:
                    message.relays.push(RelayRequest.decode(reader, reader.uint32()));
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
        message.relays = [];
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = String(object.creator);
        }
        else {
            message.creator = "";
        }
        if (object.relays !== undefined && object.relays !== null) {
            for (const e of object.relays) {
                message.relays.push(RelayRequest.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.creator !== undefined && (obj.creator = message.creator);
        if (message.relays) {
            obj.relays = message.relays.map((e) => e ? RelayRequest.toJSON(e) : undefined);
        }
        else {
            obj.relays = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseMsgProofOfWork };
        message.relays = [];
        if (object.creator !== undefined && object.creator !== null) {
            message.creator = object.creator;
        }
        else {
            message.creator = "";
        }
        if (object.relays !== undefined && object.relays !== null) {
            for (const e of object.relays) {
                message.relays.push(RelayRequest.fromPartial(e));
            }
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
