# LavaVisor
LavaVisor serves as a service manager for the Lava protocol application binaries. Its duty is to manage protocol versioning and coordinate seamless transitions to updated protocol versions. 

When an upgrade becomes necessary, either because the current protocol version has dropped below the minimum version or not compliant with the recommended target version, LavaVisor’s responsibility begin. LavaVisor orchestrates the necessary upgrade process in an automated manner, ensuring that the protocol version is aligned with the current standards and capabilities set by the minimum and target versions.

## Setup
**`make install-all`** → Creates lavavisor binary (under ~/go/bin/)

## Prerequisites
Go version above than 1.19

## Usage

### Commands:
1- **`lavavisor init`**: Initializes LavaVisor for operation.

  **Optional Flags**:
  - `--directory`
  - `--auto-download`
  - `--auto-start`

  **Example usage:**
  `lavavisor init --auto-download`

Prepares the local environment for the operation of LavaVisor.
Here are the operations that `lavavisor init` performs:

1. Verifies the existence of the `./lavavisor` directory for LavaVisor operations. If absent, it establishes the directory structure.
2. Sends a request to the Lava Network, retrieving the minimum and target protocol versions.
3. Using the acquired version, it searches for the corresponding protocol binary in `.lavavisor/upgrades/<version-tag>/`.
4. If the binary isn't found at the target location, it tries to download the binary from GitHub using the fetched version (note: the `auto-download` flag must be enabled).
5. Validates the fetched binary.
6. Verifies if a `lavap` binary already exists in the system path. If not, it copies the fetched binary to the system path.
7. Establishes a symbolic link between the fetched binary in `.lavavisor/upgrades/<version-tag>/` and the protocol binary in the system path.


**For a fully automated experience, usage of `--auto-download` flag is suggested**

### Known Issue:
Some older versions of `lavap` lack the `version` command, which LavaVisor employs to validate binary versions. When these older version binaries are downloaded using the auto-download option, LavaVisor will accept these binaries without signaling an error. However, when restarting LavaVisor with this binary, an error will be raised. This is because the binary validation will fail since the version command cannot be executed.

**Workaround**: 
- Use the latest version of the protocol.
- Set up the binaries manually.
___

2- **`lavavisor create-service`**: Creates system files according to given consumer / provider config file and configuration flags.

  **Arguments**:
  `[service-type: "provider" or "consumer"]`
  `[service-config-folder]`

  **Required Flags**:
  - `--geolocation`
  - `--from`

  **Optional Flags**:
  - `--create-link`
  - `--log-level`
  - `--directory`

  **Example usage:**
  `lavavisor create-service consumer /home/ubuntu/config/consumer-ETH1.yml --geolocation 1 --from user1 --log_level info --keyring-backend test --chain-id lava --node http://127.0.0.1:26657 --create-link`

  This command generates the service file in '.lavavisor/' directory so that consumer & provider processes can be started in a robust manner. Additionally, it creates a `./logs` directory, so that service logs can be easily accessed. Finally, it updates the `config.yml` file by appending the name of the service created by this command, allowing the created process to be read by the lavavisor start command.

 **Use the `--create-link` flag to create a symbolic link to the `/etc/systemd/system/` directory; it is highly recommended for ensuring service files are set up properly.**
___
 3- **`lavavisor start`**: Starts provider/consumer processes given with linked binary and starts lavavisor version monitor.

  **Optional Flags**:
  - `--auto-download`
  - `--directory`

  **Example usage:**
  `lavavisor start --auto-download`

This command reads the `config.yml` file to determine the list of services. Once identified, it starts each service using the linked binary. Concurrently, it also launches the LavaVisor version monitor. This monitor is designed to detect when an upgrade is necessary and will automatically carry out the upgrade process as soon as it's triggered.
Here are the operations that `lavavisor start` performs:

**Usage of `--auto-download` flag is recommended for auto-fetching of upgraded binary** (target binary must be exist in GitHub releases).
1. Ensures the `.lavavisor` directory is present.
2. Reads the `config.yml` to determine the list of services. Each service is then initiated using the binary linked by the `init` command.
3. Sets up state tracker parameters and registers its state tracker for protocol version updates.
4. Launches the version monitor as a separate go routine, which activates when the `updateTriggered` signal is set to true.
5. If an upgrade requirement is detected, the `updateTriggered` signal is enabled, prompting the version monitor to commence the auto-upgrade process.
6. Once the new binary is either retrieved from `.lavavisor/upgrades/<new-version-tag>/` or downloaded from GitHub (note: `auto-download` must be active), a new link to the binary is established, and the system daemon is restarted.
7. After rebooting the provider and consumer processes, the version monitor resumes its monitoring for potential upgrade events.


# Test

1. Run `lavavisor init --auto-download` → This will setup LavaVisor directory and link the protocol binary
2. Instead of creating service files manually, execute `lavavisor create-service` command to generate a the service files. 
3. Validate `.lavavisor/services` folder created and with generated service files.
4. Validate `config.yml` is updated and includes all of the target service names.
5. Execute `lavavisor start --auto-start`, and you should observe all services running. Additionally, the version monitor will begin validating versions.
6. Now we need to make an upgrade proposal by using `/gov` module, so that protocol version will change in the consensus and LavaVisor will detect & initiate auto-upgrade.

Here is an example `proposal.json` file:

```json
{
    "title": "Protocol Version Change",
    "description": "Update version",
    "changes": [
        {
            "subspace": "protocol",
            "key": "Version",
            "value": {
                "provider_target": "0.23.2",
                "consumer_target": "0.23.2",
                "provider_min": "0.22.0",
                "consumer_min": "0.22.0"
            }
        }
    ],
    "deposit": "10000000ulava"
}
```
Here is the script for sending version update proposal transaction (for Cosmos SDK v0.47.0):
```bash
#!/bin/bash
# upgrade script (upgrade_chain.sh)

# function to wait for next block (should be used when proposing on chains with Cosmos SDK 0.47 or higher)
function wait_next_block {
  current=$( lavad q block | jq .block.header.height)
  echo "Waiting for next block $current"
  while true; do
    sleep 0.5
    new=$( lavad q block | jq .block.header.height)
    if [[ $current != $new ]]
    then
      echo "finished waiting for block $new"
        break
    fi
  done
}

# The software upgrade proposal command. This only proposes a software upgrade. To apply the upgrade, you need to vote "yes" (like below).
lavad tx gov submit-legacy-proposal param-change proposal.json \
--gas "auto" \
--from alice \
--keyring-backend "test" \
--gas-prices "0.000000001ulava" \
--gas-adjustment "1.5" \
--yes

wait_next_block

# The vote command. Use vote ID 4 (like here) if you used the init_chain_commands.sh script. If the vote doesn't work because of a bad
# vote ID, use the 'lavad q gov proposals' query to check the latest proposal ID, and put here the latest ID + 1.
lavad tx gov vote 4 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices "0.000000001ulava"
```

(Fix proposal ID 4 according to your state - if you didn’t run ‘init_chain_commands’ you should put 1 there)

7. After the proposal passed, LavaVisor will detect the event and update the binaries. Then, it will reboot the processes with the new established symbolic link: