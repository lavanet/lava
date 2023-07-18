import { DEFAULT_NETWORKS } from "../config/default";

type ChainInfo = {
  chainName: string;
  chainID: string;
  enabledApiInterfaces: string[];
};

type ChainInfoList = {
  chainInfoList: ChainInfo[];
};
// isNetworkValid validates network param
export function isNetworkValid(network: string): boolean {
  return DEFAULT_NETWORKS.includes(network);
}

// isValidChainID validates chainID param
export function isValidChainID(
  chainID: string,
  supportedChains: ChainInfoList
): boolean {
  return !!supportedChains.chainInfoList.find(
    (chainInfo) => chainInfo.chainID === chainID
  );
}

export function validateRpcInterfaceWithChainID(
  chainID: string,
  supportedChains: ChainInfoList,
  rpcInterface: string
) {
  const targetChainInfo = supportedChains.chainInfoList.find(
    (chainInfo) => chainInfo.chainID === chainID
  );

  if (!targetChainInfo) {
    throw new Error(`ChainID ${chainID} not found.`);
  }

  if (!targetChainInfo.enabledApiInterfaces.includes(rpcInterface)) {
    throw new Error(
      `The specified RPC interface is not supported by the chain ID ${chainID}, supported interfaces include ${targetChainInfo.enabledApiInterfaces}`
    );
  }
}

// fetchRpcInterface fetches default rpcInterface for chainID
export function fetchRpcInterface(
  chainID: string,
  supportedChains: ChainInfoList
): string {
  const targetChainInfo = supportedChains.chainInfoList.find(
    (chainInfo) => chainInfo.chainID === chainID
  );

  if (!targetChainInfo) {
    throw new Error(`ChainID ${chainID} not found.`);
  }

  const preferredOrder: string[] = ["tendermintrpc", "jsonrpc", "rest"];

  for (const apiInterface of preferredOrder) {
    if (targetChainInfo.enabledApiInterfaces.includes(apiInterface)) {
      return apiInterface;
    }
  }

  throw new Error(
    `None of the preferred API interfaces were found for ChainID ${chainID}.`
  );
}
