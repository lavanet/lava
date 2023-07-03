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
        const privKey = "36781325107ad46f9f432aa47a8c29030a7a4103fa063a8821cc75debdcf2bfe";
        const endpoint = "localhost:26657";
        const chainID = "LAV1";
        const rpcInterface = "tendermintrpc";
        // Create lavaSDK
        const lavaSDK = yield new sdk_1.default({
            privateKey: privKey,
            chainID: chainID,
            lavaEndpoint: endpoint,
            rpcInterface: rpcInterface, // Optional
        });
        // Send relay
        const statusResponse = yield lavaSDK.sendRelay({
            method: "status",
            params: [],
        });
        const blockResponse = yield lavaSDK.sendRelay({
            method: "block",
            params: ["5"],
        });
        // Print relay
        console.log("statusResponse", statusResponse);
        console.log("blockResponse", blockResponse);
        setTimeout(() => __awaiter(this, void 0, void 0, function* () {
            console.log("Same epoch");
            const statusResponse = yield lavaSDK.sendRelay({
                method: "status",
                params: [],
            });
            console.log("statusResponse", statusResponse);
            return;
        }), 5000);
        setTimeout(() => __awaiter(this, void 0, void 0, function* () {
            console.log("New epoch");
            const statusResponse = yield lavaSDK.sendRelay({
                method: "status",
                params: [],
            });
            console.log("statusResponse", statusResponse);
            return;
        }), 20000);
    });
}
run()
    .then()
    .catch((err) => {
    logger_1.default.error(err);
});
