import { StateQuery } from "./state_query";
export declare class PairingUpdater {
    private stateQuery;
    private consumerSessionManagerMap;
    constructor(stateQuery: StateQuery, consumerSessionManagerMap: any);
    update(): void;
    private updateConsummerSessionManager;
    private filterPairingListByEndpoint;
}
