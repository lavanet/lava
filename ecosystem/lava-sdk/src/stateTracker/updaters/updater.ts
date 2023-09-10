import { ConsumerSessionManager } from "../../lavasession/consumerSessionManager";

export interface Updater {
  update(): void;
  registerPairing(
    consumerSessionManager: ConsumerSessionManager
  ): Promise<void | Error>;
}
