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
const logger_1 = __importDefault(require("../logger/logger"));
// Fetch from lava-sdk package
const sdk_1 = __importDefault(require("../sdk/sdk"));
function run() {
    return __awaiter(this, void 0, void 0, function* () {
        const privKey = "9deaba87285fdbfc65024731a319bacf49aa12e9147927ce3dac613395420213";
        const endpoint = "localhost:26657";
        const chainID = "LAV1";
        const rpcInterface = "rest";
        // Create lavaSDK
        const lavaSDK = yield new sdk_1.default({
            privateKey: privKey,
            chainID: chainID,
            lavaEndpoint: endpoint,
            rpcInterface: rpcInterface, // Optional
        });
        // Send rest relay
        const latestBlock = yield lavaSDK.sendRestRelay({
            method: "GET",
            url: "/blocks/latest",
        });
        console.log("latest block", latestBlock);
        const data = yield lavaSDK.sendRestRelay({
            method: "GET",
            url: "/cosmos/bank/v1beta1/denoms_metadata",
            data: {
                "pagination.count_total": true,
                "pagination.reverse": "true",
            },
        });
        console.log("data", data);
    });
}
run()
    .then()
    .catch((err) => {
    logger_1.default.error(err);
});
