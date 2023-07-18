/// <reference types="node" />
import * as https from "https";
import { grpc } from "@improbable-eng/grpc-web";
export declare function NodeHttpTransport(httpsOptions?: https.RequestOptions): grpc.TransportFactory;
