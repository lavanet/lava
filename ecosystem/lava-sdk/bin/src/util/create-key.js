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
exports.sleep = exports.createDeveloperKey = exports.getKeys = exports.getKey = exports.generateKey = exports.axiosInstance = exports.DeveloperKeyStatus = exports.MAX_ATTEMPTS = void 0;
const amino_1 = require("@cosmjs/amino");
const crypto_1 = require("@cosmjs/crypto");
const encoding_1 = require("@cosmjs/encoding");
const elliptic_1 = __importDefault(require("elliptic"));
const axios_1 = __importDefault(require("axios"));
const uuidv4_1 = require("uuidv4");
exports.MAX_ATTEMPTS = 10;
var DeveloperKeyStatus;
(function (DeveloperKeyStatus) {
    DeveloperKeyStatus["ERROR"] = "error";
    DeveloperKeyStatus["PENDING"] = "pending";
    DeveloperKeyStatus["SYNCED"] = "synced";
    DeveloperKeyStatus["DELETING"] = "deleting";
})(DeveloperKeyStatus = exports.DeveloperKeyStatus || (exports.DeveloperKeyStatus = {}));
const GW_BACKEND_URL = "https://gateway-master.lava-cybertron.xyz/sdk";
// const GW_BACKEND_URL = "http://127.0.0.1:4455/sdk";
exports.axiosInstance = axios_1.default.create({
    baseURL: `${GW_BACKEND_URL}/api/sdk`,
});
const generateKey = () => __awaiter(void 0, void 0, void 0, function* () {
    const prefix = "lava@";
    const mnemonic = crypto_1.Bip39.encode(crypto_1.Random.getBytes(32)).toString();
    const mnemonicChecked = new crypto_1.EnglishMnemonic(mnemonic);
    const seed = yield crypto_1.Bip39.mnemonicToSeed(mnemonicChecked, "");
    const hdPath = (0, amino_1.makeCosmoshubPath)(0);
    const { privkey } = crypto_1.Slip10.derivePath(crypto_1.Slip10Curve.Secp256k1, seed, hdPath);
    const privateHex = new elliptic_1.default.ec("secp256k1")
        .keyFromPrivate(privkey)
        .getPrivate("hex");
    const { pubkey } = yield crypto_1.Secp256k1.makeKeypair(privkey);
    const address = (0, encoding_1.toBech32)(prefix, (0, amino_1.rawSecp256k1PubkeyToRawAddress)(crypto_1.Secp256k1.compressPubkey(pubkey)));
    return { address, privateHex, mnemonicChecked };
});
exports.generateKey = generateKey;
function getKey(apiAccessKey, developerKey) {
    return __awaiter(this, void 0, void 0, function* () {
        return yield exports.axiosInstance.get(`/developer-keys/${developerKey}`, {
            headers: {
                "api-access-key": apiAccessKey,
            },
        });
    });
}
exports.getKey = getKey;
function getKeys(apiAccessKey) {
    return __awaiter(this, void 0, void 0, function* () {
        const { data } = yield exports.axiosInstance.get(`/developer-keys`, {
            headers: {
                "api-access-key": apiAccessKey,
            },
        });
        return data;
    });
}
exports.getKeys = getKeys;
function createDeveloperKey(apiAccessKey, developerKey) {
    return __awaiter(this, void 0, void 0, function* () {
        const keyParams = {
            projectKey: developerKey,
            name: `SDK Generated Key-${(0, uuidv4_1.uuid)().toLowerCase()}`,
        };
        return yield exports.axiosInstance.post(`/developer-keys/`, keyParams, {
            headers: { "api-access-key": apiAccessKey },
        });
    });
}
exports.createDeveloperKey = createDeveloperKey;
function sleep(delayMS) {
    return new Promise((res) => setTimeout(res, delayMS));
}
exports.sleep = sleep;
