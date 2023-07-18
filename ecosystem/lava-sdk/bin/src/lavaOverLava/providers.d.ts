import { ConsumerSessionWithProvider, SessionManager } from "../types/types";
import Relayer from "../relayer/relayer";
import { Badge } from "../grpc_web_services/pairing/relay_pb";
import { QueryShowAllChainsResponse } from "../codec/spec/query";
export interface LavaProvidersOptions {
    accountAddress: string;
    network: string;
    relayer: Relayer | null;
    geolocation: string;
    debug?: boolean;
}
export declare class LavaProviders {
    private providers;
    private network;
    private index;
    private accountAddress;
    private relayer;
    private geolocation;
    private debugMode;
    constructor(options: LavaProvidersOptions);
    updateLavaProvidersRelayersBadge(badge: Badge | undefined): void;
    init(pairingListConfig: string): Promise<void>;
    showAllChains(): Promise<QueryShowAllChainsResponse>;
    initDefaultConfig(): Promise<any>;
    initLocalConfig(path: string): Promise<any>;
    GetLavaProviders(): ConsumerSessionWithProvider[];
    GetNextLavaProvider(): ConsumerSessionWithProvider;
    getSession(chainID: string, rpcInterface: string, badge?: Badge): Promise<SessionManager>;
    private debugPrint;
    pickRandomProviders(providers: Array<ConsumerSessionWithProvider>): ConsumerSessionWithProvider[];
    pickRandomProvider(providers: Array<ConsumerSessionWithProvider>): ConsumerSessionWithProvider;
    private getPairingFromChain;
    private getMaxCuForUser;
    private getServiceApis;
    convertRestApiName(name: string): string;
    SendRelayToAllProvidersAndRace(options: any, relayCu: number, rpcInterface: string): Promise<any>;
    SendRelayWithRetry(options: any, lavaRPCEndpoint: ConsumerSessionWithProvider, relayCu: number, rpcInterface: string): Promise<any>;
    private extractBlockNumberFromError;
}
