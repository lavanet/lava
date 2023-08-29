// consumer errors
export class PairingListEmptyError extends Error {
  constructor() {
    super("Pairing list is empty");
  }
}
export class UnreachableCodeError extends Error {
  constructor() {
    super("Unreachable code");
  }
}
export class AllProviderEndpointsDisabledError extends Error {
  constructor() {
    super("All provider endpoints are disabled");
  }
}
export class MaximumNumberOfSessionsExceededError extends Error {
  constructor() {
    super("Maximum number of sessions exceeded");
  }
}
export class MaxComputeUnitsExceededError extends Error {
  constructor() {
    super("Maximum compute units exceeded");
  }
}
export class EpochMismatchError extends Error {
  constructor() {
    super("Epoch mismatch");
  }
}
export class AddressIndexWasNotFoundError extends Error {
  constructor() {
    super("Address index was not found");
  }
}
export class SessionIsAlreadyBlockListedError extends Error {
  constructor() {
    super("Session is already block listed");
  }
}
export class NegativeComputeUnitsAmountError extends Error {
  constructor() {
    super("Negative compute units amount");
  }
}
export class ReportAndBlockProviderError extends Error {
  constructor() {
    super("Report and block provider error");
  }
}
export class BlockProviderError extends Error {
  constructor() {
    super("Block provider error");
  }
}
export class SessionOutOfSyncError extends Error {
  constructor() {
    super("Session out of sync");
  }
}
export class MaximumNumberOfBlockListedSessionsError extends Error {
  constructor() {
    super("Maximum number of block listed sessions exceeded");
  }
}
export class SendRelayError extends Error {
  constructor() {
    super("Send relay error");
  }
}
