"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ResponsePrepareProposal = exports.ResponseApplySnapshotChunk = exports.ResponseLoadSnapshotChunk = exports.ResponseOfferSnapshot = exports.ResponseListSnapshots = exports.ResponseCommit = exports.ResponseEndBlock = exports.ResponseDeliverTx = exports.ResponseCheckTx = exports.ResponseBeginBlock = exports.ResponseQuery = exports.ResponseInitChain = exports.ResponseInfo = exports.ResponseFlush = exports.ResponseEcho = exports.ResponseException = exports.Response = exports.RequestProcessProposal = exports.RequestPrepareProposal = exports.RequestApplySnapshotChunk = exports.RequestLoadSnapshotChunk = exports.RequestOfferSnapshot = exports.RequestListSnapshots = exports.RequestCommit = exports.RequestEndBlock = exports.RequestDeliverTx = exports.RequestCheckTx = exports.RequestBeginBlock = exports.RequestQuery = exports.RequestInitChain = exports.RequestInfo = exports.RequestFlush = exports.RequestEcho = exports.Request = exports.responseProcessProposal_ProposalStatusToJSON = exports.responseProcessProposal_ProposalStatusFromJSON = exports.ResponseProcessProposal_ProposalStatus = exports.responseApplySnapshotChunk_ResultToJSON = exports.responseApplySnapshotChunk_ResultFromJSON = exports.ResponseApplySnapshotChunk_Result = exports.responseOfferSnapshot_ResultToJSON = exports.responseOfferSnapshot_ResultFromJSON = exports.ResponseOfferSnapshot_Result = exports.misbehaviorTypeToJSON = exports.misbehaviorTypeFromJSON = exports.MisbehaviorType = exports.checkTxTypeToJSON = exports.checkTxTypeFromJSON = exports.CheckTxType = exports.protobufPackage = void 0;
exports.ABCIApplicationClientImpl = exports.Snapshot = exports.Misbehavior = exports.ExtendedVoteInfo = exports.VoteInfo = exports.ValidatorUpdate = exports.Validator = exports.TxResult = exports.EventAttribute = exports.Event = exports.ExtendedCommitInfo = exports.CommitInfo = exports.ResponseProcessProposal = void 0;
/* eslint-disable */
const long_1 = __importDefault(require("long"));
const minimal_1 = __importDefault(require("protobufjs/minimal"));
const timestamp_1 = require("../../google/protobuf/timestamp");
const keys_1 = require("../crypto/keys");
const proof_1 = require("../crypto/proof");
const params_1 = require("../types/params");
const types_1 = require("../types/types");
exports.protobufPackage = "tendermint.abci";
var CheckTxType;
(function (CheckTxType) {
    CheckTxType[CheckTxType["NEW"] = 0] = "NEW";
    CheckTxType[CheckTxType["RECHECK"] = 1] = "RECHECK";
    CheckTxType[CheckTxType["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(CheckTxType = exports.CheckTxType || (exports.CheckTxType = {}));
function checkTxTypeFromJSON(object) {
    switch (object) {
        case 0:
        case "NEW":
            return CheckTxType.NEW;
        case 1:
        case "RECHECK":
            return CheckTxType.RECHECK;
        case -1:
        case "UNRECOGNIZED":
        default:
            return CheckTxType.UNRECOGNIZED;
    }
}
exports.checkTxTypeFromJSON = checkTxTypeFromJSON;
function checkTxTypeToJSON(object) {
    switch (object) {
        case CheckTxType.NEW:
            return "NEW";
        case CheckTxType.RECHECK:
            return "RECHECK";
        case CheckTxType.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.checkTxTypeToJSON = checkTxTypeToJSON;
var MisbehaviorType;
(function (MisbehaviorType) {
    MisbehaviorType[MisbehaviorType["UNKNOWN"] = 0] = "UNKNOWN";
    MisbehaviorType[MisbehaviorType["DUPLICATE_VOTE"] = 1] = "DUPLICATE_VOTE";
    MisbehaviorType[MisbehaviorType["LIGHT_CLIENT_ATTACK"] = 2] = "LIGHT_CLIENT_ATTACK";
    MisbehaviorType[MisbehaviorType["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(MisbehaviorType = exports.MisbehaviorType || (exports.MisbehaviorType = {}));
function misbehaviorTypeFromJSON(object) {
    switch (object) {
        case 0:
        case "UNKNOWN":
            return MisbehaviorType.UNKNOWN;
        case 1:
        case "DUPLICATE_VOTE":
            return MisbehaviorType.DUPLICATE_VOTE;
        case 2:
        case "LIGHT_CLIENT_ATTACK":
            return MisbehaviorType.LIGHT_CLIENT_ATTACK;
        case -1:
        case "UNRECOGNIZED":
        default:
            return MisbehaviorType.UNRECOGNIZED;
    }
}
exports.misbehaviorTypeFromJSON = misbehaviorTypeFromJSON;
function misbehaviorTypeToJSON(object) {
    switch (object) {
        case MisbehaviorType.UNKNOWN:
            return "UNKNOWN";
        case MisbehaviorType.DUPLICATE_VOTE:
            return "DUPLICATE_VOTE";
        case MisbehaviorType.LIGHT_CLIENT_ATTACK:
            return "LIGHT_CLIENT_ATTACK";
        case MisbehaviorType.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.misbehaviorTypeToJSON = misbehaviorTypeToJSON;
var ResponseOfferSnapshot_Result;
(function (ResponseOfferSnapshot_Result) {
    /** UNKNOWN - Unknown result, abort all snapshot restoration */
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["UNKNOWN"] = 0] = "UNKNOWN";
    /** ACCEPT - Snapshot accepted, apply chunks */
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["ACCEPT"] = 1] = "ACCEPT";
    /** ABORT - Abort all snapshot restoration */
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["ABORT"] = 2] = "ABORT";
    /** REJECT - Reject this specific snapshot, try others */
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["REJECT"] = 3] = "REJECT";
    /** REJECT_FORMAT - Reject all snapshots of this format, try others */
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["REJECT_FORMAT"] = 4] = "REJECT_FORMAT";
    /** REJECT_SENDER - Reject all snapshots from the sender(s), try others */
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["REJECT_SENDER"] = 5] = "REJECT_SENDER";
    ResponseOfferSnapshot_Result[ResponseOfferSnapshot_Result["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(ResponseOfferSnapshot_Result = exports.ResponseOfferSnapshot_Result || (exports.ResponseOfferSnapshot_Result = {}));
function responseOfferSnapshot_ResultFromJSON(object) {
    switch (object) {
        case 0:
        case "UNKNOWN":
            return ResponseOfferSnapshot_Result.UNKNOWN;
        case 1:
        case "ACCEPT":
            return ResponseOfferSnapshot_Result.ACCEPT;
        case 2:
        case "ABORT":
            return ResponseOfferSnapshot_Result.ABORT;
        case 3:
        case "REJECT":
            return ResponseOfferSnapshot_Result.REJECT;
        case 4:
        case "REJECT_FORMAT":
            return ResponseOfferSnapshot_Result.REJECT_FORMAT;
        case 5:
        case "REJECT_SENDER":
            return ResponseOfferSnapshot_Result.REJECT_SENDER;
        case -1:
        case "UNRECOGNIZED":
        default:
            return ResponseOfferSnapshot_Result.UNRECOGNIZED;
    }
}
exports.responseOfferSnapshot_ResultFromJSON = responseOfferSnapshot_ResultFromJSON;
function responseOfferSnapshot_ResultToJSON(object) {
    switch (object) {
        case ResponseOfferSnapshot_Result.UNKNOWN:
            return "UNKNOWN";
        case ResponseOfferSnapshot_Result.ACCEPT:
            return "ACCEPT";
        case ResponseOfferSnapshot_Result.ABORT:
            return "ABORT";
        case ResponseOfferSnapshot_Result.REJECT:
            return "REJECT";
        case ResponseOfferSnapshot_Result.REJECT_FORMAT:
            return "REJECT_FORMAT";
        case ResponseOfferSnapshot_Result.REJECT_SENDER:
            return "REJECT_SENDER";
        case ResponseOfferSnapshot_Result.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.responseOfferSnapshot_ResultToJSON = responseOfferSnapshot_ResultToJSON;
var ResponseApplySnapshotChunk_Result;
(function (ResponseApplySnapshotChunk_Result) {
    /** UNKNOWN - Unknown result, abort all snapshot restoration */
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["UNKNOWN"] = 0] = "UNKNOWN";
    /** ACCEPT - Chunk successfully accepted */
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["ACCEPT"] = 1] = "ACCEPT";
    /** ABORT - Abort all snapshot restoration */
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["ABORT"] = 2] = "ABORT";
    /** RETRY - Retry chunk (combine with refetch and reject) */
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["RETRY"] = 3] = "RETRY";
    /** RETRY_SNAPSHOT - Retry snapshot (combine with refetch and reject) */
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["RETRY_SNAPSHOT"] = 4] = "RETRY_SNAPSHOT";
    /** REJECT_SNAPSHOT - Reject this snapshot, try others */
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["REJECT_SNAPSHOT"] = 5] = "REJECT_SNAPSHOT";
    ResponseApplySnapshotChunk_Result[ResponseApplySnapshotChunk_Result["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(ResponseApplySnapshotChunk_Result = exports.ResponseApplySnapshotChunk_Result || (exports.ResponseApplySnapshotChunk_Result = {}));
function responseApplySnapshotChunk_ResultFromJSON(object) {
    switch (object) {
        case 0:
        case "UNKNOWN":
            return ResponseApplySnapshotChunk_Result.UNKNOWN;
        case 1:
        case "ACCEPT":
            return ResponseApplySnapshotChunk_Result.ACCEPT;
        case 2:
        case "ABORT":
            return ResponseApplySnapshotChunk_Result.ABORT;
        case 3:
        case "RETRY":
            return ResponseApplySnapshotChunk_Result.RETRY;
        case 4:
        case "RETRY_SNAPSHOT":
            return ResponseApplySnapshotChunk_Result.RETRY_SNAPSHOT;
        case 5:
        case "REJECT_SNAPSHOT":
            return ResponseApplySnapshotChunk_Result.REJECT_SNAPSHOT;
        case -1:
        case "UNRECOGNIZED":
        default:
            return ResponseApplySnapshotChunk_Result.UNRECOGNIZED;
    }
}
exports.responseApplySnapshotChunk_ResultFromJSON = responseApplySnapshotChunk_ResultFromJSON;
function responseApplySnapshotChunk_ResultToJSON(object) {
    switch (object) {
        case ResponseApplySnapshotChunk_Result.UNKNOWN:
            return "UNKNOWN";
        case ResponseApplySnapshotChunk_Result.ACCEPT:
            return "ACCEPT";
        case ResponseApplySnapshotChunk_Result.ABORT:
            return "ABORT";
        case ResponseApplySnapshotChunk_Result.RETRY:
            return "RETRY";
        case ResponseApplySnapshotChunk_Result.RETRY_SNAPSHOT:
            return "RETRY_SNAPSHOT";
        case ResponseApplySnapshotChunk_Result.REJECT_SNAPSHOT:
            return "REJECT_SNAPSHOT";
        case ResponseApplySnapshotChunk_Result.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.responseApplySnapshotChunk_ResultToJSON = responseApplySnapshotChunk_ResultToJSON;
var ResponseProcessProposal_ProposalStatus;
(function (ResponseProcessProposal_ProposalStatus) {
    ResponseProcessProposal_ProposalStatus[ResponseProcessProposal_ProposalStatus["UNKNOWN"] = 0] = "UNKNOWN";
    ResponseProcessProposal_ProposalStatus[ResponseProcessProposal_ProposalStatus["ACCEPT"] = 1] = "ACCEPT";
    ResponseProcessProposal_ProposalStatus[ResponseProcessProposal_ProposalStatus["REJECT"] = 2] = "REJECT";
    ResponseProcessProposal_ProposalStatus[ResponseProcessProposal_ProposalStatus["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(ResponseProcessProposal_ProposalStatus = exports.ResponseProcessProposal_ProposalStatus || (exports.ResponseProcessProposal_ProposalStatus = {}));
function responseProcessProposal_ProposalStatusFromJSON(object) {
    switch (object) {
        case 0:
        case "UNKNOWN":
            return ResponseProcessProposal_ProposalStatus.UNKNOWN;
        case 1:
        case "ACCEPT":
            return ResponseProcessProposal_ProposalStatus.ACCEPT;
        case 2:
        case "REJECT":
            return ResponseProcessProposal_ProposalStatus.REJECT;
        case -1:
        case "UNRECOGNIZED":
        default:
            return ResponseProcessProposal_ProposalStatus.UNRECOGNIZED;
    }
}
exports.responseProcessProposal_ProposalStatusFromJSON = responseProcessProposal_ProposalStatusFromJSON;
function responseProcessProposal_ProposalStatusToJSON(object) {
    switch (object) {
        case ResponseProcessProposal_ProposalStatus.UNKNOWN:
            return "UNKNOWN";
        case ResponseProcessProposal_ProposalStatus.ACCEPT:
            return "ACCEPT";
        case ResponseProcessProposal_ProposalStatus.REJECT:
            return "REJECT";
        case ResponseProcessProposal_ProposalStatus.UNRECOGNIZED:
        default:
            return "UNRECOGNIZED";
    }
}
exports.responseProcessProposal_ProposalStatusToJSON = responseProcessProposal_ProposalStatusToJSON;
function createBaseRequest() {
    return {
        echo: undefined,
        flush: undefined,
        info: undefined,
        initChain: undefined,
        query: undefined,
        beginBlock: undefined,
        checkTx: undefined,
        deliverTx: undefined,
        endBlock: undefined,
        commit: undefined,
        listSnapshots: undefined,
        offerSnapshot: undefined,
        loadSnapshotChunk: undefined,
        applySnapshotChunk: undefined,
        prepareProposal: undefined,
        processProposal: undefined,
    };
}
exports.Request = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.echo !== undefined) {
            exports.RequestEcho.encode(message.echo, writer.uint32(10).fork()).ldelim();
        }
        if (message.flush !== undefined) {
            exports.RequestFlush.encode(message.flush, writer.uint32(18).fork()).ldelim();
        }
        if (message.info !== undefined) {
            exports.RequestInfo.encode(message.info, writer.uint32(26).fork()).ldelim();
        }
        if (message.initChain !== undefined) {
            exports.RequestInitChain.encode(message.initChain, writer.uint32(42).fork()).ldelim();
        }
        if (message.query !== undefined) {
            exports.RequestQuery.encode(message.query, writer.uint32(50).fork()).ldelim();
        }
        if (message.beginBlock !== undefined) {
            exports.RequestBeginBlock.encode(message.beginBlock, writer.uint32(58).fork()).ldelim();
        }
        if (message.checkTx !== undefined) {
            exports.RequestCheckTx.encode(message.checkTx, writer.uint32(66).fork()).ldelim();
        }
        if (message.deliverTx !== undefined) {
            exports.RequestDeliverTx.encode(message.deliverTx, writer.uint32(74).fork()).ldelim();
        }
        if (message.endBlock !== undefined) {
            exports.RequestEndBlock.encode(message.endBlock, writer.uint32(82).fork()).ldelim();
        }
        if (message.commit !== undefined) {
            exports.RequestCommit.encode(message.commit, writer.uint32(90).fork()).ldelim();
        }
        if (message.listSnapshots !== undefined) {
            exports.RequestListSnapshots.encode(message.listSnapshots, writer.uint32(98).fork()).ldelim();
        }
        if (message.offerSnapshot !== undefined) {
            exports.RequestOfferSnapshot.encode(message.offerSnapshot, writer.uint32(106).fork()).ldelim();
        }
        if (message.loadSnapshotChunk !== undefined) {
            exports.RequestLoadSnapshotChunk.encode(message.loadSnapshotChunk, writer.uint32(114).fork()).ldelim();
        }
        if (message.applySnapshotChunk !== undefined) {
            exports.RequestApplySnapshotChunk.encode(message.applySnapshotChunk, writer.uint32(122).fork()).ldelim();
        }
        if (message.prepareProposal !== undefined) {
            exports.RequestPrepareProposal.encode(message.prepareProposal, writer.uint32(130).fork()).ldelim();
        }
        if (message.processProposal !== undefined) {
            exports.RequestProcessProposal.encode(message.processProposal, writer.uint32(138).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequest();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.echo = exports.RequestEcho.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.flush = exports.RequestFlush.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.info = exports.RequestInfo.decode(reader, reader.uint32());
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.initChain = exports.RequestInitChain.decode(reader, reader.uint32());
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.query = exports.RequestQuery.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.beginBlock = exports.RequestBeginBlock.decode(reader, reader.uint32());
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.checkTx = exports.RequestCheckTx.decode(reader, reader.uint32());
                    continue;
                case 9:
                    if (tag != 74) {
                        break;
                    }
                    message.deliverTx = exports.RequestDeliverTx.decode(reader, reader.uint32());
                    continue;
                case 10:
                    if (tag != 82) {
                        break;
                    }
                    message.endBlock = exports.RequestEndBlock.decode(reader, reader.uint32());
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.commit = exports.RequestCommit.decode(reader, reader.uint32());
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.listSnapshots = exports.RequestListSnapshots.decode(reader, reader.uint32());
                    continue;
                case 13:
                    if (tag != 106) {
                        break;
                    }
                    message.offerSnapshot = exports.RequestOfferSnapshot.decode(reader, reader.uint32());
                    continue;
                case 14:
                    if (tag != 114) {
                        break;
                    }
                    message.loadSnapshotChunk = exports.RequestLoadSnapshotChunk.decode(reader, reader.uint32());
                    continue;
                case 15:
                    if (tag != 122) {
                        break;
                    }
                    message.applySnapshotChunk = exports.RequestApplySnapshotChunk.decode(reader, reader.uint32());
                    continue;
                case 16:
                    if (tag != 130) {
                        break;
                    }
                    message.prepareProposal = exports.RequestPrepareProposal.decode(reader, reader.uint32());
                    continue;
                case 17:
                    if (tag != 138) {
                        break;
                    }
                    message.processProposal = exports.RequestProcessProposal.decode(reader, reader.uint32());
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
        return {
            echo: isSet(object.echo) ? exports.RequestEcho.fromJSON(object.echo) : undefined,
            flush: isSet(object.flush) ? exports.RequestFlush.fromJSON(object.flush) : undefined,
            info: isSet(object.info) ? exports.RequestInfo.fromJSON(object.info) : undefined,
            initChain: isSet(object.initChain) ? exports.RequestInitChain.fromJSON(object.initChain) : undefined,
            query: isSet(object.query) ? exports.RequestQuery.fromJSON(object.query) : undefined,
            beginBlock: isSet(object.beginBlock) ? exports.RequestBeginBlock.fromJSON(object.beginBlock) : undefined,
            checkTx: isSet(object.checkTx) ? exports.RequestCheckTx.fromJSON(object.checkTx) : undefined,
            deliverTx: isSet(object.deliverTx) ? exports.RequestDeliverTx.fromJSON(object.deliverTx) : undefined,
            endBlock: isSet(object.endBlock) ? exports.RequestEndBlock.fromJSON(object.endBlock) : undefined,
            commit: isSet(object.commit) ? exports.RequestCommit.fromJSON(object.commit) : undefined,
            listSnapshots: isSet(object.listSnapshots) ? exports.RequestListSnapshots.fromJSON(object.listSnapshots) : undefined,
            offerSnapshot: isSet(object.offerSnapshot) ? exports.RequestOfferSnapshot.fromJSON(object.offerSnapshot) : undefined,
            loadSnapshotChunk: isSet(object.loadSnapshotChunk)
                ? exports.RequestLoadSnapshotChunk.fromJSON(object.loadSnapshotChunk)
                : undefined,
            applySnapshotChunk: isSet(object.applySnapshotChunk)
                ? exports.RequestApplySnapshotChunk.fromJSON(object.applySnapshotChunk)
                : undefined,
            prepareProposal: isSet(object.prepareProposal)
                ? exports.RequestPrepareProposal.fromJSON(object.prepareProposal)
                : undefined,
            processProposal: isSet(object.processProposal)
                ? exports.RequestProcessProposal.fromJSON(object.processProposal)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.echo !== undefined && (obj.echo = message.echo ? exports.RequestEcho.toJSON(message.echo) : undefined);
        message.flush !== undefined && (obj.flush = message.flush ? exports.RequestFlush.toJSON(message.flush) : undefined);
        message.info !== undefined && (obj.info = message.info ? exports.RequestInfo.toJSON(message.info) : undefined);
        message.initChain !== undefined &&
            (obj.initChain = message.initChain ? exports.RequestInitChain.toJSON(message.initChain) : undefined);
        message.query !== undefined && (obj.query = message.query ? exports.RequestQuery.toJSON(message.query) : undefined);
        message.beginBlock !== undefined &&
            (obj.beginBlock = message.beginBlock ? exports.RequestBeginBlock.toJSON(message.beginBlock) : undefined);
        message.checkTx !== undefined &&
            (obj.checkTx = message.checkTx ? exports.RequestCheckTx.toJSON(message.checkTx) : undefined);
        message.deliverTx !== undefined &&
            (obj.deliverTx = message.deliverTx ? exports.RequestDeliverTx.toJSON(message.deliverTx) : undefined);
        message.endBlock !== undefined &&
            (obj.endBlock = message.endBlock ? exports.RequestEndBlock.toJSON(message.endBlock) : undefined);
        message.commit !== undefined && (obj.commit = message.commit ? exports.RequestCommit.toJSON(message.commit) : undefined);
        message.listSnapshots !== undefined &&
            (obj.listSnapshots = message.listSnapshots ? exports.RequestListSnapshots.toJSON(message.listSnapshots) : undefined);
        message.offerSnapshot !== undefined &&
            (obj.offerSnapshot = message.offerSnapshot ? exports.RequestOfferSnapshot.toJSON(message.offerSnapshot) : undefined);
        message.loadSnapshotChunk !== undefined && (obj.loadSnapshotChunk = message.loadSnapshotChunk
            ? exports.RequestLoadSnapshotChunk.toJSON(message.loadSnapshotChunk)
            : undefined);
        message.applySnapshotChunk !== undefined && (obj.applySnapshotChunk = message.applySnapshotChunk
            ? exports.RequestApplySnapshotChunk.toJSON(message.applySnapshotChunk)
            : undefined);
        message.prepareProposal !== undefined && (obj.prepareProposal = message.prepareProposal
            ? exports.RequestPrepareProposal.toJSON(message.prepareProposal)
            : undefined);
        message.processProposal !== undefined && (obj.processProposal = message.processProposal
            ? exports.RequestProcessProposal.toJSON(message.processProposal)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.Request.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseRequest();
        message.echo = (object.echo !== undefined && object.echo !== null)
            ? exports.RequestEcho.fromPartial(object.echo)
            : undefined;
        message.flush = (object.flush !== undefined && object.flush !== null)
            ? exports.RequestFlush.fromPartial(object.flush)
            : undefined;
        message.info = (object.info !== undefined && object.info !== null)
            ? exports.RequestInfo.fromPartial(object.info)
            : undefined;
        message.initChain = (object.initChain !== undefined && object.initChain !== null)
            ? exports.RequestInitChain.fromPartial(object.initChain)
            : undefined;
        message.query = (object.query !== undefined && object.query !== null)
            ? exports.RequestQuery.fromPartial(object.query)
            : undefined;
        message.beginBlock = (object.beginBlock !== undefined && object.beginBlock !== null)
            ? exports.RequestBeginBlock.fromPartial(object.beginBlock)
            : undefined;
        message.checkTx = (object.checkTx !== undefined && object.checkTx !== null)
            ? exports.RequestCheckTx.fromPartial(object.checkTx)
            : undefined;
        message.deliverTx = (object.deliverTx !== undefined && object.deliverTx !== null)
            ? exports.RequestDeliverTx.fromPartial(object.deliverTx)
            : undefined;
        message.endBlock = (object.endBlock !== undefined && object.endBlock !== null)
            ? exports.RequestEndBlock.fromPartial(object.endBlock)
            : undefined;
        message.commit = (object.commit !== undefined && object.commit !== null)
            ? exports.RequestCommit.fromPartial(object.commit)
            : undefined;
        message.listSnapshots = (object.listSnapshots !== undefined && object.listSnapshots !== null)
            ? exports.RequestListSnapshots.fromPartial(object.listSnapshots)
            : undefined;
        message.offerSnapshot = (object.offerSnapshot !== undefined && object.offerSnapshot !== null)
            ? exports.RequestOfferSnapshot.fromPartial(object.offerSnapshot)
            : undefined;
        message.loadSnapshotChunk = (object.loadSnapshotChunk !== undefined && object.loadSnapshotChunk !== null)
            ? exports.RequestLoadSnapshotChunk.fromPartial(object.loadSnapshotChunk)
            : undefined;
        message.applySnapshotChunk = (object.applySnapshotChunk !== undefined && object.applySnapshotChunk !== null)
            ? exports.RequestApplySnapshotChunk.fromPartial(object.applySnapshotChunk)
            : undefined;
        message.prepareProposal = (object.prepareProposal !== undefined && object.prepareProposal !== null)
            ? exports.RequestPrepareProposal.fromPartial(object.prepareProposal)
            : undefined;
        message.processProposal = (object.processProposal !== undefined && object.processProposal !== null)
            ? exports.RequestProcessProposal.fromPartial(object.processProposal)
            : undefined;
        return message;
    },
};
function createBaseRequestEcho() {
    return { message: "" };
}
exports.RequestEcho = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.message !== "") {
            writer.uint32(10).string(message.message);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestEcho();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.message = reader.string();
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
        return { message: isSet(object.message) ? String(object.message) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.message !== undefined && (obj.message = message.message);
        return obj;
    },
    create(base) {
        return exports.RequestEcho.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseRequestEcho();
        message.message = (_a = object.message) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseRequestFlush() {
    return {};
}
exports.RequestFlush = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestFlush();
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
        return exports.RequestFlush.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseRequestFlush();
        return message;
    },
};
function createBaseRequestInfo() {
    return { version: "", blockVersion: long_1.default.UZERO, p2pVersion: long_1.default.UZERO, abciVersion: "" };
}
exports.RequestInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.version !== "") {
            writer.uint32(10).string(message.version);
        }
        if (!message.blockVersion.isZero()) {
            writer.uint32(16).uint64(message.blockVersion);
        }
        if (!message.p2pVersion.isZero()) {
            writer.uint32(24).uint64(message.p2pVersion);
        }
        if (message.abciVersion !== "") {
            writer.uint32(34).string(message.abciVersion);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.version = reader.string();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.blockVersion = reader.uint64();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.p2pVersion = reader.uint64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.abciVersion = reader.string();
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
        return {
            version: isSet(object.version) ? String(object.version) : "",
            blockVersion: isSet(object.blockVersion) ? long_1.default.fromValue(object.blockVersion) : long_1.default.UZERO,
            p2pVersion: isSet(object.p2pVersion) ? long_1.default.fromValue(object.p2pVersion) : long_1.default.UZERO,
            abciVersion: isSet(object.abciVersion) ? String(object.abciVersion) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.version !== undefined && (obj.version = message.version);
        message.blockVersion !== undefined && (obj.blockVersion = (message.blockVersion || long_1.default.UZERO).toString());
        message.p2pVersion !== undefined && (obj.p2pVersion = (message.p2pVersion || long_1.default.UZERO).toString());
        message.abciVersion !== undefined && (obj.abciVersion = message.abciVersion);
        return obj;
    },
    create(base) {
        return exports.RequestInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseRequestInfo();
        message.version = (_a = object.version) !== null && _a !== void 0 ? _a : "";
        message.blockVersion = (object.blockVersion !== undefined && object.blockVersion !== null)
            ? long_1.default.fromValue(object.blockVersion)
            : long_1.default.UZERO;
        message.p2pVersion = (object.p2pVersion !== undefined && object.p2pVersion !== null)
            ? long_1.default.fromValue(object.p2pVersion)
            : long_1.default.UZERO;
        message.abciVersion = (_b = object.abciVersion) !== null && _b !== void 0 ? _b : "";
        return message;
    },
};
function createBaseRequestInitChain() {
    return {
        time: undefined,
        chainId: "",
        consensusParams: undefined,
        validators: [],
        appStateBytes: new Uint8Array(),
        initialHeight: long_1.default.ZERO,
    };
}
exports.RequestInitChain = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.time !== undefined) {
            timestamp_1.Timestamp.encode(toTimestamp(message.time), writer.uint32(10).fork()).ldelim();
        }
        if (message.chainId !== "") {
            writer.uint32(18).string(message.chainId);
        }
        if (message.consensusParams !== undefined) {
            params_1.ConsensusParams.encode(message.consensusParams, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.validators) {
            exports.ValidatorUpdate.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (message.appStateBytes.length !== 0) {
            writer.uint32(42).bytes(message.appStateBytes);
        }
        if (!message.initialHeight.isZero()) {
            writer.uint32(48).int64(message.initialHeight);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestInitChain();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.time = fromTimestamp(timestamp_1.Timestamp.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.chainId = reader.string();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.consensusParams = params_1.ConsensusParams.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.validators.push(exports.ValidatorUpdate.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.appStateBytes = reader.bytes();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.initialHeight = reader.int64();
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
        return {
            time: isSet(object.time) ? fromJsonTimestamp(object.time) : undefined,
            chainId: isSet(object.chainId) ? String(object.chainId) : "",
            consensusParams: isSet(object.consensusParams) ? params_1.ConsensusParams.fromJSON(object.consensusParams) : undefined,
            validators: Array.isArray(object === null || object === void 0 ? void 0 : object.validators)
                ? object.validators.map((e) => exports.ValidatorUpdate.fromJSON(e))
                : [],
            appStateBytes: isSet(object.appStateBytes) ? bytesFromBase64(object.appStateBytes) : new Uint8Array(),
            initialHeight: isSet(object.initialHeight) ? long_1.default.fromValue(object.initialHeight) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.time !== undefined && (obj.time = message.time.toISOString());
        message.chainId !== undefined && (obj.chainId = message.chainId);
        message.consensusParams !== undefined &&
            (obj.consensusParams = message.consensusParams ? params_1.ConsensusParams.toJSON(message.consensusParams) : undefined);
        if (message.validators) {
            obj.validators = message.validators.map((e) => e ? exports.ValidatorUpdate.toJSON(e) : undefined);
        }
        else {
            obj.validators = [];
        }
        message.appStateBytes !== undefined &&
            (obj.appStateBytes = base64FromBytes(message.appStateBytes !== undefined ? message.appStateBytes : new Uint8Array()));
        message.initialHeight !== undefined && (obj.initialHeight = (message.initialHeight || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.RequestInitChain.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseRequestInitChain();
        message.time = (_a = object.time) !== null && _a !== void 0 ? _a : undefined;
        message.chainId = (_b = object.chainId) !== null && _b !== void 0 ? _b : "";
        message.consensusParams = (object.consensusParams !== undefined && object.consensusParams !== null)
            ? params_1.ConsensusParams.fromPartial(object.consensusParams)
            : undefined;
        message.validators = ((_c = object.validators) === null || _c === void 0 ? void 0 : _c.map((e) => exports.ValidatorUpdate.fromPartial(e))) || [];
        message.appStateBytes = (_d = object.appStateBytes) !== null && _d !== void 0 ? _d : new Uint8Array();
        message.initialHeight = (object.initialHeight !== undefined && object.initialHeight !== null)
            ? long_1.default.fromValue(object.initialHeight)
            : long_1.default.ZERO;
        return message;
    },
};
function createBaseRequestQuery() {
    return { data: new Uint8Array(), path: "", height: long_1.default.ZERO, prove: false };
}
exports.RequestQuery = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.data.length !== 0) {
            writer.uint32(10).bytes(message.data);
        }
        if (message.path !== "") {
            writer.uint32(18).string(message.path);
        }
        if (!message.height.isZero()) {
            writer.uint32(24).int64(message.height);
        }
        if (message.prove === true) {
            writer.uint32(32).bool(message.prove);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestQuery();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.path = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.prove = reader.bool();
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
        return {
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            path: isSet(object.path) ? String(object.path) : "",
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            prove: isSet(object.prove) ? Boolean(object.prove) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.path !== undefined && (obj.path = message.path);
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.prove !== undefined && (obj.prove = message.prove);
        return obj;
    },
    create(base) {
        return exports.RequestQuery.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseRequestQuery();
        message.data = (_a = object.data) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.path = (_b = object.path) !== null && _b !== void 0 ? _b : "";
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.prove = (_c = object.prove) !== null && _c !== void 0 ? _c : false;
        return message;
    },
};
function createBaseRequestBeginBlock() {
    return { hash: new Uint8Array(), header: undefined, lastCommitInfo: undefined, byzantineValidators: [] };
}
exports.RequestBeginBlock = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.hash.length !== 0) {
            writer.uint32(10).bytes(message.hash);
        }
        if (message.header !== undefined) {
            types_1.Header.encode(message.header, writer.uint32(18).fork()).ldelim();
        }
        if (message.lastCommitInfo !== undefined) {
            exports.CommitInfo.encode(message.lastCommitInfo, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.byzantineValidators) {
            exports.Misbehavior.encode(v, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestBeginBlock();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.hash = reader.bytes();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.header = types_1.Header.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.lastCommitInfo = exports.CommitInfo.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.byzantineValidators.push(exports.Misbehavior.decode(reader, reader.uint32()));
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
        return {
            hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
            header: isSet(object.header) ? types_1.Header.fromJSON(object.header) : undefined,
            lastCommitInfo: isSet(object.lastCommitInfo) ? exports.CommitInfo.fromJSON(object.lastCommitInfo) : undefined,
            byzantineValidators: Array.isArray(object === null || object === void 0 ? void 0 : object.byzantineValidators)
                ? object.byzantineValidators.map((e) => exports.Misbehavior.fromJSON(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.hash !== undefined &&
            (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
        message.header !== undefined && (obj.header = message.header ? types_1.Header.toJSON(message.header) : undefined);
        message.lastCommitInfo !== undefined &&
            (obj.lastCommitInfo = message.lastCommitInfo ? exports.CommitInfo.toJSON(message.lastCommitInfo) : undefined);
        if (message.byzantineValidators) {
            obj.byzantineValidators = message.byzantineValidators.map((e) => e ? exports.Misbehavior.toJSON(e) : undefined);
        }
        else {
            obj.byzantineValidators = [];
        }
        return obj;
    },
    create(base) {
        return exports.RequestBeginBlock.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseRequestBeginBlock();
        message.hash = (_a = object.hash) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.header = (object.header !== undefined && object.header !== null)
            ? types_1.Header.fromPartial(object.header)
            : undefined;
        message.lastCommitInfo = (object.lastCommitInfo !== undefined && object.lastCommitInfo !== null)
            ? exports.CommitInfo.fromPartial(object.lastCommitInfo)
            : undefined;
        message.byzantineValidators = ((_b = object.byzantineValidators) === null || _b === void 0 ? void 0 : _b.map((e) => exports.Misbehavior.fromPartial(e))) || [];
        return message;
    },
};
function createBaseRequestCheckTx() {
    return { tx: new Uint8Array(), type: 0 };
}
exports.RequestCheckTx = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.tx.length !== 0) {
            writer.uint32(10).bytes(message.tx);
        }
        if (message.type !== 0) {
            writer.uint32(16).int32(message.type);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestCheckTx();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.tx = reader.bytes();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.type = reader.int32();
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
        return {
            tx: isSet(object.tx) ? bytesFromBase64(object.tx) : new Uint8Array(),
            type: isSet(object.type) ? checkTxTypeFromJSON(object.type) : 0,
        };
    },
    toJSON(message) {
        const obj = {};
        message.tx !== undefined && (obj.tx = base64FromBytes(message.tx !== undefined ? message.tx : new Uint8Array()));
        message.type !== undefined && (obj.type = checkTxTypeToJSON(message.type));
        return obj;
    },
    create(base) {
        return exports.RequestCheckTx.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseRequestCheckTx();
        message.tx = (_a = object.tx) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.type = (_b = object.type) !== null && _b !== void 0 ? _b : 0;
        return message;
    },
};
function createBaseRequestDeliverTx() {
    return { tx: new Uint8Array() };
}
exports.RequestDeliverTx = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.tx.length !== 0) {
            writer.uint32(10).bytes(message.tx);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestDeliverTx();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.tx = reader.bytes();
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
        return { tx: isSet(object.tx) ? bytesFromBase64(object.tx) : new Uint8Array() };
    },
    toJSON(message) {
        const obj = {};
        message.tx !== undefined && (obj.tx = base64FromBytes(message.tx !== undefined ? message.tx : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.RequestDeliverTx.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseRequestDeliverTx();
        message.tx = (_a = object.tx) !== null && _a !== void 0 ? _a : new Uint8Array();
        return message;
    },
};
function createBaseRequestEndBlock() {
    return { height: long_1.default.ZERO };
}
exports.RequestEndBlock = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.height.isZero()) {
            writer.uint32(8).int64(message.height);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestEndBlock();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.height = reader.int64();
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
        return { height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO };
    },
    toJSON(message) {
        const obj = {};
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.RequestEndBlock.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseRequestEndBlock();
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        return message;
    },
};
function createBaseRequestCommit() {
    return {};
}
exports.RequestCommit = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestCommit();
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
        return exports.RequestCommit.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseRequestCommit();
        return message;
    },
};
function createBaseRequestListSnapshots() {
    return {};
}
exports.RequestListSnapshots = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestListSnapshots();
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
        return exports.RequestListSnapshots.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseRequestListSnapshots();
        return message;
    },
};
function createBaseRequestOfferSnapshot() {
    return { snapshot: undefined, appHash: new Uint8Array() };
}
exports.RequestOfferSnapshot = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.snapshot !== undefined) {
            exports.Snapshot.encode(message.snapshot, writer.uint32(10).fork()).ldelim();
        }
        if (message.appHash.length !== 0) {
            writer.uint32(18).bytes(message.appHash);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestOfferSnapshot();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.snapshot = exports.Snapshot.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.appHash = reader.bytes();
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
        return {
            snapshot: isSet(object.snapshot) ? exports.Snapshot.fromJSON(object.snapshot) : undefined,
            appHash: isSet(object.appHash) ? bytesFromBase64(object.appHash) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.snapshot !== undefined && (obj.snapshot = message.snapshot ? exports.Snapshot.toJSON(message.snapshot) : undefined);
        message.appHash !== undefined &&
            (obj.appHash = base64FromBytes(message.appHash !== undefined ? message.appHash : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.RequestOfferSnapshot.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseRequestOfferSnapshot();
        message.snapshot = (object.snapshot !== undefined && object.snapshot !== null)
            ? exports.Snapshot.fromPartial(object.snapshot)
            : undefined;
        message.appHash = (_a = object.appHash) !== null && _a !== void 0 ? _a : new Uint8Array();
        return message;
    },
};
function createBaseRequestLoadSnapshotChunk() {
    return { height: long_1.default.UZERO, format: 0, chunk: 0 };
}
exports.RequestLoadSnapshotChunk = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.height.isZero()) {
            writer.uint32(8).uint64(message.height);
        }
        if (message.format !== 0) {
            writer.uint32(16).uint32(message.format);
        }
        if (message.chunk !== 0) {
            writer.uint32(24).uint32(message.chunk);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestLoadSnapshotChunk();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.height = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.format = reader.uint32();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.chunk = reader.uint32();
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
        return {
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.UZERO,
            format: isSet(object.format) ? Number(object.format) : 0,
            chunk: isSet(object.chunk) ? Number(object.chunk) : 0,
        };
    },
    toJSON(message) {
        const obj = {};
        message.height !== undefined && (obj.height = (message.height || long_1.default.UZERO).toString());
        message.format !== undefined && (obj.format = Math.round(message.format));
        message.chunk !== undefined && (obj.chunk = Math.round(message.chunk));
        return obj;
    },
    create(base) {
        return exports.RequestLoadSnapshotChunk.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseRequestLoadSnapshotChunk();
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.UZERO;
        message.format = (_a = object.format) !== null && _a !== void 0 ? _a : 0;
        message.chunk = (_b = object.chunk) !== null && _b !== void 0 ? _b : 0;
        return message;
    },
};
function createBaseRequestApplySnapshotChunk() {
    return { index: 0, chunk: new Uint8Array(), sender: "" };
}
exports.RequestApplySnapshotChunk = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.index !== 0) {
            writer.uint32(8).uint32(message.index);
        }
        if (message.chunk.length !== 0) {
            writer.uint32(18).bytes(message.chunk);
        }
        if (message.sender !== "") {
            writer.uint32(26).string(message.sender);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestApplySnapshotChunk();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.index = reader.uint32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.chunk = reader.bytes();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.sender = reader.string();
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
        return {
            index: isSet(object.index) ? Number(object.index) : 0,
            chunk: isSet(object.chunk) ? bytesFromBase64(object.chunk) : new Uint8Array(),
            sender: isSet(object.sender) ? String(object.sender) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = Math.round(message.index));
        message.chunk !== undefined &&
            (obj.chunk = base64FromBytes(message.chunk !== undefined ? message.chunk : new Uint8Array()));
        message.sender !== undefined && (obj.sender = message.sender);
        return obj;
    },
    create(base) {
        return exports.RequestApplySnapshotChunk.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseRequestApplySnapshotChunk();
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : 0;
        message.chunk = (_b = object.chunk) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.sender = (_c = object.sender) !== null && _c !== void 0 ? _c : "";
        return message;
    },
};
function createBaseRequestPrepareProposal() {
    return {
        maxTxBytes: long_1.default.ZERO,
        txs: [],
        localLastCommit: undefined,
        misbehavior: [],
        height: long_1.default.ZERO,
        time: undefined,
        nextValidatorsHash: new Uint8Array(),
        proposerAddress: new Uint8Array(),
    };
}
exports.RequestPrepareProposal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.maxTxBytes.isZero()) {
            writer.uint32(8).int64(message.maxTxBytes);
        }
        for (const v of message.txs) {
            writer.uint32(18).bytes(v);
        }
        if (message.localLastCommit !== undefined) {
            exports.ExtendedCommitInfo.encode(message.localLastCommit, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.misbehavior) {
            exports.Misbehavior.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (!message.height.isZero()) {
            writer.uint32(40).int64(message.height);
        }
        if (message.time !== undefined) {
            timestamp_1.Timestamp.encode(toTimestamp(message.time), writer.uint32(50).fork()).ldelim();
        }
        if (message.nextValidatorsHash.length !== 0) {
            writer.uint32(58).bytes(message.nextValidatorsHash);
        }
        if (message.proposerAddress.length !== 0) {
            writer.uint32(66).bytes(message.proposerAddress);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestPrepareProposal();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.maxTxBytes = reader.int64();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.txs.push(reader.bytes());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.localLastCommit = exports.ExtendedCommitInfo.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.misbehavior.push(exports.Misbehavior.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.time = fromTimestamp(timestamp_1.Timestamp.decode(reader, reader.uint32()));
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.nextValidatorsHash = reader.bytes();
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.proposerAddress = reader.bytes();
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
        return {
            maxTxBytes: isSet(object.maxTxBytes) ? long_1.default.fromValue(object.maxTxBytes) : long_1.default.ZERO,
            txs: Array.isArray(object === null || object === void 0 ? void 0 : object.txs) ? object.txs.map((e) => bytesFromBase64(e)) : [],
            localLastCommit: isSet(object.localLastCommit) ? exports.ExtendedCommitInfo.fromJSON(object.localLastCommit) : undefined,
            misbehavior: Array.isArray(object === null || object === void 0 ? void 0 : object.misbehavior)
                ? object.misbehavior.map((e) => exports.Misbehavior.fromJSON(e))
                : [],
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            time: isSet(object.time) ? fromJsonTimestamp(object.time) : undefined,
            nextValidatorsHash: isSet(object.nextValidatorsHash)
                ? bytesFromBase64(object.nextValidatorsHash)
                : new Uint8Array(),
            proposerAddress: isSet(object.proposerAddress) ? bytesFromBase64(object.proposerAddress) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.maxTxBytes !== undefined && (obj.maxTxBytes = (message.maxTxBytes || long_1.default.ZERO).toString());
        if (message.txs) {
            obj.txs = message.txs.map((e) => base64FromBytes(e !== undefined ? e : new Uint8Array()));
        }
        else {
            obj.txs = [];
        }
        message.localLastCommit !== undefined &&
            (obj.localLastCommit = message.localLastCommit ? exports.ExtendedCommitInfo.toJSON(message.localLastCommit) : undefined);
        if (message.misbehavior) {
            obj.misbehavior = message.misbehavior.map((e) => e ? exports.Misbehavior.toJSON(e) : undefined);
        }
        else {
            obj.misbehavior = [];
        }
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.time !== undefined && (obj.time = message.time.toISOString());
        message.nextValidatorsHash !== undefined &&
            (obj.nextValidatorsHash = base64FromBytes(message.nextValidatorsHash !== undefined ? message.nextValidatorsHash : new Uint8Array()));
        message.proposerAddress !== undefined &&
            (obj.proposerAddress = base64FromBytes(message.proposerAddress !== undefined ? message.proposerAddress : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.RequestPrepareProposal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e;
        const message = createBaseRequestPrepareProposal();
        message.maxTxBytes = (object.maxTxBytes !== undefined && object.maxTxBytes !== null)
            ? long_1.default.fromValue(object.maxTxBytes)
            : long_1.default.ZERO;
        message.txs = ((_a = object.txs) === null || _a === void 0 ? void 0 : _a.map((e) => e)) || [];
        message.localLastCommit = (object.localLastCommit !== undefined && object.localLastCommit !== null)
            ? exports.ExtendedCommitInfo.fromPartial(object.localLastCommit)
            : undefined;
        message.misbehavior = ((_b = object.misbehavior) === null || _b === void 0 ? void 0 : _b.map((e) => exports.Misbehavior.fromPartial(e))) || [];
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.time = (_c = object.time) !== null && _c !== void 0 ? _c : undefined;
        message.nextValidatorsHash = (_d = object.nextValidatorsHash) !== null && _d !== void 0 ? _d : new Uint8Array();
        message.proposerAddress = (_e = object.proposerAddress) !== null && _e !== void 0 ? _e : new Uint8Array();
        return message;
    },
};
function createBaseRequestProcessProposal() {
    return {
        txs: [],
        proposedLastCommit: undefined,
        misbehavior: [],
        hash: new Uint8Array(),
        height: long_1.default.ZERO,
        time: undefined,
        nextValidatorsHash: new Uint8Array(),
        proposerAddress: new Uint8Array(),
    };
}
exports.RequestProcessProposal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.txs) {
            writer.uint32(10).bytes(v);
        }
        if (message.proposedLastCommit !== undefined) {
            exports.CommitInfo.encode(message.proposedLastCommit, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.misbehavior) {
            exports.Misbehavior.encode(v, writer.uint32(26).fork()).ldelim();
        }
        if (message.hash.length !== 0) {
            writer.uint32(34).bytes(message.hash);
        }
        if (!message.height.isZero()) {
            writer.uint32(40).int64(message.height);
        }
        if (message.time !== undefined) {
            timestamp_1.Timestamp.encode(toTimestamp(message.time), writer.uint32(50).fork()).ldelim();
        }
        if (message.nextValidatorsHash.length !== 0) {
            writer.uint32(58).bytes(message.nextValidatorsHash);
        }
        if (message.proposerAddress.length !== 0) {
            writer.uint32(66).bytes(message.proposerAddress);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseRequestProcessProposal();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.txs.push(reader.bytes());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.proposedLastCommit = exports.CommitInfo.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.misbehavior.push(exports.Misbehavior.decode(reader, reader.uint32()));
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.hash = reader.bytes();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.time = fromTimestamp(timestamp_1.Timestamp.decode(reader, reader.uint32()));
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.nextValidatorsHash = reader.bytes();
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.proposerAddress = reader.bytes();
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
        return {
            txs: Array.isArray(object === null || object === void 0 ? void 0 : object.txs) ? object.txs.map((e) => bytesFromBase64(e)) : [],
            proposedLastCommit: isSet(object.proposedLastCommit) ? exports.CommitInfo.fromJSON(object.proposedLastCommit) : undefined,
            misbehavior: Array.isArray(object === null || object === void 0 ? void 0 : object.misbehavior)
                ? object.misbehavior.map((e) => exports.Misbehavior.fromJSON(e))
                : [],
            hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            time: isSet(object.time) ? fromJsonTimestamp(object.time) : undefined,
            nextValidatorsHash: isSet(object.nextValidatorsHash)
                ? bytesFromBase64(object.nextValidatorsHash)
                : new Uint8Array(),
            proposerAddress: isSet(object.proposerAddress) ? bytesFromBase64(object.proposerAddress) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.txs) {
            obj.txs = message.txs.map((e) => base64FromBytes(e !== undefined ? e : new Uint8Array()));
        }
        else {
            obj.txs = [];
        }
        message.proposedLastCommit !== undefined &&
            (obj.proposedLastCommit = message.proposedLastCommit ? exports.CommitInfo.toJSON(message.proposedLastCommit) : undefined);
        if (message.misbehavior) {
            obj.misbehavior = message.misbehavior.map((e) => e ? exports.Misbehavior.toJSON(e) : undefined);
        }
        else {
            obj.misbehavior = [];
        }
        message.hash !== undefined &&
            (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.time !== undefined && (obj.time = message.time.toISOString());
        message.nextValidatorsHash !== undefined &&
            (obj.nextValidatorsHash = base64FromBytes(message.nextValidatorsHash !== undefined ? message.nextValidatorsHash : new Uint8Array()));
        message.proposerAddress !== undefined &&
            (obj.proposerAddress = base64FromBytes(message.proposerAddress !== undefined ? message.proposerAddress : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.RequestProcessProposal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f;
        const message = createBaseRequestProcessProposal();
        message.txs = ((_a = object.txs) === null || _a === void 0 ? void 0 : _a.map((e) => e)) || [];
        message.proposedLastCommit = (object.proposedLastCommit !== undefined && object.proposedLastCommit !== null)
            ? exports.CommitInfo.fromPartial(object.proposedLastCommit)
            : undefined;
        message.misbehavior = ((_b = object.misbehavior) === null || _b === void 0 ? void 0 : _b.map((e) => exports.Misbehavior.fromPartial(e))) || [];
        message.hash = (_c = object.hash) !== null && _c !== void 0 ? _c : new Uint8Array();
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.time = (_d = object.time) !== null && _d !== void 0 ? _d : undefined;
        message.nextValidatorsHash = (_e = object.nextValidatorsHash) !== null && _e !== void 0 ? _e : new Uint8Array();
        message.proposerAddress = (_f = object.proposerAddress) !== null && _f !== void 0 ? _f : new Uint8Array();
        return message;
    },
};
function createBaseResponse() {
    return {
        exception: undefined,
        echo: undefined,
        flush: undefined,
        info: undefined,
        initChain: undefined,
        query: undefined,
        beginBlock: undefined,
        checkTx: undefined,
        deliverTx: undefined,
        endBlock: undefined,
        commit: undefined,
        listSnapshots: undefined,
        offerSnapshot: undefined,
        loadSnapshotChunk: undefined,
        applySnapshotChunk: undefined,
        prepareProposal: undefined,
        processProposal: undefined,
    };
}
exports.Response = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.exception !== undefined) {
            exports.ResponseException.encode(message.exception, writer.uint32(10).fork()).ldelim();
        }
        if (message.echo !== undefined) {
            exports.ResponseEcho.encode(message.echo, writer.uint32(18).fork()).ldelim();
        }
        if (message.flush !== undefined) {
            exports.ResponseFlush.encode(message.flush, writer.uint32(26).fork()).ldelim();
        }
        if (message.info !== undefined) {
            exports.ResponseInfo.encode(message.info, writer.uint32(34).fork()).ldelim();
        }
        if (message.initChain !== undefined) {
            exports.ResponseInitChain.encode(message.initChain, writer.uint32(50).fork()).ldelim();
        }
        if (message.query !== undefined) {
            exports.ResponseQuery.encode(message.query, writer.uint32(58).fork()).ldelim();
        }
        if (message.beginBlock !== undefined) {
            exports.ResponseBeginBlock.encode(message.beginBlock, writer.uint32(66).fork()).ldelim();
        }
        if (message.checkTx !== undefined) {
            exports.ResponseCheckTx.encode(message.checkTx, writer.uint32(74).fork()).ldelim();
        }
        if (message.deliverTx !== undefined) {
            exports.ResponseDeliverTx.encode(message.deliverTx, writer.uint32(82).fork()).ldelim();
        }
        if (message.endBlock !== undefined) {
            exports.ResponseEndBlock.encode(message.endBlock, writer.uint32(90).fork()).ldelim();
        }
        if (message.commit !== undefined) {
            exports.ResponseCommit.encode(message.commit, writer.uint32(98).fork()).ldelim();
        }
        if (message.listSnapshots !== undefined) {
            exports.ResponseListSnapshots.encode(message.listSnapshots, writer.uint32(106).fork()).ldelim();
        }
        if (message.offerSnapshot !== undefined) {
            exports.ResponseOfferSnapshot.encode(message.offerSnapshot, writer.uint32(114).fork()).ldelim();
        }
        if (message.loadSnapshotChunk !== undefined) {
            exports.ResponseLoadSnapshotChunk.encode(message.loadSnapshotChunk, writer.uint32(122).fork()).ldelim();
        }
        if (message.applySnapshotChunk !== undefined) {
            exports.ResponseApplySnapshotChunk.encode(message.applySnapshotChunk, writer.uint32(130).fork()).ldelim();
        }
        if (message.prepareProposal !== undefined) {
            exports.ResponsePrepareProposal.encode(message.prepareProposal, writer.uint32(138).fork()).ldelim();
        }
        if (message.processProposal !== undefined) {
            exports.ResponseProcessProposal.encode(message.processProposal, writer.uint32(146).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponse();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.exception = exports.ResponseException.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.echo = exports.ResponseEcho.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.flush = exports.ResponseFlush.decode(reader, reader.uint32());
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.info = exports.ResponseInfo.decode(reader, reader.uint32());
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.initChain = exports.ResponseInitChain.decode(reader, reader.uint32());
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.query = exports.ResponseQuery.decode(reader, reader.uint32());
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.beginBlock = exports.ResponseBeginBlock.decode(reader, reader.uint32());
                    continue;
                case 9:
                    if (tag != 74) {
                        break;
                    }
                    message.checkTx = exports.ResponseCheckTx.decode(reader, reader.uint32());
                    continue;
                case 10:
                    if (tag != 82) {
                        break;
                    }
                    message.deliverTx = exports.ResponseDeliverTx.decode(reader, reader.uint32());
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.endBlock = exports.ResponseEndBlock.decode(reader, reader.uint32());
                    continue;
                case 12:
                    if (tag != 98) {
                        break;
                    }
                    message.commit = exports.ResponseCommit.decode(reader, reader.uint32());
                    continue;
                case 13:
                    if (tag != 106) {
                        break;
                    }
                    message.listSnapshots = exports.ResponseListSnapshots.decode(reader, reader.uint32());
                    continue;
                case 14:
                    if (tag != 114) {
                        break;
                    }
                    message.offerSnapshot = exports.ResponseOfferSnapshot.decode(reader, reader.uint32());
                    continue;
                case 15:
                    if (tag != 122) {
                        break;
                    }
                    message.loadSnapshotChunk = exports.ResponseLoadSnapshotChunk.decode(reader, reader.uint32());
                    continue;
                case 16:
                    if (tag != 130) {
                        break;
                    }
                    message.applySnapshotChunk = exports.ResponseApplySnapshotChunk.decode(reader, reader.uint32());
                    continue;
                case 17:
                    if (tag != 138) {
                        break;
                    }
                    message.prepareProposal = exports.ResponsePrepareProposal.decode(reader, reader.uint32());
                    continue;
                case 18:
                    if (tag != 146) {
                        break;
                    }
                    message.processProposal = exports.ResponseProcessProposal.decode(reader, reader.uint32());
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
        return {
            exception: isSet(object.exception) ? exports.ResponseException.fromJSON(object.exception) : undefined,
            echo: isSet(object.echo) ? exports.ResponseEcho.fromJSON(object.echo) : undefined,
            flush: isSet(object.flush) ? exports.ResponseFlush.fromJSON(object.flush) : undefined,
            info: isSet(object.info) ? exports.ResponseInfo.fromJSON(object.info) : undefined,
            initChain: isSet(object.initChain) ? exports.ResponseInitChain.fromJSON(object.initChain) : undefined,
            query: isSet(object.query) ? exports.ResponseQuery.fromJSON(object.query) : undefined,
            beginBlock: isSet(object.beginBlock) ? exports.ResponseBeginBlock.fromJSON(object.beginBlock) : undefined,
            checkTx: isSet(object.checkTx) ? exports.ResponseCheckTx.fromJSON(object.checkTx) : undefined,
            deliverTx: isSet(object.deliverTx) ? exports.ResponseDeliverTx.fromJSON(object.deliverTx) : undefined,
            endBlock: isSet(object.endBlock) ? exports.ResponseEndBlock.fromJSON(object.endBlock) : undefined,
            commit: isSet(object.commit) ? exports.ResponseCommit.fromJSON(object.commit) : undefined,
            listSnapshots: isSet(object.listSnapshots) ? exports.ResponseListSnapshots.fromJSON(object.listSnapshots) : undefined,
            offerSnapshot: isSet(object.offerSnapshot) ? exports.ResponseOfferSnapshot.fromJSON(object.offerSnapshot) : undefined,
            loadSnapshotChunk: isSet(object.loadSnapshotChunk)
                ? exports.ResponseLoadSnapshotChunk.fromJSON(object.loadSnapshotChunk)
                : undefined,
            applySnapshotChunk: isSet(object.applySnapshotChunk)
                ? exports.ResponseApplySnapshotChunk.fromJSON(object.applySnapshotChunk)
                : undefined,
            prepareProposal: isSet(object.prepareProposal)
                ? exports.ResponsePrepareProposal.fromJSON(object.prepareProposal)
                : undefined,
            processProposal: isSet(object.processProposal)
                ? exports.ResponseProcessProposal.fromJSON(object.processProposal)
                : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.exception !== undefined &&
            (obj.exception = message.exception ? exports.ResponseException.toJSON(message.exception) : undefined);
        message.echo !== undefined && (obj.echo = message.echo ? exports.ResponseEcho.toJSON(message.echo) : undefined);
        message.flush !== undefined && (obj.flush = message.flush ? exports.ResponseFlush.toJSON(message.flush) : undefined);
        message.info !== undefined && (obj.info = message.info ? exports.ResponseInfo.toJSON(message.info) : undefined);
        message.initChain !== undefined &&
            (obj.initChain = message.initChain ? exports.ResponseInitChain.toJSON(message.initChain) : undefined);
        message.query !== undefined && (obj.query = message.query ? exports.ResponseQuery.toJSON(message.query) : undefined);
        message.beginBlock !== undefined &&
            (obj.beginBlock = message.beginBlock ? exports.ResponseBeginBlock.toJSON(message.beginBlock) : undefined);
        message.checkTx !== undefined &&
            (obj.checkTx = message.checkTx ? exports.ResponseCheckTx.toJSON(message.checkTx) : undefined);
        message.deliverTx !== undefined &&
            (obj.deliverTx = message.deliverTx ? exports.ResponseDeliverTx.toJSON(message.deliverTx) : undefined);
        message.endBlock !== undefined &&
            (obj.endBlock = message.endBlock ? exports.ResponseEndBlock.toJSON(message.endBlock) : undefined);
        message.commit !== undefined && (obj.commit = message.commit ? exports.ResponseCommit.toJSON(message.commit) : undefined);
        message.listSnapshots !== undefined &&
            (obj.listSnapshots = message.listSnapshots ? exports.ResponseListSnapshots.toJSON(message.listSnapshots) : undefined);
        message.offerSnapshot !== undefined &&
            (obj.offerSnapshot = message.offerSnapshot ? exports.ResponseOfferSnapshot.toJSON(message.offerSnapshot) : undefined);
        message.loadSnapshotChunk !== undefined && (obj.loadSnapshotChunk = message.loadSnapshotChunk
            ? exports.ResponseLoadSnapshotChunk.toJSON(message.loadSnapshotChunk)
            : undefined);
        message.applySnapshotChunk !== undefined && (obj.applySnapshotChunk = message.applySnapshotChunk
            ? exports.ResponseApplySnapshotChunk.toJSON(message.applySnapshotChunk)
            : undefined);
        message.prepareProposal !== undefined && (obj.prepareProposal = message.prepareProposal
            ? exports.ResponsePrepareProposal.toJSON(message.prepareProposal)
            : undefined);
        message.processProposal !== undefined && (obj.processProposal = message.processProposal
            ? exports.ResponseProcessProposal.toJSON(message.processProposal)
            : undefined);
        return obj;
    },
    create(base) {
        return exports.Response.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseResponse();
        message.exception = (object.exception !== undefined && object.exception !== null)
            ? exports.ResponseException.fromPartial(object.exception)
            : undefined;
        message.echo = (object.echo !== undefined && object.echo !== null)
            ? exports.ResponseEcho.fromPartial(object.echo)
            : undefined;
        message.flush = (object.flush !== undefined && object.flush !== null)
            ? exports.ResponseFlush.fromPartial(object.flush)
            : undefined;
        message.info = (object.info !== undefined && object.info !== null)
            ? exports.ResponseInfo.fromPartial(object.info)
            : undefined;
        message.initChain = (object.initChain !== undefined && object.initChain !== null)
            ? exports.ResponseInitChain.fromPartial(object.initChain)
            : undefined;
        message.query = (object.query !== undefined && object.query !== null)
            ? exports.ResponseQuery.fromPartial(object.query)
            : undefined;
        message.beginBlock = (object.beginBlock !== undefined && object.beginBlock !== null)
            ? exports.ResponseBeginBlock.fromPartial(object.beginBlock)
            : undefined;
        message.checkTx = (object.checkTx !== undefined && object.checkTx !== null)
            ? exports.ResponseCheckTx.fromPartial(object.checkTx)
            : undefined;
        message.deliverTx = (object.deliverTx !== undefined && object.deliverTx !== null)
            ? exports.ResponseDeliverTx.fromPartial(object.deliverTx)
            : undefined;
        message.endBlock = (object.endBlock !== undefined && object.endBlock !== null)
            ? exports.ResponseEndBlock.fromPartial(object.endBlock)
            : undefined;
        message.commit = (object.commit !== undefined && object.commit !== null)
            ? exports.ResponseCommit.fromPartial(object.commit)
            : undefined;
        message.listSnapshots = (object.listSnapshots !== undefined && object.listSnapshots !== null)
            ? exports.ResponseListSnapshots.fromPartial(object.listSnapshots)
            : undefined;
        message.offerSnapshot = (object.offerSnapshot !== undefined && object.offerSnapshot !== null)
            ? exports.ResponseOfferSnapshot.fromPartial(object.offerSnapshot)
            : undefined;
        message.loadSnapshotChunk = (object.loadSnapshotChunk !== undefined && object.loadSnapshotChunk !== null)
            ? exports.ResponseLoadSnapshotChunk.fromPartial(object.loadSnapshotChunk)
            : undefined;
        message.applySnapshotChunk = (object.applySnapshotChunk !== undefined && object.applySnapshotChunk !== null)
            ? exports.ResponseApplySnapshotChunk.fromPartial(object.applySnapshotChunk)
            : undefined;
        message.prepareProposal = (object.prepareProposal !== undefined && object.prepareProposal !== null)
            ? exports.ResponsePrepareProposal.fromPartial(object.prepareProposal)
            : undefined;
        message.processProposal = (object.processProposal !== undefined && object.processProposal !== null)
            ? exports.ResponseProcessProposal.fromPartial(object.processProposal)
            : undefined;
        return message;
    },
};
function createBaseResponseException() {
    return { error: "" };
}
exports.ResponseException = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.error !== "") {
            writer.uint32(10).string(message.error);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseException();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.error = reader.string();
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
        return { error: isSet(object.error) ? String(object.error) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.error !== undefined && (obj.error = message.error);
        return obj;
    },
    create(base) {
        return exports.ResponseException.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseException();
        message.error = (_a = object.error) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseResponseEcho() {
    return { message: "" };
}
exports.ResponseEcho = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.message !== "") {
            writer.uint32(10).string(message.message);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseEcho();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.message = reader.string();
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
        return { message: isSet(object.message) ? String(object.message) : "" };
    },
    toJSON(message) {
        const obj = {};
        message.message !== undefined && (obj.message = message.message);
        return obj;
    },
    create(base) {
        return exports.ResponseEcho.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseEcho();
        message.message = (_a = object.message) !== null && _a !== void 0 ? _a : "";
        return message;
    },
};
function createBaseResponseFlush() {
    return {};
}
exports.ResponseFlush = {
    encode(_, writer = minimal_1.default.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseFlush();
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
        return exports.ResponseFlush.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(_) {
        const message = createBaseResponseFlush();
        return message;
    },
};
function createBaseResponseInfo() {
    return {
        data: "",
        version: "",
        appVersion: long_1.default.UZERO,
        lastBlockHeight: long_1.default.ZERO,
        lastBlockAppHash: new Uint8Array(),
    };
}
exports.ResponseInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.data !== "") {
            writer.uint32(10).string(message.data);
        }
        if (message.version !== "") {
            writer.uint32(18).string(message.version);
        }
        if (!message.appVersion.isZero()) {
            writer.uint32(24).uint64(message.appVersion);
        }
        if (!message.lastBlockHeight.isZero()) {
            writer.uint32(32).int64(message.lastBlockHeight);
        }
        if (message.lastBlockAppHash.length !== 0) {
            writer.uint32(42).bytes(message.lastBlockAppHash);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.data = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.version = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.appVersion = reader.uint64();
                    continue;
                case 4:
                    if (tag != 32) {
                        break;
                    }
                    message.lastBlockHeight = reader.int64();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.lastBlockAppHash = reader.bytes();
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
        return {
            data: isSet(object.data) ? String(object.data) : "",
            version: isSet(object.version) ? String(object.version) : "",
            appVersion: isSet(object.appVersion) ? long_1.default.fromValue(object.appVersion) : long_1.default.UZERO,
            lastBlockHeight: isSet(object.lastBlockHeight) ? long_1.default.fromValue(object.lastBlockHeight) : long_1.default.ZERO,
            lastBlockAppHash: isSet(object.lastBlockAppHash) ? bytesFromBase64(object.lastBlockAppHash) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined && (obj.data = message.data);
        message.version !== undefined && (obj.version = message.version);
        message.appVersion !== undefined && (obj.appVersion = (message.appVersion || long_1.default.UZERO).toString());
        message.lastBlockHeight !== undefined && (obj.lastBlockHeight = (message.lastBlockHeight || long_1.default.ZERO).toString());
        message.lastBlockAppHash !== undefined &&
            (obj.lastBlockAppHash = base64FromBytes(message.lastBlockAppHash !== undefined ? message.lastBlockAppHash : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.ResponseInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseResponseInfo();
        message.data = (_a = object.data) !== null && _a !== void 0 ? _a : "";
        message.version = (_b = object.version) !== null && _b !== void 0 ? _b : "";
        message.appVersion = (object.appVersion !== undefined && object.appVersion !== null)
            ? long_1.default.fromValue(object.appVersion)
            : long_1.default.UZERO;
        message.lastBlockHeight = (object.lastBlockHeight !== undefined && object.lastBlockHeight !== null)
            ? long_1.default.fromValue(object.lastBlockHeight)
            : long_1.default.ZERO;
        message.lastBlockAppHash = (_c = object.lastBlockAppHash) !== null && _c !== void 0 ? _c : new Uint8Array();
        return message;
    },
};
function createBaseResponseInitChain() {
    return { consensusParams: undefined, validators: [], appHash: new Uint8Array() };
}
exports.ResponseInitChain = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.consensusParams !== undefined) {
            params_1.ConsensusParams.encode(message.consensusParams, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.validators) {
            exports.ValidatorUpdate.encode(v, writer.uint32(18).fork()).ldelim();
        }
        if (message.appHash.length !== 0) {
            writer.uint32(26).bytes(message.appHash);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseInitChain();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.consensusParams = params_1.ConsensusParams.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.validators.push(exports.ValidatorUpdate.decode(reader, reader.uint32()));
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.appHash = reader.bytes();
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
        return {
            consensusParams: isSet(object.consensusParams) ? params_1.ConsensusParams.fromJSON(object.consensusParams) : undefined,
            validators: Array.isArray(object === null || object === void 0 ? void 0 : object.validators)
                ? object.validators.map((e) => exports.ValidatorUpdate.fromJSON(e))
                : [],
            appHash: isSet(object.appHash) ? bytesFromBase64(object.appHash) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.consensusParams !== undefined &&
            (obj.consensusParams = message.consensusParams ? params_1.ConsensusParams.toJSON(message.consensusParams) : undefined);
        if (message.validators) {
            obj.validators = message.validators.map((e) => e ? exports.ValidatorUpdate.toJSON(e) : undefined);
        }
        else {
            obj.validators = [];
        }
        message.appHash !== undefined &&
            (obj.appHash = base64FromBytes(message.appHash !== undefined ? message.appHash : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.ResponseInitChain.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseResponseInitChain();
        message.consensusParams = (object.consensusParams !== undefined && object.consensusParams !== null)
            ? params_1.ConsensusParams.fromPartial(object.consensusParams)
            : undefined;
        message.validators = ((_a = object.validators) === null || _a === void 0 ? void 0 : _a.map((e) => exports.ValidatorUpdate.fromPartial(e))) || [];
        message.appHash = (_b = object.appHash) !== null && _b !== void 0 ? _b : new Uint8Array();
        return message;
    },
};
function createBaseResponseQuery() {
    return {
        code: 0,
        log: "",
        info: "",
        index: long_1.default.ZERO,
        key: new Uint8Array(),
        value: new Uint8Array(),
        proofOps: undefined,
        height: long_1.default.ZERO,
        codespace: "",
    };
}
exports.ResponseQuery = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.code !== 0) {
            writer.uint32(8).uint32(message.code);
        }
        if (message.log !== "") {
            writer.uint32(26).string(message.log);
        }
        if (message.info !== "") {
            writer.uint32(34).string(message.info);
        }
        if (!message.index.isZero()) {
            writer.uint32(40).int64(message.index);
        }
        if (message.key.length !== 0) {
            writer.uint32(50).bytes(message.key);
        }
        if (message.value.length !== 0) {
            writer.uint32(58).bytes(message.value);
        }
        if (message.proofOps !== undefined) {
            proof_1.ProofOps.encode(message.proofOps, writer.uint32(66).fork()).ldelim();
        }
        if (!message.height.isZero()) {
            writer.uint32(72).int64(message.height);
        }
        if (message.codespace !== "") {
            writer.uint32(82).string(message.codespace);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseQuery();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.code = reader.uint32();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.log = reader.string();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.info = reader.string();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.index = reader.int64();
                    continue;
                case 6:
                    if (tag != 50) {
                        break;
                    }
                    message.key = reader.bytes();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.value = reader.bytes();
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.proofOps = proof_1.ProofOps.decode(reader, reader.uint32());
                    continue;
                case 9:
                    if (tag != 72) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 10:
                    if (tag != 82) {
                        break;
                    }
                    message.codespace = reader.string();
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
        return {
            code: isSet(object.code) ? Number(object.code) : 0,
            log: isSet(object.log) ? String(object.log) : "",
            info: isSet(object.info) ? String(object.info) : "",
            index: isSet(object.index) ? long_1.default.fromValue(object.index) : long_1.default.ZERO,
            key: isSet(object.key) ? bytesFromBase64(object.key) : new Uint8Array(),
            value: isSet(object.value) ? bytesFromBase64(object.value) : new Uint8Array(),
            proofOps: isSet(object.proofOps) ? proof_1.ProofOps.fromJSON(object.proofOps) : undefined,
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            codespace: isSet(object.codespace) ? String(object.codespace) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.code !== undefined && (obj.code = Math.round(message.code));
        message.log !== undefined && (obj.log = message.log);
        message.info !== undefined && (obj.info = message.info);
        message.index !== undefined && (obj.index = (message.index || long_1.default.ZERO).toString());
        message.key !== undefined &&
            (obj.key = base64FromBytes(message.key !== undefined ? message.key : new Uint8Array()));
        message.value !== undefined &&
            (obj.value = base64FromBytes(message.value !== undefined ? message.value : new Uint8Array()));
        message.proofOps !== undefined && (obj.proofOps = message.proofOps ? proof_1.ProofOps.toJSON(message.proofOps) : undefined);
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.codespace !== undefined && (obj.codespace = message.codespace);
        return obj;
    },
    create(base) {
        return exports.ResponseQuery.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f;
        const message = createBaseResponseQuery();
        message.code = (_a = object.code) !== null && _a !== void 0 ? _a : 0;
        message.log = (_b = object.log) !== null && _b !== void 0 ? _b : "";
        message.info = (_c = object.info) !== null && _c !== void 0 ? _c : "";
        message.index = (object.index !== undefined && object.index !== null) ? long_1.default.fromValue(object.index) : long_1.default.ZERO;
        message.key = (_d = object.key) !== null && _d !== void 0 ? _d : new Uint8Array();
        message.value = (_e = object.value) !== null && _e !== void 0 ? _e : new Uint8Array();
        message.proofOps = (object.proofOps !== undefined && object.proofOps !== null)
            ? proof_1.ProofOps.fromPartial(object.proofOps)
            : undefined;
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.codespace = (_f = object.codespace) !== null && _f !== void 0 ? _f : "";
        return message;
    },
};
function createBaseResponseBeginBlock() {
    return { events: [] };
}
exports.ResponseBeginBlock = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.events) {
            exports.Event.encode(v, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseBeginBlock();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.events.push(exports.Event.decode(reader, reader.uint32()));
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
        return { events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => exports.Event.fromJSON(e)) : [] };
    },
    toJSON(message) {
        const obj = {};
        if (message.events) {
            obj.events = message.events.map((e) => e ? exports.Event.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        return obj;
    },
    create(base) {
        return exports.ResponseBeginBlock.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseBeginBlock();
        message.events = ((_a = object.events) === null || _a === void 0 ? void 0 : _a.map((e) => exports.Event.fromPartial(e))) || [];
        return message;
    },
};
function createBaseResponseCheckTx() {
    return {
        code: 0,
        data: new Uint8Array(),
        log: "",
        info: "",
        gasWanted: long_1.default.ZERO,
        gasUsed: long_1.default.ZERO,
        events: [],
        codespace: "",
        sender: "",
        priority: long_1.default.ZERO,
        mempoolError: "",
    };
}
exports.ResponseCheckTx = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.code !== 0) {
            writer.uint32(8).uint32(message.code);
        }
        if (message.data.length !== 0) {
            writer.uint32(18).bytes(message.data);
        }
        if (message.log !== "") {
            writer.uint32(26).string(message.log);
        }
        if (message.info !== "") {
            writer.uint32(34).string(message.info);
        }
        if (!message.gasWanted.isZero()) {
            writer.uint32(40).int64(message.gasWanted);
        }
        if (!message.gasUsed.isZero()) {
            writer.uint32(48).int64(message.gasUsed);
        }
        for (const v of message.events) {
            exports.Event.encode(v, writer.uint32(58).fork()).ldelim();
        }
        if (message.codespace !== "") {
            writer.uint32(66).string(message.codespace);
        }
        if (message.sender !== "") {
            writer.uint32(74).string(message.sender);
        }
        if (!message.priority.isZero()) {
            writer.uint32(80).int64(message.priority);
        }
        if (message.mempoolError !== "") {
            writer.uint32(90).string(message.mempoolError);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseCheckTx();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.code = reader.uint32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.log = reader.string();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.info = reader.string();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.gasWanted = reader.int64();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.gasUsed = reader.int64();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.events.push(exports.Event.decode(reader, reader.uint32()));
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.codespace = reader.string();
                    continue;
                case 9:
                    if (tag != 74) {
                        break;
                    }
                    message.sender = reader.string();
                    continue;
                case 10:
                    if (tag != 80) {
                        break;
                    }
                    message.priority = reader.int64();
                    continue;
                case 11:
                    if (tag != 90) {
                        break;
                    }
                    message.mempoolError = reader.string();
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
        return {
            code: isSet(object.code) ? Number(object.code) : 0,
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            log: isSet(object.log) ? String(object.log) : "",
            info: isSet(object.info) ? String(object.info) : "",
            gasWanted: isSet(object.gas_wanted) ? long_1.default.fromValue(object.gas_wanted) : long_1.default.ZERO,
            gasUsed: isSet(object.gas_used) ? long_1.default.fromValue(object.gas_used) : long_1.default.ZERO,
            events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => exports.Event.fromJSON(e)) : [],
            codespace: isSet(object.codespace) ? String(object.codespace) : "",
            sender: isSet(object.sender) ? String(object.sender) : "",
            priority: isSet(object.priority) ? long_1.default.fromValue(object.priority) : long_1.default.ZERO,
            mempoolError: isSet(object.mempoolError) ? String(object.mempoolError) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.code !== undefined && (obj.code = Math.round(message.code));
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.log !== undefined && (obj.log = message.log);
        message.info !== undefined && (obj.info = message.info);
        message.gasWanted !== undefined && (obj.gas_wanted = (message.gasWanted || long_1.default.ZERO).toString());
        message.gasUsed !== undefined && (obj.gas_used = (message.gasUsed || long_1.default.ZERO).toString());
        if (message.events) {
            obj.events = message.events.map((e) => e ? exports.Event.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        message.codespace !== undefined && (obj.codespace = message.codespace);
        message.sender !== undefined && (obj.sender = message.sender);
        message.priority !== undefined && (obj.priority = (message.priority || long_1.default.ZERO).toString());
        message.mempoolError !== undefined && (obj.mempoolError = message.mempoolError);
        return obj;
    },
    create(base) {
        return exports.ResponseCheckTx.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f, _g, _h;
        const message = createBaseResponseCheckTx();
        message.code = (_a = object.code) !== null && _a !== void 0 ? _a : 0;
        message.data = (_b = object.data) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.log = (_c = object.log) !== null && _c !== void 0 ? _c : "";
        message.info = (_d = object.info) !== null && _d !== void 0 ? _d : "";
        message.gasWanted = (object.gasWanted !== undefined && object.gasWanted !== null)
            ? long_1.default.fromValue(object.gasWanted)
            : long_1.default.ZERO;
        message.gasUsed = (object.gasUsed !== undefined && object.gasUsed !== null)
            ? long_1.default.fromValue(object.gasUsed)
            : long_1.default.ZERO;
        message.events = ((_e = object.events) === null || _e === void 0 ? void 0 : _e.map((e) => exports.Event.fromPartial(e))) || [];
        message.codespace = (_f = object.codespace) !== null && _f !== void 0 ? _f : "";
        message.sender = (_g = object.sender) !== null && _g !== void 0 ? _g : "";
        message.priority = (object.priority !== undefined && object.priority !== null)
            ? long_1.default.fromValue(object.priority)
            : long_1.default.ZERO;
        message.mempoolError = (_h = object.mempoolError) !== null && _h !== void 0 ? _h : "";
        return message;
    },
};
function createBaseResponseDeliverTx() {
    return {
        code: 0,
        data: new Uint8Array(),
        log: "",
        info: "",
        gasWanted: long_1.default.ZERO,
        gasUsed: long_1.default.ZERO,
        events: [],
        codespace: "",
    };
}
exports.ResponseDeliverTx = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.code !== 0) {
            writer.uint32(8).uint32(message.code);
        }
        if (message.data.length !== 0) {
            writer.uint32(18).bytes(message.data);
        }
        if (message.log !== "") {
            writer.uint32(26).string(message.log);
        }
        if (message.info !== "") {
            writer.uint32(34).string(message.info);
        }
        if (!message.gasWanted.isZero()) {
            writer.uint32(40).int64(message.gasWanted);
        }
        if (!message.gasUsed.isZero()) {
            writer.uint32(48).int64(message.gasUsed);
        }
        for (const v of message.events) {
            exports.Event.encode(v, writer.uint32(58).fork()).ldelim();
        }
        if (message.codespace !== "") {
            writer.uint32(66).string(message.codespace);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseDeliverTx();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.code = reader.uint32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.log = reader.string();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.info = reader.string();
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.gasWanted = reader.int64();
                    continue;
                case 6:
                    if (tag != 48) {
                        break;
                    }
                    message.gasUsed = reader.int64();
                    continue;
                case 7:
                    if (tag != 58) {
                        break;
                    }
                    message.events.push(exports.Event.decode(reader, reader.uint32()));
                    continue;
                case 8:
                    if (tag != 66) {
                        break;
                    }
                    message.codespace = reader.string();
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
        return {
            code: isSet(object.code) ? Number(object.code) : 0,
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            log: isSet(object.log) ? String(object.log) : "",
            info: isSet(object.info) ? String(object.info) : "",
            gasWanted: isSet(object.gas_wanted) ? long_1.default.fromValue(object.gas_wanted) : long_1.default.ZERO,
            gasUsed: isSet(object.gas_used) ? long_1.default.fromValue(object.gas_used) : long_1.default.ZERO,
            events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => exports.Event.fromJSON(e)) : [],
            codespace: isSet(object.codespace) ? String(object.codespace) : "",
        };
    },
    toJSON(message) {
        const obj = {};
        message.code !== undefined && (obj.code = Math.round(message.code));
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.log !== undefined && (obj.log = message.log);
        message.info !== undefined && (obj.info = message.info);
        message.gasWanted !== undefined && (obj.gas_wanted = (message.gasWanted || long_1.default.ZERO).toString());
        message.gasUsed !== undefined && (obj.gas_used = (message.gasUsed || long_1.default.ZERO).toString());
        if (message.events) {
            obj.events = message.events.map((e) => e ? exports.Event.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        message.codespace !== undefined && (obj.codespace = message.codespace);
        return obj;
    },
    create(base) {
        return exports.ResponseDeliverTx.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d, _e, _f;
        const message = createBaseResponseDeliverTx();
        message.code = (_a = object.code) !== null && _a !== void 0 ? _a : 0;
        message.data = (_b = object.data) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.log = (_c = object.log) !== null && _c !== void 0 ? _c : "";
        message.info = (_d = object.info) !== null && _d !== void 0 ? _d : "";
        message.gasWanted = (object.gasWanted !== undefined && object.gasWanted !== null)
            ? long_1.default.fromValue(object.gasWanted)
            : long_1.default.ZERO;
        message.gasUsed = (object.gasUsed !== undefined && object.gasUsed !== null)
            ? long_1.default.fromValue(object.gasUsed)
            : long_1.default.ZERO;
        message.events = ((_e = object.events) === null || _e === void 0 ? void 0 : _e.map((e) => exports.Event.fromPartial(e))) || [];
        message.codespace = (_f = object.codespace) !== null && _f !== void 0 ? _f : "";
        return message;
    },
};
function createBaseResponseEndBlock() {
    return { validatorUpdates: [], consensusParamUpdates: undefined, events: [] };
}
exports.ResponseEndBlock = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.validatorUpdates) {
            exports.ValidatorUpdate.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.consensusParamUpdates !== undefined) {
            params_1.ConsensusParams.encode(message.consensusParamUpdates, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.events) {
            exports.Event.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseEndBlock();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.validatorUpdates.push(exports.ValidatorUpdate.decode(reader, reader.uint32()));
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.consensusParamUpdates = params_1.ConsensusParams.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.events.push(exports.Event.decode(reader, reader.uint32()));
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
        return {
            validatorUpdates: Array.isArray(object === null || object === void 0 ? void 0 : object.validatorUpdates)
                ? object.validatorUpdates.map((e) => exports.ValidatorUpdate.fromJSON(e))
                : [],
            consensusParamUpdates: isSet(object.consensusParamUpdates)
                ? params_1.ConsensusParams.fromJSON(object.consensusParamUpdates)
                : undefined,
            events: Array.isArray(object === null || object === void 0 ? void 0 : object.events) ? object.events.map((e) => exports.Event.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.validatorUpdates) {
            obj.validatorUpdates = message.validatorUpdates.map((e) => e ? exports.ValidatorUpdate.toJSON(e) : undefined);
        }
        else {
            obj.validatorUpdates = [];
        }
        message.consensusParamUpdates !== undefined && (obj.consensusParamUpdates = message.consensusParamUpdates
            ? params_1.ConsensusParams.toJSON(message.consensusParamUpdates)
            : undefined);
        if (message.events) {
            obj.events = message.events.map((e) => e ? exports.Event.toJSON(e) : undefined);
        }
        else {
            obj.events = [];
        }
        return obj;
    },
    create(base) {
        return exports.ResponseEndBlock.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseResponseEndBlock();
        message.validatorUpdates = ((_a = object.validatorUpdates) === null || _a === void 0 ? void 0 : _a.map((e) => exports.ValidatorUpdate.fromPartial(e))) || [];
        message.consensusParamUpdates =
            (object.consensusParamUpdates !== undefined && object.consensusParamUpdates !== null)
                ? params_1.ConsensusParams.fromPartial(object.consensusParamUpdates)
                : undefined;
        message.events = ((_b = object.events) === null || _b === void 0 ? void 0 : _b.map((e) => exports.Event.fromPartial(e))) || [];
        return message;
    },
};
function createBaseResponseCommit() {
    return { data: new Uint8Array(), retainHeight: long_1.default.ZERO };
}
exports.ResponseCommit = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.data.length !== 0) {
            writer.uint32(18).bytes(message.data);
        }
        if (!message.retainHeight.isZero()) {
            writer.uint32(24).int64(message.retainHeight);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseCommit();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.data = reader.bytes();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.retainHeight = reader.int64();
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
        return {
            data: isSet(object.data) ? bytesFromBase64(object.data) : new Uint8Array(),
            retainHeight: isSet(object.retainHeight) ? long_1.default.fromValue(object.retainHeight) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.data !== undefined &&
            (obj.data = base64FromBytes(message.data !== undefined ? message.data : new Uint8Array()));
        message.retainHeight !== undefined && (obj.retainHeight = (message.retainHeight || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.ResponseCommit.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseCommit();
        message.data = (_a = object.data) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.retainHeight = (object.retainHeight !== undefined && object.retainHeight !== null)
            ? long_1.default.fromValue(object.retainHeight)
            : long_1.default.ZERO;
        return message;
    },
};
function createBaseResponseListSnapshots() {
    return { snapshots: [] };
}
exports.ResponseListSnapshots = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.snapshots) {
            exports.Snapshot.encode(v, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseListSnapshots();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.snapshots.push(exports.Snapshot.decode(reader, reader.uint32()));
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
        return {
            snapshots: Array.isArray(object === null || object === void 0 ? void 0 : object.snapshots) ? object.snapshots.map((e) => exports.Snapshot.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        if (message.snapshots) {
            obj.snapshots = message.snapshots.map((e) => e ? exports.Snapshot.toJSON(e) : undefined);
        }
        else {
            obj.snapshots = [];
        }
        return obj;
    },
    create(base) {
        return exports.ResponseListSnapshots.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseListSnapshots();
        message.snapshots = ((_a = object.snapshots) === null || _a === void 0 ? void 0 : _a.map((e) => exports.Snapshot.fromPartial(e))) || [];
        return message;
    },
};
function createBaseResponseOfferSnapshot() {
    return { result: 0 };
}
exports.ResponseOfferSnapshot = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.result !== 0) {
            writer.uint32(8).int32(message.result);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseOfferSnapshot();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.result = reader.int32();
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
        return { result: isSet(object.result) ? responseOfferSnapshot_ResultFromJSON(object.result) : 0 };
    },
    toJSON(message) {
        const obj = {};
        message.result !== undefined && (obj.result = responseOfferSnapshot_ResultToJSON(message.result));
        return obj;
    },
    create(base) {
        return exports.ResponseOfferSnapshot.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseOfferSnapshot();
        message.result = (_a = object.result) !== null && _a !== void 0 ? _a : 0;
        return message;
    },
};
function createBaseResponseLoadSnapshotChunk() {
    return { chunk: new Uint8Array() };
}
exports.ResponseLoadSnapshotChunk = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.chunk.length !== 0) {
            writer.uint32(10).bytes(message.chunk);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseLoadSnapshotChunk();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.chunk = reader.bytes();
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
        return { chunk: isSet(object.chunk) ? bytesFromBase64(object.chunk) : new Uint8Array() };
    },
    toJSON(message) {
        const obj = {};
        message.chunk !== undefined &&
            (obj.chunk = base64FromBytes(message.chunk !== undefined ? message.chunk : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.ResponseLoadSnapshotChunk.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseLoadSnapshotChunk();
        message.chunk = (_a = object.chunk) !== null && _a !== void 0 ? _a : new Uint8Array();
        return message;
    },
};
function createBaseResponseApplySnapshotChunk() {
    return { result: 0, refetchChunks: [], rejectSenders: [] };
}
exports.ResponseApplySnapshotChunk = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.result !== 0) {
            writer.uint32(8).int32(message.result);
        }
        writer.uint32(18).fork();
        for (const v of message.refetchChunks) {
            writer.uint32(v);
        }
        writer.ldelim();
        for (const v of message.rejectSenders) {
            writer.uint32(26).string(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseApplySnapshotChunk();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.result = reader.int32();
                    continue;
                case 2:
                    if (tag == 16) {
                        message.refetchChunks.push(reader.uint32());
                        continue;
                    }
                    if (tag == 18) {
                        const end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.refetchChunks.push(reader.uint32());
                        }
                        continue;
                    }
                    break;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.rejectSenders.push(reader.string());
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
        return {
            result: isSet(object.result) ? responseApplySnapshotChunk_ResultFromJSON(object.result) : 0,
            refetchChunks: Array.isArray(object === null || object === void 0 ? void 0 : object.refetchChunks) ? object.refetchChunks.map((e) => Number(e)) : [],
            rejectSenders: Array.isArray(object === null || object === void 0 ? void 0 : object.rejectSenders) ? object.rejectSenders.map((e) => String(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.result !== undefined && (obj.result = responseApplySnapshotChunk_ResultToJSON(message.result));
        if (message.refetchChunks) {
            obj.refetchChunks = message.refetchChunks.map((e) => Math.round(e));
        }
        else {
            obj.refetchChunks = [];
        }
        if (message.rejectSenders) {
            obj.rejectSenders = message.rejectSenders.map((e) => e);
        }
        else {
            obj.rejectSenders = [];
        }
        return obj;
    },
    create(base) {
        return exports.ResponseApplySnapshotChunk.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseResponseApplySnapshotChunk();
        message.result = (_a = object.result) !== null && _a !== void 0 ? _a : 0;
        message.refetchChunks = ((_b = object.refetchChunks) === null || _b === void 0 ? void 0 : _b.map((e) => e)) || [];
        message.rejectSenders = ((_c = object.rejectSenders) === null || _c === void 0 ? void 0 : _c.map((e) => e)) || [];
        return message;
    },
};
function createBaseResponsePrepareProposal() {
    return { txs: [] };
}
exports.ResponsePrepareProposal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        for (const v of message.txs) {
            writer.uint32(10).bytes(v);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponsePrepareProposal();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.txs.push(reader.bytes());
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
        return { txs: Array.isArray(object === null || object === void 0 ? void 0 : object.txs) ? object.txs.map((e) => bytesFromBase64(e)) : [] };
    },
    toJSON(message) {
        const obj = {};
        if (message.txs) {
            obj.txs = message.txs.map((e) => base64FromBytes(e !== undefined ? e : new Uint8Array()));
        }
        else {
            obj.txs = [];
        }
        return obj;
    },
    create(base) {
        return exports.ResponsePrepareProposal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponsePrepareProposal();
        message.txs = ((_a = object.txs) === null || _a === void 0 ? void 0 : _a.map((e) => e)) || [];
        return message;
    },
};
function createBaseResponseProcessProposal() {
    return { status: 0 };
}
exports.ResponseProcessProposal = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.status !== 0) {
            writer.uint32(8).int32(message.status);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseResponseProcessProposal();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.status = reader.int32();
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
        return { status: isSet(object.status) ? responseProcessProposal_ProposalStatusFromJSON(object.status) : 0 };
    },
    toJSON(message) {
        const obj = {};
        message.status !== undefined && (obj.status = responseProcessProposal_ProposalStatusToJSON(message.status));
        return obj;
    },
    create(base) {
        return exports.ResponseProcessProposal.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseResponseProcessProposal();
        message.status = (_a = object.status) !== null && _a !== void 0 ? _a : 0;
        return message;
    },
};
function createBaseCommitInfo() {
    return { round: 0, votes: [] };
}
exports.CommitInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.round !== 0) {
            writer.uint32(8).int32(message.round);
        }
        for (const v of message.votes) {
            exports.VoteInfo.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseCommitInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.round = reader.int32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.votes.push(exports.VoteInfo.decode(reader, reader.uint32()));
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
        return {
            round: isSet(object.round) ? Number(object.round) : 0,
            votes: Array.isArray(object === null || object === void 0 ? void 0 : object.votes) ? object.votes.map((e) => exports.VoteInfo.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.round !== undefined && (obj.round = Math.round(message.round));
        if (message.votes) {
            obj.votes = message.votes.map((e) => e ? exports.VoteInfo.toJSON(e) : undefined);
        }
        else {
            obj.votes = [];
        }
        return obj;
    },
    create(base) {
        return exports.CommitInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseCommitInfo();
        message.round = (_a = object.round) !== null && _a !== void 0 ? _a : 0;
        message.votes = ((_b = object.votes) === null || _b === void 0 ? void 0 : _b.map((e) => exports.VoteInfo.fromPartial(e))) || [];
        return message;
    },
};
function createBaseExtendedCommitInfo() {
    return { round: 0, votes: [] };
}
exports.ExtendedCommitInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.round !== 0) {
            writer.uint32(8).int32(message.round);
        }
        for (const v of message.votes) {
            exports.ExtendedVoteInfo.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseExtendedCommitInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.round = reader.int32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.votes.push(exports.ExtendedVoteInfo.decode(reader, reader.uint32()));
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
        return {
            round: isSet(object.round) ? Number(object.round) : 0,
            votes: Array.isArray(object === null || object === void 0 ? void 0 : object.votes) ? object.votes.map((e) => exports.ExtendedVoteInfo.fromJSON(e)) : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.round !== undefined && (obj.round = Math.round(message.round));
        if (message.votes) {
            obj.votes = message.votes.map((e) => e ? exports.ExtendedVoteInfo.toJSON(e) : undefined);
        }
        else {
            obj.votes = [];
        }
        return obj;
    },
    create(base) {
        return exports.ExtendedCommitInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseExtendedCommitInfo();
        message.round = (_a = object.round) !== null && _a !== void 0 ? _a : 0;
        message.votes = ((_b = object.votes) === null || _b === void 0 ? void 0 : _b.map((e) => exports.ExtendedVoteInfo.fromPartial(e))) || [];
        return message;
    },
};
function createBaseEvent() {
    return { type: "", attributes: [] };
}
exports.Event = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.type !== "") {
            writer.uint32(10).string(message.type);
        }
        for (const v of message.attributes) {
            exports.EventAttribute.encode(v, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEvent();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.type = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.attributes.push(exports.EventAttribute.decode(reader, reader.uint32()));
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
        return {
            type: isSet(object.type) ? String(object.type) : "",
            attributes: Array.isArray(object === null || object === void 0 ? void 0 : object.attributes)
                ? object.attributes.map((e) => exports.EventAttribute.fromJSON(e))
                : [],
        };
    },
    toJSON(message) {
        const obj = {};
        message.type !== undefined && (obj.type = message.type);
        if (message.attributes) {
            obj.attributes = message.attributes.map((e) => e ? exports.EventAttribute.toJSON(e) : undefined);
        }
        else {
            obj.attributes = [];
        }
        return obj;
    },
    create(base) {
        return exports.Event.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseEvent();
        message.type = (_a = object.type) !== null && _a !== void 0 ? _a : "";
        message.attributes = ((_b = object.attributes) === null || _b === void 0 ? void 0 : _b.map((e) => exports.EventAttribute.fromPartial(e))) || [];
        return message;
    },
};
function createBaseEventAttribute() {
    return { key: "", value: "", index: false };
}
exports.EventAttribute = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.key !== "") {
            writer.uint32(10).string(message.key);
        }
        if (message.value !== "") {
            writer.uint32(18).string(message.value);
        }
        if (message.index === true) {
            writer.uint32(24).bool(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseEventAttribute();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.key = reader.string();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.value = reader.string();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.index = reader.bool();
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
        return {
            key: isSet(object.key) ? String(object.key) : "",
            value: isSet(object.value) ? String(object.value) : "",
            index: isSet(object.index) ? Boolean(object.index) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.key !== undefined && (obj.key = message.key);
        message.value !== undefined && (obj.value = message.value);
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    create(base) {
        return exports.EventAttribute.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c;
        const message = createBaseEventAttribute();
        message.key = (_a = object.key) !== null && _a !== void 0 ? _a : "";
        message.value = (_b = object.value) !== null && _b !== void 0 ? _b : "";
        message.index = (_c = object.index) !== null && _c !== void 0 ? _c : false;
        return message;
    },
};
function createBaseTxResult() {
    return { height: long_1.default.ZERO, index: 0, tx: new Uint8Array(), result: undefined };
}
exports.TxResult = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.height.isZero()) {
            writer.uint32(8).int64(message.height);
        }
        if (message.index !== 0) {
            writer.uint32(16).uint32(message.index);
        }
        if (message.tx.length !== 0) {
            writer.uint32(26).bytes(message.tx);
        }
        if (message.result !== undefined) {
            exports.ResponseDeliverTx.encode(message.result, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseTxResult();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.index = reader.uint32();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.tx = reader.bytes();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.result = exports.ResponseDeliverTx.decode(reader, reader.uint32());
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
        return {
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            index: isSet(object.index) ? Number(object.index) : 0,
            tx: isSet(object.tx) ? bytesFromBase64(object.tx) : new Uint8Array(),
            result: isSet(object.result) ? exports.ResponseDeliverTx.fromJSON(object.result) : undefined,
        };
    },
    toJSON(message) {
        const obj = {};
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.index !== undefined && (obj.index = Math.round(message.index));
        message.tx !== undefined && (obj.tx = base64FromBytes(message.tx !== undefined ? message.tx : new Uint8Array()));
        message.result !== undefined &&
            (obj.result = message.result ? exports.ResponseDeliverTx.toJSON(message.result) : undefined);
        return obj;
    },
    create(base) {
        return exports.TxResult.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseTxResult();
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.index = (_a = object.index) !== null && _a !== void 0 ? _a : 0;
        message.tx = (_b = object.tx) !== null && _b !== void 0 ? _b : new Uint8Array();
        message.result = (object.result !== undefined && object.result !== null)
            ? exports.ResponseDeliverTx.fromPartial(object.result)
            : undefined;
        return message;
    },
};
function createBaseValidator() {
    return { address: new Uint8Array(), power: long_1.default.ZERO };
}
exports.Validator = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.address.length !== 0) {
            writer.uint32(10).bytes(message.address);
        }
        if (!message.power.isZero()) {
            writer.uint32(24).int64(message.power);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseValidator();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.address = reader.bytes();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.power = reader.int64();
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
        return {
            address: isSet(object.address) ? bytesFromBase64(object.address) : new Uint8Array(),
            power: isSet(object.power) ? long_1.default.fromValue(object.power) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.address !== undefined &&
            (obj.address = base64FromBytes(message.address !== undefined ? message.address : new Uint8Array()));
        message.power !== undefined && (obj.power = (message.power || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Validator.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseValidator();
        message.address = (_a = object.address) !== null && _a !== void 0 ? _a : new Uint8Array();
        message.power = (object.power !== undefined && object.power !== null) ? long_1.default.fromValue(object.power) : long_1.default.ZERO;
        return message;
    },
};
function createBaseValidatorUpdate() {
    return { pubKey: undefined, power: long_1.default.ZERO };
}
exports.ValidatorUpdate = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.pubKey !== undefined) {
            keys_1.PublicKey.encode(message.pubKey, writer.uint32(10).fork()).ldelim();
        }
        if (!message.power.isZero()) {
            writer.uint32(16).int64(message.power);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseValidatorUpdate();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.pubKey = keys_1.PublicKey.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.power = reader.int64();
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
        return {
            pubKey: isSet(object.pubKey) ? keys_1.PublicKey.fromJSON(object.pubKey) : undefined,
            power: isSet(object.power) ? long_1.default.fromValue(object.power) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.pubKey !== undefined && (obj.pubKey = message.pubKey ? keys_1.PublicKey.toJSON(message.pubKey) : undefined);
        message.power !== undefined && (obj.power = (message.power || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.ValidatorUpdate.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        const message = createBaseValidatorUpdate();
        message.pubKey = (object.pubKey !== undefined && object.pubKey !== null)
            ? keys_1.PublicKey.fromPartial(object.pubKey)
            : undefined;
        message.power = (object.power !== undefined && object.power !== null) ? long_1.default.fromValue(object.power) : long_1.default.ZERO;
        return message;
    },
};
function createBaseVoteInfo() {
    return { validator: undefined, signedLastBlock: false };
}
exports.VoteInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.validator !== undefined) {
            exports.Validator.encode(message.validator, writer.uint32(10).fork()).ldelim();
        }
        if (message.signedLastBlock === true) {
            writer.uint32(16).bool(message.signedLastBlock);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseVoteInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.validator = exports.Validator.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.signedLastBlock = reader.bool();
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
        return {
            validator: isSet(object.validator) ? exports.Validator.fromJSON(object.validator) : undefined,
            signedLastBlock: isSet(object.signedLastBlock) ? Boolean(object.signedLastBlock) : false,
        };
    },
    toJSON(message) {
        const obj = {};
        message.validator !== undefined &&
            (obj.validator = message.validator ? exports.Validator.toJSON(message.validator) : undefined);
        message.signedLastBlock !== undefined && (obj.signedLastBlock = message.signedLastBlock);
        return obj;
    },
    create(base) {
        return exports.VoteInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a;
        const message = createBaseVoteInfo();
        message.validator = (object.validator !== undefined && object.validator !== null)
            ? exports.Validator.fromPartial(object.validator)
            : undefined;
        message.signedLastBlock = (_a = object.signedLastBlock) !== null && _a !== void 0 ? _a : false;
        return message;
    },
};
function createBaseExtendedVoteInfo() {
    return { validator: undefined, signedLastBlock: false, voteExtension: new Uint8Array() };
}
exports.ExtendedVoteInfo = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.validator !== undefined) {
            exports.Validator.encode(message.validator, writer.uint32(10).fork()).ldelim();
        }
        if (message.signedLastBlock === true) {
            writer.uint32(16).bool(message.signedLastBlock);
        }
        if (message.voteExtension.length !== 0) {
            writer.uint32(26).bytes(message.voteExtension);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseExtendedVoteInfo();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 10) {
                        break;
                    }
                    message.validator = exports.Validator.decode(reader, reader.uint32());
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.signedLastBlock = reader.bool();
                    continue;
                case 3:
                    if (tag != 26) {
                        break;
                    }
                    message.voteExtension = reader.bytes();
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
        return {
            validator: isSet(object.validator) ? exports.Validator.fromJSON(object.validator) : undefined,
            signedLastBlock: isSet(object.signedLastBlock) ? Boolean(object.signedLastBlock) : false,
            voteExtension: isSet(object.voteExtension) ? bytesFromBase64(object.voteExtension) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.validator !== undefined &&
            (obj.validator = message.validator ? exports.Validator.toJSON(message.validator) : undefined);
        message.signedLastBlock !== undefined && (obj.signedLastBlock = message.signedLastBlock);
        message.voteExtension !== undefined &&
            (obj.voteExtension = base64FromBytes(message.voteExtension !== undefined ? message.voteExtension : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.ExtendedVoteInfo.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseExtendedVoteInfo();
        message.validator = (object.validator !== undefined && object.validator !== null)
            ? exports.Validator.fromPartial(object.validator)
            : undefined;
        message.signedLastBlock = (_a = object.signedLastBlock) !== null && _a !== void 0 ? _a : false;
        message.voteExtension = (_b = object.voteExtension) !== null && _b !== void 0 ? _b : new Uint8Array();
        return message;
    },
};
function createBaseMisbehavior() {
    return { type: 0, validator: undefined, height: long_1.default.ZERO, time: undefined, totalVotingPower: long_1.default.ZERO };
}
exports.Misbehavior = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (message.type !== 0) {
            writer.uint32(8).int32(message.type);
        }
        if (message.validator !== undefined) {
            exports.Validator.encode(message.validator, writer.uint32(18).fork()).ldelim();
        }
        if (!message.height.isZero()) {
            writer.uint32(24).int64(message.height);
        }
        if (message.time !== undefined) {
            timestamp_1.Timestamp.encode(toTimestamp(message.time), writer.uint32(34).fork()).ldelim();
        }
        if (!message.totalVotingPower.isZero()) {
            writer.uint32(40).int64(message.totalVotingPower);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseMisbehavior();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.type = reader.int32();
                    continue;
                case 2:
                    if (tag != 18) {
                        break;
                    }
                    message.validator = exports.Validator.decode(reader, reader.uint32());
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.height = reader.int64();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.time = fromTimestamp(timestamp_1.Timestamp.decode(reader, reader.uint32()));
                    continue;
                case 5:
                    if (tag != 40) {
                        break;
                    }
                    message.totalVotingPower = reader.int64();
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
        return {
            type: isSet(object.type) ? misbehaviorTypeFromJSON(object.type) : 0,
            validator: isSet(object.validator) ? exports.Validator.fromJSON(object.validator) : undefined,
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.ZERO,
            time: isSet(object.time) ? fromJsonTimestamp(object.time) : undefined,
            totalVotingPower: isSet(object.totalVotingPower) ? long_1.default.fromValue(object.totalVotingPower) : long_1.default.ZERO,
        };
    },
    toJSON(message) {
        const obj = {};
        message.type !== undefined && (obj.type = misbehaviorTypeToJSON(message.type));
        message.validator !== undefined &&
            (obj.validator = message.validator ? exports.Validator.toJSON(message.validator) : undefined);
        message.height !== undefined && (obj.height = (message.height || long_1.default.ZERO).toString());
        message.time !== undefined && (obj.time = message.time.toISOString());
        message.totalVotingPower !== undefined &&
            (obj.totalVotingPower = (message.totalVotingPower || long_1.default.ZERO).toString());
        return obj;
    },
    create(base) {
        return exports.Misbehavior.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b;
        const message = createBaseMisbehavior();
        message.type = (_a = object.type) !== null && _a !== void 0 ? _a : 0;
        message.validator = (object.validator !== undefined && object.validator !== null)
            ? exports.Validator.fromPartial(object.validator)
            : undefined;
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.ZERO;
        message.time = (_b = object.time) !== null && _b !== void 0 ? _b : undefined;
        message.totalVotingPower = (object.totalVotingPower !== undefined && object.totalVotingPower !== null)
            ? long_1.default.fromValue(object.totalVotingPower)
            : long_1.default.ZERO;
        return message;
    },
};
function createBaseSnapshot() {
    return { height: long_1.default.UZERO, format: 0, chunks: 0, hash: new Uint8Array(), metadata: new Uint8Array() };
}
exports.Snapshot = {
    encode(message, writer = minimal_1.default.Writer.create()) {
        if (!message.height.isZero()) {
            writer.uint32(8).uint64(message.height);
        }
        if (message.format !== 0) {
            writer.uint32(16).uint32(message.format);
        }
        if (message.chunks !== 0) {
            writer.uint32(24).uint32(message.chunks);
        }
        if (message.hash.length !== 0) {
            writer.uint32(34).bytes(message.hash);
        }
        if (message.metadata.length !== 0) {
            writer.uint32(42).bytes(message.metadata);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof minimal_1.default.Reader ? input : minimal_1.default.Reader.create(input);
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = createBaseSnapshot();
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if (tag != 8) {
                        break;
                    }
                    message.height = reader.uint64();
                    continue;
                case 2:
                    if (tag != 16) {
                        break;
                    }
                    message.format = reader.uint32();
                    continue;
                case 3:
                    if (tag != 24) {
                        break;
                    }
                    message.chunks = reader.uint32();
                    continue;
                case 4:
                    if (tag != 34) {
                        break;
                    }
                    message.hash = reader.bytes();
                    continue;
                case 5:
                    if (tag != 42) {
                        break;
                    }
                    message.metadata = reader.bytes();
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
        return {
            height: isSet(object.height) ? long_1.default.fromValue(object.height) : long_1.default.UZERO,
            format: isSet(object.format) ? Number(object.format) : 0,
            chunks: isSet(object.chunks) ? Number(object.chunks) : 0,
            hash: isSet(object.hash) ? bytesFromBase64(object.hash) : new Uint8Array(),
            metadata: isSet(object.metadata) ? bytesFromBase64(object.metadata) : new Uint8Array(),
        };
    },
    toJSON(message) {
        const obj = {};
        message.height !== undefined && (obj.height = (message.height || long_1.default.UZERO).toString());
        message.format !== undefined && (obj.format = Math.round(message.format));
        message.chunks !== undefined && (obj.chunks = Math.round(message.chunks));
        message.hash !== undefined &&
            (obj.hash = base64FromBytes(message.hash !== undefined ? message.hash : new Uint8Array()));
        message.metadata !== undefined &&
            (obj.metadata = base64FromBytes(message.metadata !== undefined ? message.metadata : new Uint8Array()));
        return obj;
    },
    create(base) {
        return exports.Snapshot.fromPartial(base !== null && base !== void 0 ? base : {});
    },
    fromPartial(object) {
        var _a, _b, _c, _d;
        const message = createBaseSnapshot();
        message.height = (object.height !== undefined && object.height !== null)
            ? long_1.default.fromValue(object.height)
            : long_1.default.UZERO;
        message.format = (_a = object.format) !== null && _a !== void 0 ? _a : 0;
        message.chunks = (_b = object.chunks) !== null && _b !== void 0 ? _b : 0;
        message.hash = (_c = object.hash) !== null && _c !== void 0 ? _c : new Uint8Array();
        message.metadata = (_d = object.metadata) !== null && _d !== void 0 ? _d : new Uint8Array();
        return message;
    },
};
class ABCIApplicationClientImpl {
    constructor(rpc, opts) {
        this.service = (opts === null || opts === void 0 ? void 0 : opts.service) || "tendermint.abci.ABCIApplication";
        this.rpc = rpc;
        this.Echo = this.Echo.bind(this);
        this.Flush = this.Flush.bind(this);
        this.Info = this.Info.bind(this);
        this.DeliverTx = this.DeliverTx.bind(this);
        this.CheckTx = this.CheckTx.bind(this);
        this.Query = this.Query.bind(this);
        this.Commit = this.Commit.bind(this);
        this.InitChain = this.InitChain.bind(this);
        this.BeginBlock = this.BeginBlock.bind(this);
        this.EndBlock = this.EndBlock.bind(this);
        this.ListSnapshots = this.ListSnapshots.bind(this);
        this.OfferSnapshot = this.OfferSnapshot.bind(this);
        this.LoadSnapshotChunk = this.LoadSnapshotChunk.bind(this);
        this.ApplySnapshotChunk = this.ApplySnapshotChunk.bind(this);
        this.PrepareProposal = this.PrepareProposal.bind(this);
        this.ProcessProposal = this.ProcessProposal.bind(this);
    }
    Echo(request) {
        const data = exports.RequestEcho.encode(request).finish();
        const promise = this.rpc.request(this.service, "Echo", data);
        return promise.then((data) => exports.ResponseEcho.decode(minimal_1.default.Reader.create(data)));
    }
    Flush(request) {
        const data = exports.RequestFlush.encode(request).finish();
        const promise = this.rpc.request(this.service, "Flush", data);
        return promise.then((data) => exports.ResponseFlush.decode(minimal_1.default.Reader.create(data)));
    }
    Info(request) {
        const data = exports.RequestInfo.encode(request).finish();
        const promise = this.rpc.request(this.service, "Info", data);
        return promise.then((data) => exports.ResponseInfo.decode(minimal_1.default.Reader.create(data)));
    }
    DeliverTx(request) {
        const data = exports.RequestDeliverTx.encode(request).finish();
        const promise = this.rpc.request(this.service, "DeliverTx", data);
        return promise.then((data) => exports.ResponseDeliverTx.decode(minimal_1.default.Reader.create(data)));
    }
    CheckTx(request) {
        const data = exports.RequestCheckTx.encode(request).finish();
        const promise = this.rpc.request(this.service, "CheckTx", data);
        return promise.then((data) => exports.ResponseCheckTx.decode(minimal_1.default.Reader.create(data)));
    }
    Query(request) {
        const data = exports.RequestQuery.encode(request).finish();
        const promise = this.rpc.request(this.service, "Query", data);
        return promise.then((data) => exports.ResponseQuery.decode(minimal_1.default.Reader.create(data)));
    }
    Commit(request) {
        const data = exports.RequestCommit.encode(request).finish();
        const promise = this.rpc.request(this.service, "Commit", data);
        return promise.then((data) => exports.ResponseCommit.decode(minimal_1.default.Reader.create(data)));
    }
    InitChain(request) {
        const data = exports.RequestInitChain.encode(request).finish();
        const promise = this.rpc.request(this.service, "InitChain", data);
        return promise.then((data) => exports.ResponseInitChain.decode(minimal_1.default.Reader.create(data)));
    }
    BeginBlock(request) {
        const data = exports.RequestBeginBlock.encode(request).finish();
        const promise = this.rpc.request(this.service, "BeginBlock", data);
        return promise.then((data) => exports.ResponseBeginBlock.decode(minimal_1.default.Reader.create(data)));
    }
    EndBlock(request) {
        const data = exports.RequestEndBlock.encode(request).finish();
        const promise = this.rpc.request(this.service, "EndBlock", data);
        return promise.then((data) => exports.ResponseEndBlock.decode(minimal_1.default.Reader.create(data)));
    }
    ListSnapshots(request) {
        const data = exports.RequestListSnapshots.encode(request).finish();
        const promise = this.rpc.request(this.service, "ListSnapshots", data);
        return promise.then((data) => exports.ResponseListSnapshots.decode(minimal_1.default.Reader.create(data)));
    }
    OfferSnapshot(request) {
        const data = exports.RequestOfferSnapshot.encode(request).finish();
        const promise = this.rpc.request(this.service, "OfferSnapshot", data);
        return promise.then((data) => exports.ResponseOfferSnapshot.decode(minimal_1.default.Reader.create(data)));
    }
    LoadSnapshotChunk(request) {
        const data = exports.RequestLoadSnapshotChunk.encode(request).finish();
        const promise = this.rpc.request(this.service, "LoadSnapshotChunk", data);
        return promise.then((data) => exports.ResponseLoadSnapshotChunk.decode(minimal_1.default.Reader.create(data)));
    }
    ApplySnapshotChunk(request) {
        const data = exports.RequestApplySnapshotChunk.encode(request).finish();
        const promise = this.rpc.request(this.service, "ApplySnapshotChunk", data);
        return promise.then((data) => exports.ResponseApplySnapshotChunk.decode(minimal_1.default.Reader.create(data)));
    }
    PrepareProposal(request) {
        const data = exports.RequestPrepareProposal.encode(request).finish();
        const promise = this.rpc.request(this.service, "PrepareProposal", data);
        return promise.then((data) => exports.ResponsePrepareProposal.decode(minimal_1.default.Reader.create(data)));
    }
    ProcessProposal(request) {
        const data = exports.RequestProcessProposal.encode(request).finish();
        const promise = this.rpc.request(this.service, "ProcessProposal", data);
        return promise.then((data) => exports.ResponseProcessProposal.decode(minimal_1.default.Reader.create(data)));
    }
}
exports.ABCIApplicationClientImpl = ABCIApplicationClientImpl;
var tsProtoGlobalThis = (() => {
    if (typeof globalThis !== "undefined") {
        return globalThis;
    }
    if (typeof self !== "undefined") {
        return self;
    }
    if (typeof window !== "undefined") {
        return window;
    }
    if (typeof global !== "undefined") {
        return global;
    }
    throw "Unable to locate global object";
})();
function bytesFromBase64(b64) {
    if (tsProtoGlobalThis.Buffer) {
        return Uint8Array.from(tsProtoGlobalThis.Buffer.from(b64, "base64"));
    }
    else {
        const bin = tsProtoGlobalThis.atob(b64);
        const arr = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; ++i) {
            arr[i] = bin.charCodeAt(i);
        }
        return arr;
    }
}
function base64FromBytes(arr) {
    if (tsProtoGlobalThis.Buffer) {
        return tsProtoGlobalThis.Buffer.from(arr).toString("base64");
    }
    else {
        const bin = [];
        arr.forEach((byte) => {
            bin.push(String.fromCharCode(byte));
        });
        return tsProtoGlobalThis.btoa(bin.join(""));
    }
}
function toTimestamp(date) {
    const seconds = numberToLong(date.getTime() / 1000);
    const nanos = (date.getTime() % 1000) * 1000000;
    return { seconds, nanos };
}
function fromTimestamp(t) {
    let millis = t.seconds.toNumber() * 1000;
    millis += t.nanos / 1000000;
    return new Date(millis);
}
function fromJsonTimestamp(o) {
    if (o instanceof Date) {
        return o;
    }
    else if (typeof o === "string") {
        return new Date(o);
    }
    else {
        return fromTimestamp(timestamp_1.Timestamp.fromJSON(o));
    }
}
function numberToLong(number) {
    return long_1.default.fromNumber(number);
}
if (minimal_1.default.util.Long !== long_1.default) {
    minimal_1.default.util.Long = long_1.default;
    minimal_1.default.configure();
}
function isSet(value) {
    return value !== null && value !== undefined;
}
