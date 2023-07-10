"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.BadgeManager = exports.TimoutFailureFetchingBadgeError = void 0;
const badges_pb_service_1 = require("../grpc_web_services/pairing/badges_pb_service");
const badges_pb_1 = require("../grpc_web_services/pairing/badges_pb");
const grpc_web_1 = require("@improbable-eng/grpc-web");
const browser_1 = __importDefault(require("../util/browser"));
const BadBadgeUsageWhileNotActiveError = new Error("Bad BadgeManager usage detected, trying to use badge manager while not active");
exports.TimoutFailureFetchingBadgeError = new Error("Failed fetching badge, exceeded timeout duration");
class BadgeManager {
    constructor(options) {
        this.badgeServerAddress = "";
        this.projectId = "";
        this.active = true;
        if (!options) {
            this.active = false;
            return;
        }
        this.badgeServerAddress = options.badgeServerAddress;
        this.projectId = options.projectId;
        if (options.authentication) {
            this.authentication = new Map([
                ["Authorization", options.authentication],
            ]);
        }
    }
    isActive() {
        return this.active;
    }
    fetchBadge(badgeUser) {
        return __awaiter(this, void 0, void 0, function* () {
            if (!this.active) {
                throw BadBadgeUsageWhileNotActiveError;
            }
            const request = new badges_pb_1.GenerateBadgeRequest();
            request.setBadgeAddress(badgeUser);
            request.setProjectId(this.projectId);
            const requestPromise = new Promise((resolve, reject) => {
                grpc_web_1.grpc.invoke(badges_pb_service_1.BadgeGenerator.GenerateBadge, {
                    request: request,
                    host: this.badgeServerAddress,
                    metadata: this.authentication ? this.authentication : {},
                    transport: browser_1.default,
                    onMessage: (message) => {
                        resolve(message);
                    },
                    onEnd: (code, msg) => {
                        if (code == grpc_web_1.grpc.Code.OK || msg == undefined) {
                            return;
                        }
                        reject(new Error("Failed Fetching badge, message: " + msg));
                    },
                });
            });
            return this.relayWithTimeout(5000, requestPromise);
        });
    }
    timeoutPromise(timeout) {
        return new Promise((resolve, reject) => {
            setTimeout(() => {
                reject(new Error("Timeout exceeded"));
            }, timeout);
        });
    }
    relayWithTimeout(timeLimit, task) {
        return __awaiter(this, void 0, void 0, function* () {
            const response = yield Promise.race([
                task,
                this.timeoutPromise(timeLimit),
            ]);
            return response;
        });
    }
}
exports.BadgeManager = BadgeManager;
