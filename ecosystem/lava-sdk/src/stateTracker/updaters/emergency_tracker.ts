import { StateQuery } from "../stateQuery/state_query";
import { Params as DowntimeParams } from "../../grpc_web_services/lavanet/lava/downtime/v1/downtime_pb";
import { Logger } from "../../logger/logger";
import { StateBadgeQuery } from "../stateQuery/state_badge_query";

export interface EmergencyTrackerInf {
  getVirtualEpoch(): number;
}

export class EmergencyTracker {
  private stateQuery: StateQuery;
  private latestEpoch: number;
  private latestEpochTime: number;
  private downtimeParams: DowntimeParams | undefined;

  constructor(stateQuery: StateQuery) {
    Logger.debug("Initialization of Emergency Tracker started");

    this.stateQuery = stateQuery;
    this.latestEpochTime = Date.now();
    this.latestEpoch = 0;
  }

  public async update() {
    const latestEpoch = this.stateQuery.getCurrentEpoch();
    if (latestEpoch == undefined || latestEpoch <= this.latestEpoch) {
      return;
    }

    this.latestEpochTime = Date.now();
    this.latestEpoch = latestEpoch;
  }

  public getVirtualEpoch(): number {
    // receive virtual epoch directly from badge query
    if (this.stateQuery instanceof StateBadgeQuery) {
      return (this.stateQuery as StateBadgeQuery).getVirtualEpoch();
    }

    if (this.downtimeParams == undefined) {
      return 0;
    }

    const downtimeDuration = this.downtimeParams.getDowntimeDuration();
    const epochDuration = this.downtimeParams.getEpochDuration();
    if (downtimeDuration == undefined || epochDuration == undefined) {
      return 0;
    }

    // check if emergency mode has started
    const delay = Date.now() - this.latestEpochTime;
    if (delay < downtimeDuration.getSeconds() * 1000) {
      return 0;
    }

    // division without rounding up to skip normal epoch,
    // if time since latestEpochTime > epochDuration => virtual epoch has started
    const virtualEpoch = delay / (epochDuration.getSeconds() * 1000);
    return virtualEpoch;
  }
}
