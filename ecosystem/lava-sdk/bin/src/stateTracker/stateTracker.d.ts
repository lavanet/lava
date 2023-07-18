import { ConsumerSessionWithProvider, SessionManager } from "../types/types";
import { AccountData } from "@cosmjs/proto-signing";
import { LavaProviders } from "../lavaOverLava/providers";
import Relayer from "../relayer/relayer";
export declare class StateTracker {
    private lavaProviders;
    private relayer;
    constructor(lavaProviders: LavaProviders | null, relayer: Relayer | null);
    getSession(account: AccountData, chainID: string, rpcInterface: string): Promise<SessionManager>;
    pickRandomProvider(providers: Array<ConsumerSessionWithProvider>): ConsumerSessionWithProvider;
    private getPairingFromChain;
    private getMaxCuForUser;
    private getServiceApis;
    convertRestApiName(name: string): string;
    sendRelayWithRetry(options: any, // TODO add type
    lavaRPCEndpoint: ConsumerSessionWithProvider): Promise<any>;
}
