"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.NodeHttpTransport = void 0;
const http = __importStar(require("http"));
const https = __importStar(require("https"));
const url = __importStar(require("url"));
const grpc_web_1 = require("@improbable-eng/grpc-web");
function NodeHttpTransport(httpsOptions) {
    return (opts) => {
        return new NodeHttp(opts, httpsOptions);
    };
}
exports.NodeHttpTransport = NodeHttpTransport;
class NodeHttp {
    constructor(transportOptions, httpsOptions) {
        this.httpsOptions = httpsOptions;
        this.options = transportOptions;
    }
    sendMessage(msgBytes) {
        if (!this.options.methodDefinition.requestStream &&
            !this.options.methodDefinition.responseStream) {
            // Disable chunked encoding if we are not using streams
            this.request.setHeader("Content-Length", msgBytes.byteLength);
        }
        this.request.write(toBuffer(msgBytes));
        this.request.end();
    }
    finishSend() {
        return;
    }
    responseCallback(response) {
        this.options.debug && console.log("NodeHttp.response", response.statusCode);
        const headers = filterHeadersForUndefined(response.headers);
        this.options.onHeaders(new grpc_web_1.grpc.Metadata(headers), response.statusCode);
        response.on("data", (chunk) => {
            this.options.debug && console.log("NodeHttp.data", chunk);
            this.options.onChunk(toArrayBuffer(chunk));
        });
        response.on("end", () => {
            this.options.debug && console.log("NodeHttp.end");
            this.options.onEnd();
        });
    }
    start(metadata) {
        const headers = {};
        metadata.forEach((key, values) => {
            headers[key] = values.join(", ");
        });
        const parsedUrl = url.parse(this.options.url);
        const httpOptions = {
            host: parsedUrl.hostname,
            port: parsedUrl.port ? parseInt(parsedUrl.port) : undefined,
            path: parsedUrl.path,
            headers: headers,
            method: "POST",
        };
        if (parsedUrl.protocol === "https:") {
            this.request = https.request(Object.assign(Object.assign({}, httpOptions), this === null || this === void 0 ? void 0 : this.httpsOptions), this.responseCallback.bind(this));
        }
        else {
            this.request = http.request(httpOptions, this.responseCallback.bind(this));
        }
        this.request.on("error", (err) => {
            this.options.debug && console.log("NodeHttp.error", err);
            this.options.onEnd(err);
        });
    }
    cancel() {
        this.options.debug && console.log("NodeHttp.abort");
        this.request.abort();
    }
}
function filterHeadersForUndefined(headers) {
    const filteredHeaders = {};
    for (const key in headers) {
        const value = headers[key];
        if (headers.hasOwnProperty(key)) {
            if (value !== undefined) {
                filteredHeaders[key] = value;
            }
        }
    }
    return filteredHeaders;
}
function toArrayBuffer(buf) {
    const view = new Uint8Array(buf.length);
    for (let i = 0; i < buf.length; i++) {
        view[i] = buf[i];
    }
    return view;
}
function toBuffer(ab) {
    const buf = Buffer.alloc(ab.byteLength);
    for (let i = 0; i < buf.length; i++) {
        buf[i] = ab[i];
    }
    return buf;
}
