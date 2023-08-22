"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.PairingUpdater = void 0;
class PairingUpdater {
    constructor(stateQuery, consumerSessionManagerMap) {
        // Save arguments
        this.stateQuery = stateQuery;
        this.consumerSessionManagerMap = consumerSessionManagerMap;
    }
    update() {
        const keys = Object.keys(this.consumerSessionManagerMap);
        for (const chainID of keys) {
            const consumerSessionManagerList = this.consumerSessionManagerMap[chainID];
            // Your code using chainID and consumerSessionManagerList
            const pairing = this.stateQuery.getPairing(chainID);
            // Update each consumer session menager with matching pairing list
            for (const consumerSessionManager of consumerSessionManagerList) {
                this.consumerSessionManagerMap(pairing, consumerSessionManager);
            }
        }
    }
    // updateConsummerSessionManager filters pairing list and update consuemr session manager
    updateConsummerSessionManager(pairing, consumerSessionManager) {
        // Filter pairing list for specific consumer session manager
        const pairingListForThisCSM = this.filterPairingListByEndpoint(pairing, consumerSessionManager.RPCEndpoint());
        // Update specific consumer session manager
        consumerSessionManager.UpdateAllProviders(pairingListForThisCSM);
    }
    // filterPairingListByEndpoint filters pairing list and return only the once for rpcInterface
    filterPairingListByEndpoint(pairing, rpcInterface) {
        return;
    }
}
exports.PairingUpdater = PairingUpdater;
