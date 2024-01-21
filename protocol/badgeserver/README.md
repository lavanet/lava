# Bade generator

The purpose of this service is to generate and provide a badge, of which consumers can use when communication with providers.

## Configuration

Run the command

```
lavad badgeserver [config_file] --port=8080 --log_level=debug  --chain-id=lava  --grpc-url=127.0.0.1:9090
```

---

## Env Variables explained

1.  BADGE_DEFAULT_GEOLOCATION
    > this is really important because if for some reason we don't find which country the users is calling from than we use the default value.
        this value should be on the BADGE_USER_DATA json.
2.  BADGE_COUNTRIES_FILE_PATH

    > this is a csv file with all countries and lava-geolocation link for example.
    > It contains four colums country-code;country-name,continent code,lava-geolocation  
    > for example:

    ```
    AD;Andorra;EU;2
    AE;United Arab Emirates;AS;2
    AF;Afghanistan;AS;2
    AL;Albania;EU;2
    AM;Armenia;AS;2
    AO;Angola;AF;2
    ```

3.  BADGE_IP_FILE_PATH
    > this is a tsv file with all ip ranges and country code that they belong.
    > it can be downloaded from here [ip](https://iptoasn.com/) pls download ip2asn-v4.tsv
    > It contains 4/5 colums range_start;range_end;AS_number;country_code AS_description  
    > for example:
    ```
    1.0.0.0 1.0.0.255	13335	US	CLOUDFLARENET
    1.0.1.0	1.0.3.255	0	None	Not routed
    1.0.4.0	1.0.5.255	38803	AU	WPL-AS-AP Wirefreebroadband Pty Ltd
    1.0.6.0	1.0.7.255	38803	AU	WPL-AS-AP Wirefreebroadband Pty Ltd
    1.0.8.0	1.0.15.255	0	None	Not routed
    ```
4.  BADGE_USER_DATA
    > a json that link geolocation, public key of a project and private key to use for the encryption.
    ```
    {
      "1": {
        "default": {
          "project_public_key": "test111111",
          "private_key": "123456",
          "epochs_max_cu": 1
        },
        "projectId2": {
          "project_public_key": "test1122",
          "private_key": "12345678",
          "epochs_max_cu": 1
        }
      },
      "2": {
        "default": {
          "project_public_key": "test123",
          "private_key": "123456",
          "epochs_max_cu": 1
        }
      }
    }
    ```
