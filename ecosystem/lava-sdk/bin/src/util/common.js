"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.generateRPCData = exports.base64ToUint8Array = void 0;
function base64ToUint8Array(str) {
    const buffer = Buffer.from(str, "base64");
    return new Uint8Array(buffer);
}
exports.base64ToUint8Array = base64ToUint8Array;
function generateRPCData(method, params) {
    const stringifyMethod = JSON.stringify(method);
    const stringifyParam = JSON.stringify(params, (key, value) => {
        if (typeof value === "bigint") {
            return value.toString();
        }
        return value;
    });
    // TODO make id changable
    return ('{"jsonrpc": "2.0", "id": 1, "method": ' +
        stringifyMethod +
        ', "params": ' +
        stringifyParam +
        "}");
}
exports.generateRPCData = generateRPCData;
