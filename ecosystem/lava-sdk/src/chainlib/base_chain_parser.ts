import {
  FUNCTION_TAG,
  ParseDirective,
  ApiCollection,
  Api,
  Header,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { Metadata } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { Spec } from "../grpc_web_services/lavanet/lava/spec/spec_pb";
import { Logger } from "../logger/logger";
import Long from "long";

export const APIInterfaceJsonRPC = "jsonrpc";
export const APIInterfaceTendermintRPC = "tendermintrpc";
export const APIInterfaceRest = "rest";
export const APIInterfaceGrpc = "grpc";
export const HeadersPassSend = Header.HeaderType.PASS_SEND;

/**
 * Options for sending RPC relay.
 */
export interface SendRelayOptions {
  method: string; // Required: The RPC method to be called
  params: Array<any>; // Required: An array of parameters to be passed to the RPC method
  chainId?: string; // Optional: the chain id to send the request to, if only one chain is initialized it will be chosen by default
  metadata?: Metadata[]; // Optional headers to be sent with the request.
}

/**
 * Options for sending Rest relay.
 */
export interface SendRestRelayOptions {
  connectionType: string; // Required: The HTTP method to be used (e.g., "GET", "POST")
  url: string; // Required: The API Path (URL) (e.g Cosmos: "/cosmos/base/tendermint/v1beta1/blocks/latest", Aptos: "/transactions" )
  // eslint-disable-next-line
  data?: Record<string, any>; // Optional: An object containing data to be sent in the request body (applicable for methods like "POST" and "PUT")
  chainId?: string; // Optional: the chain id to send the request to, if only one chain is initialized it will be chosen by default
  metadata?: Metadata[]; // Optional headers to be sent with the request.
}

export interface ApiKey {
  name: string;
  connectionType: string;
}

export type ApiKeyString = string;

export function ApiKeyToString(key: ApiKey): ApiKeyString {
  return JSON.stringify(key);
}

interface TaggedContainer {
  parsing: ParseDirective;
  apiCollection: ApiCollection;
}

export interface CollectionKey {
  connectionType: string;
  internalPath: string;
  addon: string;
}

export type CollectionKeyString = string;

export function CollectionKeyToString(key: CollectionKey): CollectionKeyString {
  return JSON.stringify(key);
}

interface ApiContainer {
  api: Api;
  collectionKey: CollectionKey;
  apiKey: ApiKey;
}

interface HeaderContainer {
  header: Header;
  apiKey: ApiKey;
}

interface VerificationKey {
  extension: string;
  addon: string;
}

interface VerificationContainer {
  connectionType: string;
  name: string;
  parseDirective: ParseDirective;
  value: string;
  latestDistance: Long;
  verificationKey: VerificationKey;
}

type VerificationKeyString = string;

function VerificationKeyToString(key: VerificationKey): VerificationKeyString {
  return JSON.stringify(key);
}

interface HeadersHandler {
  filteredHeaders: Metadata[];
  overwriteRequestedBlock: string;
  ignoredMetadata: Metadata[];
}

export interface ChainBlockStats {
  allowedBlockLagForQosSync: number;
  averageBlockTime: number;
  blockDistanceForFinalizedData: number;
  blocksInFinalizationProof: number;
}

interface DataReliabilityParams {
  enabled: boolean;
  dataReliabilityThreshold: number;
}

export abstract class BaseChainParser {
  protected taggedApis: Map<number, TaggedContainer>;
  protected spec: Spec | undefined;
  protected serverApis: Map<ApiKeyString, ApiContainer>;
  protected headers: Map<ApiKeyString, HeaderContainer>;
  protected apiCollections: Map<CollectionKeyString, ApiCollection>;
  // TODO: implement addons.
  protected allowedAddons: Set<string>;
  // private extensionParser: ExtensionParser;
  public apiInterface = "";
  protected verifications: Map<VerificationKeyString, VerificationContainer[]>;

  constructor() {
    this.taggedApis = new Map();
    this.serverApis = new Map();
    this.apiCollections = new Map();
    this.headers = new Map();
    this.allowedAddons = new Set();
    this.verifications = new Map();
  }

  protected getSupportedApi(
    name: string,
    connectionType: string
  ): ApiContainer {
    const apiKey: ApiKey = {
      name,
      connectionType,
    };
    const apiCont = this.serverApis.get(ApiKeyToString(apiKey));
    if (!apiCont) {
      throw Logger.fatal("api not supported", name, connectionType);
    }
    if (!apiCont.api.getEnabled()) {
      throw Logger.fatal("api is disabled in spec", name, connectionType);
    }
    return apiCont;
  }

  protected getApiCollection(collectionKey: CollectionKey): ApiCollection {
    const key = CollectionKeyToString(collectionKey);
    const collection = this.apiCollections.get(key);
    if (!collection) {
      throw Logger.fatal("Api not supported", collectionKey);
    }
    if (collection.getEnabled()) {
      throw Logger.fatal("Api disabled in spec", collectionKey);
    }
    return collection;
  }

  public dataReliabilityParams(): DataReliabilityParams {
    // TODO: implement this
    const spec = this.spec;
    if (spec == undefined) {
      throw new Error("spec undefined can't get stats");
    }
    return {
      enabled: spec.getDataReliabilityEnabled(),
      dataReliabilityThreshold: spec.getReliabilityThreshold(),
    };
  }

  // initialize the base chain parser with the spec information
  public init(spec: Spec) {
    if (this.apiInterface == "") {
      throw Logger.fatal("Chain parser apiInterface is not set");
    }
    this.spec = spec;

    if (spec.getEnabled()) {
      for (const apiCollection of spec.getApiCollectionsList()) {
        if (!apiCollection.getEnabled()) {
          continue;
        }
        if (
          apiCollection.getCollectionData()?.getApiInterface() !=
          this.apiInterface
        ) {
          continue;
        }

        const connectionType = apiCollection.getCollectionData()?.getType();
        if (connectionType == undefined) {
          //TODO change message
          throw Logger.fatal(
            "Missing verification parseDirective data in BaseChainParser constructor"
          );
        }
        const internalPath = apiCollection
          .getCollectionData()
          ?.getInternalPath();
        if (internalPath == undefined) {
          //TODO change message
          throw Logger.fatal(
            "Missing verification parseDirective data in BaseChainParser constructor"
          );
        }
        const addon = apiCollection.getCollectionData()?.getAddOn();
        if (addon == undefined) {
          //TODO change message
          throw Logger.fatal(
            "Missing verification parseDirective data in BaseChainParser constructor"
          );
        }
        const collectionKey: CollectionKey = {
          connectionType: connectionType,
          internalPath: internalPath,
          addon: addon,
        };

        // parse directives
        for (const parsing of apiCollection.getParseDirectivesList()) {
          this.taggedApis.set(parsing.getFunctionTag(), {
            parsing: parsing,
            apiCollection: apiCollection,
          });
        }

        // parse api collection
        for (const api of apiCollection.getApisList()) {
          if (!api.getEnabled()) {
            continue;
          }
          let apiName = api.getName();
          if (this.apiInterface == APIInterfaceRest) {
            const re = /{[^}]+}/;
            apiName = api.getName().replace(re, "replace-me-with-regex");
            apiName = apiName.replace(/replace-me-with-regex/g, "[^\\/\\s]+");
            apiName = this.escapeRegExp(apiName); // Assuming you have a RegExp.escape function
          }
          const apiKey: ApiKey = {
            name: apiName,
            connectionType: collectionKey.connectionType,
          };
          this.serverApis.set(ApiKeyToString(apiKey), {
            apiKey: apiKey,
            api: api,
            collectionKey: collectionKey,
          });
        }

        // Parse headers
        for (const header of apiCollection.getHeadersList()) {
          const apiKeyHeader: ApiKey = {
            name: header.getName(),
            connectionType: collectionKey.connectionType,
          };
          this.headers.set(ApiKeyToString(apiKeyHeader), {
            header: header,
            apiKey: apiKeyHeader,
          });
        }

        for (const verification of apiCollection.getVerificationsList()) {
          for (const parseValue of verification.getValuesList()) {
            const addons = apiCollection.getCollectionData()?.getAddOn();
            if (addons == undefined) {
              //TODO change message
              throw Logger.fatal(
                "Missing verification parseDirective data in BaseChainParser constructor"
              );
            }

            const value = parseValue.toObject();
            const verificationKey: VerificationKey = {
              extension: value.extension,
              addon: addons,
            };
            if (!verification.getParseDirective()) {
              throw Logger.fatal(
                "Missing verification parseDirective data in BaseChainParser constructor",
                verification
              );
            }
            const connectionType = apiCollection.getCollectionData()?.getType();
            if (connectionType == undefined) {
              throw Logger.fatal(
                "Missing verification parseDirective data in BaseChainParser constructor"
              );
            }
            const parseDirective = verification.getParseDirective();
            if (parseDirective == undefined) {
              throw Logger.fatal(
                "Missing verification parseDirective data in BaseChainParser constructor"
              );
            }

            const verificationContainer: VerificationContainer = {
              connectionType: connectionType,
              name: verification.getName(),
              parseDirective: parseDirective,
              value: parseValue.getExpectedValue(),
              latestDistance: Long.fromNumber(parseValue.getLatestDistance()),
              verificationKey: verificationKey,
            };
            const vfkey = VerificationKeyToString(verificationKey);
            const existingVerifications = this.verifications.get(vfkey);
            if (!existingVerifications) {
              this.verifications.set(vfkey, [verificationContainer]);
            } else {
              existingVerifications.push(verificationContainer);
            }
          }
        }
        this.apiCollections.set(
          CollectionKeyToString(collectionKey),
          apiCollection
        );
      }
    }
  }

  protected isRest(
    options: SendRelayOptions | SendRestRelayOptions
  ): options is SendRestRelayOptions {
    return "connectionType" in options; // how to check which options were given
  }

  protected handleHeaders(
    metadata: Metadata[] | undefined,
    apiCollection: ApiCollection,
    headersDirection: number
  ): HeadersHandler {
    if (!metadata || metadata.length == 0) {
      return {
        filteredHeaders: [],
        overwriteRequestedBlock: "",
        ignoredMetadata: [],
      };
    }
    const retMetaData: Metadata[] = [];
    const ignoredMetadata: Metadata[] = [];
    let overwriteRequestedBlock = "";
    for (const header of metadata) {
      const headerName = header.getName().toLowerCase();
      if (!apiCollection.getCollectionData()) {
        throw Logger.fatal(
          "Missing api collection data in handleHeaders",
          apiCollection
        );
      }
      const connectionType = apiCollection.getCollectionData()?.getType();
      if (connectionType == undefined) {
        // TODO fix
        throw Logger.fatal(
          "Missing api collection data in handleHeaders",
          apiCollection
        );
      }
      const apiKey: ApiKey = {
        name: headerName,
        connectionType: connectionType,
      };

      const headerDirective = this.headers.get(ApiKeyToString(apiKey));
      if (!headerDirective) {
        continue; // this header is not handled
      }
      if (
        headerDirective.header.getKind() ==
          <number>(<unknown>headersDirection) ||
        headerDirective.header.getKind() == Header.HeaderType.PASS_BOTH
      ) {
        retMetaData.push(header);
        if (
          headerDirective.header.getFunctionTag() ==
          FUNCTION_TAG.SET_LATEST_IN_METADATA
        ) {
          overwriteRequestedBlock = header.getValue();
        }
      } else if (
        headerDirective.header.getKind() == Header.HeaderType.PASS_IGNORE
      ) {
        ignoredMetadata.push(header);
      }
    }

    return {
      filteredHeaders: retMetaData,
      ignoredMetadata: ignoredMetadata,
      overwriteRequestedBlock: overwriteRequestedBlock,
    };
  }

  protected isAddon(addon: string): boolean {
    return this.allowedAddons.has(addon);
  }

  protected escapeRegExp(s: string): string {
    return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }

  protected matchSpecApiByName(
    name: string,
    connectionType: string
  ): [ApiContainer | undefined, boolean] {
    let foundNameOnDifferentConnectionType: string | undefined = undefined;
    for (const [, api] of this.serverApis.entries()) {
      const re = new RegExp(`^${api.apiKey.name}$`);
      if (re.test(name)) {
        if (api.apiKey.connectionType === connectionType) {
          return [api, true];
        } else {
          foundNameOnDifferentConnectionType = api.apiKey.connectionType;
        }
      }
    }
    if (foundNameOnDifferentConnectionType) {
      Logger.warn(
        `Found the api on a different connection type, found: ${foundNameOnDifferentConnectionType}, requested: ${connectionType}`
      );
    }
    return [undefined, false];
  }

  abstract parseMsg(
    options: SendRelayOptions | SendRestRelayOptions
  ): ChainMessage;

  public chainBlockStats(): ChainBlockStats {
    const averageBlockTime = this.spec?.getAverageBlockTime();
    if (!averageBlockTime) {
      throw Logger.fatal("no average block time in spec", this.spec);
    }
    const allowedLag = this.spec?.getAllowedBlockLagForQosSync();
    if (!allowedLag) {
      throw Logger.fatal("no allowed lag in spec", this.spec);
    }

    const blockDistanceForFinalizedData =
      this.spec?.getBlockDistanceForFinalizedData();
    if (!blockDistanceForFinalizedData) {
      throw Logger.fatal(
        "no block distance for finalized data in spec",
        this.spec
      );
    }
    const blocksInFinalizationProof = this.spec?.getBlocksInFinalizationProof();
    if (!blocksInFinalizationProof) {
      throw Logger.fatal("no block in finalization proof in spec", this.spec);
    }
    return {
      allowedBlockLagForQosSync: allowedLag,
      averageBlockTime: averageBlockTime,
      blockDistanceForFinalizedData: blockDistanceForFinalizedData,
      blocksInFinalizationProof: blocksInFinalizationProof,
    };
  }
}

export interface RawRequestData {
  url: string;
  data: string;
}

export class ChainMessage {
  private requestedBlock: number;
  private api: Api;
  private apiCollection: ApiCollection;
  private messageData: string;
  private messageUrl: string;
  public headers: Metadata[] = [];
  constructor(
    requestedBlock: number,
    api: Api,
    apiCollection: ApiCollection,
    data: string,
    messageUrl: string
  ) {
    this.requestedBlock = requestedBlock;
    this.apiCollection = apiCollection;
    this.api = api;
    this.messageData = data;
    this.messageUrl = messageUrl;
  }

  public getRawRequestData(): RawRequestData {
    return { url: this.messageUrl, data: this.messageData };
  }

  public getMessageUrl(): string {
    return this.messageUrl;
  }

  public getRequestedBlock(): number {
    return this.requestedBlock;
  }

  public updateLatestBlockInMessage(
    latestBlock: number,
    modififyContent: boolean
  ): boolean {
    return false; // TODO: implement
  }

  public appendHeader(metaData: Metadata[]) {
    this.headers = [...this.headers, ...metaData];
  }

  public getApi(): Api {
    return this.api;
  }

  public getApiCollection(): ApiCollection {
    return this.apiCollection;
  }
}
