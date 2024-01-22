# Bade generator

The purpose of this service is to generate and provide a badge, of which consumers can use when communication with providers.

## Configuration

The badge server uses a config file named `badgeserver.yml` that provides the projects data and some other basic configuration.
An example for a configuration file:

```yaml
projects-data:
	2:
		myWeb3Project:
			epochs-max-cu: 100000

default-geolocation: 2

countries-file-path: ""
ip-file-path: ""
```

`projects-data` - hold a map of maps of geolocation to project to project's data. In this example, there is only one project, under geolocation `2` , named `myWeb3Project`.
`epochs-max-cu` - defines the max CU that can be used for this project per epoch.
`default-geolocation` - the default geolocation to use, when one cannot be determined by the badge server.
`countries-file-path` - path of the countries file
`ip-file-path` - path of the IP file

### Countries File

This is a CSV file with all countries and lava-geolocation link for example.
It contains four columns: country-code, country-name, continent code and lava-geolocation.
Example:

```
AD;Andorra;EU;2
AE;United Arab Emirates;AS;2
AF;Afghanistan;AS;2
AL;Albania;EU;2
AM;Armenia;AS;2
AO;Angola;AF;2
```

### IP File

This is a TSV file with all IP ranges and country code that they belong to.
Tt can be downloaded from [this](https://iptoasn.com/) website, under `ip2asn-v4.tsv`.
It contains 5 columns: range_start, range_end, AS_number, country_code and AS_description.
Example:

```
1.0.0.0	1.0.0.255	13335	US	CLOUDFLARENET
1.0.1.0	1.0.3.255	0	None	Not routed
1.0.4.0	1.0.5.255	38803	AU	WPL-AS-AP Wirefreebroadband Pty Ltd
1.0.6.0	1.0.7.255	38803	AU	WPL-AS-AP Wirefreebroadband Pty Ltd
1.0.8.0	1.0.15.255	0	None	Not routed
```

## Running the `badgeserver`

To run the badge server, run this command:

```
lavad badgeserver [config-path] --port=[port] --log_level=[log level] --chain-id=[chain id]
```

Where `config-path` is the directory which contains the `badgeserver.yml` file.

## Generating a Badge

To generate a badge, use the RPC endpoint `lavanet.lava.pairing.BadgeGenerator/GenerateBadge`.
The input for this endpoint is:
`badge_adress` - the client wallet address that uses the badge. This address must match to the transactions signer address.
`project_id` - the project ID to use.
`spec_id` - the spec to generate the badge for.

The response of this request is:
`badge` - and object, holding the following data:

- `cu_allocation`
- `epoch`
- `address`
- `lava_chain_id`
- `project_sig`
- `virtual_epoch`

`get_pairing_response` - the response from a pairing query, made automatically by the badge server on badge generation.
`badge_signer_address` - the public address of the badge signer.
`spec` - the full spec of the requested `spec_id`
