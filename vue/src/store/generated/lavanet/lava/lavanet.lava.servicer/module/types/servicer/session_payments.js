/* eslint-disable */
import { UserPaymentStorage } from "../servicer/user_payment_storage";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "lavanet.lava.servicer";
const baseSessionPayments = { index: "" };
export const SessionPayments = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        for (const v of message.usersPayments) {
            UserPaymentStorage.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseSessionPayments };
        message.usersPayments = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                case 2:
                    message.usersPayments.push(UserPaymentStorage.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseSessionPayments };
        message.usersPayments = [];
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        if (object.usersPayments !== undefined && object.usersPayments !== null) {
            for (const e of object.usersPayments) {
                message.usersPayments.push(UserPaymentStorage.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        if (message.usersPayments) {
            obj.usersPayments = message.usersPayments.map((e) => e ? UserPaymentStorage.toJSON(e) : undefined);
        }
        else {
            obj.usersPayments = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseSessionPayments };
        message.usersPayments = [];
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        if (object.usersPayments !== undefined && object.usersPayments !== null) {
            for (const e of object.usersPayments) {
                message.usersPayments.push(UserPaymentStorage.fromPartial(e));
            }
        }
        return message;
    },
};
