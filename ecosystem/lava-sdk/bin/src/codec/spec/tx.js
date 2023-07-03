"use strict";
/* eslint-disable */
Object.defineProperty(exports, "__esModule", { value: true });
exports.MsgClientImpl = exports.protobufPackage = void 0;
exports.protobufPackage = "lavanet.lava.spec";
class MsgClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.spec.Msg";
        this.rpc = rpc;
    }
}
exports.MsgClientImpl = MsgClientImpl;
