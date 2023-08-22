import { BadgeManager } from "../badge/fetchBadge";
import Relayer from "../relayer/relayer";
export interface ChainIDRpcInterface {
    chainID: string;
    rpcInterface: string;
}
export interface Config {
    geolocation: string;
    network: string;
    accountAddress: string;
    debug: boolean;
}
export declare class StateTracker {
    private updaters;
    private stateQuery;
    constructor(badgeManager: BadgeManager, pairingListConfig: string, relayer: Relayer, chainIDRpcInterface: ChainIDRpcInterface[], config: Config, consumerSessionManagerMap: any);
    private registerForUpdates;
    executeUpdateOnNewEpoch(): Promise<void>;
}
