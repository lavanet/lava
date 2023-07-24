import {
  makeCosmoshubPath,
  rawSecp256k1PubkeyToRawAddress,
} from "@cosmjs/amino";
import {
  Bip39,
  Random,
  EnglishMnemonic,
  Slip10,
  Slip10Curve,
  Secp256k1,
} from "@cosmjs/crypto";
import { toBech32 } from "@cosmjs/encoding";
import elliptic from "elliptic";
import axios from "axios";
import { uuid } from "uuidv4";

export const MAX_ATTEMPTS = 10;

export enum DeveloperKeyStatus {
  ERROR = "error",
  PENDING = "pending",
  SYNCED = "synced",
  DELETING = "deleting",
}

export type GeneratedKeyType = {
  address: string;
  privateHex: string;
  mnemonicChecked: EnglishMnemonic;
};

export type DeveloperKeyParams = {
  projectKey: string;
  name: string;
};

const GW_BACKEND_URL = "https://gateway-master.lava-cybertron.xyz/sdk";
// const GW_BACKEND_URL = "http://127.0.0.1:4455/sdk";

export const axiosInstance = axios.create({
  baseURL: `${GW_BACKEND_URL}/api/sdk`,
});

export const generateKey = async (): Promise<GeneratedKeyType> => {
  const prefix = "lava@";
  const mnemonic = Bip39.encode(Random.getBytes(32)).toString();
  const mnemonicChecked = new EnglishMnemonic(mnemonic);
  const seed = await Bip39.mnemonicToSeed(mnemonicChecked, "");
  const hdPath = makeCosmoshubPath(0);

  const { privkey } = Slip10.derivePath(Slip10Curve.Secp256k1, seed, hdPath);
  const privateHex = new elliptic.ec("secp256k1")
    .keyFromPrivate(privkey)
    .getPrivate("hex");
  const { pubkey } = await Secp256k1.makeKeypair(privkey);
  const address = toBech32(
    prefix,
    rawSecp256k1PubkeyToRawAddress(Secp256k1.compressPubkey(pubkey))
  );

  return { address, privateHex, mnemonicChecked };
};

export async function getKey(
  apiAccessKey: string,
  developerKey: string
): Promise<any> {
  return await axiosInstance.get(`/developer-keys/${developerKey}`, {
    headers: {
      "api-access-key": apiAccessKey,
    },
  });
}

export async function getKeys(apiAccessKey: string): Promise<any> {
  const { data } = await axiosInstance.get(`/developer-keys`, {
    headers: {
      "api-access-key": apiAccessKey,
    },
  });
  return data;
}

export async function createDeveloperKey(
  apiAccessKey: string,
  developerKey: string
) {
  const keyParams: DeveloperKeyParams = {
    projectKey: developerKey,
    name: `SDK Generated Key-${uuid().toLowerCase()}`,
  };
  return await axiosInstance.post(`/developer-keys/`, keyParams, {
    headers: { "api-access-key": apiAccessKey },
  });
}

export function sleep(delayMS: number) {
  return new Promise((res) => setTimeout(res, delayMS));
}
