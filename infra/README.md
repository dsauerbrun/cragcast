# infra

This directory contains the infrastructure automation for cragcast.

## Setup

This automation relies on the following tools, also described in [shell.nix](../shell.nix):

* [Pulumi](https://github.com/pulumi/pulumi)
    * [pulumi-aws](https://github.com/pulumi/pulumi-aws)
    * [aws-cli](https://github.com/aws/aws-cli)

By default, Pulumi will use configuration (and credentials) in the AWS CLI. Run `aws configure` to set this up.

By default, this configuration will assume an SSH key named `cragcast` exists in your target AWS account.

## Usage

1. Run `pulumi up`.
1. Wait for infrastructure to bootstrap.
1. Using the exported `public-ip`, you may SSH in:
    ```
    ssh -i ~/.ssh/cragcast.pem root@<public-ip>
    ```
1. When done, run `pulumi destroy`.

