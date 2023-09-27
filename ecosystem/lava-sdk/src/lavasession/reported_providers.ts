import Long from "long";
import { ReportedProvider } from "../grpc_web_services/lavanet/lava/pairing/relay_pb";

export class ReportedProviders {
  private addedToPurgeAndReport: Map<string, ReportedProviderEntry> = new Map<
    string,
    ReportedProviderEntry
  >(); // list of purged providers to report for QoS unavailability. (easier to search maps.)
  public reset() {
    this.addedToPurgeAndReport = new Map<string, ReportedProviderEntry>();
  }
  public GetReportedProviders(): Array<ReportedProvider> {
    const reportedProviders = new Array<ReportedProvider>(
      this.addedToPurgeAndReport.size // allocating space before inserting values has better performance.
    );
    let index = 0;
    for (const [
      provider,
      reportedProviderEntry,
    ] of this.addedToPurgeAndReport.entries()) {
      const reportedProvider = new ReportedProvider();
      reportedProvider.setAddress(provider);
      reportedProvider.setDisconnections(reportedProviderEntry.disconnections);
      reportedProvider.setErrors(reportedProviderEntry.errors);
      reportedProvider.setTimestampS(reportedProviderEntry.addedTime);
      reportedProviders[index] = reportedProvider;
      index++;
    }
    return reportedProviders;
  }
  public isReported(address: string): boolean {
    const reportedProvider = this.addedToPurgeAndReport.get(address);
    if (reportedProvider == undefined) {
      return false;
    }
    return true;
  }
  public reportedProvider(
    address: string,
    errors: number,
    disconnections: number
  ) {
    let reportedProvider: ReportedProviderEntry;
    const reportedProviderValue = this.addedToPurgeAndReport.get(address);
    if (reportedProviderValue == undefined) {
      reportedProvider = new ReportedProviderEntry();
      reportedProvider.addedTime = performance.now();
    } else {
      reportedProvider = reportedProviderValue;
    }
    reportedProvider.disconnections += disconnections;
    reportedProvider.errors += errors;
    this.addedToPurgeAndReport.set(address, reportedProvider);
  }
}

class ReportedProviderEntry {
  public disconnections = 0;
  public errors = 0;
  public addedTime = 0;
}
