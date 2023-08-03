export class SessionManager {
  PairingList: ConsumerSessionWithProvider[];
  PairingListExtended: ConsumerSessionWithProvider[]; // this list contains other geo-locations, and used if failed using pairing list
  NextEpochStart: Date;
  Apis: Map<string, number>;

  constructor(
    pairingList: ConsumerSessionWithProvider[],
    pairingListExtended: ConsumerSessionWithProvider[],
    nextEpochStart: Date,
    apis: Map<string, number>
  ) {
    this.NextEpochStart = nextEpochStart;
    this.PairingList = pairingList;
    this.PairingListExtended = pairingListExtended;
    this.Apis = apis;
  }

  getCuSumFromApi(name: string, chainID: string): number | undefined {
    if (chainID === "rest") {
      this.Apis.forEach((value: number, key: string) => {
        if (name.match(key)) {
          return value;
        }
      });
    }
    return this.Apis.get(name);
  }
}

export class ConsumerSessionWithProvider {
  ConsumerAddress: string;
  Endpoints: Array<Endpoint>;
  Session: SingleConsumerSession;
  MaxComputeUnits: number;
  UsedComputeUnits: number;
  ReliabilitySent: boolean;

  constructor(
    acc: string,
    endpoints: Array<Endpoint>,
    session: SingleConsumerSession,
    maxComputeUnits: number,
    usedComputeUnits: number,
    reliabilitySent: boolean
  ) {
    this.ConsumerAddress = acc;
    this.Endpoints = endpoints;
    this.Session = session;
    this.MaxComputeUnits = maxComputeUnits;
    this.UsedComputeUnits = usedComputeUnits;
    this.ReliabilitySent = reliabilitySent;
  }
}

export class SingleConsumerSession {
  ProviderAddress: string;
  CuSum: number;
  LatestRelayCu: number;
  SessionId: number;
  RelayNum: number;
  Endpoint: Endpoint;
  PairingEpoch: number;

  constructor(
    cuSum: number,
    latestRelayCu: number,
    relayNum: number,
    endpoint: Endpoint,
    pairingEpoch: number,
    providerAddress: string
  ) {
    this.CuSum = cuSum;
    this.LatestRelayCu = latestRelayCu;
    this.SessionId = this.getNewSessionId();
    this.RelayNum = relayNum;
    this.Endpoint = endpoint;
    this.PairingEpoch = pairingEpoch;
    this.ProviderAddress = providerAddress;
  }

  getNewSessionId(): number {
    return this.generateRandomUint();
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

export class Endpoint {
  Addr: string;
  Enabled: boolean;
  ConnectionRefusals: number;

  constructor(addr: string, enabled: boolean, connectionRefusals: number) {
    this.Addr = addr;
    this.Enabled = enabled;
    this.ConnectionRefusals = connectionRefusals;
  }
}
