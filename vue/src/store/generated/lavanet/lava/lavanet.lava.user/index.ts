import { txClient, queryClient, MissingWalletError , registry} from './module'
// @ts-ignore
import { SpVuexError } from '@starport/vuex'

import { BlockDeadlineForCallback } from "./module/types/user/block_deadline_for_callback"
import { BlockNum } from "./module/types/user/block_num"
import { Params } from "./module/types/user/params"
import { SpecName } from "./module/types/user/spec_name"
import { SpecStakeStorage } from "./module/types/user/spec_stake_storage"
import { StakeStorage } from "./module/types/user/stake_storage"
import { UnstakingUsersAllSpecs } from "./module/types/user/unstaking_users_all_specs"
import { UserStake } from "./module/types/user/user_stake"


export { BlockDeadlineForCallback, BlockNum, Params, SpecName, SpecStakeStorage, StakeStorage, UnstakingUsersAllSpecs, UserStake };

async function initTxClient(vuexGetters) {
	return await txClient(vuexGetters['common/wallet/signer'], {
		addr: vuexGetters['common/env/apiTendermint']
	})
}

async function initQueryClient(vuexGetters) {
	return await queryClient({
		addr: vuexGetters['common/env/apiCosmos']
	})
}

function mergeResults(value, next_values) {
	for (let prop of Object.keys(next_values)) {
		if (Array.isArray(next_values[prop])) {
			value[prop]=[...value[prop], ...next_values[prop]]
		}else{
			value[prop]=next_values[prop]
		}
	}
	return value
}

function getStructure(template) {
	let structure = { fields: [] }
	for (const [key, value] of Object.entries(template)) {
		let field: any = {}
		field.name = key
		field.type = typeof value
		structure.fields.push(field)
	}
	return structure
}

const getDefaultState = () => {
	return {
				Params: {},
				UserStake: {},
				UserStakeAll: {},
				SpecStakeStorage: {},
				SpecStakeStorageAll: {},
				BlockDeadlineForCallback: {},
				UnstakingUsersAllSpecs: {},
				UnstakingUsersAllSpecsAll: {},
				StakedUsers: {},
				
				_Structure: {
						BlockDeadlineForCallback: getStructure(BlockDeadlineForCallback.fromPartial({})),
						BlockNum: getStructure(BlockNum.fromPartial({})),
						Params: getStructure(Params.fromPartial({})),
						SpecName: getStructure(SpecName.fromPartial({})),
						SpecStakeStorage: getStructure(SpecStakeStorage.fromPartial({})),
						StakeStorage: getStructure(StakeStorage.fromPartial({})),
						UnstakingUsersAllSpecs: getStructure(UnstakingUsersAllSpecs.fromPartial({})),
						UserStake: getStructure(UserStake.fromPartial({})),
						
		},
		_Registry: registry,
		_Subscriptions: new Set(),
	}
}

// initial state
const state = getDefaultState()

export default {
	namespaced: true,
	state,
	mutations: {
		RESET_STATE(state) {
			Object.assign(state, getDefaultState())
		},
		QUERY(state, { query, key, value }) {
			state[query][JSON.stringify(key)] = value
		},
		SUBSCRIBE(state, subscription) {
			state._Subscriptions.add(JSON.stringify(subscription))
		},
		UNSUBSCRIBE(state, subscription) {
			state._Subscriptions.delete(JSON.stringify(subscription))
		}
	},
	getters: {
				getParams: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.Params[JSON.stringify(params)] ?? {}
		},
				getUserStake: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.UserStake[JSON.stringify(params)] ?? {}
		},
				getUserStakeAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.UserStakeAll[JSON.stringify(params)] ?? {}
		},
				getSpecStakeStorage: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.SpecStakeStorage[JSON.stringify(params)] ?? {}
		},
				getSpecStakeStorageAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.SpecStakeStorageAll[JSON.stringify(params)] ?? {}
		},
				getBlockDeadlineForCallback: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.BlockDeadlineForCallback[JSON.stringify(params)] ?? {}
		},
				getUnstakingUsersAllSpecs: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.UnstakingUsersAllSpecs[JSON.stringify(params)] ?? {}
		},
				getUnstakingUsersAllSpecsAll: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.UnstakingUsersAllSpecsAll[JSON.stringify(params)] ?? {}
		},
				getStakedUsers: (state) => (params = { params: {}}) => {
					if (!(<any> params).query) {
						(<any> params).query=null
					}
			return state.StakedUsers[JSON.stringify(params)] ?? {}
		},
				
		getTypeStructure: (state) => (type) => {
			return state._Structure[type].fields
		},
		getRegistry: (state) => {
			return state._Registry
		}
	},
	actions: {
		init({ dispatch, rootGetters }) {
			console.log('Vuex module: lavanet.lava.user initialized!')
			if (rootGetters['common/env/client']) {
				rootGetters['common/env/client'].on('newblock', () => {
					dispatch('StoreUpdate')
				})
			}
		},
		resetState({ commit }) {
			commit('RESET_STATE')
		},
		unsubscribe({ commit }, subscription) {
			commit('UNSUBSCRIBE', subscription)
		},
		async StoreUpdate({ state, dispatch }) {
			state._Subscriptions.forEach(async (subscription) => {
				try {
					const sub=JSON.parse(subscription)
					await dispatch(sub.action, sub.payload)
				}catch(e) {
					throw new SpVuexError('Subscriptions: ' + e.message)
				}
			})
		},
		
		
		
		 		
		
		
		async QueryParams({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryParams()).data
				
					
				commit('QUERY', { query: 'Params', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryParams', payload: { options: { all }, params: {...key},query }})
				return getters['getParams']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryParams', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryUserStake({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryUserStake( key.index)).data
				
					
				commit('QUERY', { query: 'UserStake', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryUserStake', payload: { options: { all }, params: {...key},query }})
				return getters['getUserStake']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryUserStake', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryUserStakeAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryUserStakeAll(query)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await queryClient.queryUserStakeAll({...query, 'pagination.key':(<any> value).pagination.next_key})).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'UserStakeAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryUserStakeAll', payload: { options: { all }, params: {...key},query }})
				return getters['getUserStakeAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryUserStakeAll', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QuerySpecStakeStorage({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.querySpecStakeStorage( key.index)).data
				
					
				commit('QUERY', { query: 'SpecStakeStorage', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QuerySpecStakeStorage', payload: { options: { all }, params: {...key},query }})
				return getters['getSpecStakeStorage']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QuerySpecStakeStorage', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QuerySpecStakeStorageAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.querySpecStakeStorageAll(query)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await queryClient.querySpecStakeStorageAll({...query, 'pagination.key':(<any> value).pagination.next_key})).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'SpecStakeStorageAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QuerySpecStakeStorageAll', payload: { options: { all }, params: {...key},query }})
				return getters['getSpecStakeStorageAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QuerySpecStakeStorageAll', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryBlockDeadlineForCallback({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryBlockDeadlineForCallback()).data
				
					
				commit('QUERY', { query: 'BlockDeadlineForCallback', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryBlockDeadlineForCallback', payload: { options: { all }, params: {...key},query }})
				return getters['getBlockDeadlineForCallback']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryBlockDeadlineForCallback', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryUnstakingUsersAllSpecs({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryUnstakingUsersAllSpecs( key.id)).data
				
					
				commit('QUERY', { query: 'UnstakingUsersAllSpecs', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryUnstakingUsersAllSpecs', payload: { options: { all }, params: {...key},query }})
				return getters['getUnstakingUsersAllSpecs']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryUnstakingUsersAllSpecs', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryUnstakingUsersAllSpecsAll({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryUnstakingUsersAllSpecsAll(query)).data
				
					
				while (all && (<any> value).pagination && (<any> value).pagination.next_key!=null) {
					let next_values=(await queryClient.queryUnstakingUsersAllSpecsAll({...query, 'pagination.key':(<any> value).pagination.next_key})).data
					value = mergeResults(value, next_values);
				}
				commit('QUERY', { query: 'UnstakingUsersAllSpecsAll', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryUnstakingUsersAllSpecsAll', payload: { options: { all }, params: {...key},query }})
				return getters['getUnstakingUsersAllSpecsAll']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryUnstakingUsersAllSpecsAll', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		
		
		 		
		
		
		async QueryStakedUsers({ commit, rootGetters, getters }, { options: { subscribe, all} = { subscribe:false, all:false}, params, query=null }) {
			try {
				const key = params ?? {};
				const queryClient=await initQueryClient(rootGetters)
				let value= (await queryClient.queryStakedUsers( key.specName)).data
				
					
				commit('QUERY', { query: 'StakedUsers', key: { params: {...key}, query}, value })
				if (subscribe) commit('SUBSCRIBE', { action: 'QueryStakedUsers', payload: { options: { all }, params: {...key},query }})
				return getters['getStakedUsers']( { params: {...key}, query}) ?? {}
			} catch (e) {
				throw new SpVuexError('QueryClient:QueryStakedUsers', 'API Node Unavailable. Could not perform query: ' + e.message)
				
			}
		},
		
		
		async sendMsgStakeUser({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgStakeUser(value)
				const result = await txClient.signAndBroadcast([msg], {fee: { amount: fee, 
	gas: "200000" }, memo})
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgStakeUser:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgStakeUser:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		async sendMsgUnstakeUser({ rootGetters }, { value, fee = [], memo = '' }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgUnstakeUser(value)
				const result = await txClient.signAndBroadcast([msg], {fee: { amount: fee, 
	gas: "200000" }, memo})
				return result
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgUnstakeUser:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgUnstakeUser:Send Could not broadcast Tx: '+ e.message)
				}
			}
		},
		
		async MsgStakeUser({ rootGetters }, { value }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgStakeUser(value)
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgStakeUser:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgStakeUser:Create Could not create message: ' + e.message)
				}
			}
		},
		async MsgUnstakeUser({ rootGetters }, { value }) {
			try {
				const txClient=await initTxClient(rootGetters)
				const msg = await txClient.msgUnstakeUser(value)
				return msg
			} catch (e) {
				if (e == MissingWalletError) {
					throw new Error('TxClient:MsgUnstakeUser:Init Could not initialize signing client. Wallet is required.')
				}else{
					throw new Error('TxClient:MsgUnstakeUser:Create Could not create message: ' + e.message)
				}
			}
		},
		
	}
}
