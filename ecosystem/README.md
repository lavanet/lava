## LavaVisor
LavaVisor serves as a service manager for the Lava protocol application binaries. Its duty is to manage protocol versioning and coordinate seamless transitions to updated protocol versions. 

When an upgrade becomes necessary, either because the current protocol version has dropped below the minimum version or not compliant with the recommended target version, LavaVisor’s responsibility begin. LavaVisor orchestrates the necessary upgrade process in an automated manner, ensuring that the protocol version is aligned with the current standards and capabilities set by the minimum and target versions.

## Setup
To set up LavaVisor, navigate to the `./lava/ecosystem/lavavisor/` directory and execute the `./scripts/build.sh` script. This will generate an executable binary located at `~/go/bin`.

## Usage

### Commands:
- **`lavavisor init`**: Initializes LavaVisor for operation.

  **Optional Flags**:
  - `--directory`
  - `--auto-download`
  - `--auto-start`

  **Example usage:**
  `lavavisor init --auto-download --auto-start``

Prepares the local environment for the operation of LavaVisor.
Here are the operations that `lavavisor init` performs:

1. Verifies the existence of the `./lavavisor` directory for LavaVisor operations. If absent, it establishes the directory structure and includes a sample `config.yml` file.
2. Sends a request to the Lava Network, retrieving the minimum and target protocol versions.
3. Using the acquired version, it searches for the corresponding protocol binary in `.lavavisor/upgrades/<version-tag>/`.
4. If the binary isn't found at the target location, it tries to download the binary from GitHub using the fetched version (note: the `auto-download` flag must be enabled).
5. Validates the fetched binary.
6. Verifies if a `lava-protocol` binary already exists in the system path. If not, it copies the fetched binary to the system path.
7. Establishes a symbolic link between the fetched binary in `.lavavisor/upgrades/<version-tag>/` and the protocol binary in the system path.

### Known Issue:
Some older versions of `lava-protocol` lack the `version` command, which LavaVisor employs to validate binary versions. When these older version binaries are downloaded using the auto-download option, LavaVisor will accept these binaries without signaling an error. However, when restarting LavaVisor with this binary, an error will be raised. This is because the binary validation will fail since the version command cannot be executed.

**Workaround**: 
- Use the latest version of the protocol.
- Set up the binaries manually.

- **`lavavisor start`**: Starts provider/consumer processes given with linked binary and starts lavavisor version monitor.

  **Optional Flags**:
  - `--directory`
  - `--auto-download`

  **Example usage:**
  `lavavisor start --auto-download`

This command reads the `config.yml` file to determine the list of services. Once identified, it starts each service using the linked binary. Concurrently, it also launches the LavaVisor version monitor. This monitor is designed to detect when an upgrade is necessary and will automatically carry out the upgrade process as soon as it's triggered.
Here are the operations that `lavavisor start` performs:

1. Ensures the `.lavavisor` directory is present.
2. Reads the `config.yml` to determine the list of services. Each service is then initiated using the binary linked by the `init` command.
3. Sets up state tracker parameters and registers its state tracker for protocol version updates.
4. Launches the version monitor as a separate go routine, which activates when the `updateTriggered` signal is set to true.
5. If an upgrade requirement is detected, the `updateTriggered` signal is enabled, prompting the version monitor to commence the auto-upgrade process.
6. Once the new binary is either retrieved from `.lavavisor/upgrades/<new-version-tag>/` or downloaded from GitHub (note: `auto-download` must be active), a new link to the binary is established, and the system daemon is restarted.
7. After rebooting the provider and consumer processes, the version monitor resumes its monitoring for potential upgrade events.

