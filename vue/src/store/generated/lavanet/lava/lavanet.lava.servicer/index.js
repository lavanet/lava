import { txClient, queryClient, MissingWalletError, registry } from './module';
// @ts-ignore
import { SpVuexError } from '@starport/vuex';
import { BlockDeadlineForCallback } from "./module/types/servicer/block_deadline_for_callback";
import { BlockNum } from "./module/types/servicer/block_num";
import { ClientRequest } from "./module/types/servicer/client_request";
import { CurrentSessionStart } from "./module/types/servicer/current_session_start";
import { EarliestSessionStart } from "./module/types/servicer/earliest_session_start";
import { Params } from "./module/types/servicer/params";
import { PreviousSessionBlocks } from "./module/types/servicer/previous_session_blocks";
import { SessionID } from "./module/types/servicer/session_id";
import { SessionPayments } from "./module/types/servicer/session_payments";
import { SessionStorageForSpec } from "./module/types/servicer/session_storage_for_spec";
import { SpecName } from "./module/types/servicer/spec_name";
import { SpecStakeStorage } from "./module/types/servicer/spec_stake_storage";
import { StakeMap } from "./module/types/servicer/stake_map";
import { StakeStorage } from "./module/types/servicer/stake_storage";
import { UniquePaymentStorageUserServicer } from "./module/types/servicer/unique_payment_storage_user_servicer";
import { UnstakingServicersAllSpecs } from "./module/types/servicer/unstaking_servicers_all_specs";
import { UserPaymentStorage } from "./module/types/servicer/user_payment_storage";
import { WorkProof } from "./module/types/servicer/work_proof";
export { BlockDeadlineForCallback, BlockNum, ClientRequest, CurrentSessionStart, EarliestSessionStart, Params, PreviousSessionBlocks, SessionID, SessionPayments, SessionStorageForSpec, SpecName, SpecStakeStorage, StakeMap, StakeStorage, UniquePaymentStorageUserServicer, UnstakingServicersAllSpecs, UserPaymentStorage, WorkProof };
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
        BlockDeadlineForCallback: {},
        UnstakingServicersAllSpecs: {},
        UnstakingServicersAllSpecsAll: {},
        GetPairing: {},
        CurrentSessionStart: {},
        PreviousSessionBlocks: {},
        SessionStorageForSpec: {},
        SessionStorageForSpecAll: {},
        SessionStorageForAllSpecs: {},
        AllSessionStoragesForSpec: {},
        EarliestSessionStart: {},
        VerifyPairing: {},
        UniquePaymentStorageUserServicer: {},
        UniquePaymentStorageUserServicerAll: {},
        UserPaymentStorage: {},
        UserPaymentStorageAll: {},
        SessionPayments: {},
        SessionPaymentsAll: {},
        _Structure: {
            BlockDeadlineForCallback: getStructure(BlockDeadlineForCallback.fromPartial({})),
            BlockNum: getStructure(BlockNum.fromPartial({})),
            ClientRequest: getStructure(ClientRequest.fromPartial({})),
            CurrentSessionStart: getStructure(CurrentSessionStart.fromPartial({})),
            EarliestSessionStart: getStructure(EarliestSessionStart.fromPartial({})),
            Params: getStructure(Params.fromPartial({})),
            PreviousSessionBlocks: getStructure(PreviousSessionBlocks.fromPartial({})),
            SessionID: getStructure(SessionID.fromPartial({})),
            SessionPayments: getStructure(SessionPayments.fromPartial({})),
            SessionStorageForSpec: getStructure(SessionStorageForSpec.fromPartial({})),
            SpecName: getStructure(SpecName.fromPartial({})),
            SpecStakeStorage: getStructure(SpecStakeStorage.fromPartial({})),
            StakeMap: getStructure(StakeMap.fromPartial({})),
            StakeStorage: getStructure(StakeStorage.fromPartial({})),
            UniquePaymentStorageUserServicer: getStructure(UniquePaymentStorageUserServicer.fromPartial({})),
            UnstakingServicersAllSpecs: getStructure(UnstakingServicersAllSpecs.fromPartial({})),
            UserPaymentStorage: getStructure(UserPaymentStorage.fromPartial({})),
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
        getBlockDeadlineForCallback: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.BlockDeadlineForCallback[JSON.stringify(params)] ?? {};
        },
        getUnstakingServicersAllSpecs: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.UnstakingServicersAllSpecs[JSON.stringify(params)] ?? {};
        },
        getUnstakingServicersAllSpecsAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.UnstakingServicersAllSpecsAll[JSON.stringify(params)] ?? {};
        },
        getGetPairing: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.GetPairing[JSON.stringify(params)] ?? {};
        },
        getCurrentSessionStart: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.CurrentSessionStart[JSON.stringify(params)] ?? {};
        },
        getPreviousSessionBlocks: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.PreviousSessionBlocks[JSON.stringify(params)] ?? {};
        },
        getSessionStorageForSpec: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SessionStorageForSpec[JSON.stringify(params)] ?? {};
        },
        getSessionStorageForSpecAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SessionStorageForSpecAll[JSON.stringify(params)] ?? {};
        },
        getSessionStorageForAllSpecs: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SessionStorageForAllSpecs[JSON.stringify(params)] ?? {};
        },
        getAllSessionStoragesForSpec: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.AllSessionStoragesForSpec[JSON.stringify(params)] ?? {};
        },
        getEarliestSessionStart: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.EarliestSessionStart[JSON.stringify(params)] ?? {};
        },
        getVerifyPairing: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.VerifyPairing[JSON.stringify(params)] ?? {};
        },
        getUniquePaymentStorageUserServicer: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.UniquePaymentStorageUserServicer[JSON.stringify(params)] ?? {};
        },
        getUniquePaymentStorageUserServicerAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.UniquePaymentStorageUserServicerAll[JSON.stringify(params)] ?? {};
        },
        getUserPaymentStorage: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.UserPaymentStorage[JSON.stringify(params)] ?? {};
        },
        getUserPaymentStorageAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.UserPaymentStorageAll[JSON.stringify(params)] ?? {};
        },
        getSessionPayments: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SessionPayments[JSON.stringify(params)] ?? {};
        },
        getSessionPaymentsAll: (state) => (params = { params: {} }) => {
            if (!params.query) {
                params.query = null;
            }
            return state.SessionPaymentsAll[JSON.stringify(params)] ?? {};
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
        async QueryBlockDeadlineForCallback({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryBlockDeadlineForCallback()).data;
                commit('QUERY', { query: 'BlockDeadlineForCallback', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryBlockDeadlineForCallback', payload: { options: { all }, params: { ...key }, query } });
                return getters['getBlockDeadlineForCallback']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryBlockDeadlineForCallback', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryUnstakingServicersAllSpecs({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryUnstakingServicersAllSpecs(key.id)).data;
                commit('QUERY', { query: 'UnstakingServicersAllSpecs', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryUnstakingServicersAllSpecs', payload: { options: { all }, params: { ...key }, query } });
                return getters['getUnstakingServicersAllSpecs']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryUnstakingServicersAllSpecs', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryUnstakingServicersAllSpecsAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryUnstakingServicersAllSpecsAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.queryUnstakingServicersAllSpecsAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'UnstakingServicersAllSpecsAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryUnstakingServicersAllSpecsAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getUnstakingServicersAllSpecsAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryUnstakingServicersAllSpecsAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryGetPairing({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryGetPairing(key.specName, key.userAddr)).data;
                commit('QUERY', { query: 'GetPairing', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryGetPairing', payload: { options: { all }, params: { ...key }, query } });
                return getters['getGetPairing']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryGetPairing', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryCurrentSessionStart({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryCurrentSessionStart()).data;
                commit('QUERY', { query: 'CurrentSessionStart', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryCurrentSessionStart', payload: { options: { all }, params: { ...key }, query } });
                return getters['getCurrentSessionStart']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryCurrentSessionStart', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryPreviousSessionBlocks({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryPreviousSessionBlocks()).data;
                commit('QUERY', { query: 'PreviousSessionBlocks', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryPreviousSessionBlocks', payload: { options: { all }, params: { ...key }, query } });
                return getters['getPreviousSessionBlocks']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryPreviousSessionBlocks', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySessionStorageForSpec({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySessionStorageForSpec(key.index)).data;
                commit('QUERY', { query: 'SessionStorageForSpec', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySessionStorageForSpec', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSessionStorageForSpec']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySessionStorageForSpec', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySessionStorageForSpecAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySessionStorageForSpecAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.querySessionStorageForSpecAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'SessionStorageForSpecAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySessionStorageForSpecAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSessionStorageForSpecAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySessionStorageForSpecAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySessionStorageForAllSpecs({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySessionStorageForAllSpecs(key.blockNum)).data;
                commit('QUERY', { query: 'SessionStorageForAllSpecs', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySessionStorageForAllSpecs', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSessionStorageForAllSpecs']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySessionStorageForAllSpecs', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryAllSessionStoragesForSpec({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryAllSessionStoragesForSpec(key.specName)).data;
                commit('QUERY', { query: 'AllSessionStoragesForSpec', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryAllSessionStoragesForSpec', payload: { options: { all }, params: { ...key }, query } });
                return getters['getAllSessionStoragesForSpec']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryAllSessionStoragesForSpec', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryEarliestSessionStart({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryEarliestSessionStart()).data;
                commit('QUERY', { query: 'EarliestSessionStart', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryEarliestSessionStart', payload: { options: { all }, params: { ...key }, query } });
                return getters['getEarliestSessionStart']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryEarliestSessionStart', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryVerifyPairing({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryVerifyPairing(key.spec, key.userAddr, key.servicerAddr, key.blockNum)).data;
                commit('QUERY', { query: 'VerifyPairing', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryVerifyPairing', payload: { options: { all }, params: { ...key }, query } });
                return getters['getVerifyPairing']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryVerifyPairing', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryUniquePaymentStorageUserServicer({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryUniquePaymentStorageUserServicer(key.index)).data;
                commit('QUERY', { query: 'UniquePaymentStorageUserServicer', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryUniquePaymentStorageUserServicer', payload: { options: { all }, params: { ...key }, query } });
                return getters['getUniquePaymentStorageUserServicer']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryUniquePaymentStorageUserServicer', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryUniquePaymentStorageUserServicerAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryUniquePaymentStorageUserServicerAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.queryUniquePaymentStorageUserServicerAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'UniquePaymentStorageUserServicerAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryUniquePaymentStorageUserServicerAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getUniquePaymentStorageUserServicerAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryUniquePaymentStorageUserServicerAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryUserPaymentStorage({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryUserPaymentStorage(key.index)).data;
                commit('QUERY', { query: 'UserPaymentStorage', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryUserPaymentStorage', payload: { options: { all }, params: { ...key }, query } });
                return getters['getUserPaymentStorage']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryUserPaymentStorage', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QueryUserPaymentStorageAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.queryUserPaymentStorageAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.queryUserPaymentStorageAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'UserPaymentStorageAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QueryUserPaymentStorageAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getUserPaymentStorageAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QueryUserPaymentStorageAll', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySessionPayments({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySessionPayments(key.index)).data;
                commit('QUERY', { query: 'SessionPayments', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySessionPayments', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSessionPayments']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySessionPayments', 'API Node Unavailable. Could not perform query: ' + e.message);
            }
        },
        async QuerySessionPaymentsAll({ commit, rootGetters, getters }, { options: { subscribe, all } = { subscribe: false, all: false }, params, query = null }) {
            try {
                const key = params ?? {};
                const queryClient = await initQueryClient(rootGetters);
                let value = (await queryClient.querySessionPaymentsAll(query)).data;
                while (all && value.pagination && value.pagination.next_key != null) {
                    let next_values = (await queryClient.querySessionPaymentsAll({ ...query, 'pagination.key': value.pagination.next_key })).data;
                    value = mergeResults(value, next_values);
                }
                commit('QUERY', { query: 'SessionPaymentsAll', key: { params: { ...key }, query }, value });
                if (subscribe)
                    commit('SUBSCRIBE', { action: 'QuerySessionPaymentsAll', payload: { options: { all }, params: { ...key }, query } });
                return getters['getSessionPaymentsAll']({ params: { ...key }, query }) ?? {};
            }
            catch (e) {
                throw new SpVuexError('QueryClient:QuerySessionPaymentsAll', 'API Node Unavailable. Could not perform query: ' + e.message);
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
    }
};
