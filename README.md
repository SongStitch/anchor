<div align="center">

# Anchor

![Go Version](https://img.shields.io/github/go-mod/go-version/SongStitch/anchor?style=flat-square&logo=go)
![Docker](https://img.shields.io/badge/Docker-%232496ED.svg?logo=docker&logoColor=white&style=flat-square)
[![CI status](https://img.shields.io/github/actions/workflow/status/songstitch/anchor/ci.yaml?branch=main&style=flat-square&logo=github)](https://github.com/SongStitch/anchor/actions?query=branch%3Amain)
[![License](https://img.shields.io/github/license/SongStitch/anchor?style=flat-square)](/LICENSE)
[![Release](https://img.shields.io/github/v/release/SongStitch/anchor?style=flat-square)](https://github.com/SongStitch/anchor/releases/latest)

A tool for anchoring dependencies in dockerfiles

</div>

<!-- toc -->

- [Installation](#installation)
  - [Via Go Install](#via-go-install)
  - [Via GitHub Releases](#via-github-releases)
- [What is Anchor, and How Does it Work?](#what-is-anchor-and-how-does-it-work)
  - [By Example](#by-example)
- [Supported Operating Systems Package Managers](#supported-operating-systems-package-managers)
- [Recommended Workflow](#recommended-workflow)
- [Usage](#usage)
  - [Default Usage](#default-usage)
  - [Specifying Input and Output Files](#specifying-input-and-output-files)
  - [Non-Interactive Mode (CI/CD Pipelines)](#non-interactive-mode-cicd-pipelines)
  - [Printing the Output Instead of Writing to a File](#printing-the-output-instead-of-writing-to-a-file)
- [License](#license)

<!-- tocstop -->

# Installation

## Via Go Install

```shell
go install github.com/songstitch/anchor@latest
```

## Via GitHub Releases

Download the latest binary from the [releases page](https://github.com/SongStitch/anchor/releases/latest)

# What is Anchor, and How Does it Work?

Anchor is a tool for anchoring Dockerfiles (not unlike pinning in lock files). It allows for reproducible builds by ensuring that the versions of dependencies are fixed. This is done in two ways

- Replacing docker image tags referenced in a Dockerfile with the digest of the image
- Replacing package versions in a Dockerfile with the version of the package. The parent digest image is used resolve the package versions to ensure that the package versions are consistent with the parent image.

Anchor is designed that with the generated `Dockerfile`, no changes are needed on one's CI or build process.

Note that `docker` must be installed and running on the system for `anchor` to work.

## By Example

Given this `Dockerfile`

```dockerfile
# Comments are preserved
FROM golang:1.22-bookworm as builder

RUN apt-get update \
    && apt-get install --no-install-recommends -y curl wget \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
```

Running `anchor` will generate the following `Dockerfile`

```dockerfile
# Comments are preserved
FROM golang:1.22-bookworm@sha256:5c56bd47228dd572d8a82971cf1f946cd8bb1862a8ec6dc9f3d387cc94136976 as builder

RUN apt-get update \
    && dpkg --add-architecture arm64 && apt-get update && \
    apt-get install --no-install-recommends -y curl:arm64=7.88.1-10+deb12u5 wget:arm64=1.21.3-1+b1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
```

# Supported Operating Systems Package Managers

Currently, Anchor only supports the `apt` package manager. Support for other OS package managers is planned.

# Recommended Workflow

The recommended workflow for using `anchor` is as follows:

- Name your Dockerfile `Dockerfile.template`
- Run `anchor` in the same directory as the `Dockerfile.template`
- Commit the generated `Dockerfile` to your repository
- Use the generated `Dockerfile` in your CI/CD pipeline to ensure repoducible builds
- Do not modify the generated `Dockerfile` manually
- If you need to make changes to the Dockerfile, make them in the `Dockerfile.template` and run `anchor` again
- If you need to update the dependencies, run `anchor` again

# Usage

## Default Usage

Running `anchor` without any flags will use the default input and output files. It looks for a file named `Dockerfile.template` in the current directory and outputs the result to `Dockerfile`.

```shell
anchor
```

## Specifying Input and Output Files

You can specify the input and output files using the `-i` and `-o` flags respectively.

```shell
anchor -i Dockerfile.template -o Dockerfile
```

## Non-Interactive Mode (CI/CD Pipelines)

You can use the `--yes` flag to automatically accept the changes made by `anchor`. This is useful for CI/CD pipelines.

```shell
anchor -i Dockerfile.template -o Dockerfile --yes
```

Without the `--yes` flag, `anchor` will prompt you to accept any overwrites.

## Printing the Output Instead of Writing to a File

You can print the output to stdout by using the `-p` flag.

```shell
anchor -i Dockerfile.template --dry-run
```

# License

This project is licensed under the GPL-2.0 License - see the [LICENSE](/LICENSE) file for details.
