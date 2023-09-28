import { JsonrpcMessage } from "../chainlib/chainproxy/rpcInterfaceMessages/json_rpc_message";

export function base64ToUint8Array(str: string): Uint8Array {
  const buffer = Buffer.from(str, "base64");

  return new Uint8Array(buffer);
}

export function generateRPCData(rpcMessage: JsonrpcMessage): string {
  const stringifyId = JSON.stringify(rpcMessage.id);
  const stringifyVersion = JSON.stringify(rpcMessage.version);
  const stringifyMethod = JSON.stringify(rpcMessage.method);
  const stringifyParam = JSON.stringify(rpcMessage.params, (key, value) => {
    if (typeof value === "bigint") {
      return value.toString();
    }
    return value;
  });
  return `{"jsonrpc": ${stringifyVersion}, "id": ${stringifyId}, "method": ${stringifyMethod}, "params": ${stringifyParam}}`;
}

export function parseLong(long: Long): number {
  /**
   * this function will parse long to a 64bit number,
   * this assumes all systems running the sdk will run on 64bit systems
   * @param long A long number to parse into number
   */
  const high = Number(long.high);
  const low = Number(long.low);
  const parsedNumber = (high << 32) + low;
  if (high > 0) {
    console.log("MAYBE AN ISSUE", high);
  }
  return parsedNumber;
}

export function debugPrint(
  debugMode: boolean,
  message?: any,
  ...optionalParams: any[]
) {
  if (debugMode) {
    console.log(message, ...optionalParams);
  }
}

export function generateRandomInt(): number {
  return Math.floor(Math.random() * (Number.MAX_SAFE_INTEGER + 1));
}

export function sleep(ms: number): Promise<void> {
  if (ms <= 0) {
    return Promise.resolve();
  }

  return new Promise((resolve) => {
    const timeout = setTimeout(() => {
      resolve();
      clearTimeout(timeout);
    }, ms);
  });
}

export function median(values: number[]): number {
  // First, we need to sort the array in ascending order
  const sortedValues = values.slice().sort((a, b) => a - b);

  // Calculate the middle index
  const middleIndex = Math.floor(sortedValues.length / 2);

  // Check if the array length is even or odd
  if (sortedValues.length % 2 === 0) {
    // If it's even, return the average of the two middle values
    return (sortedValues[middleIndex - 1] + sortedValues[middleIndex]) / 2;
  } else {
    // If it's odd, return the middle value
    return sortedValues[middleIndex];
  }
}

export function encodeUtf8(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

export function byteArrayToString(byteArray: Uint8Array): string {
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
}
