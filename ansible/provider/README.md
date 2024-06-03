# RPC Provider Service Deployment

This repository includes Ansible playbooks and supporting files for deploying and managing the RPC Provider Service. The service uses Docker containers managed via Docker Compose to ensure easy and scalable deployments across multiple environments.

## Prerequisites

- **Ansible 2.9+**: Ensure Ansible is installed on your control machine.
- **Docker**: Must be installed on the target hosts.
- **Docker Compose**: Required for managing Dockerized applications.
- **SSH Access**: Root or sudo access on the target hosts.

## Repository Structure

- **`group_vars`** and **`host_vars`**: Contains variables specific to hosts and groups. Customize these to fit the deployment context.
- **`roles`**: Contains the tasks used for setting up the RPC provider.
- **`templates`**: Jinja2 templates for generating Docker Compose and environment configuration files.
- **`inventory`**: Hosts file defining the servers on which the RPC service will be deployed.

## Installation and Setup

### Clone the Repository

Start by cloning this repository to your Ansible control machine:

```bash
git clone <repository-url>
cd <repository-dir>
```

### Configure Inventory

Edit the `inventory/hosts` file to add the IP addresses or hostnames of the machines where the service should be deployed.

Example:

```yaml
all:
  children:
    lava_provider_eu:
      hosts:
        192.168.1.100:
          ansible_user: root
          ansible_ssh_private_key_file: ~/.ssh/id_rsa
```

> You can declare certain parameters in the `ansible.cfg` configuration file or in `group_vars/all.yml`. These settings will be applied to all hosts at each startup

ansible.cfg
```ini
[defaults]
private_key_file = ~/.ssh/id_rsa
```

group_vars/all.yml

```yml
# group_vars/all.yml
---
ansible_user: root
ansible_port: 22
```

### Set Role Variables
Adjust role-specific variables in `group_vars/all.yml` and `host_vars/*.yml` to match your environment:

## Deployment
The deployment process involves setting directly configuring Docker networks, generating necessary configuration claims, and managing Docker containers through Docker Compose.

### Deploy the Service
To deploy the RPC Provider Service:

```bash
ansible-playbook main.yml --tags deploy
```

> Note that by default ```anisble-playbook main.yml``` command deploys and runs the service.

## Managing

Start the Service: Ensure the service is up and running:

```bash
ansible-playbook main.yml --tags start
```

Stop the Service: Safely stop the service when needed:

```bash
ansible-playbook main.yml --tags stop
```

Restart the Service: Restart the service to apply updates or changes:

```bash
ansible-playbook main.yml --tags restart
```

## Configuration Files

* Docker Compose Configuration: Located at `{{ project_path }}/docker-compose.yml`, it defines the service setup, including image, ports, and environment variables.
* Environment Variables: Stored in `{{ project_path }}/provider.env`, this file includes environment-specific variables like log level and cache expiration.
* Chains configuration: Stored in `{{ volume_path }}`. This config includes specified chain settings for the Lava RPC Provider.