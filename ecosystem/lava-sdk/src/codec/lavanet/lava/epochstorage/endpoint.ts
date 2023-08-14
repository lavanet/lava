/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "lavanet.lava.epochstorage";

export interface Endpoint {
  iPPORT: string;
  geolocation: Long;
  addons: string[];
  apiInterfaces: string[];
  extensions: string[];
}

function createBaseEndpoint(): Endpoint {
  return { iPPORT: "", geolocation: Long.UZERO, addons: [], apiInterfaces: [], extensions: [] };
}

export const Endpoint = {
  encode(message: Endpoint, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.iPPORT !== "") {
      writer.uint32(10).string(message.iPPORT);
    }
    if (!message.geolocation.isZero()) {
      writer.uint32(24).uint64(message.geolocation);
    }
    for (const v of message.addons) {
      writer.uint32(34).string(v!);
    }
    for (const v of message.apiInterfaces) {
      writer.uint32(42).string(v!);
    }
    for (const v of message.extensions) {
      writer.uint32(50).string(v!);
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
          if (tag != 10) {
            break;
          }

          message.iPPORT = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.geolocation = reader.uint64() as Long;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.addons.push(reader.string());
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.apiInterfaces.push(reader.string());
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.extensions.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Endpoint {
    return {
      iPPORT: isSet(object.iPPORT) ? String(object.iPPORT) : "",
      geolocation: isSet(object.geolocation) ? Long.fromValue(object.geolocation) : Long.UZERO,
      addons: Array.isArray(object?.addons) ? object.addons.map((e: any) => String(e)) : [],
      apiInterfaces: Array.isArray(object?.apiInterfaces) ? object.apiInterfaces.map((e: any) => String(e)) : [],
      extensions: Array.isArray(object?.extensions) ? object.extensions.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: Endpoint): unknown {
    const obj: any = {};
    message.iPPORT !== undefined && (obj.iPPORT = message.iPPORT);
    message.geolocation !== undefined && (obj.geolocation = (message.geolocation || Long.UZERO).toString());
    if (message.addons) {
      obj.addons = message.addons.map((e) => e);
    } else {
      obj.addons = [];
    }
    if (message.apiInterfaces) {
      obj.apiInterfaces = message.apiInterfaces.map((e) => e);
    } else {
      obj.apiInterfaces = [];
    }
    if (message.extensions) {
      obj.extensions = message.extensions.map((e) => e);
    } else {
      obj.extensions = [];
    }
    return obj;
  },

  create<I extends Exact<DeepPartial<Endpoint>, I>>(base?: I): Endpoint {
    return Endpoint.fromPartial(base ?? {});
  },

  fromPartial<I extends Exact<DeepPartial<Endpoint>, I>>(object: I): Endpoint {
    const message = createBaseEndpoint();
    message.iPPORT = object.iPPORT ?? "";
    message.geolocation = (object.geolocation !== undefined && object.geolocation !== null)
      ? Long.fromValue(object.geolocation)
      : Long.UZERO;
    message.addons = object.addons?.map((e) => e) || [];
    message.apiInterfaces = object.apiInterfaces?.map((e) => e) || [];
    message.extensions = object.extensions?.map((e) => e) || [];
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
