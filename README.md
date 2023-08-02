# Terraform Provider for Pure Storage Fusion

[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/PureStorage-OpenConnect/terraform-provider-fusion?label=release&style=for-the-badge)](https://github.com/PureStorage-OpenConnect/terraform-provider-fusion/releases/latest) [![License](https://img.shields.io/github/license/PureStorage-OpenConnect/terraform-provider-fusion.svg?style=for-the-badge)](LICENSE)

The Terraform Provider for [Fusion][what-is-fusion] is a plugin for Terraform that allows you to interact with Fusion.  Fusion is a Pure Storage product that allows you to provision, manage, and consume enterprise storage with the simple on-demand provisioning, with effortless scale, and self-management of the cloud.  Read more about fusion [here][what-is-fusion]. This provider can be used to manage consumer oriented resources such as volumes, hosts, placement groups, tenant spaces.


Learn More:

* Read the provider [documentation][provider-documentation].
* Get help from [Pure Storage Customer Support][customer-support]

## Requirements

* [Terraform 0.15+][terraform-install]

    For general information about Terraform, visit [terraform.io][terraform-install] and [the project][terraform-github] on GitHub.

* Fusion credentials

In order to access the fusion API, you will need the required credentials and configuration.  Namely you will need the `api_host`, `private_key_file`, and `issuer_id`.  These need to be specified in the provider block in your terraform configuration.  Alternatively, these values can also be supplied using the environment variables: `FUSION_API_HOST`, `FUSION_PRIVATE_KEY_FILE` and `FUSION_ISSUER_ID`

## Using the provider

It is highly reccomended that you use the pre-built providers.  You should include a block in your terraform config like this:

    terraform {
      required_providers {
        fusion = {
          source  = "PureStorage-OpenConnect/fusion"
          version = "1.0.0"
        }
      }
    }

Then you should be able to just run `terraform init` and it should automatically install the right provider version.  Please check out examples from the [documentation][provider-documentation]  Note: The version number specified here is not the most up-to-date version, please refer to the [documentation][provider-documentation] for the latest version information.

## Getting support

Please don't hesitate to reach out to [Pure Storage Customer Support][customer-support].  If you are having trouble, please try to save and provide the terraform logs.  You can get those logs by setting the `TF_LOG`/`TF_LOG_PATH` envionment variables, for example:

    export TF_LOG=TRACE
    export TF_LOG_PATH=/tmp/terraform-logs
    terraform apply
    <....>
    gzip /tmp/terraform-log

Then the logs will be located at /tmp/terraform-log.gz

## Developing on the provider

Unit tests can be run like normal go tests, for example:
```
go test ./... -v
```

> :warning: **Caution! Acceptance tests may wipe your data or delete Fusion infrastructure!**

To run acceptance tests, you must have pre-created infrastructure:
  - Region
  - Availability Zone in this Region (All resources which depend on this AZ will be deleted)
  - At least one array in this Availability Zone
  
You also need to set environment variables.
  - `FUSION_API_HOST=http://your-fusion-control-plane:8080` This needs to be set to your controlplane endpoint
  - `FUSION_ISSUER_ID=pure1:apikey:abcdefghigjlkmnop` Set this to your fusion issuer ID
  - `FUSION_PRIVATE_KEY_FILE=/tmp/your-fusion-key.pem` Set this to the path of your fusion private key file
  - `FUSION_CONFIG=<path-to-your-fusion-config>` Set this to the path of your fusion config file
  - `TEST_EXISTING_REGION=<name-of-pre-created-az>` Set this to to the pre-created region name. By default `pure-us-west`
  - `TEST_EXISTING_AVAILABILITY_ZONE=<name-of-pre-created-az>` Set this to the pre-created availability zone name. By default `az1`

Acceptance test can be run with `go test` using `TF_ACC` environment variable: 
```
TF_ACC=1 go test ./... -v -timeout 0
```

[terraform-install]: https://www.terraform.io/downloads.html
[terraform-github]: https://github.com/hashicorp/terraform
[provider-documentation]: https://registry.terraform.io/providers/PureStorage-OpenConnect/fusion/latest/docs
[customer-support]: https://pure1.purestorage.com/support/cases
[what-is-fusion]: https://www.purestorage.com/enable/pure-fusion.html
