export type ErrorResult<T, E extends Error = Error> = {
  error: E;
} & Partial<T>;

export type SuccessResult<T> = T & { error?: undefined };

export type Result<T = unknown, E extends Error = Error> =
  | SuccessResult<T>
  | ErrorResult<T, E>;
