"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
// import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
const transportNode_1 = require("./transportNode");
const grpc_web_1 = require("@improbable-eng/grpc-web");
let transport;
if (typeof window !== "undefined") {
    // We are running in a browser
    transport = grpc_web_1.grpc.CrossBrowserHttpTransport({ withCredentials: false });
}
else if (typeof process !== "undefined") {
    // We are running in Node.js
    transport = (0, transportNode_1.NodeHttpTransport)();
    // transportAllowInsecure = NodeHttpTransport({ rejectUnauthorized: false });
}
else {
    // If we are not running in the browser or node.js
    // We are running in a Web Worker
    // Assume the transport is same as for browser
    // Not tested
    transport = grpc_web_1.grpc.CrossBrowserHttpTransport({ withCredentials: false });
}
exports.default = transport;
