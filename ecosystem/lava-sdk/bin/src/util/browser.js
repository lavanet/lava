"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const grpc_web_node_http_transport_1 = require("@improbable-eng/grpc-web-node-http-transport");
const grpc_web_1 = require("@improbable-eng/grpc-web");
let transport;
if (typeof window !== "undefined") {
    // We are running in a browser
    transport = grpc_web_1.grpc.CrossBrowserHttpTransport({ withCredentials: false });
}
else if (typeof process !== "undefined") {
    // We are running in Node.js
    transport = (0, grpc_web_node_http_transport_1.NodeHttpTransport)();
}
else {
    // If we are not running in the browser or node.js
    // We are running in a Web Worker
    // Assume the transport is same as for browser
    // Not tested
    transport = grpc_web_1.grpc.CrossBrowserHttpTransport({ withCredentials: false });
}
exports.default = transport;
