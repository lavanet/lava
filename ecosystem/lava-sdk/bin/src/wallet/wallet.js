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
exports.createDynamicWallet = exports.createWallet = exports.LavaWallet = void 0;
const amino_1 = require("@cosmjs/amino");
const launchpad_1 = require("@cosmjs/launchpad");
const errors_1 = __importDefault(require("./errors"));
const logger_1 = __importDefault(require("../logger/logger"));
const encoding_1 = require("@cosmjs/encoding");
const crypto_1 = require("@cosmjs/crypto");
const encoding_2 = require("@cosmjs/encoding");
// prefix for lava accounts
const lavaPrefix = "lava@";
class LavaWallet {
    constructor(privKey) {
        this.privKey = privKey;
        this.wallet = errors_1.default.errWalletNotInitialized;
    }
    // Initialize client
    init() {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                this.wallet = yield amino_1.Secp256k1Wallet.fromKey((0, encoding_1.fromHex)(this.privKey), lavaPrefix);
            }
            catch (err) {
                throw errors_1.default.errInvalidPrivateKey;
            }
        });
    }
    // Get consumer account from the wallet
    getConsumerAccount() {
        return __awaiter(this, void 0, void 0, function* () {
            try {
                // Check if wallet was initialized
                if (this.wallet instanceof Error) {
                    throw errors_1.default.errWalletNotInitialized;
                }
                // Fetch account
                const account = yield this.wallet.getAccounts();
                // Check if zero account exists
                if (account[0] == undefined) {
                    throw errors_1.default.errZeroAccountDoesNotExists;
                }
                // Return zero account from wallet
                return account[0];
            }
            catch (err) {
                throw err;
            }
        });
    }
    // Print account details
    printAccount(AccountData) {
        logger_1.default.info("INFO:");
        logger_1.default.info("Address: " + AccountData.address);
        logger_1.default.info("Public key: " + AccountData.pubkey);
        logger_1.default.emptyLine();
    }
}
exports.LavaWallet = LavaWallet;
function createWallet(privKey) {
    return __awaiter(this, void 0, void 0, function* () {
        // Create lavaSDK
        const wallet = new LavaWallet(privKey);
        // Initialize wallet
        yield wallet.init();
        return wallet;
    });
}
exports.createWallet = createWallet;
function createDynamicWallet() {
    return __awaiter(this, void 0, void 0, function* () {
        const walletWithRandomSeed = yield launchpad_1.Secp256k1HdWallet.generate(undefined, {
            prefix: lavaPrefix,
        });
        const walletPrivKey = yield getWalletPrivateKey(walletWithRandomSeed.mnemonic);
        const privKey = Array.from(walletPrivKey.privkey)
            .map((byte) => byte.toString(16).padStart(2, "0"))
            .join("");
        const wallet = yield createWallet(privKey);
        return { wallet, privKey };
    });
}
exports.createDynamicWallet = createDynamicWallet;
function getWalletPrivateKey(walletMnemonic) {
    return __awaiter(this, void 0, void 0, function* () {
        const { privkey, pubkey } = yield getKeyPair([crypto_1.Slip10RawIndex.normal(0)], walletMnemonic);
        const address = (0, encoding_2.toBech32)(lavaPrefix, rawSecp256k1PubkeyToRawAddress(pubkey));
        return {
            algo: "secp256k1",
            privkey: privkey,
            pubkey: pubkey,
            address: address,
        };
    });
}
function getKeyPair(hdPath, walletMnemonic) {
    return __awaiter(this, void 0, void 0, function* () {
        const mnemonicChecked = new crypto_1.EnglishMnemonic(walletMnemonic);
        const seed = yield crypto_1.Bip39.mnemonicToSeed(mnemonicChecked);
        const { privkey } = crypto_1.Slip10.derivePath(crypto_1.Slip10Curve.Secp256k1, seed, hdPath);
        const { pubkey } = yield crypto_1.Secp256k1.makeKeypair(privkey);
        return {
            privkey: privkey,
            pubkey: crypto_1.Secp256k1.compressPubkey(pubkey),
        };
    });
}
function rawSecp256k1PubkeyToRawAddress(pubkeyData) {
    if (pubkeyData.length !== 33) {
        throw new Error(`Invalid Secp256k1 pubkey length (compressed): ${pubkeyData.length}`);
    }
    return (0, crypto_1.ripemd160)((0, crypto_1.sha256)(pubkeyData));
}
