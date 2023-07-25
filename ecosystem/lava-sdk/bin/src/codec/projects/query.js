"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.QueryDeveloperResponse = exports.QueryDeveloperRequest = exports.QueryInfoResponse = exports.QueryInfoRequest = exports.QueryParamsResponse = exports.QueryParamsRequest = exports.protobufPackage = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const params_1 = require("./params");
const project_1 = require("./project");
exports.protobufPackage = "lavanet.lava.projects";
function createBaseQueryParamsRequest() {
    return {};
}
exports.QueryParamsRequest = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(_) {
        return {};
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    create(base) {
        return exports.QueryParamsRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseQueryParamsRequest();
        return message;
    },
};
function createBaseQueryParamsResponse() {
    return { params: undefined };
}
exports.QueryParamsResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.params !== undefined) {
            params_1.Params.encode(message.params, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryParamsResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.params = params_1.Params.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { params: isSet(object.params) ? params_1.Params.fromJSON(object.params) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.params !== undefined && (obj.params = message.params ? params_1.Params.toJSON(message.params) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryParamsResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryParamsResponse();
        message.params = (object.params !== undefined && object.params !== null)
            ? params_1.Params.fromPartial(object.params)
            : undefined;
        return message;
    },
};
function createBaseQueryInfoRequest() {
    return { project: "" };
}
exports.QueryInfoRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.project !== "") {
            writer.uint32(10).string(message.project);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryInfoRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.project = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { project: isSet(object.project) ? String(object.project) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.project !== undefined && (obj.project = message.project);
        return obj;
    },
    create(base) {
        return exports.QueryInfoRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryInfoRequest();
        message.project = (_a = object.project) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryInfoResponse() {
    return { project: undefined };
}
exports.QueryInfoResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.project !== undefined) {
            project_1.Project.encode(message.project, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryInfoResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.project = project_1.Project.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { project: isSet(object.project) ? project_1.Project.fromJSON(object.project) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.project !== undefined && (obj.project = message.project ? project_1.Project.toJSON(message.project) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryInfoResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryInfoResponse();
        message.project = (object.project !== undefined && object.project !== null)
            ? project_1.Project.fromPartial(object.project)
            : undefined;
        return message;
    },
};
function createBaseQueryDeveloperRequest() {
    return { developer: "" };
}
exports.QueryDeveloperRequest = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.developer !== "") {
            writer.uint32(10).string(message.developer);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryDeveloperRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.developer = reader.string();
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { developer: isSet(object.developer) ? String(object.developer) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.developer !== undefined && (obj.developer = message.developer);
        return obj;
    },
    create(base) {
        return exports.QueryDeveloperRequest.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseQueryDeveloperRequest();
        message.developer = (_a = object.developer) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseQueryDeveloperResponse() {
    return { project: undefined };
}
exports.QueryDeveloperResponse = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.project !== undefined) {
            project_1.Project.encode(message.project, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseQueryDeveloperResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.project = project_1.Project.decode(reader, reader.uint32());
                    continue;
            }
            if ((tag & 7) == 4 || tag == 0) {
                break;
            }
            reader.skipType(tag & 7);
        }
        return message;
    },
    fromJSON(object) {
        return { project: isSet(object.project) ? project_1.Project.fromJSON(object.project) : undefined };
    },
    toJSON(message) {
        const obj = {};
        message.project !== undefined && (obj.project = message.project ? project_1.Project.toJSON(message.project) : undefined);
        return obj;
    },
    create(base) {
        return exports.QueryDeveloperResponse.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseQueryDeveloperResponse();
        message.project = (object.project !== undefined && object.project !== null)
            ? project_1.Project.fromPartial(object.project)
            : undefined;
        return message;
    },
};
class QueryClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "lavanet.lava.projects.Query";
        this.rpc = rpc;
        this.Params = this.Params.bind(this);
        this.Info = this.Info.bind(this);
        this.Developer = this.Developer.bind(this);
    }
    Params(request) {
        const data = exports.QueryParamsRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Params", data);
        return promise.then((data) => exports.QueryParamsResponse.decode(minimal_1.default.Reader.create(data)));
    }
    Info(request) {
        const data = exports.QueryInfoRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Info", data);
        return promise.then((data) => exports.QueryInfoResponse.decode(minimal_1.default.Reader.create(data)));
    }
    Developer(request) {
        const data = exports.QueryDeveloperRequest.encode(request).finish();
        const promise = this.rpc.request(this.service, "Developer", data);
        return promise.then((data) => exports.QueryDeveloperResponse.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.QueryClientImpl = QueryClientImpl;
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
