export declare class LavaSDK {
    private privKey;
    private badge;
    private isBadge;
    private chainID;
    private rpcInterface;
    private network;
    private pairingListConfig;
    private geolocation;
    private lavaChainId;
    private lavaProviders;
    private account;
    private relayer;
    private secure;
    private activeSessionManager;
    /**
     * Create Lava-SDK instance
     *
     * Use Lava-SDK for dAccess with a supported network. You can find a list of supported networks and their chain IDs at (url).
     *
     * @async
     * @param {LavaSDKOptions} options The options to use for initializing the LavaSDK.
     *
     * @returns A promise that resolves when the LavaSDK has been successfully initialized, returns LavaSDK object.
     */
    constructor(options: LavaSDKOptions);
    private init;
    private handleRpcRelay;
    private handleRestRelay;
    private sendRelayWithRetries;
    /**
     * Send relay to network through providers.
     *
     * @async
     * @param options The options to use for sending relay.
     *
     * @returns A promise that resolves when the relay response has been returned, and returns a JSON string
     *
     */
    sendRelay(options: SendRelayOptions | SendRestRelayOptions): Promise<string>;
    private generateRPCData;
    private decodeRelayResponse;
    private getCuSumForMethod;
    private getConsumerProviderSession;
    private newEpochStarted;
    private isRest;
    private base64ToUint8Array;
}
/**
 * Options for sending RPC relay.
 */
export interface SendRelayOptions {
    method: string;
    params: Array<any>;
}
/**
 * Options for sending Rest relay.
 */
export interface SendRestRelayOptions {
    method: string;
    url: string;
    data?: Record<string, any>;
}
/**
 * Options for initializing the LavaSDK.
 */
export interface LavaSDKOptions {
    privateKey?: string;
    badge?: {
        badgeServerAddress: string;
        projectId: string;
    };
    chainID: string;
    rpcInterface?: string;
    pairingListConfig?: string;
    network?: string;
    geolocation?: string;
    lavaChainId?: string;
    secure?: boolean;
}
