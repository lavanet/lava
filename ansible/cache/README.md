# Cache Service Deployment

This `cache` directory contains Ansible playbooks and templates for deploying the configuration to set up a service based on the `lava` cache, using Docker and Docker Compose.

## Requirements

- **Ansible 2.9** or higher.
- **Docker** installed on the host machine.
- **Docker Compose** installed on the host machine.
- **Access** to the host machine as a user with sudo privileges.

## Setup

1. **Clone the repository**:
   Ensure that you have cloned this repository to your local machine or to the server where you want to run the playbook.

2. **Configure Hosts**:
   Update the `hosts` file in the `inventory` directory to match the target deployment environment's IP address or hostname.

3. **Set Variables**:
   Customize variables in `group_vars/all.yml` and `host_vars/<hostname>.yml` to match your deployment settings. These settings include the Docker image, network configurations, and resource limits.

## Deployment

The deployment process involves setting directly configuring Docker networks, generating necessary configuration claims, and managing Docker containers through Docker Compose.

> Note that by default ```anisble-playbook main.yml``` command deploys and runs the service.

### Deploying the Cache Service

To deploy the cache service, run the following command:

```bash
ansible-playbook main.yml --tags deploy
```

This command executes the role that sets up directories, configures Docker Compose, and ensures that the network is ready for the service to run.

## Managing
The managing process involves such operations as: starting the service, stopping and restarting. They runs using corresponding tags.

### Starting the Service
To start the cache service if it is not already running, use the following command:

```bash
ansible-playbook main.yml --tags start
```

### Stopping the Service
To stop the cache service, execute:

```bash
ansible-playbook main.yml --tags stop
```

This command will stop the Docker container without removing the configuration, allowing you to restart it later without reconfiguration.

### Restarting the Service
If you need to restart the cache service, you can use:

```bash
ansible-playbook main.yml --tags restart
```

## Configuration Files

* Docker Compose Configuration: Located at `{{ project_path }}/docker-compose.yml`, it defines the service setup, including image, ports, and environment variables.
* Environment Variables: Stored in `{{ project_path }}/cache.env`, this file includes environment-specific variables like log level and cache expiration.
