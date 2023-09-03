import {
  functionTag,
  ParseDirective,
  ApiCollection,
  Api,
  Header,
  Extension,
} from "../codec/lavanet/lava/spec/api_collection";
import { Spec } from "../codec/lavanet/lava/spec/spec";

const APIInterfaceJsonRPC = "jsonrpc";
const APIInterfaceTendermintRPC = "tendermintrpc";
const APIInterfaceRest = "rest";
const APIInterfaceGrpc = "grpc";

interface ApiKey {
  name: string;
  connectionType: string;
}

interface TaggedContainer {
  parsing: ParseDirective;
  apiCollection: ApiCollection;
}

interface CollectionKey {
  connectionType: string;
  internalPath: string;
  addon: string;
}

interface ApiContainer {
  api: Api;
  collectionKey: CollectionKey;
}

interface ExtensionKey {
  Extension: string;
  ConnectionType: string;
  InternalPath: string;
  Addon: string;
}

interface ExtensionParser {
  allowedExtensions: Map<string, null>;
  configuredExtensions: Map<ExtensionKey, Extension>;
}

class BaseChainParser {
  private taggedApis: Map<functionTag, TaggedContainer>;
  private spec: Spec;
  private serverApis: Map<ApiKey, ApiContainer>;
  private apiCollections: Map<CollectionKey, ApiCollection>;
  private headers: Map<ApiKey, Header>;
  private allowedAddons: Map<string, null>;
  // private extensionParser: ExtensionParser; TODO support extensions
  private apiInterface: string;

  constructor(spec: Spec, apiInterface: string) {
    this.apiInterface = apiInterface;
    this.taggedApis = new Map();
    this.serverApis = new Map();
    this.apiCollections = new Map();
    this.headers = new Map();
    this.allowedAddons = new Map();
    this.spec = spec;

    if (spec.enabled) {
      for (const apiCollection of spec.apiCollections) {
        if (!apiCollection.enabled) {
          continue;
        }
        if (apiCollection.collectionData?.apiInterface != apiInterface) {
          continue;
        }
        const collectionKey: CollectionKey = {
          connectionType: apiCollection.collectionData.type,
          internalPath: apiCollection.collectionData.internalPath,
          addon: apiCollection.collectionData.addOn,
        };

        for (const parsing of apiCollection.parseDirectives) {
          this.taggedApis.set(parsing.functionTag, {
            parsing: parsing,
            apiCollection: apiCollection,
          });
        }

        for (const api of apiCollection.apis) {
          if (!api.enabled) {
            continue;
          }
          let apiName = api.name;
          if (apiInterface == APIInterfaceRest) {
            const re = /{[^}]+}/;
            apiName = api.name.replace(re, "replace-me-with-regex");
            apiName = apiName.replace(/replace-me-with-regex/g, "[^\\/\\s]+");
            apiName = this.escapeRegExp(apiName); // Assuming you have a RegExp.escape function
          }
          this.serverApis.set(
            {
              name: apiName,
              connectionType: collectionKey.connectionType,
            },
            {
              api: api,
              collectionKey: collectionKey,
            }
          );
        }

        for (const header of apiCollection.headers) {
          this.headers.set(
            {
              name: header.name,
              connectionType: collectionKey.connectionType,
            },
            header
          );
        }
      }
    }
  }

  escapeRegExp(s: string): string {
    return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }
}
