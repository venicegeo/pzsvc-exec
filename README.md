# pzsvc-exec

## Table of Contents

- [Overview](#overview)
- Installing and Running: You need this if you want to run/administrate a pzsvc-exec instance
- Config File Format: You need this if you want to have any control over the instance you're running.  May be useful for understanding how pzsvc-exec works.
- Service Endpoints: a listing of the service endpoints that pzsvc-exec makes available and what they are useful for.
- Execute Endpoint Request Format: You need this if you want to make use of the /execute endpoint of pzsvc-exec with any control at all.
- Pzsvc-Taskworker: What you need to know to use the associated pzsvc-taskworker app (and, by extension, to take advantage of the Piazza task queue for scalability)

## Overview

Pzsvc-exec is a microservice written in the Go programming langauge.  It's purpose is to bring algorithms and other non-scalable applications and web services to the enterprise.  It provides load balancing capabilities to enable algorithms and applications to be scalable in the Enterprise.

"pzsvc-exec" is designed to serve command-line programs to Piazza, based on the contents of a config file.

Pzsvc-exec is at its most basic level something that publishes the exec() call as a service to Piazza (with lots of useful bells and whistles).

When it is launched, it is given a config file, from which it derives all persistent information.  If the config file allows, it will start by automatically registering itself as a service to a specified Piazza instance.  Regardless, it will then begin to serve.

When a request comes in, it has up to three parts - a set of files to download, a command string to execute, and a set of files to upload.  It will generate a temporary folder, download the files into the folder, execute its command in the folder, upload the files from the folder, reply to the service request, and then delete the folder.  The command it attempts to execute is the `CliCmd` parameter from the config file, with the `cmd` from the service request appended on.  The reply to the service request will take the form of a JSON string, and contains a list of the files downloaded, a list of the files uploaded, and the stdout return of command executed.

The idea of this meta-service is to simplify the task of launch and maintenance on Pz services.  If you have execute access to an algorithm or similar program, its meaningful inputs consist of files and a command-line call, and its meaningful outputs consist of files, stderr, and stdout, you can provide it as a Piazza service.  All you should have to do is fill out the config file properly (and have a Piazza instance to connect to) and pzsvc-exec will take care of the rest.

As a secondary benefit, pzsvc-exec will be kept current with the existing Piazza interface, meaning that it can serve as living example code for those of you who find its limitations overly constraining.  For those of you writing in Go, it even contains a library (pzsvc) built to handle interactions with Piazza.

Additionally, and associated, pzsvc-exec contains a secondary application of pzsvc-taskworker.  Pzvc-taskworker is designed to run off the same config file that pzsvc-exec does and coordinate with pzsvc-exec in such a way as to take advantage of the Piazza task manager functionality, offering improvements in things like security and scalability.  Pzsvc-taskworker is optional, like much of the functionality associated with pzsvc-exec, and will be described more in depth in its own section.

## Installing and Running

Make sure you have Go installed on you machine, and an appropriate GOPATH (environment variable) set.

Use `go get` to install the latest version of both the CLI and the library.
	`$ go get -v github.com/venicegeo/pzsvc-exec/...`

To install:
	`$ go install github.com/venicegeo/pzsvc-exec/...`

Alternate install:
	navigate to `GOPATH/src/github.com/venicegeo/pzsvc-exec`
	then call `$ go install .`

To Run:
	`GOPATH/bin/pzsvc-exec <configfile.txt>`, where <configfile.txt> represents the path to an appropriately formatted config file, indicating what command line function to use, and where to find Piazza for registration.  Additionally, make sure that whatever application you wish to access is in path.

## Config File Format

The example config file in this directory includes all pertinent potential entries, and may be used as an example, though some entries are left as 0/false/"".  Some entries are redundant with one another or mutually exclusive.  In cases like that, there is no behavioral difference between, for example, setting PzAddr to the empty string or leaving that entry out altogether.  Additional entries are meaningless but nonharmful, as long as standard JSON format is maintained.  No entries are strictly speaking mandatory, but leaving them out will often disable one or more pieces of of the pzsvc-exec functionality.

CliCmd: The initial parameters to feed to the exec call.  For security reasons, you are strongly encouraged to define this entry as something other than whitespace or the empty string, thus limiting your service to a single application.  If you do not, you are essentially offering open command-line access on the serving computer to anyone capable of calling your service.  Should be spaced normally, as if entering into the command line directly

VersionStr: Version of the software pointed to, in the form of a string.  Added to the service data in autoregistration and to the file metadata for uploaded files.  The version string is also available through the "/version" endpoint.  Redundant with VersionCmd.

VersionCmd: as with versionStr, except that this is a command line call which expects the version string as a return.  Reloads fresh each time pzsvc-exec is called.  Redundant with VersionStr.

PzAddr: For use with a Piazza instance.  This is the base https address of the chosen Piazza instance.  It is useful for file upload and Pz file download.  It is necessary for autoregistration and taskworker.

PzAddrEnVar: Environment variable, containing the piazza address.  When defined and non-empty, overwrites PzAddr.  Intended for cases where, for example, multiple domains exist using the same set of seeds.

APIKeyEnVar: The name of the environment variable that will contain your Piazza API key.  It is useful for file upload and Pz file download.  It is necessary for autoregistration and taskworker.

SvcName: This is the name by which the service will identify itself.  Maintaining SvcName uniqueness among your services is important, as it will be used to determine on execution whether a service is being launched for the first time, or whether it is a continuation of a previous service.  Maintaining SvcName uniqueness in general is not as critical, as identity of launching user will also be used as a component.  It is added to file metadata for uploaded files, and is necessary for autoregistration.

URL: This is the URL that the service will be served on.  It is necessary for autoregistration when not registering as a task manager service.

Port: The port this service serves on.  If not defined, will default to 8080.

PortEnVar: Environment variable, containing a port number.  When defined and non-empty, overwrites Port.  Intended for systems where the buid/push process may call for an arbitrary port.

Description: A simple text description of your service.  Used in registration, and also available through the "/description" endpoint.

Attributes: A block of freeform key/value pairs for you to set additional descriptive attributes.  Used in registration, and also available through the "/attributes" endpoint.  Primarily intended to aid communication between services and service consumers with respect to the details of a service.  Information provided might be things like service type (so that the correct service consumers can identify you), interface (so they know how to interact with you) and image requirements (so they know what sorts of images to send you).

NumProcs: Integer.  The maximum number of simultaneous jobs to allow.  This will generally depend on the amount of data you are uploading and downloading, the overall computational load of the command you are executing, and the resources each instance has available to draw on.  If the service is crashing regularly from overload, you want to drop this number.  If it is running at low load but processing too slowly, you'll want to increase it.  Defaults to no thread control, allowing all jobs to run as they arrive.

CanUpload: Boolean.  If false, does not allow uploads after processing.  Defaults to false.

CanDownlPz: Boolean.  If false, does not allow Piazza downloads before processing.  Defaults to false.

CanDownlExt: Boolean.  If false, does not allow external downloads before processing.  Defaults to false.

RegForTaskMgr: Boolean.  If true, registers as a Task Manager service.  This is the Pz feature that taskworker was designed to take advantage of.  Required for taskworker. 

MaxRunTime: int.  Only applicable when registering for task manager.  Indicates how long Pz should wait after a job has been taken before assuming that the process has failed.

LocalOnly: Boolean.  If true, this pzsvc-exec instance will only accept connections from localhost.  Intended as an additional security measure when using a local takworker

LogAudit: Boolean.  If true, pzsvc-exec will produce audit logs.  If false, does not produce audit logs.  Audit logs are useful as an added security feature, if you can manage them properly, but add significant bulk to the log outputs.

LimitUserData: Boolean.  If true, this instance reduces the overall amount of data it returns to the user significantly, causing any attempts at meaningful debugging to require access to the logs.  This is intended to serve as something of a security measure - reducing the ability of an external attacker to discover details of how the system works.

ExtRetryOn202: Boolean.  If true, pzsvc-exec will respond to HTTP Code 202 responses on external file downloads by waiting a minute and trying again, for up to an hour.  This is offered as a way to enable dealings with systems like Planetlabs, where files must be activated before they are made available.

DocURL: string.  Specifies a URL to provide to autoregistration and to the /documentation REST endpoint.  This URL shoudl point to some sort of online documentation about this pzsvc-exec instance.

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

## Pzsvc-taskworker

Pzsvc-taskworker exists inside of the pzsvc-taskworker subfolder of the pzsvc-exec folder, and can be installed via `go get` or `go install` appropriately.  it can be run with `GOPATH/bin/pzsvc-taskworker <configfile.txt>`. It should be called using the same config file as was used for the instance of pzsvc-exec it has been paired with.

Pzsvc-taskworker is designed to connect to the task manager feature of Piazza.  In that system, rather than passing jobs through to a service directly, Piazza stores them in a queue, and waits for a service to request jobs to complete.  This can be useful for scalability and security purposes.  It allows arbitrary scalability while letting each pzsvc-exec/pzsvc-taskworker pair control the number of jobs they're running at a time.

Currently, pszvc-taskworker is designed to work with a colocated copy of pzsvc-exec.  It is possible to use it to work with a colocated copy of some other REST service, but that requires additional effort and is not supported.  Support for that feature may be expanded in the future if there is enough demand.  

A copy of pzsvc-taskworker requires only the config file it is called on and the environment variables specified by same.  No other input is necessary
