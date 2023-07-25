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
import { v4 } from "uuid";

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

const GW_BACKEND_URL_PROD = "https://gateway-master.lavanet.xyz/sdk";

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

export async function getDeveloperKey(
  apiAccessKey: string,
  developerKey: string,
  url: string = GW_BACKEND_URL_PROD
): Promise<any> {
  const response = await fetch(
    `${url}/api/sdk/developer-keys/${developerKey}`,
    {
      headers: {
        "api-access-key": apiAccessKey,
        "Content-Type": "application/json",
      },
    }
  );
  return await response.json();
}

export async function createDeveloperKey(
  apiAccessKey: string,
  developerKey: string,
  url: string = GW_BACKEND_URL_PROD
) {
  const keyParams: DeveloperKeyParams = {
    projectKey: developerKey,
    name: `SDK Generated Key-${v4().toLowerCase()}`,
  };

  const response = await fetch(`${url}/api/sdk/developer-keys/`, {
    method: "POST",
    headers: {
      "api-access-key": apiAccessKey,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(keyParams),
  });
  return await response.json();
}

export function sleep(delayMS: number) {
  return new Promise((res) => setTimeout(res, delayMS));
}
