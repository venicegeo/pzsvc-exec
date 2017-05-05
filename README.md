# pzsvc-exec

## Contents

- [Overview](#overview)
- [Development Environment](#development-environment)
- [Installing and Running](#installing-and-running)
- [Configuration File Definition](#configuration-file-definition) 
- [Service Endpoints](#service-endpoints)
- [Execute Endpoint Request Format](#execute-endpoint-request-format)
- [Incorporating Scalability](#incorporating-scalability)

## Overview

Pzsvc-exec is a microservice written in the Go programming langauge.  It's purpose is to bring algorithms and other non-scalable applications and web services to the enterprise.  It provides load balancing capabilities to enable algorithms and applications to be scalable in the Enterprise.

Ppzsvc-exec is designed to serve command-line programs to Piazza, based on the contents of a config file.  Piazza is an open-source framework and tool-suite enabling rapid geospatial information systems (GIS) solutions for the enterprise.   It is designed to do the heavy lifting needed by developers moving solutions to the cloud.  Piazza leverages pzsvc-exec so algorithm developers can have their algorithms deployed to the cloud so developers can discover and utilize these algorithms within the GIS solutions they develop.   For more details on Piazza, see https://pz-docs.geointservices.io/ for details.

When pzsvc-exec is launched, it reads from a configuration file, from which it derives all persistent information.  Based on settings in the configuration file, pzsvc-exec starts automatically registering itself as a service to a specified Piazza instance.  

When a request comes in, it has up to three parts - a set of files to download, a command string to execute, and a set of files to upload.  It will generate a temporary folder, download the files into the folder, execute its command in the folder, upload the files from the folder, reply to the service request, and then delete the folder.  The command it attempts to execute is the `CliCmd` parameter from the config file, with the `cmd` from the service request appended on.  The reply to the service request will take the form of a JSON string, and contains a list of the files downloaded, a list of the files uploaded, and the stdout return of command executed.

The idea of this meta-service is to simplify the task of launch and maintenance on Pz services.  If you have execute access to an algorithm or similar program, its meaningful inputs consist of files and a command-line call, and its meaningful outputs consist of files, stderr, and stdout, you can provide it as a Piazza service.  All you should have to do is fill out the config file properly (and have a Piazza instance to connect to) and pzsvc-exec will take care of the rest.

As a secondary benefit, pzsvc-exec will be kept current with the existing Piazza interface, meaning that it can serve as living example code for those of you who find its limitations overly constraining.  For those of you writing in Go, it even contains a library (pzsvc) built to handle interactions with Piazza.

Additionally, and associated, pzsvc-exec contains a secondary application of pzsvc-taskworker.  Pzvc-taskworker is designed to run off the same config file that pzsvc-exec does and coordinate with pzsvc-exec in such a way as to take advantage of the Piazza task manager functionality, offering improvements in things like security and scalability.  Pzsvc-taskworker is optional, like much of the functionality associated with pzsvc-exec, and will be described more in depth in its own section.

## Development Environment

Pzsvc-exec is written in the go programming language.  To develop capabilities in pzsvc-exec, do the following:

### 1. Install Go

Pzsvc-exec  is built using the Go, version 1.8.x. For details on installing Go, see https://golang.org/dl/.  Once Go is instaled, make sure the Go tool is on your path once the install is done.

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

**URL**: The URL for where pzsvc-exec is running.  This address is used during autoregistration with Piazza so the service can be discovered.  **Required**

**Port**: The port this service serves on.  If not defined, will default to `8080`.

**PortEnVar**: Environment variable, containing a port number. 
Either one of the `Port` parameters is required or else your user service will be default to port `8080`.  Specifying `PortEnVar` overwrites the value of `Port`.

**Description**: A simple text description of your service.   This is used as metadata when your service is registered with Piazza's user service registry.

**Attributes**: A block of freeform key/value pairs for you to set additional descriptive attributes.  This is primarily intended to aid communication between services and service consumers with respect to the details of a service.  Information provided might be things like service type (so that the correct service consumers can identify you), interface (so they know how to interact with you) and image requirements (so they know what sorts of images to send you).

**NumProcs**: An integer vaue specifying the maximum number of simultaneous jobs to allow.  This will generally depend on the amount of data you are uploading and downloading, the overall computational load of the command you are executing, and the resources each instance has available to draw on.  If the service is crashing regularly from overload, you want to drop this number.  If it is running at low load but processing too slowly, you'll want to increase it.  Defaults to no thread control, allowing all jobs to run as they arrive.

**CanUpload: Boolean**: A boolean indicating if data results should be loaded after your service processes a request.  Defaults to false.

**CanDownlPz**:  A boolean indicating whether data can downloaded by Piazza before processsing. Defaults to false.

**CanDownlExt**: A boolean indicating whether external downloads can be done before processing.  Defaults to false.

**RegForTaskMgr**: A boolean indicating whether your service should run as a task managed service.  This is the Piazza feature that taskworker was designed to take advantage of.  For more details on how task management works, see the [Piazza the  Guide](https://pz-docs.stage.geointservices.io/userguide/index.html#_task_management) . **Required for Task Managed Service**

**MaxRunTime**: An integer which is used when registering for task manager.  Indicates how long Piazza should wait after a job has been taken before assuming that the process has failed.  **Required for Task Managed Service**

**LocalOnly**: A boolean indicating whether pzsvc-exec should accept connections externally or locally.  Specifying `true` means your service will only accept calls from requests coming from the same host.  Intended as an additional security measure when using a local taskworker.  **Required**

**LogAudit**: A boolean indicating whether pzsvc-exec should produce audit logs.

**LimitUserData**: A boolean indicating whether logs should be obfuscated to prevent reverse engineering and determining the details of how your service works. 

**ExtRetryOn202:** Boolean.  If true, pzsvc-exec will respond to HTTP Code 202 responses on external file downloads by waiting a minute and trying again, for up to an hour.  This is offered as a way to enable dealings with systems like Planetlabs, where files must be activated before they are made available.

**DocURL**: A string specifying a URL to provide to autoregistration and to the /documentation REST endpoint.  This URL should point to some sort of online documentation about your service instance.

## Service Endpoints

The pzsvc-exec service endpoints are as follows:

- '/': The entry point.  Displays the 'CliCmd' parameter, if any, and suggests other endpoints.
- '/execute': The meat of the program.  Downloads files, executes on them, and uploads the results.
See the Execute Endpoint Request Format section of this Readme for interface details.
- '/description': When enabled, provides a description of this particular pzsvc-exec instance.
- '/documentation': When enabled, provides a url containing documentation for this particular pzsvc-exec instance.
- '/attributes': When enabled, provides a list of key/value attributes for this pzsvc-exec instance.
- '/version': When enabled, provides version number for the application served by this pzsvc-exec instance.
- '/help': Provides the Service Endpoint data available here.

## Execute Endpoint Request Format

The intended use of the '/execute' endpoint is through the Piazza service, but it can also be used as a standalone service.  It currently requires POST calls.  Parameters should be as a json-formatted body. Format as follows:


Input Format:
```
cmd           string    // Command string - appended to CliCmd from the config file
userID        string    // Unique ID of initiating user - used for logging purposes
inPzFiles     []string  // Pz dataIds for files to download before processing
inExtFiles    []string  // URLs for external files to download before processing
inPzNames     []string  // Parallel to InPzFiles - name for each file
inExtNames    []string  // Parallel to inExtFiles - name for each file
outTiffs      []string  // Filenames of GeoTIFFs to ingest after processing
outTxts       []string  // Filenames of text files to ingest after processing
outGeoJson    []string  // Filenames of GeoJSON files to ingest after processing
inExtAuthKey  string    // Auth key for accessing external files
pzAuthKey     string    // Auth key for accessing Piazza
pzAddr        string    // URL for the targeted Pz instance
```

As an example (fully functional as an input to pzsvc-ossim, other than the auth key):
```
{
"cmd":"shoreline --image coastal.TIF,swir1.TIF --projection geo-scaled shoreline.json",
"inExtFiles":["https://landsat-pds.s3.amazonaws.com/L8/090/089/LC80900892015290LGN00/LC80900892015290LGN00_B1.TIF","https://landsat-pds.s3.amazonaws.com/L8/090/089/LC80900892015290LGN00/LC80900892015290LGN00_B6.TIF"],
"inExtNames":["coastal.TIF","swir1.TIF"],
"outGeoJson":["shoreline.json"],
"pzAuthKey":"******"
"inExtAuthKey":"******"
}
```

## Incorporating Scalability

Pzsvc-taskworker exists inside of the pzsvc-taskworker subfolder of the pzsvc-exec folder, and can be installed via `go get` or `go install` appropriately.  it can be run with `GOPATH/bin/pzsvc-taskworker <configuration file>`. It should be called using the same config file as was used for the instance of pzsvc-exec it has been paired with.

For details on how task management works, see the [Piazza User's Guide](https://pz-docs.stage.geointservices.io/userguide/index.html#_task_management).

Pzsvc-taskworker is designed to connect to the task manager feature of Piazza.  In that system, rather than passing jobs through to a service directly, Piazza stores them in a queue, and waits for a service to request jobs to complete.  This can be useful for scalability and security purposes.  It allows arbitrary scalability while letting each pzsvc-exec/pzsvc-taskworker pair control the number of jobs they're running at a time.

Currently, pszvc-taskworker is designed to work with a colocated copy of pzsvc-exec.  It is possible to use it to work with a colocated copy of some other REST service, but that requires additional effort and is not supported.  Support for that feature may be expanded in the future if there is enough demand.  

A copy of pzsvc-taskworker requires only the config file it is called on and the environment variables specified by same.  No other input is necessary
