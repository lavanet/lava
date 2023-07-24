import { EnglishMnemonic } from "@cosmjs/crypto";
export declare const MAX_ATTEMPTS = 10;
export declare enum DeveloperKeyStatus {
    ERROR = "error",
    PENDING = "pending",
    SYNCED = "synced",
    DELETING = "deleting"
}
export declare type GeneratedKeyType = {
    address: string;
    privateHex: string;
    mnemonicChecked: EnglishMnemonic;
};
export declare type DeveloperKeyParams = {
    projectKey: string;
    name: string;
};
export declare const axiosInstance: import("axios").AxiosInstance;
export declare const generateKey: () => Promise<GeneratedKeyType>;
export declare function getKey(apiAccessKey: string, developerKey: string): Promise<any>;
export declare function getKeys(apiAccessKey: string): Promise<any>;
export declare function createDeveloperKey(publicKey: string, apiAccessKey: string): Promise<import("axios").AxiosResponse<any, any>>;
export declare function timeout(delayMS: number): Promise<unknown>;
