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
const supportedChains_json_1 = __importDefault(require("../../supportedChains.json"));
const chains_1 = require("./chains");
describe("Make sure supportedChains.json is valid", () => {
    it("All entities are valid", () => __awaiter(void 0, void 0, void 0, function* () {
        supportedChains_json_1.default.filter((item) => {
            // Each entry needs to have 3 attributes (name, chainID, defaultRPC)
            expect(item.name).toBeDefined();
            expect(item.chainID).toBeDefined();
            expect(item.defaultRPC).toBeDefined();
            // No attribute can be empty
            expect(item.name).not.toBe("");
            expect(item.chainID).not.toBe("");
            expect(item.defaultRPC).not.toBe("");
            // For default jsonRPC only these values are supported [jsonrpc,tendermintrpc,rest]
            expect(["jsonrpc", "tendermintrpc", "rest"].includes(item.defaultRPC)).toBe(true);
        });
    }));
    it("No duplicates for same chainID", () => __awaiter(void 0, void 0, void 0, function* () {
        // Create map with all chainIDs
        const chainIDs = supportedChains_json_1.default.map((obj) => obj.chainID);
        // Create array of unique chainIDs
        const uniquechainIDs = Array.from(new Set(chainIDs));
        // SupportedChain.json and uniquechainIDs needs to be equal
        expect(uniquechainIDs.length).toBe(supportedChains_json_1.default.length);
    }));
});
describe("Test isNetworkValid", () => {
    it("Network is valid", () => __awaiter(void 0, void 0, void 0, function* () {
        expect((0, chains_1.isNetworkValid)("testnet")).toBe(true);
        expect((0, chains_1.isNetworkValid)("mainnet")).toBe(true);
    }));
    it("Network is not valid", () => __awaiter(void 0, void 0, void 0, function* () {
        expect((0, chains_1.isNetworkValid)("randomNetwork")).toBe(false);
    }));
});
