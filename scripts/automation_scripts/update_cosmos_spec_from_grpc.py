import argparse
import json
import os
import re


def parse_endpoints_from_spec(lava_spec_file: str) -> dict[str, list[str]]:
    print("### Parsing endpoints from current spec file")

    endpoints: dict[str, list[str]] = {"grpc": [], "rest": []}

    with open(lava_spec_file, "r", encoding="utf-8") as f:
        content = json.load(f)

    for api_collection in (
        content.get("proposal", {}).get("specs", [])[0].get("api_collections", [])
    ):
        interface_type = api_collection.get("collection_data", {}).get(
            "api_interface", ""
        )

        if interface_type not in endpoints:
            continue

        for api in api_collection.get("apis", []):
            api_name = api.get("name", "")
            if not (
                "cosmos" in api_name or "cosmwasm" in api_name or "/ibc/" in api_name
            ):
                endpoints[interface_type].append(api_name)

    print(f"    ### {len(endpoints['grpc'])} gRPC endpoints")
    print(f"    ### {len(endpoints['rest'])} Rest endpoints")

    return endpoints


def parse_endpoints_from_grpcurl(grpc_url: str) -> dict[str, list[str]]:
    print("### Parsing endpoints from gRPC service")

    endpoints: dict[str, list[str]] = {"grpc": [], "rest": []}
    content = os.popen(f"grpcurl -plaintext {grpc_url} describe").read()

    # Regex pattern to find services starting with their corresponding rpc and rest paths
    grpc_pattern = re.compile(
        r"(\S+) is a service:(.*?)(?=^\S+ is a service:|\Z)",
        re.DOTALL | re.MULTILINE,
    )

    rpc_pattern = re.compile(r"rpc (\w+) \(")
    rest_pattern = re.compile(r'option\s*\(\s*.*?\s*\)\s*=\s*{\s*get:\s*"(.*?)"\s*}', re.DOTALL)

    # Finding all services that start with 'lavanet'
    for service_match in grpc_pattern.finditer(content):
        service_name, service_content = service_match.groups()

        if (
            "cosmos" in service_content
            or "cosmwasm" in service_content
            or "ibc" in service_content
            or "grpc.reflection.v1alpha" in service_content
        ):
            continue

        # Extracting all grpc paths
        for rpc_match in rpc_pattern.finditer(service_content):
            rpc_method = rpc_match.group(1)
            endpoints["grpc"].append(f"{service_name}/{rpc_method}")

        # Extracting all rest paths
        for rest_match in rest_pattern.finditer(service_content):
            rest_path = rest_match.group(1)
            endpoints["rest"].append(rest_path)

    print(f"    ### {len(endpoints['grpc'])} gRPC endpoints")
    print(f"    ### {len(endpoints['rest'])} Rest endpoints")

    return endpoints


def update_spec_file(
    lava_spec_file: str,
    grpc_endpoints_to_update: set[str],
    rest_endpoints_to_update: set[str],
):
    print("### Updating current spec file")

    with open(lava_spec_file, "r", encoding="utf-8") as f:
        content = json.load(f)

    for api_collection in content["proposal"]["specs"][0]["api_collections"]:
        interface_type = api_collection["collection_data"]["api_interface"]

        if interface_type not in ["grpc", "rest"]:
            continue

        endpoints_to_update = (
            grpc_endpoints_to_update
            if interface_type == "grpc"
            else rest_endpoints_to_update
        )

        api_collection["apis"].extend(
            create_api_entry(endpoint) for endpoint in endpoints_to_update
        )

    with open(lava_spec_file, "w", encoding="utf-8") as f:
        json.dump(content, f, indent=4)


def create_api_entry(endpoint: str) -> dict:
    return {
        "name": endpoint,
        "block_parsing": {"parser_arg": ["latest"], "parser_func": "DEFAULT"},
        "compute_units": 10,
        "enabled": True,
        "category": {
            "deterministic": True,
            "local": False,
            "subscription": False,
            "stateful": 0,
        },
        "extra_compute_units": 0,
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Update Cosmos-based spec file")
    parser.add_argument("spec_file", help="Path to the spec file to update")
    parser.add_argument("grpc_url", help="GRPC URL to fetch the endpoints from")
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Perform a dry run without making any changes.",
    )
    args = parser.parse_args()
    try:
        grpcurl_endpoints = parse_endpoints_from_grpcurl(args.grpc_url)
        spec_endpoints = parse_endpoints_from_spec(args.spec_file)

        grpc_endpoints_to_update = set(grpcurl_endpoints["grpc"]) - set(
            spec_endpoints["grpc"]
        )
        rest_endpoints_to_update = set(grpcurl_endpoints["rest"]) - set(
            spec_endpoints["rest"]
        )

        if args.dry_run:
            if grpc_endpoints_to_update or rest_endpoints_to_update:
                print("### There are endpoints to update in Lava's spec file!")
                exit(1)
        else:
            update_spec_file(
                args.spec_file, grpc_endpoints_to_update, rest_endpoints_to_update
            )

        print("### Done")

    except FileNotFoundError:
        print(f"Error: File not found - {args.lava_spec_file}")
        exit(1)
    except json.JSONDecodeError:
        print(f"Error: Invalid JSON in the file - {args.lava_spec_file}")
        exit(1)
