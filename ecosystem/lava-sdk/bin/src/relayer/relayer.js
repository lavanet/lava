"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const crypto_1 = require("@cosmjs/crypto");
const encoding_1 = require("@cosmjs/encoding");
const grpc_web_1 = require("@improbable-eng/grpc-web");
const relay_pb_1 = require("../grpc_web_services/pairing/relay_pb");
const relay_pb_service_1 = require("../grpc_web_services/pairing/relay_pb_service");
const browser_1 = __importDefault(require("../util/browser"));
class Relayer {
    constructor(chainID, privKey, lavaChainId, secure, badge) {
        this.prefix = "http";
        this.byteArrayToString = (byteArray) => {
            let output = "";
            for (let i = 0; i < byteArray.length; i++) {
                const byte = byteArray[i];
                if (byte === 0x09) {
                    output += "\\t";
                }
                else if (byte === 0x0a) {
                    output += "\\n";
                }
                else if (byte === 0x0d) {
                    output += "\\r";
                }
                else if (byte === 0x5c) {
                    output += "\\\\";
                }
                else if (byte >= 0x20 && byte <= 0x7e) {
                    output += String.fromCharCode(byte);
                }
                else {
                    output += "\\" + byte.toString(8).padStart(3, "0");
                }
            }
            return output;
        };
        this.chainID = chainID;
        this.privKey = privKey;
        this.lavaChainId = lavaChainId;
        if (secure) {
            this.prefix = "https";
        }
        this.badge = badge;
    }
    sendRelay(options, consumerProviderSession, cuSum, apiInterface) {
        return __awaiter(this, void 0, void 0, function* () {
            // Extract attributes from options
            const { data, url, connectionType } = options;
            const enc = new TextEncoder();
            const consumerSession = consumerProviderSession.Session;
            // Increase used compute units
            consumerProviderSession.UsedComputeUnits =
                consumerProviderSession.UsedComputeUnits + cuSum;
            // create request private data
            const requestPrivateData = new relay_pb_1.RelayPrivateData();
            requestPrivateData.setConnectionType(connectionType);
            requestPrivateData.setApiUrl(url);
            requestPrivateData.setData(enc.encode(data));
            requestPrivateData.setRequestBlock(0);
            requestPrivateData.setApiInterface(apiInterface);
            requestPrivateData.setSalt(consumerSession.getNewSalt());
            const contentHash = this.calculateContentHashForRelayData(requestPrivateData);
            // create request session
            const requestSession = new relay_pb_1.RelaySession();
            requestSession.setSpecId(this.chainID);
            requestSession.setSessionId(consumerSession.getNewSessionId());
            requestSession.setCuSum(cuSum);
            requestSession.setProvider(consumerSession.ProviderAddress);
            requestSession.setRelayNum(consumerSession.RelayNum);
            requestSession.setEpoch(consumerSession.PairingEpoch);
            requestSession.setUnresponsiveProviders(new Uint8Array());
            requestSession.setContentHash(contentHash);
            requestSession.setSig(new Uint8Array());
            requestSession.setLavaChainId(this.lavaChainId);
            // Sign data
            const signedMessage = yield this.signRelay(requestSession, this.privKey);
            requestSession.setSig(signedMessage);
            if (this.badge) {
                // Badge is separated from the signature!
                requestSession.setBadge(this.badge);
            }
            // Create request
            const request = new relay_pb_1.RelayRequest();
            request.setRelaySession(requestSession);
            request.setRelayData(requestPrivateData);
            const requestPromise = new Promise((resolve, reject) => {
                grpc_web_1.grpc.invoke(relay_pb_service_1.Relayer.Relay, {
                    request: request,
                    host: this.prefix + "://" + consumerSession.Endpoint.Addr,
                    transport: browser_1.default,
                    onMessage: (message) => {
                        resolve(message);
                    },
                    onEnd: (code, msg) => {
                        if (code == grpc_web_1.grpc.Code.OK || msg == undefined) {
                            return;
                        }
                        // underflow guard
                        if (consumerProviderSession.UsedComputeUnits > cuSum) {
                            consumerProviderSession.UsedComputeUnits =
                                consumerProviderSession.UsedComputeUnits - cuSum;
                        }
                        else {
                            consumerProviderSession.UsedComputeUnits = 0;
                        }
                        let additionalInfo = "";
                        if (msg.includes("Response closed without headers")) {
                            additionalInfo =
                                additionalInfo +
                                    ", provider iPPORT: " +
                                    consumerProviderSession.Session.Endpoint.Addr +
                                    ", provider address: " +
                                    consumerProviderSession.Session.ProviderAddress;
                        }
                        const errMessage = this.extractErrorMessage(msg) + additionalInfo;
                        reject(new Error(errMessage));
                    },
                });
            });
            return this.relayWithTimeout(2000, requestPromise);
        });
    }
    extractErrorMessage(error) {
        // Regular expression to match the desired pattern
        const regex = /desc = (.*?)(?=:\s*rpc error|$)/s;
        // Try to match the error message to the regular expression
        const match = error.match(regex);
        // If there is a match, return it; otherwise return the original error message
        if (match && match[1]) {
            return match[1].trim();
        }
        else {
            return error;
        }
    }
    relayWithTimeout(timeLimit, task) {
        return __awaiter(this, void 0, void 0, function* () {
            let timeout;
            const timeoutPromise = new Promise((resolve, reject) => {
                timeout = setTimeout(() => {
                    reject(new Error("Timeout exceeded"));
                }, timeLimit);
            });
            const response = yield Promise.race([task, timeoutPromise]);
            if (timeout) {
                //the code works without this but let's be safe and clean up the timeout
                clearTimeout(timeout);
            }
            return response;
        });
    }
    // Sign relay request using priv key
    signRelay(request, privKey) {
        return __awaiter(this, void 0, void 0, function* () {
            const message = this.prepareRequest(request);
            const sig = yield crypto_1.Secp256k1.createSignature(message, (0, encoding_1.fromHex)(privKey));
            const recovery = sig.recovery;
            const r = sig.r(32); // if r is not 32 bytes, add padding
            const s = sig.s(32); // if s is not 32 bytes, add padding
            // TODO consider adding compression in the signing
            // construct signature
            // <(byte of 27+public key solution)>< padded bytes for signature R><padded bytes for signature S>
            return Uint8Array.from([27 + recovery, ...r, ...s]);
        });
    }
    calculateContentHashForRelayData(relayRequestData) {
        const requestBlock = relayRequestData.getRequestBlock();
        const requestBlockBytes = this.convertRequestedBlockToUint8Array(requestBlock);
        const apiInterfaceBytes = this.encodeUtf8(relayRequestData.getApiInterface());
        const connectionTypeBytes = this.encodeUtf8(relayRequestData.getConnectionType());
        const apiUrlBytes = this.encodeUtf8(relayRequestData.getApiUrl());
        const dataBytes = relayRequestData.getData();
        const dataUint8Array = dataBytes instanceof Uint8Array ? dataBytes : this.encodeUtf8(dataBytes);
        const saltBytes = relayRequestData.getSalt();
        const saltUint8Array = saltBytes instanceof Uint8Array ? saltBytes : this.encodeUtf8(saltBytes);
        const msgData = this.concatUint8Arrays([
            apiInterfaceBytes,
            connectionTypeBytes,
            apiUrlBytes,
            dataUint8Array,
            requestBlockBytes,
            saltUint8Array,
        ]);
        const hash = (0, crypto_1.sha256)(msgData);
        return hash;
    }
    convertRequestedBlockToUint8Array(requestBlock) {
        const requestBlockBytes = new Uint8Array(8);
        let number = BigInt(requestBlock);
        if (requestBlock < 0) {
            // Convert the number to its 64-bit unsigned representation
            const maxUint64 = BigInt(2) ** BigInt(64);
            number = maxUint64 + BigInt(requestBlock);
        }
        // Copy the bytes from the unsigned representation to the byte array
        for (let i = 0; i < 8; i++) {
            requestBlockBytes[i] = Number((number >> BigInt(8 * i)) & BigInt(0xff));
        }
        return requestBlockBytes;
    }
    encodeUtf8(str) {
        return new TextEncoder().encode(str);
    }
    concatUint8Arrays(arrays) {
        const totalLength = arrays.reduce((acc, arr) => acc + arr.length, 0);
        const result = new Uint8Array(totalLength);
        let offset = 0;
        arrays.forEach((arr) => {
            result.set(arr, offset);
            offset += arr.length;
        });
        return result;
    }
    prepareRequest(request) {
        const enc = new TextEncoder();
        const jsonMessage = JSON.stringify(request.toObject(), (key, value) => {
            if (key == "content_hash") {
                const dataBytes = request.getContentHash();
                const dataUint8Array = dataBytes instanceof Uint8Array
                    ? dataBytes
                    : this.encodeUtf8(dataBytes);
                let stringByte = this.byteArrayToString(dataUint8Array);
                if (stringByte.endsWith(",")) {
                    stringByte += ",";
                }
                return stringByte;
            }
            if (value !== null && value !== 0 && value !== "")
                return value;
        });
        const messageReplaced = jsonMessage
            .replace(/\\\\/g, "\\")
            .replace(/,"/g, ' "')
            .replace(/, "/g, ',"')
            .replace(/"(\w+)"\s*:/g, "$1:")
            .slice(1, -1);
        const encodedMessage = enc.encode(messageReplaced + " ");
        const hash = (0, crypto_1.sha256)(encodedMessage);
        return hash;
    }
}
exports.default = Relayer;
