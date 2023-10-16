export interface Updater {
  update(virtualEpoch: number): Promise<any>;
}
