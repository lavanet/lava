export type ChainIDsToInit = string | ChainIdSpecification;

/**
   * @param ChainIdWithSpecificAPIInterfaces Use ChainIdWithSpecificAPIInterfaces if you want to initialize only a specific api interface 
    for example, some chains are initializing 2-3 api interfaces (tendermint, rest, jsonrpc)
    use this functionality to initialize specific api interfaces for less traffic if you are not using them. (the sdk will not probe these interfaces)
  */
export type ChainIdSpecification =
  | string[]
  | ChainIdWithSpecificAPIInterfaces[]; // chainId or an array of chain ids to initialize sdk for.

/**
 * @param chainId chain id to initialize
 * @param apiInterfaces list of api interfaces to initialize
 */
export interface ChainIdWithSpecificAPIInterfaces {
  chainId: string;
  apiInterfaces: string[];
}
