export function base64ToUint8Array(str: string): Uint8Array {
  const buffer = Buffer.from(str, "base64");

  return new Uint8Array(buffer);
}

let globalId = 0;

export function generateRPCData(method: string, params: Array<any>): string {
  const stringifyMethod = JSON.stringify(method);
  const stringifyParam = JSON.stringify(params, (key, value) => {
    if (typeof value === "bigint") {
      return value.toString();
    }
    return value;
  });
  globalId += 1;
  return `{"jsonrpc": "2.0", "id": ${globalId}, "method": ${stringifyMethod}, "params": ${stringifyParam}}`;
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
  if (values.length === 0) {
    return 0;
  }

  values = [...values].sort((a, b) => a - b);
  const half = Math.floor(values.length / 2);

  if (values.length % 2) {
    return values[half];
  }

  return (values[half - 1] + values[half]) / 2;
}
