# Lava Provider Helm Chart

Running a lava provider on Kubernetes is a 2 step process.

1. Run the provider using the Helm chart
2. Create a staking transaction for the provider

## 1. Run the provider using the Helm chart

#### Helm Command

```bash
helm install myprovider \
    oci://us-central1-docker.pkg.dev/lavanet-public/charts/lava-provider \
    --values myvalues.yaml
```

#### Prerequisite

Before running the provider we need to create a kubernetes secret with your account key (`lavad keys export ...`) and the password.
Please see the [secret.example.yaml](secret.example.yaml) file for an example.

#### Required Config
To run the Helm chart you need to provider some `REQUIRED` values. Anything marked at required in the [values.yaml](provider/values.yaml) file is required.

Please see [values.example.yaml](values.example.yaml) file for an example for what values you will need to provide.

Please see our [docs](https://docs.lavanet.xyz/provider-setup) for more information on configuration. 

The `configYaml` section in the [values.example.yaml](values.example.yaml) file has an example config for a provider that works on `LAV1`


## 2. Create a staking transaction for the provider

Once you deploy the provider into kubernetes using the helm file, and the deployment is marked as ready and running by kubernetes you can stake your provider.

```bash
lavad tx pairing stake-provider [chain-id] [amount] [endpoint endpoint ...] [geolocation] [flags]
```

*Check the output for the status of the staking operation. A successful operation will have a code **`0`**.*

#### Parameters Description

- **`chain-id`** - The ID of the serviced chain (e.g., **`COS4`** or **`FTM250`**).
- **`amount`** - Stake amount for the specific chain (e.g., **`2010ulava`**).
- **`endpoint`** - Provider host listener, composed of `provider-host:provider-port,geolocation`.
- **`geolocation`** - Indicates the geographical location where the process is located (e.g., **`1`** for US or **`2`** for EU).

#### Geolocations

```javascript    
    USC = 1; // US-Center
    EU = 2; // Europe
    USE = 4; // US-East
    USW = 8; // US-West
    AF = 16; // Africa
    AS = 32; // Asia
    AU = 64;  // (Australia, includes NZ)
    GL = 65535; // Global
```


#### Flags Details

- **`--from`** - The account to be used for the provider staking (e.g., **`my_account`**).
- **`--provider-moniker`** - Provider’s public name
- **`--keyring-backend`** - A keyring-backend of your choosing (e.g., **`test`**).
- **`--chain-id`** - The chain_id of the network (e.g., **`lava-testnet-2`**).
- **`--gas`** - The gas limit for the transaction (e.g., **`"auto"`**).
- **`--gas-adjustment`** - The gas adjustment factor (e.g., **`"1.5"`**).
- **`--node`** - A RPC node for Lava (e.g., **`https://public-rpc-testnet2.lavanet.xyz:443/rpc/`**).


For full docs on [how to stake as a provider](https://docs.lavanet.xyz/provider-setup#step-2-stake-as-provider) please see our docs: https://docs.lavanet.xyz/provider-setup#step-2-stake-as-provider

## Resources

- Provider setup docs: https://docs.lavanet.xyz/provider-setup#step-2-stake-as-provider
