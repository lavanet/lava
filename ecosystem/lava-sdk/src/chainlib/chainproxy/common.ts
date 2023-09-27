import { Metadata } from "../../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { ParseDirective } from "../../grpc_web_services/lavanet/lava/spec/api_collection_pb";

export interface BaseMessageOptions {
  headers: Metadata[];
  latestBlockHeaderSetter: ParseDirective | undefined;
}

export abstract class BaseMessage {
  private headers: Metadata[];
  private latestBlockHeaderSetter: ParseDirective | undefined;

  constructor({ headers = [], latestBlockHeaderSetter }: BaseMessageOptions) {
    this.headers = headers ?? [];
    this.latestBlockHeaderSetter = latestBlockHeaderSetter;
  }

  appendHeader(metadata: Metadata[]) {
    const existing: { [key: string]: boolean } = {};
    for (const metadataEntry of this.headers) {
      existing[metadataEntry.getName()] = true;
    }
    for (const metadataEntry of metadata) {
      if (!existing[metadataEntry.getName()]) {
        this.headers.push(metadataEntry);
      }
    }
  }

  setLatestBlockWithHeader(
    latestBlock: number,
    modifyContent: boolean
  ): boolean {
    if (!this.latestBlockHeaderSetter) {
      return false;
    }
    const headerValue = `${this.latestBlockHeaderSetter.getFunctionTemplate()}${latestBlock}`;
    for (let idx = 0; idx < this.headers.length; idx++) {
      const header = this.headers[idx];
      if (header.getName() === this.latestBlockHeaderSetter.getApiName()) {
        if (modifyContent) {
          this.headers[idx].setValue(headerValue);
        }
        return true;
      }
    }
    if (modifyContent) {
      const metadata = new Metadata();
      metadata.setName(this.latestBlockHeaderSetter.getApiName());
      metadata.setValue(headerValue);
      this.headers.push(metadata);
    }
    return true;
  }

  getHeaders(): Metadata[] {
    return this.headers;
  }
}
