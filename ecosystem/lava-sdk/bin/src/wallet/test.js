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
const wallet_1 = require("./wallet");
const errors_1 = __importDefault(require("./errors"));
describe("Fetching account from private key", () => {
    it("Successfully fetch account", () => __awaiter(void 0, void 0, void 0, function* () {
        const privateKey = "885c3ebe355979d68d16f51e267040eb91e39021db07a9608ad881782d546009";
        const expectedAddress = "lava@194hjlf7swpm9c0rmktswt55p6xhhj6huzxnhaj";
        // Create lava wallet instance
        const lavaWallet = yield (0, wallet_1.createWallet)(privateKey);
        // Expect no error
        expect(() => __awaiter(void 0, void 0, void 0, function* () {
            yield lavaWallet.getConsumerAccount();
        })).not.toThrow(Error);
        // Fetch account
        const accountData = yield lavaWallet.getConsumerAccount();
        // Check if account address match expected address
        expect(accountData.address).toBe(expectedAddress);
    }));
    it("Invalid private key, can not create wallet", () => __awaiter(void 0, void 0, void 0, function* () {
        const privateKey = "";
        // Wallet was never initialized, expect error
        try {
            // Create lava wallet instance
            yield (0, wallet_1.createWallet)(privateKey);
        }
        catch (err) {
            expect(err).toBe(errors_1.default.errInvalidPrivateKey);
        }
    }));
});
