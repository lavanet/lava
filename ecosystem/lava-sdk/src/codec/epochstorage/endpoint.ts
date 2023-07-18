/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.epochstorage";

export interface Endpoint {
  iPPORT: string;
  useType: string;
  geolocation: Long;
}

function createBaseEndpoint(): Endpoint {
  return { iPPORT: "", useType: "", geolocation: Long.UZERO };
}

export const Endpoint = {
  encode(message: Endpoint, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.iPPORT !== "") {
      writer.uint32(10).string(message.iPPORT);
    }
    if (message.useType !== "") {
      writer.uint32(18).string(message.useType);
    }
    if (!message.geolocation.isZero()) {
      writer.uint32(24).uint64(message.geolocation);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Endpoint {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEndpoint();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.iPPORT = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.useType = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.geolocation = reader.uint64() as Long;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Endpoint {
    return {
      iPPORT: isSet(object.iPPORT) ? String(object.iPPORT) : "",
      useType: isSet(object.useType) ? String(object.useType) : "",
      geolocation: isSet(object.geolocation) ? Long.fromValue(object.geolocation) : Long.UZERO,
    };
  },

  toJSON(message: Endpoint): unknown {
    const obj: any = {};
    message.iPPORT !== undefined && (obj.iPPORT = message.iPPORT);
    message.useType !== undefined && (obj.useType = message.useType);
    message.geolocation !== undefined && (obj.geolocation = (message.geolocation || Long.UZERO).toString());
    return obj;
  },

  create<I extends Exact<DeepPartial<Endpoint>, I>>(base?: I): Endpoint {
    return Endpoint.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Endpoint>, I>>(object: I): Endpoint {
    const message = createBaseEndpoint();
    message.iPPORT = object.iPPORT ?? "";
    message.useType = object.useType ?? "";
    message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
      ? Long.fromValue(object.geolocation)
      : Long.UZERO;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
