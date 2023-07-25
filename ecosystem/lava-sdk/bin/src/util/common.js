"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseLong = exports.generateRPCData = exports.base64ToUint8Array = void 0;
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
function parseLong(long) {
    /**
     * this function will parse long to a 64bit number,
     * this assumes all systems running the sdk will run on 64bit systems
     * @param long A long number to parse into number
     */
    const high = Number(long.high);
    const low = Number(long.low);
    const parsedNumber = (high << 32) + low;
    if (high > 0) {
        console.log("MAYBE AN ISSUE", high);
    }
    return parsedNumber;
}
exports.parseLong = parseLong;
