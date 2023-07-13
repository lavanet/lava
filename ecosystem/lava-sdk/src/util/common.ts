export function base64ToUint8Array(str: string): Uint8Array {
  const buffer = Buffer.from(str, "base64");

  return new Uint8Array(buffer);
}

export function generateRPCData(method: string, params: Array<any>): string {
  const stringifyMethod = JSON.stringify(method);
  const stringifyParam = JSON.stringify(params, (key, value) => {
    if (typeof value === "bigint") {
      return value.toString();
    }
    return value;
  });
  // TODO make id changable
  return (
    '{"jsonrpc": "2.0", "id": 1, "method": ' +
    stringifyMethod +
    ', "params": ' +
    stringifyParam +
    "}"
  );
}
