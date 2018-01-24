# Piazza Service Executor

## Contents

- [Overview](#overview)
- [Development Environment](#development-environment)
- [Installing and Running](#installing-and-running)
- [Configuration File Definition](#configuration-file-definition) 
- [Service Endpoints](#service-endpoints)
- [Execute Endpoint Request Format](#execute-endpoint-request-format)
- [Incorporating Scalability](#incorporating-scalability)

## Overview

Piazza Service Executor is a microservice written in the Go programming langauge whose purpose is to bring algorithms and other non-scalable applications to the enterprise by using Piazza and Cloud Foundry load balancing capabilities to enable algorithms and applications to be scalable in the Enterprise.

The Service Executor is designed to serve command-line programs to Piazza, based on the contents of a config file.  Piazza is an open-source framework and tool-suite enabling rapid geospatial information systems (GIS) solutions for the enterprise.  It is designed to do the heavy lifting needed by developers moving solutions to the cloud.  Piazza leverages this Service Executor so algorithm developers can have their algorithms deployed to the cloud so developers can discover and utilize these algorithms within the GIS solutions they develop. 

When the Service Executor is launched, it reads from a configuration file, from which it derives all persistent information (in addition to some information from Environment Variables).  Based on settings in the configuration file and the environment, the Service Executor starts automatically registering itself as a Service to a specified Piazza instance.  

Service Executor has two main components: the Dispatcher and the Worker.  Service Executor's Dispatcher component will begin polling Piazza, on startup, for work that it has received for its particular Service. When a Service Job request comes in, it has up to three parts - a set of files to download, a command string to execute, and a set of files to upload.  The Dispatcher component will use Cloud Foundry Tasks in order to spin up a new container that will run the Service Executor's Worker component. This Worker component will read the input Job request and execute the CLI algorithm in this Task Container, and send any results or status updates back to Piazza where the requesting user can read the status of the algorithm, and fetch results. 

## Development Environment

Pzsvc-exec is written in the go programming language.  To develop capabilities in pzsvc-exec, do the following:

### 1. Install Go

Pzsvc-exec is built using the Go, version 1.8.x. For details on installing Go, see https://golang.org/dl/.  Once Go is instaled, make sure the Go tool is on your path once the install is done.

### 2. Set up Go environment variables

Before developing using Go, certain environment variables must set. To see all the relevant environment variables, run the *go env* command. Below is a list of key Go environment variables:

- `GOROOT` - Should be set to point to the base directory at which Go is installed
- `GOPATH` - Should be set to point to a directory that is to serve as your development environment. This is where this code and dependencies will live.
- `GOBIN` - Should be set to point to a directory where the executable will live.  If not set, this defaults to the $GOPATH/bin directory.

### 3. Clone the Pzsvc-exec repository

To clone the pzsvc-exec repository, do the following commands

1. `$ mkdir -p $GOPATH/src/github.com/venicegeo`

2. `$ cd $GOPATH/src/github.com/venicegeo`

3. `$ git clone git@github.com:venicegeo/pzsvc-exec.git`

## Installing and Running

Before installing pzsvc-exec, make sure you have Go installed on you machine, and the environment variables set.

To __*install*__ pzsvc-exec, do the following:
	`$ go install github.com/venicegeo/pzsvc-exec/`

Alternate install:
	navigate to `GOPATH/src/github.com/venicegeo/pzsvc-exec`
	then call `$ go install .`

To __*run*__ pzsvc-exec, do the following:
	`$GOBIN/pzsvc-exec <configuration file>`
	
 where `<configuration file>` represents the path to an appropriately formatted configuration file, indicating what command line function to use and the information to register with Piazza.  Additionally, when running pzsvc-exec, make sure that whatever application you wish to access is in path.

## Configuration File Definition

An example configuration file, `examplecfg.txt` is located in the root directory of this repository.  Below is a list of the parameters that should be specified within your configuration file.  

**CliCmd**: The command line to execute when called.  This should include any parameters that are necessary for running the algoirthm.  **Required**

**VersionStr**: The version of the software pointed to, in the form of a string.  If provided, this is added as metadata about the service when registered with Piazza.  

**VersionCmd**: The command line command to use to obtain the version of the algorithm.

**PzAddr**: The URL for where pzsvc-exec is running.  This address is used during autoregistration with Piazza so the service can be discovered. 

**PzAddrEnVar**: Environment variable, containing the piazza address.  

Either one of the `PzAddr` parameters is required.  Specifying `PzAddrEnVar` overwrites the value of `PzAddr`.

**APIKeyEnVar**: The name of the environment variable that will contain your Piazza API key.  When using Piazza, an API key is necessary.  For details see on obtaining the key and using Piazza, see the [Piazza User's Guide](https://pz-docs.stage.geointservices.io/userguide/index.html).  **Required**

**SvcName**: A unique name of the service.  The name specified is used to register your web service so it can be discovered.  This name is also added as metadata for any loaded files.   **Required**

**Description**: A simple text description of your service.   This is used as metadata when your service is registered with Piazza's user service registry.

**Attributes**: A block of freeform key/value pairs for you to set additional descriptive attributes.  This is primarily intended to aid communication between services and service consumers with respect to the details of a service.  Information provided might be things like service type (so that the correct service consumers can identify you), interface (so they know how to interact with you) and image requirements (so they know what sorts of images to send you).

**CanUpload: Boolean**: A boolean indicating if data results should be loaded after your service processes a request.  Defaults to false.

**CanDownlPz**:  A boolean indicating whether data can downloaded by Piazza before processsing. Defaults to false.

**CanDownlExt**: A boolean indicating whether external downloads can be done before processing.  Defaults to false.

**MaxRunTime**: An integer which is used when registering for task manager.  Indicates how long Piazza should wait after a job has been taken before assuming that the process has failed.  **Required for Task Managed Service**

**LogAudit**: A boolean indicating whether pzsvc-exec should produce audit logs.

## Environment Variables

In addition to the config, certain environment variables are required. The `CF_API`, `CF_USER`, and `CF_PASS` variables are required in order to spin up the Cloud Foundry Task container. 

Additionally, the `TASK_LIMIT` environment variable can be used to tune the number of simultaneous Cloud Foundry Tasks that the Dispatcher will be allowed to create.  By default this value is 5. The number of Cloud Foundry Task containers is limited only by the available resources in a CF organization, so it is recommended to supply a realistic limit for this value, depending on your organization. 

