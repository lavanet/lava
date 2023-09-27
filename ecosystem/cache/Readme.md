# Lava Protocol Cache Service 

Lava's caching service is used to cut costs and improve the overall performance of your application, both provider and consumer processes benefit from the caching service.

In order to use the caching service all you need to do is run the following process:


```bash
ListenAddress="127.0.0.1:7777"
ListenMetricsAddress="127.0.0.1:5747"
lavap cache $ListenAddress --metrics_address $ListenMetricsAddress --log_level debug
```

now the cache service is running in the background. all you need to do is plug in the caching service with the consumer / provider process like so:

```bash
lavap rpcprovider <your-regular-cli-options> --cache-be $ListenAddress
```
