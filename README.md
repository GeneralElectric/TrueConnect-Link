# TrueConnect-Link
## Overview
This is the TrueConnect client for deployment on remote servers to facilitate the upload of files to TrueConnect.
The client has several modes of operation allowing it to meet the need of several scenarios.

### Single File Upload
In this mode the client can be called via command line or from a script in a similar way to using curl. The benefit over
using curl is that the standard metadata fields such as file size, sha256 hash, original file name are populated
automatically. Also the fact the file has been uploaded is logged in such a way that duplicate uploads can be prevented.

### Onetime Collection
In this mode the client will search a configured locations for files that match a particular patterns, collect any
additional metadata contained in the file path, the configuration can contain multiple enabled and disabled target
directories to search, optionally these targets can be specified by name when invoking this mode, in which case only
the named targets will be searched regardless of whether the target is enabled or not. Once all the matching files have
been uploaded the process will exit with a zero exit code or 1 if there were errors.

### As a Service
This mode is similar to [onetime collection](#onetime-collection), but with searches
restricted to enabled targets at a configured interval.


## Configuration
The configuration for the client is a json file, each user will have their own configuration file containing
the targets they want searched. The configuration specifying which targets are to be searched can be overridden at
the command line when the client is executed

[Configuration Schema](./docs/LinkConfigurationSchema.json)

[Configuration Example](./aviation_trueconnect-tenancytest3_dev.json)

## Logging
The client logs the status of operations such as file uploads in a structured csv file allowing for the log to be parsed
programmatically to asserting the upload status. The headings for this log file are as following:

[Date Time],[System Name],[Operation],[Status],[Context],[Comments]

**Date Time:**
This is the time at which the log was made. ISO RFC3339 format

**System Name:**
This is the name of the application carrying out the operation. Such as TrueConnect-Link

**Operation:**
The operation being performed for example File Upload

**Status:**
This indicates the condition of the operation, e.g. Started, Failed, Success, Finished, etc.

**Context:**
This is an id that uniquely identifies this operation compared to others of the same type, this is useful to match up
 start and stop status updated for the same operation.
 
**Comments:**
This is used to contain additional information

## Installation

You can download the source via git or from the [releases](./releases), compile this with Go version 1.8.3+

Place the executable file along side a configuration file in a folder where it may be executed.

## Execution
```
        	USAGE:

                Trueconnectlink -c:<command> [arguments]

            The commands are:

                Upload		used to upload a single file
                Start		searches the configured targets, uploads all matching files and exits
                Auto		for each configured target: uploads all matching files and waits the configured period and repeats
                            after the configured interval
                Test		Performs a test of network connectivity to TrueConnect and validates all local configurations
                            against allowed permissions on target tenants

            Upload:
                Trueconnectlink -c:Upload -u:<user> -p:<secret> -tenant:<tenant> -datatype:<datatype> -dataformat:<dataformat>
                                -path:<path> -e:<endpoint> -tokurl:<tokenurl>

                Args:
                    user 		The UAA clientID used to connect to TrueConnect
                    secret		The UAA secret for the associated clientID
                    tenant		The tenant that the upload data will belong to
                    datatype	Sets the datatype meta data tag associated with this file
                    dataformat	Sets the data format meta data tag associated with this file
                    path		Set the full path of the file to upload
                    e	        Sets the url TrueConnect service where the file is to be uploaded
                    tokurl	    Sets the OAuth2 service URL from where a bearer token needed for TrueConnect may be obtained

            Start:
                Trueconnectlink -c:Start -u:<user> [-t:<Target>]

                Args:
                    user 		The UAA clientID used to connect to TrueConnect (The configuration file will be
                                called <user>.json and found in the same folder as the executable)
                    target		OPTIONAL, MULTIPLE, Names which targets will be searched in the config. If none are named only
                                those configured as active will be searched. If targets are named, only they will be searched
                                regardless of whether they are configured as active or not.
            Auto:
                Trueconnectlink -c:Auto -u:<user>

                Args:
                    user 		The UAA clientID used to connect to TrueConnect (The configuration file will be
                                called <user>.json and found in the same folder as the executable)
            Test:
                Trueconnectlink -c:Test
```
