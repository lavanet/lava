import {
  EARLIEST_BLOCK,
  FINALIZED_BLOCK,
  LATEST_BLOCK,
  NOT_APPLICABLE,
  SAFE_BLOCK,
} from "../common/common";
import {
  RelayRequest,
  RelaySession,
  RelayPrivateData,
  QualityOfServiceReport,
  RelayReply,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";
import { SingleConsumerSession } from "../lavasession/consumerTypes";
import { sha256 } from "@cosmjs/crypto";
import { Decimal } from "@cosmjs/math";

export interface SendRelayData {
  connectionType: string;
  data: string;
  url: string;
  apiInterface: string;
  chainId: string;
  requestedBlock: number;
}

export function newRelayData(relayData: SendRelayData): RelayPrivateData {
  const { data, url, connectionType } = relayData;
  // create request private data
  const enc = new TextEncoder();
  const requestPrivateData = new RelayPrivateData();
  requestPrivateData.setConnectionType(connectionType);
  requestPrivateData.setApiUrl(url);
  requestPrivateData.setData(enc.encode(data));
  requestPrivateData.setRequestBlock(-1); // TODO: when block parsing is implemented, replace this with the request parsed block. -1 == not applicable
  requestPrivateData.setApiInterface(relayData.apiInterface);
  requestPrivateData.setSalt(getNewSalt());
  return requestPrivateData;
}

function getNewSalt(): Uint8Array {
  const salt = generateRandomUint();
  const nonceBytes = new Uint8Array(8);
  const dataView = new DataView(nonceBytes.buffer);

  // use LittleEndian
  dataView.setBigUint64(0, BigInt(salt), true);

  return nonceBytes;
}

function generateRandomUint(): number {
  const min = 1;
  const max = Number.MAX_SAFE_INTEGER;
  return Math.floor(Math.random() * (max - min) + min);
}

export function constructRelayRequest(
  lavaChainID: string,
  chainID: string,
  relayData: RelayPrivateData,
  providerAddress: string,
  singleConsumerSession: SingleConsumerSession,
  epoch: number,
  reportedProviders: string
): RelayRequest {
  const relayRequest = new RelayRequest();
  relayRequest.setRelayData(relayData);
  const relaySession = constructRelaySession(
    lavaChainID,
    chainID,
    relayData,
    providerAddress,
    singleConsumerSession,
    epoch,
    reportedProviders
  );
  relayRequest.setRelaySession(relaySession);
  return relayRequest;
}

function constructRelaySession(
  lavaChainID: string,
  chainID: string,
  relayData: RelayPrivateData,
  providerAddress: string,
  singleConsumerSession: SingleConsumerSession,
  epoch: number,
  reportedProviders: string
): RelaySession {
  const lastQos = singleConsumerSession.qoSInfo.lastQoSReport;
  const newQualityOfServiceReport = new QualityOfServiceReport();
  if (lastQos != undefined) {
    // TODO: needs to serialize the QoS report value like a serialized Dec
    newQualityOfServiceReport.setLatency(
      Decimal.fromUserInput(lastQos.latency.toString(), 0).toString()
    );
    newQualityOfServiceReport.setAvailability(
      Decimal.fromUserInput(lastQos.availability.toString(), 0).toString()
    );
    newQualityOfServiceReport.setSync(
      Decimal.fromUserInput(lastQos.sync.toString(), 0).toString()
    );
  }
  const lastQosExcellence =
    singleConsumerSession.qoSInfo.lastExcellenceQoSReport;
  const newQualityOfServiceReportExcellence = new QualityOfServiceReport();
  if (lastQosExcellence != undefined) {
    // TODO: needs to serialize the QoS report value like a serialized Dec
    newQualityOfServiceReportExcellence.setLatency(
      Decimal.fromUserInput(lastQosExcellence.latency.toString(), 0).toString()
    );
    newQualityOfServiceReportExcellence.setAvailability(
      Decimal.fromUserInput(
        lastQosExcellence.availability.toString(),
        0
      ).toString()
    );
    newQualityOfServiceReportExcellence.setSync(
      Decimal.fromUserInput(lastQosExcellence.sync.toString(), 0).toString()
    );
  }
  const relaySession = new RelaySession();
  relaySession.setSpecId(chainID);
  relaySession.setLavaChainId(lavaChainID);
  relaySession.setSessionId(singleConsumerSession.sessionId);
  relaySession.setCuSum(
    singleConsumerSession.cuSum + singleConsumerSession.sessionId
  );
  relaySession.setProvider(providerAddress);
  relaySession.setQosReport(newQualityOfServiceReport);
  relaySession.setEpoch(epoch);
  relaySession.setUnresponsiveProviders(reportedProviders);
  relaySession.setQosExcellenceReport(newQualityOfServiceReportExcellence);
  relaySession.setContentHash(calculateContentHash(relayData));
  return relaySession;
}

function calculateContentHash(relayRequestData: RelayPrivateData): Uint8Array {
  const requestBlock = relayRequestData.getRequestBlock();
  const requestBlockBytes = convertRequestedBlockToUint8Array(requestBlock);

  const apiInterfaceBytes = encodeUtf8(relayRequestData.getApiInterface());
  const connectionTypeBytes = encodeUtf8(relayRequestData.getConnectionType());
  const apiUrlBytes = encodeUtf8(relayRequestData.getApiUrl());
  const dataBytes = relayRequestData.getData();
  const dataUint8Array =
    dataBytes instanceof Uint8Array ? dataBytes : encodeUtf8(dataBytes);
  const saltBytes = relayRequestData.getSalt();
  const saltUint8Array =
    saltBytes instanceof Uint8Array ? saltBytes : encodeUtf8(saltBytes);

  const msgData = concatUint8Arrays([
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

function encodeUtf8(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

function concatUint8Arrays(arrays: Uint8Array[]): Uint8Array {
  const totalLength = arrays.reduce((acc, arr) => acc + arr.length, 0);
  const result = new Uint8Array(totalLength);
  let offset = 0;
  arrays.forEach((arr) => {
    result.set(arr, offset);
    offset += arr.length;
  });
  return result;
}

function convertRequestedBlockToUint8Array(requestBlock: number): Uint8Array {
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

export function UpdateRequestedBlock(
  request: RelayPrivateData,
  response: RelayReply
) {
  request.setRequestBlock(
    ReplaceRequestedBlock(request.getRequestBlock(), response.getLatestBlock())
  );
  return;
}

export function ReplaceRequestedBlock(
  requestedBlock: number,
  latestBlock: number
): number {
  switch (requestedBlock) {
    case LATEST_BLOCK:
    case SAFE_BLOCK:
    case FINALIZED_BLOCK:
      return latestBlock;
    case EARLIEST_BLOCK:
      // TODO: add support for earliest block reliability
      return NOT_APPLICABLE;
    default:
      return requestedBlock;
  }
}

export function IsFinalizedBlock(
  requestedBlock: number,
  latestBlock: number,
  finalizationCriteria: number
): boolean {
  switch (requestedBlock) {
    case NOT_APPLICABLE:
      return false;
    default:
      if (requestedBlock < 0) {
        return false;
      }
      if (requestedBlock <= latestBlock - finalizationCriteria) {
        return true;
      }
  }
  return false;
}

export function verifyRelayReply(
  reply: RelayReply,
  relayRequest: RelayRequest,
  providerAddress: string
): Error | undefined {
  // TODO: implement signature verificaion
  return;
}

interface FinalizationData {
  finalizedBlocks: Map<number, string>;
  finalizationConflict: undefined;
}

export function verifyFinalizationData(
  reply: RelayReply,
  relayRequest: RelayRequest,
  providerPublicAddress: string,
  consumerAddress: string,
  existingSessionLatestBlock: number,
  blockDistanceForFinalizedData: number
): FinalizationData | Error {
  return new Error("implement me");
}
