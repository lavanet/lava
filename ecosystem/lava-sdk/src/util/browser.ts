// import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { NodeHttpTransport } from "./transportNode";
import { grpc } from "@improbable-eng/grpc-web";

let transport: grpc.TransportFactory;

if (typeof window !== "undefined") {
  // We are running in a browser
  transport = grpc.CrossBrowserHttpTransport({ withCredentials: false });
} else if (typeof process !== "undefined") {
  // We are running in Node.js
  transport = NodeHttpTransport();
  // transportAllowInsecure = NodeHttpTransport({ rejectUnauthorized: false });
} else {
  // If we are not running in the browser or node.js
  // We are running in a Web Worker
  // Assume the transport is same as for browser
  // Not tested
  transport = grpc.CrossBrowserHttpTransport({ withCredentials: false });
}

export default transport;
