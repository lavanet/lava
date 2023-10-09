export class ValueNotSetError extends Error {
  constructor() {
    super(
      "When trying to parse, the value that we attempted to parse did not exist"
    );
  }
}

export class InvalidBlockValue extends Error {
  constructor(block: string) {
    super(`Invalid block value: ${block}`);
  }
}

export class UnsupportedBlockParser extends Error {
  constructor(blockParser: string) {
    super(`Unsupported block parser: ${blockParser}`);
  }
}

export class InvalidInputFormat extends Error {
  constructor(message: string) {
    super(`Invalid input format: ${message}`);
  }
}
