import { Extension } from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";

interface ExtensionKey {
  Extension: string;
  ConnectionType: string;
  InternalPath: string;
  Addon: string;
}

// TODO: implement extension parser.
export class ExtensionParser {
  private allowedExtensions: Set<string>;
  private configuredExtensions: Map<ExtensionKey, Extension>;
  constructor() {
    this.allowedExtensions = new Set();
    this.configuredExtensions = new Map();
  }
}
