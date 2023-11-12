export const NOT_APPLICABLE = -1;
export const LATEST_BLOCK = -2;
export const EARLIEST_BLOCK = -3;
export const PENDING_BLOCK = -4;
export const SAFE_BLOCK = -5;
export const FINALIZED_BLOCK = -6;

export const BACKOFF_TIME_ON_FAILURE = 3000; // 3 seconds

export const DEFAULT_PARSED_RESULT_INDEX = 0;

export function IsString(value: any): boolean {
  return typeof value === "string" || value instanceof String;
}

export function IsNumber(value: any): boolean {
  return typeof value === "number" || value instanceof Number;
}

export enum HttpMethod {
  GET = "GET",
  HEAD = "HEAD",
  POST = "POST",
  PUT = "PUT",
  PATCH = "PATCH", // RFC 5789
  DELETE = "DELETE",
  CONNECT = "CONNECT",
  OPTIONS = "OPTIONS",
  TRACE = "TRACE",
}

export type StringToArrayMap = { [key: string]: Array<string> };
