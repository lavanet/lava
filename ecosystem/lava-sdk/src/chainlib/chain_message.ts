import { LATEST_BLOCK, NOT_APPLICABLE } from "../common/common";
import { Metadata } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import {
  Api,
  ApiCollection,
  Extension,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { GenericMessage } from "./chainproxy/rpcInterfaceMessages/common";

interface RawRequestData {
  url: string;
  data: string;
}

export interface UpdatableRPCInput extends GenericMessage {
  updateLatestBlockInMessage(
    latestBlock: number,
    modifyContent: boolean
  ): boolean;
  appendHeader(metadata: Metadata[]): void;
}

export class BaseChainMessageContainer {
  private api: Api;
  private requestedBlock: number;
  private msg: UpdatableRPCInput;
  private apiCollection: ApiCollection;
  private extensions: Extension[];
  private messageUrl: string;
  private messageData: string;

  constructor(
    api: Api,
    requestedBlock: number,
    msg: UpdatableRPCInput,
    apiCollection: ApiCollection,
    messageData: string,
    messageUrl?: string,
    extensions?: Extension[]
  ) {
    this.api = api;
    this.requestedBlock = requestedBlock;
    this.msg = msg;
    this.apiCollection = apiCollection;
    this.messageData = messageData;
    this.messageUrl = messageUrl ?? "";
    this.extensions = extensions ?? [];
  }

  public getRawRequestData(): RawRequestData {
    return { url: this.messageUrl, data: this.messageData };
  }

  public appendHeader(metadata: Metadata[]): void {
    this.msg.appendHeader(metadata);
  }

  public getApi(): Api {
    return this.api;
  }

  public getApiCollection(): ApiCollection {
    return this.apiCollection;
  }

  public getRequestedBlock(): number {
    return this.requestedBlock;
  }

  public getRPCMessage(): GenericMessage {
    return this.msg;
  }

  public updateLatestBlockInMessage(
    latestBlock: number,
    modifyContent: boolean
  ): boolean {
    const requestedBlock = this.getRequestedBlock();
    if (latestBlock <= NOT_APPLICABLE || requestedBlock !== LATEST_BLOCK) {
      return false;
    }

    const success = this.msg.updateLatestBlockInMessage(
      latestBlock,
      modifyContent
    );

    if (success) {
      this.requestedBlock = latestBlock;
      return true;
    }
    return false;
  }

  public getExtensions(): Extension[] {
    return this.extensions;
  }

  public setExtension(extension: Extension): void {
    if (this.extensions.length > 0) {
      for (const ext of this.extensions) {
        if (ext.getName() === extension.getName()) {
          // Already existing, no need to add
          return;
        }
      }
      this.extensions.push(extension);
    } else {
      this.extensions = [extension];
    }
  }
}
