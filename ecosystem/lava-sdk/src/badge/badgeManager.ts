import { BadgeGeneratorClient } from "../grpc_web_services/lavanet/lava/pairing/badges_pb_service";
import {
  GenerateBadgeRequest,
  GenerateBadgeResponse,
} from "../grpc_web_services/lavanet/lava/pairing/badges_pb";
import { grpc } from "@improbable-eng/grpc-web";
import transport from "../util/browser";
import { Logger } from "../logger/logger";
import { ServiceError } from "../grpc_web_services/lavanet/lava/pairing/badges_pb_service";
const BadBadgeUsageWhileNotActiveError = new Error(
  "Bad BadgeManager usage detected, trying to use badge manager while not active"
);
export const TimoutFailureFetchingBadgeError = new Error(
  "Failed fetching badge, exceeded timeout duration"
);
/**
 * Interface for managing Badges
 */
export interface BadgeOptions {
  badgeServerAddress: string;
  projectId: string;
  authentication?: string;
}

export class BadgeManager {
  private badgeServerAddress = "";
  private projectId = "";
  private authentication: grpc.Metadata | undefined;
  private active = true;
  private transport: grpc.TransportFactory | undefined;
  private badgeGeneratorClient?: BadgeGeneratorClient;
  constructor(
    options: BadgeOptions | undefined,
    transport?: grpc.TransportFactory
  ) {
    if (!options) {
      this.active = false;
      return;
    }
    this.badgeServerAddress = options.badgeServerAddress;
    this.projectId = options.projectId;
    this.authentication = new grpc.Metadata();
    if (options.authentication) {
      this.authentication.append("Authorization", options.authentication);
    }
    this.transport = transport;

    this.badgeGeneratorClient = new BadgeGeneratorClient(
      this.badgeServerAddress,
      this.getTransportWrapped()
    );
  }

  public isActive(): boolean {
    return this.active;
  }

  public async fetchBadge(
    badgeUser: string,
    specId: string
  ): Promise<GenerateBadgeResponse | Error> {
    if (!this.active) {
      throw BadBadgeUsageWhileNotActiveError;
    }
    const request = new GenerateBadgeRequest();
    request.setBadgeAddress(badgeUser);
    request.setProjectId(this.projectId);
    request.setSpecId(specId);
    const requestPromise = new Promise<GenerateBadgeResponse>(
      (resolve, reject) => {
        if (!this.badgeGeneratorClient || !this.authentication) {
          // type fix with checks as they cant really be undefined.
          throw BadBadgeUsageWhileNotActiveError;
        }
        this.badgeGeneratorClient.generateBadge(
          request,
          this.authentication,
          (err: ServiceError | null, result: GenerateBadgeResponse | null) => {
            if (err != null) {
              Logger.error("failed fetching badge", err);
              reject(err);
            }

            if (result != null) {
              resolve(result);
            }
            reject(new Error("Didn't get an error nor result"));
          }
        );
      }
    );
    return this.relayWithTimeout(5000, requestPromise);
  }

  getTransport() {
    if (this.transport) {
      return this.transport;
    }
    return transport;
  }

  getTransportWrapped() {
    return {
      // if allow insecure we use a transport with rejectUnauthorized disabled
      // otherwise normal transport (default to rejectUnauthorized = true));}
      transport: this.getTransport(),
    };
  }

  private timeoutPromise<T>(timeout: number): Promise<T> {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        reject(new Error("Timeout exceeded"));
      }, timeout);
    });
  }

  private async relayWithTimeout(
    timeLimit: number,
    task: Promise<GenerateBadgeResponse>
  ): Promise<GenerateBadgeResponse | Error> {
    const response = await Promise.race([
      task,
      this.timeoutPromise<Error>(timeLimit),
    ]);
    return response;
  }
}
