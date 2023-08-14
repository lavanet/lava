import { BadgeGenerator } from "../grpc_web_services/lavanet/lava/pairing/badges_pb_service";
import {
  GenerateBadgeRequest,
  GenerateBadgeResponse,
} from "../grpc_web_services/lavanet/lava/pairing/badges_pb";
import { grpc } from "@improbable-eng/grpc-web";
import transport from "../util/browser";

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
  private authentication: Map<string, string> | undefined;
  private active = true;
  constructor(options: BadgeOptions | undefined) {
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

  public isActive(): boolean {
    return this.active;
  }

  public async fetchBadge(
    badgeUser: string
  ): Promise<GenerateBadgeResponse | Error> {
    if (!this.active) {
      throw BadBadgeUsageWhileNotActiveError;
    }
    const request = new GenerateBadgeRequest();
    request.setBadgeAddress(badgeUser);
    request.setProjectId(this.projectId);
    const requestPromise = new Promise<GenerateBadgeResponse>(
      (resolve, reject) => {
        grpc.invoke(BadgeGenerator.GenerateBadge, {
          request: request,
          host: this.badgeServerAddress,
          metadata: this.authentication ? this.authentication : {}, // providing the authentication headers
          transport: transport,
          onMessage: (message: GenerateBadgeResponse) => {
            resolve(message);
          },
          onEnd: (code: grpc.Code, msg: string | undefined) => {
            if (code == grpc.Code.OK || msg == undefined) {
              return;
            }
            reject(
              new Error(
                "Failed fetching a badge from the badge server, message: " + msg
              )
            );
          },
        });
      }
    );
    return this.relayWithTimeout(5000, requestPromise);
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
