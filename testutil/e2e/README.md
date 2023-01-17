# Lava E2E

Run using the following command

```
go test ./testutil/e2e/ -v -timeout 1200s
```

This E2E performs the steps below to test if the system is working as expected.

1. Start lava in developer mode (equivalent to running the command "ignite chain serve" ).
2. Check if lava is done booting up by sending a GRPC request.
3. Send Spec proposals and stake providers and clients.
4. Check if the proposals and stakes are properly executed.
5. Start the JSONRPC Proxy.
6. Start the JSONRPC Provider.
7. Start the JSONRPC Gateway.
8. Check if the JSONRPC Gateway is working as expected by sending a query through the gateway.
9. Start the Tendermint Provider using the running lava process as RPC.
10. Start the Tenderming Gateway.
11. Check if the Tendermint Gateway is working as expected by sending a query through the gateway.
12. Start the REST Provider using the running lava process as RPC.
13. Start the REST Provider.
14. Check if the REST Gateway is working as expected by sending a query through the gateway.
15. Send multiple requests through each gateway.
16. Check if a gateway responds with an error.
17. Check if payments are paid.

After the steps above are finished (even if a step fails and the E2E ends) the E2E will save all the captured logs.

# Allowed Error List

The allowed error list contains a list of errors that is allowed to happen during tests. The key is the error substring that can be seen in the logs. The value is the description on why this error is allowed. 