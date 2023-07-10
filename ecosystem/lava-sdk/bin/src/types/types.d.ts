export declare class SessionManager {
    PairingList: ConsumerSessionWithProvider[];
    NextEpochStart: Date;
    Apis: Map<string, number>;
    constructor(pairingList: ConsumerSessionWithProvider[], nextEpochStart: Date, apis: Map<string, number>);
    getCuSumFromApi(name: string, chainID: string): number | undefined;
}
export declare class ConsumerSessionWithProvider {
    Acc: string;
    Endpoints: Array<Endpoint>;
    Session: SingleConsumerSession;
    MaxComputeUnits: number;
    UsedComputeUnits: number;
    ReliabilitySent: boolean;
    constructor(acc: string, endpoints: Array<Endpoint>, session: SingleConsumerSession, maxComputeUnits: number, usedComputeUnits: number, reliabilitySent: boolean);
}
export declare class SingleConsumerSession {
    ProviderAddress: string;
    CuSum: number;
    LatestRelayCu: number;
    SessionId: number;
    RelayNum: number;
    Endpoint: Endpoint;
    PairingEpoch: number;
    constructor(cuSum: number, latestRelayCu: number, relayNum: number, endpoint: Endpoint, pairingEpoch: number, providerAddress: string);
    getNewSessionId(): number;
    getNewSalt(): Uint8Array;
    private generateRandomUint;
}
export declare class Endpoint {
    Addr: string;
    Enabled: boolean;
    ConnectionRefusals: number;
    constructor(addr: string, enabled: boolean, connectionRefusals: number);
}
