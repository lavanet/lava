import {
  RelayRequest,
  RelayReply,
  RelaySession,
  RelayPrivateData,
  Badge,
  ProbeRequest,
  ProbeReply,
} from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

export interface SendRelayData {
  connectionType: string;
  data: string;
  url: string;
  apiInterface: string;
  chainId: string;
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
