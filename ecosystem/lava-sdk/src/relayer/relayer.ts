import { ConsumerSessionWithProvider } from "../types/types";
import { Secp256k1, sha256 } from "@cosmjs/crypto";
import { fromHex } from "@cosmjs/encoding";
import { grpc } from "@improbable-eng/grpc-web";
import {
  RelayRequest,
  RelayReply,
  RelaySession,
  RelayPrivateData,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { Relayer as RelayerService } from "../grpc_web_services/lavanet/lava/pairing/relay_pb_service";
import { Badge } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import transport from "../util/browser";
import transportAllowInsecure from "../util/browserAllowInsecure";

class Relayer {
  private chainID: string;
  private privKey: string;
  private lavaChainId: string;
  private prefix: string;
  private allowInsecureTransport: boolean;
  private badge?: Badge;

  constructor(
    chainID: string,
    privKey: string,
    lavaChainId: string,
    secure: boolean,
    allowInsecureTransport?: boolean,
    badge?: Badge
  ) {
    this.chainID = chainID;
    this.privKey = privKey;
    this.lavaChainId = lavaChainId;
    this.prefix = secure ? "https" : "http";
    this.allowInsecureTransport = allowInsecureTransport ?? false;
    this.badge = badge;
  }

  // when an epoch changes we need to update the badge
  public setBadge(badge: Badge | undefined) {
    if (this.badge && !badge) {
      // we have a badge and trying to set it to undefined
      throw new Error(
        "Trying to set an undefined badge to an existing badge, bad flow"
      );
    }
    this.badge = badge;
  }

  async sendRelay(
    options: SendRelayOptions,
    consumerProviderSession: ConsumerSessionWithProvider,
    cuSum: number,
    apiInterface: string
  ): Promise<RelayReply> {
    // Extract attributes from options
    const { data, url, connectionType } = options;

    const enc = new TextEncoder();

    const consumerSession = consumerProviderSession.Session;
    // Increase used compute units
    consumerProviderSession.UsedComputeUnits =
      consumerProviderSession.UsedComputeUnits + cuSum;

    // create request private data
    const requestPrivateData = new RelayPrivateData();
    requestPrivateData.setConnectionType(connectionType);
    requestPrivateData.setApiUrl(url);
    requestPrivateData.setData(enc.encode(data));
    requestPrivateData.setRequestBlock(-1); // TODO: when block parsing is implemented, replace this with the request parsed block. -1 == not applicable
    requestPrivateData.setApiInterface(apiInterface);
    requestPrivateData.setSalt(consumerSession.getNewSalt());

    const contentHash =
      this.calculateContentHashForRelayData(requestPrivateData);

    // create request session
    const requestSession = new RelaySession();
    requestSession.setSpecId(this.chainID);
    requestSession.setSessionId(consumerSession.getNewSessionId());
    requestSession.setCuSum(cuSum);
    requestSession.setProvider(consumerSession.ProviderAddress);
    requestSession.setRelayNum(consumerSession.RelayNum);
    requestSession.setEpoch(consumerSession.PairingEpoch);
    requestSession.setUnresponsiveProviders(new Uint8Array());
    requestSession.setContentHash(contentHash);
    requestSession.setSig(new Uint8Array());
    requestSession.setLavaChainId(this.lavaChainId);

    // Sign data
    const signedMessage = await this.signRelay(requestSession, this.privKey);

    requestSession.setSig(signedMessage);

    if (this.badge) {
      // Badge is separated from the signature!
      requestSession.setBadge(this.badge);
    }

    // Create request
    const request = new RelayRequest();
    request.setRelaySession(requestSession);
    request.setRelayData(requestPrivateData);

    const requestPromise = new Promise<RelayReply>((resolve, reject) => {
      grpc.invoke(RelayerService.Relay, {
        request: request,
        host: this.prefix + "://" + consumerSession.Endpoint.Addr,
        transport: this.allowInsecureTransport
          ? transportAllowInsecure // if allow insecure we use a transport with rejectUnauthorized disabled
          : transport, // otherwise normal transport (default to rejectUnauthorized = true)
        onMessage: (message: RelayReply) => {
          resolve(message);
        },
        onEnd: (code: grpc.Code, msg: string | undefined) => {
          if (code == grpc.Code.OK || msg == undefined) {
            return;
          }
          // underflow guard
          if (consumerProviderSession.UsedComputeUnits > cuSum) {
            consumerProviderSession.UsedComputeUnits =
              consumerProviderSession.UsedComputeUnits - cuSum;
          } else {
            consumerProviderSession.UsedComputeUnits = 0;
          }

          let additionalInfo = "";
          if (msg.includes("Response closed without headers")) {
            additionalInfo =
              additionalInfo +
              ", provider iPPORT: " +
              consumerProviderSession.Session.Endpoint.Addr +
              ", provider address: " +
              consumerProviderSession.Session.ProviderAddress;
          }

          const errMessage = this.extractErrorMessage(msg) + additionalInfo;
          reject(new Error(errMessage));
        },
      });
    });

    return this.relayWithTimeout(5000, requestPromise);
  }

  extractErrorMessage(error: string) {
    // Regular expression to match the desired pattern
    const regex = /desc = (.*?)(?=:\s*rpc error|$)/s;

    // Try to match the error message to the regular expression
    const match = error.match(regex);

    // If there is a match, return it; otherwise return the original error message
    if (match && match[1]) {
      return match[1].trim();
    } else {
      return error;
    }
  }

  async relayWithTimeout(timeLimit: number, task: any) {
    let timeout;
    const timeoutPromise = new Promise((resolve, reject) => {
      timeout = setTimeout(() => {
        reject(new Error("Timeout exceeded"));
      }, timeLimit);
    });
    const response = await Promise.race([task, timeoutPromise]);
    if (timeout) {
      //the code works without this but let's be safe and clean up the timeout
      clearTimeout(timeout);
    }
    return response;
  }

  byteArrayToString = (byteArray: Uint8Array): string => {
    let output = "";
    for (let i = 0; i < byteArray.length; i++) {
      const byte = byteArray[i];
      if (byte === 0x09) {
        output += "\\t";
      } else if (byte === 0x0a) {
        output += "\\n";
      } else if (byte === 0x0d) {
        output += "\\r";
      } else if (byte === 0x5c) {
        output += "\\\\";
      } else if (byte >= 0x20 && byte <= 0x7e) {
        output += String.fromCharCode(byte);
      } else {
        output += "\\" + byte.toString(8).padStart(3, "0");
      }
    }
    return output;
  };

  // Sign relay request using priv key
  async signRelay(request: RelaySession, privKey: string): Promise<Uint8Array> {
    const message = this.prepareRequest(request);

    const sig = await Secp256k1.createSignature(message, fromHex(privKey));

    const recovery = sig.recovery;
    const r = sig.r(32); // if r is not 32 bytes, add padding
    const s = sig.s(32); // if s is not 32 bytes, add padding

    // TODO consider adding compression in the signing
    // construct signature
    // <(byte of 27+public key solution)>< padded bytes for signature R><padded bytes for signature S>
    return Uint8Array.from([27 + recovery, ...r, ...s]);
  }

  calculateContentHashForRelayData(
    relayRequestData: RelayPrivateData
  ): Uint8Array {
    const requestBlock = relayRequestData.getRequestBlock();
    const requestBlockBytes =
      this.convertRequestedBlockToUint8Array(requestBlock);

    const apiInterfaceBytes = this.encodeUtf8(
      relayRequestData.getApiInterface()
    );
    const connectionTypeBytes = this.encodeUtf8(
      relayRequestData.getConnectionType()
    );
    const apiUrlBytes = this.encodeUtf8(relayRequestData.getApiUrl());
    const dataBytes = relayRequestData.getData();
    const dataUint8Array =
      dataBytes instanceof Uint8Array ? dataBytes : this.encodeUtf8(dataBytes);
    const saltBytes = relayRequestData.getSalt();
    const saltUint8Array =
      saltBytes instanceof Uint8Array ? saltBytes : this.encodeUtf8(saltBytes);

    const msgData = this.concatUint8Arrays([
      apiInterfaceBytes,
      connectionTypeBytes,
      apiUrlBytes,
      dataUint8Array,
      requestBlockBytes,
      saltUint8Array,
    ]);

    const hash = sha256(msgData);

    return hash;
  }

  convertRequestedBlockToUint8Array(requestBlock: number): Uint8Array {
    const requestBlockBytes = new Uint8Array(8);
    let number = BigInt(requestBlock);
    if (requestBlock < 0) {
      // Convert the number to its 64-bit unsigned representation
      const maxUint64 = BigInt(2) ** BigInt(64);
      number = maxUint64 + BigInt(requestBlock);
    }

    // Copy the bytes from the unsigned representation to the byte array
    for (let i = 0; i < 8; i++) {
      requestBlockBytes[i] = Number((number >> BigInt(8 * i)) & BigInt(0xff));
    }

    return requestBlockBytes;
  }

  encodeUtf8(str: string): Uint8Array {
    return new TextEncoder().encode(str);
  }

  concatUint8Arrays(arrays: Uint8Array[]): Uint8Array {
    const totalLength = arrays.reduce((acc, arr) => acc + arr.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    arrays.forEach((arr) => {
      result.set(arr, offset);
      offset += arr.length;
    });
    return result;
  }

  prepareRequest(request: RelaySession): Uint8Array {
    const enc = new TextEncoder();

    const jsonMessage = JSON.stringify(request.toObject(), (key, value) => {
      if (key == "content_hash") {
        const dataBytes = request.getContentHash();
        const dataUint8Array =
          dataBytes instanceof Uint8Array
            ? dataBytes
            : this.encodeUtf8(dataBytes);

        let stringByte = this.byteArrayToString(dataUint8Array);

        if (stringByte.endsWith(",")) {
          stringByte += ",";
        }
        return stringByte;
      }
      if (value !== null && value !== 0 && value !== "") return value;
    });

    const messageReplaced = jsonMessage
      .replace(/\\\\/g, "\\")
      .replace(/,"/g, ' "')
      .replace(/, "/g, ',"')
      .replace(/"(\w+)"\s*:/g, "$1:")
      .slice(1, -1);

    const encodedMessage = enc.encode(messageReplaced + " ");

    const hash = sha256(encodedMessage);

    return hash;
  }
}

/**
 * Options for send relay method.
 */
interface SendRelayOptions {
  data: string;
  url: string;
  connectionType: string;
}

export default Relayer;
