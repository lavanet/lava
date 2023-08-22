import { BadgeManager } from "../badge/fetchBadge";
import { ChainIDRpcInterface, Config } from "./stateTracker";
import Relayer from "../relayer/relayer";
import { ConsumerSessionWithProvider } from "../types/types";
export declare class StateQuery {
    private badgeManager;
    private pairingListConfig;
    private relayer;
    private chainIDRpcInterfaces;
    private lavaProviders;
    private config;
    private pairing;
    constructor(badgeManager: BadgeManager, pairingListConfig: string, chainIdRpcInterfaces: ChainIDRpcInterface[], relayer: Relayer, config: Config);
    private fetchLavaProviders;
    fetchPairing(): Promise<number>;
    getPairing(chainID: string): ConsumerSessionWithProvider[] | undefined;
    private getLatestBlockFromProviders;
    private validatePairingData;
    private fetchLocalLavaPairingList;
    private fetchDefaultLavaPairingList;
    private constructLavaPairing;
    private SendRelayToAllProvidersAndRace;
    private SendRelayWithRetry;
    private extractBlockNumberFromError;
}
