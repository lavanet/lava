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
export declare const generateKey: () => Promise<GeneratedKeyType>;
export declare function getDeveloperKey(apiAccessKey: string, developerKey: string, url?: string): Promise<any>;
export declare function createDeveloperKey(apiAccessKey: string, developerKey: string, url?: string): Promise<any>;
export declare function sleep(delayMS: number): Promise<unknown>;
