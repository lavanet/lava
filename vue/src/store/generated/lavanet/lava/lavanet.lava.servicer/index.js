import { txClient, queryClient, MissingWalletError, registry } from './module';
// @ts-ignore
import { SpVuexError } from '@starport/vuex';
import { BlockNum } from "./module/types/servicer/block_num";
import { ClientRequest } from "./module/types/servicer/client_request";
import { Params } from "./module/types/servicer/params";
import { SessionID } from "./module/types/servicer/session_id";
import { SpecName } from "./module/types/servicer/spec_name";
import { SpecStakeStorage } from "./module/types/servicer/spec_stake_storage";
import { StakeMap } from "./module/types/servicer/stake_map";
import { StakeStorage } from "./module/types/servicer/stake_storage";
import { WorkProof } from "./module/types/servicer/work_proof";
export { BlockNum, ClientRequest, Params, SessionID, SpecName, SpecStakeStorage, StakeMap, StakeStorage, WorkProof };
async function initTxClient(vuexGetters) {
    return await txClient(vuexGetters['common/wallet/signer'], {
        addr: vuexGetters['common/env/apiTendermint']
    });
}
async function initQueryClient(vuexGetters) {
    return await queryClient({
        addr: vuexGetters['common/env/apiCosmos']
    });
}
function mergeResults(value, next_values) {
    for (let prop of Object.keys(next_values)) {
        if (Array.isArray(next_values[prop])) {
            value[prop] = [...value[prop], ...next_values[prop]];
        }
        else {
            value[prop] = next_values[prop];
        }
    }
    return value;
}
function getStructure(template) {
    let structure = { fields: [] };
    for (const [key, value] of Object.entries(template)) {
        let field = {};
        field.name = key;
        field.type = typeof value;
        structure.fields.push(field);
    }
    return structure;
}
const getDefaultState = () => {
    return {
        Params: {},
        StakeMap: {},
        StakeMapAll: {},
        SpecStakeStorage: {},
        SpecStakeStorageAll: {},
        StakedServicers: {},
        _Structure: {
            BlockNum: getStructure(BlockNum.fromPartial({})),
            ClientRequest: getStructure(ClientRequest.fromPartial({})),
            Params: getStructure(Params.fromPartial({})),
            SessionID: getStructure(SessionID.fromPartial({})),
            SpecName: getStructure(SpecName.fromPartial({})),
            SpecStakeStorage: getStructure(SpecStakeStorage.fromPartial({})),
            StakeMap: getStructure(StakeMap.fromPartial({})),
            StakeStorage: getStructure(StakeStorage.fromPartial({})),
            WorkProof: getStructure(WorkProof.fromPartial({})),
        },
        _Registry: registry,
        _Subscriptions: new Set(),
    };
};
// initial state
const state = getDefaultState();
export default {
    namespaced: true,
    state,
    mutations: {
        RESET_STATE(state) {
            Object.assign(state, getDefaultState());
        },
        QUERY(state, { query, key, value }) {
            state[query][JSON.stringify(key)] = value;
        },
        SUBSCRIBE(state, subscription) {
            state._Subscriptions.add(JSON.stringify(subscription));
        },
        UNSUBSCRIBE(state, subscription) {
            state._Subscriptions.delete(JSON.stringify(subscription));
        }
    },
    getters: {
        getParams: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.Params[JSON.stringify(params)] ?? {};
        },
        getStakeMap: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.StakeMap[JSON.stringify(params)] ?? {};
        },
        getStakeMapAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.StakeMapAll[JSON.stringify(params)] ?? {};
        },
        getSpecStakeStorage: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SpecStakeStorage[JSON.stringify(params)] ?? {};
        },
        getSpecStakeStorageAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SpecStakeStorageAll[JSON.stringify(params)] ?? {};
        },
        getStakedServicers: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.StakedServicers[JSON.stringify(params)] ?? {};
        },
        getTypeStructure: (state) => (type) => {
            return state._Structure[type].fields;
        },
        getRegistry: (state) => {
            return state._Registry;
        }
    },
    actions: {
        init({ dispatch, rootGetters }) {
            console.log('Vuex module: lavanet.lava.servicer initialized!');
            if (rootGetters['common/env/client']) {
                rootGetters['common/env/client'].on('newblock', () => {
                    dispatch('StoreUpdate');
                });
            }
        },
        resetState({ commit }) {
            commit('RESET_STATE');
        },
        unsubscribe({ commit }, subscription) {
            commit('UNSUBSCRIBE', subscription);
        },
        async StoreUpdate({ state, dispatch }) {
            state._Subscriptions.forEach(async (subscription) => {
                try {
                    const sub = JSON.parse(subscription);
                    await dispatch(sub.action, sub.payload);
                }
                catch (e) {
                    throw new SpVuexError('Subscriptions: ' + e.message);
                }
            });
        },
        async QueryParams({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryParams()).data;
                commit('QUERY', { query: 'Params', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: { ...key }, query } });
                return getters['getParams']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryParams', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryStakeMap({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryStakeMap(key.index)).data;
                commit('QUERY', { query: 'StakeMap', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryStakeMap', payload: { options: { all }, params: { ...key }, query } });
                return getters['getStakeMap']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryStakeMap', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryStakeMapAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryStakeMapAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.queryStakeMapAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'StakeMapAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryStakeMapAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getStakeMapAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryStakeMapAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySpecStakeStorage({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySpecStakeStorage(key.index)).data;
                commit('QUERY', { query: 'SpecStakeStorage', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySpecStakeStorage', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSpecStakeStorage']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySpecStakeStorage', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySpecStakeStorageAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySpecStakeStorageAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.querySpecStakeStorageAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'SpecStakeStorageAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySpecStakeStorageAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSpecStakeStorageAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySpecStakeStorageAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryStakedServicers({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryStakedServicers(key.specName)).data;
                commit('QUERY', { query: 'StakedServicers', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryStakedServicers', payload: { options: { all }, params: { ...key }, query } });
                return getters['getStakedServicers']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryStakedServicers', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async sendMsgProofOfWork({ rootGetters }, { value, fee = [], memo = '' }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgProofOfWork(value);
                const result = await txClient.signAndBroadcast([msg], { fee: { amount: fee,
                        gas: "200000" }, memo });
                return result;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgProofOfWork:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgProofOfWork:Send', 'Could not broadcast Tx: ' + e.message);
                }
            }
        },
        async sendMsgUnstakeServicer({ rootGetters }, { value, fee = [], memo = '' }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgUnstakeServicer(value);
                const result = await txClient.signAndBroadcast([msg], { fee: { amount: fee,
                        gas: "200000" }, memo });
                return result;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgUnstakeServicer:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgUnstakeServicer:Send', 'Could not broadcast Tx: ' + e.message);
                }
            }
        },
        async sendMsgStakeServicer({ rootGetters }, { value, fee = [], memo = '' }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgStakeServicer(value);
                const result = await txClient.signAndBroadcast([msg], { fee: { amount: fee,
                        gas: "200000" }, memo });
                return result;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgStakeServicer:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgStakeServicer:Send', 'Could not broadcast Tx: ' + e.message);
                }
            }
        },
        async MsgProofOfWork({ rootGetters }, { value }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgProofOfWork(value);
                return msg;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgProofOfWork:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgProofOfWork:Create', 'Could not create message: ' + e.message);
                }
            }
        },
        async MsgUnstakeServicer({ rootGetters }, { value }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgUnstakeServicer(value);
                return msg;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgUnstakeServicer:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgUnstakeServicer:Create', 'Could not create message: ' + e.message);
                }
            }
        },
        async MsgStakeServicer({ rootGetters }, { value }) {
            try {
                const txClient = await initTxClient(rootGetters);
                const msg = await txClient.msgStakeServicer(value);
                return msg;
            }
            catch (e) {
                if (e == MissingWalletError) {
                    throw new SpVuexError('TxClient:MsgStakeServicer:Init', 'Could not initialize signing client. Wallet is required.');
                }
                else {
                    throw new SpVuexError('TxClient:MsgStakeServicer:Create', 'Could not create message: ' + e.message);
                }
            }
        },
    }
};
