import { BOOT_RETRY_ATTEMPTS } from "../config/default";
import { Secp256k1, sha256 } from "@cosmjs/crypto";
import { fromHex } from "@cosmjs/encoding";
import { grpc } from "@improbable-eng/grpc-web";
import {
  RelayRequest,
  RelayReply,
  RelaySession,
  RelayPrivateData,
  Badge,
  ProbeRequest,
  ProbeReply,
  ReportedProvider,
  QualityOfServiceReport,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import {
  Relayer as RelayerService,
  RelayerClient,
  ServiceError,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb_service";
import transport from "../util/browser";
import transportAllowInsecure from "../util/browserAllowInsecure";
import { SingleConsumerSession } from "../lavasession/consumerTypes";
import SDKErrors from "../sdk/errors";

export interface RelayerOptions {
  privKey: string;
  secure: boolean;
  allowInsecureTransport: boolean;
  lavaChainId: string;
  transport?: grpc.TransportFactory;
}

export class Relayer {
  private privKey: string;
  private lavaChainId: string;
  private prefix: string;
  private allowInsecureTransport: boolean;
  private badge?: Badge;
  private transport: grpc.TransportFactory | undefined;

  constructor(relayerOptions: RelayerOptions) {
    this.privKey = relayerOptions.privKey;
    this.lavaChainId = relayerOptions.lavaChainId;
    this.prefix = relayerOptions.secure ? "https" : "http";
    this.allowInsecureTransport =
      relayerOptions.allowInsecureTransport ?? false;
    if (relayerOptions.transport) {
      this.transport = relayerOptions.transport;
    }
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

  async probeProvider(
    providerAddress: string,
    apiInterface: string,
    guid: number,
    specId: string
  ): Promise<ProbeReply> {
    const client = new RelayerClient(
      this.prefix + "://" + providerAddress,
      this.getTransportWrapped()
    );
    const request = new ProbeRequest();
    request.setGuid(guid);
    request.setApiInterface(apiInterface);
    request.setSpecId(specId);
    const requestPromise = new Promise<ProbeReply>((resolve, reject) => {
      client.probe(
        request,
        (err: ServiceError | null, result: ProbeReply | null) => {
          if (err != null) {
            console.log("failed sending probe", err);
            reject(err);
          }

          if (result != null) {
            resolve(result);
          }
          reject(new Error("Didn't get an error nor result"));
        }
      );
    });
    return this.relayWithTimeout(5000, requestPromise);
  }

  public async sendRelay(
    client: RelayerClient,
    relayRequest: RelayRequest,
    timeout: number
  ): Promise<RelayReply | Error> {
    const requestSession = relayRequest.getRelaySession();
    if (requestSession == undefined) {
      return new Error("empty request session");
    }
    requestSession.setSig(new Uint8Array());
    // Sign data
    const signedMessage = await this.signRelay(requestSession, this.privKey);
    requestSession.setSig(signedMessage);
    if (this.badge) {
      // Badge is separated from the signature!
      requestSession.setBadge(this.badge);
    }
    relayRequest.setRelaySession(requestSession);
    const requestPromise = new Promise<RelayReply>((resolve, reject) => {
      client.relay(
        relayRequest,
        (err: ServiceError | null, result: RelayReply | null) => {
          if (err != null) {
            console.log("failed sending relay", err);
            reject(err);
          }

          if (result != null) {
            resolve(result);
          }
          reject(new Error("Didn't get an error nor result"));
        }
      );
    });

    return this.relayWithTimeout(timeout, requestPromise);
  }

  public async constructAndSendRelay(
    options: SendRelayOptions,
    singleConsumerSession: SingleConsumerSession
  ): Promise<RelayReply> {
    // Extract attributes from options
    const { data, url, connectionType } = options;

    const enc = new TextEncoder();

    // create request private data
    const requestPrivateData = new RelayPrivateData();
    requestPrivateData.setConnectionType(connectionType);
    requestPrivateData.setApiUrl(url);
    requestPrivateData.setData(enc.encode(data));
    requestPrivateData.setRequestBlock(-1); // TODO: when block parsing is implemented, replace this with the request parsed block. -1 == not applicable
    requestPrivateData.setApiInterface(options.apiInterface);
    requestPrivateData.setSalt(this.getNewSalt());

    const contentHash =
      this.calculateContentHashForRelayData(requestPrivateData);

    // create request session
    const requestSession = new RelaySession();
    requestSession.setSpecId(options.chainId);
    requestSession.setSessionId(singleConsumerSession.sessionId);
    requestSession.setCuSum(singleConsumerSession.cuSum);
    requestSession.setProvider(options.publicProviderLavaAddress);
    requestSession.setRelayNum(singleConsumerSession.relayNum);
    requestSession.setEpoch(options.epoch);
    requestSession.setUnresponsiveProvidersList(new Array<ReportedProvider>());
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
    const transportation = this.getTransport();
    const requestPromise = new Promise<RelayReply>((resolve, reject) => {
      grpc.invoke(RelayerService.Relay, {
        request: request,
        host:
          this.prefix + "://" + singleConsumerSession.endpoint.networkAddress,
        transport: transportation, // otherwise normal transport (default to rejectUnauthorized = true)
        onMessage: (message: RelayReply) => {
          resolve(message);
        },
        onEnd: (code: grpc.Code, msg: string | undefined) => {
          if (code == grpc.Code.OK || msg == undefined) {
            return;
          }
          let additionalInfo = "";
          if (msg.includes("Response closed without headers")) {
            additionalInfo =
              additionalInfo +
              ", provider iPPORT: " +
              singleConsumerSession.endpoint.networkAddress +
              ", provider address: " +
              options.publicProviderLavaAddress;
          }
          const errMessage = this.extractErrorMessage(msg) + additionalInfo;
          reject(new Error(errMessage));
        },
      });
    });

    return this.relayWithTimeout(5000, requestPromise);
  }

  getTransport() {
    if (this.transport) {
      return this.transport;
    }
    return this.allowInsecureTransport ? transportAllowInsecure : transport;
  }

  getTransportWrapped() {
    return {
      // if allow insecure we use a transport with rejectUnauthorized disabled
      // otherwise normal transport (default to rejectUnauthorized = true));}
      transport: this.getTransport(),
    };
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
        reject(SDKErrors.relayTimeout);
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
      } else if (byte === 0x22) {
        output += '\\"';
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
    // TODO: we serialize the message here the same way gogo proto serializes there's no straightforward implementation available, but we should compile this code into wasm and import it here because it's ugly
    let serializedRequest = "";
    for (const [key, valueInner] of Object.entries(request.toObject())) {
      serializedRequest += ((
        key: string,
        value:
          | string
          | number
          | Uint8Array
          | QualityOfServiceReport.AsObject
          | ReportedProvider.AsObject[]
          | Badge.AsObject
      ) => {
        function handleNumStr(key: string, value: number | string): string {
          switch (typeof value) {
            case "string":
              if (value == "") {
                return "";
              }
              return key + ':"' + value + '" ';
            case "number":
              if (value == 0) {
                return "";
              }
              return key + ":" + value + " ";
          }
        }
        if (value == undefined) {
          return "";
        }
        switch (typeof value) {
          case "string":
          case "number":
            return handleNumStr(key, value);
          case "object":
            let valueInnerStr = "";
            if (value instanceof Uint8Array) {
              valueInnerStr = this.byteArrayToString(value);
              return key + ':"' + valueInnerStr + '" ';
            }
            if (value instanceof Array) {
              let retst = "";
              for (const arrayVal of Object.values(value)) {
                let arrayValstr = "";
                const entries = Object.entries(arrayVal);
                for (const [objkey, objVal] of entries) {
                  const objValStr = handleNumStr(objkey, objVal);
                  if (objValStr != "") {
                    arrayValstr += objValStr;
                  }
                }
                if (arrayValstr != "") {
                  retst += key + ":<" + arrayValstr + "> ";
                }
              }
              return retst;
            }
            const entries = Object.entries(value);
            if (entries.length == 0) {
              return "";
            }
            let retst = "";
            for (const [objkey, objVal] of entries) {
              let objValStr = "";
              switch (typeof objVal) {
                case "string":
                case "number":
                  objValStr = handleNumStr(objkey, objVal);
                  break;
                case "object":
                  objValStr = objkey + ":" + this.byteArrayToString(objVal);
                  break;
              }
              if (objValStr != "") {
                retst += objValStr;
              }
            }
            if (retst != "") {
              return key + ":<" + retst + "> ";
            }
            return "";
        }
      })(key, valueInner);
    }
    // console.log("message: " + serializedRequest);
    const encodedMessage = enc.encode(serializedRequest);
    // console.log("encodedMessage: " + encodedMessage);
    const hash = sha256(encodedMessage);

    return hash;
  }

  // SendRelayToAllProvidersAndRace sends relay to all lava providers and returns first response
  public async SendRelayToAllProvidersAndRace(
    batch: BatchRelays[]
  ): Promise<any> {
    console.log("Started sending to all providers and race");
    let lastError;
    for (
      let retryAttempt = 0;
      retryAttempt < BOOT_RETRY_ATTEMPTS;
      retryAttempt++
    ) {
      const allRelays: Map<string, Promise<any>> = new Map();
      for (const provider of batch) {
        const uniqueKey =
          provider.options.publicProviderLavaAddress +
          String(Math.floor(Math.random() * 10000000));
        const providerRelayPromise = this.constructAndSendRelay(
          provider.options,
          provider.singleConsumerSession
        );
        allRelays.set(uniqueKey, providerRelayPromise);
      }

      while (allRelays.size > 0) {
        const returnedResponse = await Promise.race([...allRelays.values()]);
        if (returnedResponse) {
          console.log("Ended sending to all providers and race");
          return returnedResponse;
        }
        // Handle removal of completed promises separately (Optional and based on your needs)
        allRelays.forEach((promise, key) => {
          promise
            .then(() => allRelays.delete(key))
            .catch(() => allRelays.delete(key));
        });
      }
    }
    throw new Error(
      "Failed all promises SendRelayToAllProvidersAndRace: " + String(lastError)
    );
  }

  getNewSalt(): Uint8Array {
    const salt = this.generateRandomUint();
    const nonceBytes = new Uint8Array(8);
    const dataView = new DataView(nonceBytes.buffer);

    // use LittleEndian
    dataView.setBigUint64(0, BigInt(salt), true);

    return nonceBytes;
  }

  private generateRandomUint(): number {
    const min = 1;
    const max = Number.MAX_SAFE_INTEGER;
    return Math.floor(Math.random() * (max - min) + min);
  }
}

export interface BatchRelays {
  options: SendRelayOptions;
  singleConsumerSession: SingleConsumerSession;
}

/**
 * Options for send relay method.
 */
export interface SendRelayOptions {
  data: string;
  url: string;
  connectionType: string;
  apiInterface: string;
  chainId: string;
  publicProviderLavaAddress: string;
  epoch: number;
}
