import { GenerateBadgeResponse } from "../grpc_web_services/pairing/badges_pb";
export declare function fetchBadge(serverAddress: string, badgeUser: string, projectKey: string): Promise<GenerateBadgeResponse>;
