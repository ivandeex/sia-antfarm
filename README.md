# sia-antfarm
[![Build Status](https://github.com/ivandeex/sia-antfarm/badges/master/pipeline.svg)](https://github.com/ivandeex/sia-antfarm/actions)
[![GoDoc](https://pkg.go.dev/go.sia.tech/siad?status.svg)](https://pkg.go.dev/go.sia.tech/siad)
[![License MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://img.shields.io/badge/License-MIT-brightgreen.svg)

sia-antfarm is a collection of utilities for performing complex, end-to-end
tests of the [Sia](https://sia.tech) platform. These tests are long-running
and offer superior insight into high-level network behaviour
than Sia's existing automated testing suite.

# Sia-AntFarm Docker image

sia-antfarm is also available as a docker image [ivandeex/sia-antfarm](https://hub.docker.com/r/ivandeex/sia-antfarm)

## Supported Tags

### Latest
* **latest**

### 1.2.0 (iva02)
* Sia AntFarm `v1.2.0` based on Sia `v1.5.7`

### 1.1.3
* Sia AntFarm `v1.1.3` with AntFarm stability updates and updated build test

### 1.1.1
* Sia AntFarm `v1.1.1` based on Sia `v1.5.5`

### 1.1.0
* Sia AntFarm `v1.1.0` based on Sia `v1.5.4`

### 1.0.5
* Allows publishing multiple ant HTTP API ports

### 1.0.4
* Sia Ant Farm `v1.0.4` based on Sia `v1.5.3`

### 1.0.3
* Sia Ant Farm `v1.0.3` based on Sia `v1.5.2`

### 1.0.2
* Sia Ant Farm `v1.0.2` based on Sia `v1.5.1`

### 1.0.1
* Sia Ant Farm `v1.0.1` based on Sia `v1.5.0`

## Running Ant Farm in Docker container

### Basic Usage
To start Ant Farm with default configuration
(`config/basic-renter-5-hosts-docker.json`) execute:
```
docker run \
    --publish 127.0.0.1:9980:9980 \
    nebulouslabs/siaantfarm
```
Port `127.0.0.1:9980` above is the renter API address which you can use to
issue commands to the renter. For security reasons you should bind the port to
localhost (see `127.0.0.1:9980` above).

### Custom Configuration
By default the Sia Ant Farm docker image has a copy of
`config/basic-renter-5-hosts-docker.json` configuration file.

If you want to execute Ant Farm with a custom configuration, create your custom
configuration e.g. `config/custom-config.json`, mount your config directory and
set `CONFIG` environment variable to your custom configuration by executing:
```
docker run \
    --publish 127.0.0.1:9980:9980 \
    --volume $(pwd)/config:/sia-antfarm/config \
    --env CONFIG=config/custom-config.json \
    nebulouslabs/siaantfarm
```

### Change Port
To change port on which you can access the renter (e.g. to 39980) execute:
```
docker run \
    --publish 127.0.0.1:39980:9980 \
    nebulouslabs/siaantfarm
```

### Open multiple ports
In default configuration only renter's HTTP API port is accessible from outside
of the docker container. If you want to configure access to more or all the
ants, you need to use custom configuration file (described above) and each
ant's HTTP API port needs to be set in 2 places:
* In the configuration file
* Pubished when starting docker container

#### Specify port in configuration file
`APIAddr` setting needs to be set in the configuration file same way as it is
set for renter ant in default configuration file
`config/basic-renter-5-hosts-docker.json`. Hostname part can only have one of
two values: `127.0.0.1` or `localhost`.

Example snippet:
```
        ...
		{
			"AllowHostLocalNetAddress": true,
            "APIAddr": "127.0.0.1:10980",
			"Name": "host1",
			"Jobs": [
				"host"
			],
			"DesiredCurrency": 100000
		},
        ...
```

#### Publish port when starting docker
Once you have prepared configuration file, you can start the container. You
need to set the path to custom configuration via `CONFIG` environment variable
and publish each port via `--publish` flag.

Example docker run command:
```
docker run \
    --publish 127.0.0.1:9980:9980 \
    --publish 127.0.0.1:10980:10980 \
    --volume $(pwd)/config:/sia-antfarm/config \
    --env CONFIG=config/custom-config.json \
    nebulouslabs/siaantfarm
```

### Persistent Ant Farm Data
There are several ways how to persist Ant Farm data. To store `antfarm-data` in
the current directory can be done the following way:
```
docker run \
    --publish 127.0.0.1:9980:9980 \
    --volume $(pwd):/sia-antfarm/data \
    nebulouslabs/siaantfarm
```

# AntFarm Requirements

## Generic
- Go installed. Sia Antfarm is tested extensively to run successfully on Go
  `1.15` on Linux and should be running well also on MacOS.
- `$GOPATH/bin` should be added to the `$PATH` so that built siad binary can be
  found and executed.
- On Linux Sia Antfarm requires `ss` command line utility to be installed.

## Version Test Requirements

Sia Antfarm is capable of building and testing different released and custom
versions of siad. Examples are in directories `foundation-test` (for Foundation
hardfork tests) and `version-test` (for basic version tests and for renter and
hosts upgrade tests).

Antfarm clones `Sia` repo if it doesn't already exist at
`$GOPATH/src/gitlab.com/NebulousLabs/Sia`. The local `Sia` repo should be in a
state that allows Antfarm to checkout different releases or custom branches,
i.e. all changes should be committed. If there are any uncommitted, unstashed
changes, they will be reset and lost.

# Install

To install release version of the Sia Antfarm which loads release constants
from Sia repository, execute:

```shell
go get -u gitlab.com/NebulousLabs/Sia-Ant-Farm/...
cd $GOPATH/src/gitlab.com/NebulousLabs/Sia-Ant-Farm
make dependencies && make
```

Sia Antfarm dev version loads dev constants from the Sia repository (it has
e.g. faster blocks) execute one of the dev targets below.

To install dev version of Sia Antfarm without debug logs enabled, execute:

```shell
make dependencies && make install-dev
```

To install dev version of Sia Antfarm with debug logs enabled, execute:

```shell
make dependencies && make install-dev-debug
```

If `siad` (at `$GOPATH/src/go.sia.tech/siad/cmd/siad`) is updated
and should be used with Antfarm, `make dependencies` needs to be rerun.

# Running a sia-antfarm

This repository contains one utility, `sia-antfarm`. `sia-antfarm` starts up
a number of `siad` instances, using jobs and configuration options parsed from
the input `config.json`. `sia-antfarm` takes a flag, `-config`, which is a path
to a JSON file defining the ants and their jobs. See `nebulous-configs/` for
some examples that we use to test Sia.

An example `config.json`:

`config.json:`
```json
{
	"antconfigs": 
	[ 
		{
			"jobs": [
				"gateway"
			]
		},
		{
			"jobs": [
				"gateway"
			]
		},
		{
			"jobs": [
				"gateway"
			]
		},
		{
			"jobs": [
				"gateway"
			]
		},
		{
			"apiaddr": "127.0.0.1:9980",
			"jobs": [
				"gateway",
				"miner"
			]
		}
	],
	"autoconnect": true
}
```

This `config.json` creates 5 ants, with four running the `gateway` job and one
running a `gateway` and a `miner` job.  If `HostAddr`, `APIAddr`, `RPCAddr`,
`SiamuxAddr`, or `SiamuxWsAddr` are not specified, they will be set to a random
port. If `autoconnect` is set to `false`, the ants will not automatically be
made peers of each other.

Note that if you have UPnP enabled on your router, the ants connect to each
other over the public Internet. If you do not have UPnP enabled on your router
and want the ants connect to each other over public Internet, you must
configure your system so that the ants' `RPCAddr` and `HostAddr` ports are
accessible from the Internet, i.e. to forward ports from your public IP. You
can run ant farm local IP range (then you do not need UPnP enabled router or
manual port forwarding) if you set `AllowHostLocalNetAddress` to `true`.

When you installed the Antfarm binary (see section Install) you can start the
Antfarm executing e.g. with one of our configs:

```shell
sia-antfarm -config nebulous-configs/basic-renter-host-5.json
```

or with debug logs on:

```shell
sia-antfarm-debug -config nebulous-configs/basic-renter-host-5.json
```
## Antfarm configuration options

```json
{
	'ListenAddress': 'localhost:9900' // string
	'AntConfigs': [
		<Ant Config 1>,
		<Ant Config 2>,
		...
	]
	'AutoConnect': true  // bool
	'ExternalFarms': [
		'localhost:9901' // string
		'localhost:9902' // string
		...
	]
	'WaitForSync': true  // bool
}
```

**ListenAddress**  
The listen address that the `sia-antfarm` API listens on.

**AntConfigs**  
An array of `AntConfig` objects, defining the ants to run on this antfarm. See
below.

**AutoConnect**  
A boolean which automatically bootstraps the antfarm if provided.

**ExternalFarms**  
An array of strings, where each string is the api address of an external
antfarm to connect to.

**WaitForSync**  
Wait with all non-mining jobs until the ASIC hardfork block height is reached
and all ants are in sync, defaults to false. This helps to prevent 2 known
issues happening around hardfork height:

* A renter contract is formed before hardfork, the block with the transaction
  is reverted, the transaction is being broadcasted after the hardfork height,
  but it has invalid signature. This issue might be fixed with upcoming utreexo
  changes.
* A block with host transaction is reverted and there is an issue: `wallet has
  coins spent in incomplete transactions - not enough remaining coins`. This
  issue is a known wallet issue which is fixed by itself after longer time
  period passes in the network.

## Ant configuration options

`AntConfig`s have the following options (with example values):
```json
{
	'APIAddr':                       'localhost:9980' // string
	'APIPassword':                   'a pass word'    // string
	'RPCAddr':                       'localhost:9981' // string
	'HostAddr':                      'localhost:9982' // string
	'SiamuxAddr':                    'localhost:9983' // string
	'SiamuxWsAddr':                  'localhost:9984' // string
	'AllowHostLocalNetAddress':      true             // bool
	'RenterDisableIPViolationCheck': true             // bool
	'SiaDirectory':                  'ant_0'          // string
	'SiadPath':                      'siad-dev'       // string
	'Name':                          'miner1'         // string
	'Jobs': [
		'gateway',                                    // string
		'miner',                                      // string
		...
	]
	'DesiredCurrency':               100000           // int
}
```

**APIAddr**  
The API address for the ant to listen on, by default an unused localhost bind
address will be used.

**APIPassword**  
The password to be used for authenticating certain calls to the ant.

**RPCAddr**  
The RPC address for the ant to listen on, by default an unused bind address
will be used.

**HostAddr**  
The Host address for the ant to listen on, by default an unused bind address
will be used.

**SiamuxAddr**  
The SiaMux address for the ant to listen on, by default an unused bind address
will be used.

**SiamuxWsAddr**  
The SiaMux websocket address for the ant to listen on, by default an unused
bind address will be used.

**AllowHostLocalNetAddress**  
If set to true allows hosts to announce on local network without Antfarm being
hosted on host with public IP, port forwarding from public IP to host or need
of UPnP enabled router with public IP.

**RenterDisableIPViolationCheck**  
Relevant only for renter, if set to true allows renter to rent on hosts on the
same IP subnets by disabling the `IPViolationCheck` for the renter.

**SiaDirectory**  
The data directory to use for this ant, by default a unique directory in
`./antfarm-data` will be generated and used.

**SiadPath**  
The path to the `siad` binary, by default the `siad-dev` in your path will be
used.

**Name**  
Human readable name of the ant.

**Jobs**  
An array of jobs for this ant to run. Available jobs include:
- `miner`
- `host`
- `noAllowanceRenter`
- `renter`
- `autoRenter`
- `gateway`

`noAllowanceRenter` job starts the renter and waits for renter wallet to be
filled.  
`renter` job starts the renter, sets default allowance and waits till the
renter is upload ready, it doesn't starts any renter's background activity.  
`autoRenter` does the same as 'renter' job and then starts renter's periodic
file uploads, downloads, and deletions.

**DesiredCurrency**  
A minimum amount (integer) of SiaCoins that this Ant will attempt to maintain
by mining currency. This is mutually exclusive with the `miner` job.

# License

The MIT License (MIT)
