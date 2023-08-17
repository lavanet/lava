export interface ErrorResult {
  error: Error;
  result?: null;
}

export interface SuccessResult<T> {
  result: T;
  error?: null;
}

export type Result<T> = ErrorResult | SuccessResult<T>;
