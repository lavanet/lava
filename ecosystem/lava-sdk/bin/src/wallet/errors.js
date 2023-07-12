"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
class WalletErrors {
}
WalletErrors.errWalletNotInitialized = new Error("Wallet was not initialized");
WalletErrors.errZeroAccountDoesNotExists = new Error("Zero account does not exists in wallet");
WalletErrors.errInvalidPrivateKey = new Error("Invalid private key");
exports.default = WalletErrors;
