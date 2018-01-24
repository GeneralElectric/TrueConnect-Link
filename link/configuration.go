package link

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

// Commands
const (
	UploadCommand    = "Upload"
	StartCommand     = "Start"
	AutoCommand      = "Auto"
	HelpCommand      = "help"
	TestCommand      = "Test"
	SetConfigCommand = "SetConfig"
	GetConfigCommand = "GetConfig"
	Usage            = `TrueConnect-Link v1.0.1 
https://github.com/GeneralElectric/TrueConnect-Link
Use this tool to upload data to TrueConnect
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
        			e			Sets the url TrueConnect service where the file is to be uploaded
        			tokurl		Sets the OAuth2 service URL from where a bearer token needed for TrueConnect may be obtained

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
	`
)

// Configuration is the root element of the configuration for the TrueConnect client each client_id (user) must
// use there own configuration. This configuration controls what files are looked for and to which tenant they should be uploaded
// against
type Configuration struct {
	// when set to true the client will run as a service, this is overridden when the client is executed with the
	// command AUTO
	RunAsService bool `json:"runasservice"`

	// This is the client id, this is also present in the file name (user name)
	ClientID string `json:"clientid"`

	// password
	Secret string `json:"secret"`

	// The URL for the OAUTH =2 service that provides the bearer token
	TokenURL string `json:"tokenurl"`

	// The TrueConnect endpoint url to connect to
	Endpoint string `json:"endpoint"`

	// The collection of targets to search for files to upload
	Targets []Target `json:"targets"`

	// The maximum number of concurrent uploads allowed
	ConcurrentUploads int `json:"concurrentuploads"`

	// The mode of execution, set via command line argument
	command string
}

// Target is the configuration used to specify a location to search and what data to find there
type Target struct {
	// The name of the target, this is allow the target to be referred to by name from the command line
	Name string `json:"name"`

	// This value set whether the target will be searched whilst in Auto mode
	Active bool `json:"active,bool"`

	// The tenant to assign any files uploaded from this target location
	Tenant string `json:"tenant"`

	// The path to search for files in
	Location string `json:"location"`

	// When set the location directory and all sub directories will be set
	Recursive bool `json:"recursive"`

	// A regular expression used to describe which file to include for upload
	Match string `json:"match"`

	// The data type of the file that match the regular expression in this target location
	DataType string `json:"datatype"`

	// The data format of the file that match the regular expression in this target location
	DataFormat string `json:"dataformat"`

	// A list of metadata tag that can be extracted from the file path
	PathEncodedMetaDataTags []PathEncodedMetaDataTag `json:"pathencodedmetadatatags"`

	// a list of static meta data values
	StaticTags []StaticMetaData `json:"statictags"`

	// The number of seconds between the time the last file found was uploaded and the next time we should check for
	// files that haven't been uploaded yet
	PollInterval int `json:"pollinterval"`

	// Command or script to be run on successful upload of the file, the full path of the file uploaded will be added to the
	// command as the first argument after the command wrapped in double quoates. The file storage reference will be
	// added as the second argument.
	OnSuccess string `json:"onsuccess"`
}

// PathEncodedMetaDataTag configuration used to describe a metadata tag whose value can be found in the file path
type PathEncodedMetaDataTag struct {
	// the name of the metadata tag
	Tag string `json:"tag"`

	// The regular expression used to define a group that will contain the value of the metadata tag
	Match string `json:"match"`
}

// StaticMetaData that will accompany all uploads on associated target
type StaticMetaData struct {
	// the name of the metadata tag
	Tag string `json:"tag"`

	// the vale to assign this tag
	Value string `json:"value"`
}

func (linkClient *linkClient) loadConfigWithTargets() error {
	var empty struct{}
	// this map is used to store the list of targets passed in on command line, a map was used rather than a slice
	// because this collection will be search for a matching name for each target contained in the configuration
	// file, so i wanted to used the map as it had the most efficient search algorithm
	targetNames := make(map[string]struct{})
	if len(linkClient.configuration.Targets) > 0 {
		for _, target := range linkClient.configuration.Targets {
			targetNames[target.Name] = empty
		}
	}
	err := linkClient.loadConfig(linkClient.configuration.ClientID + ".json")
	if err != nil {
		return err
	}

	if len(targetNames) > 0 {
		var targets []Target
		for _, target := range linkClient.configuration.Targets {
			if _, ok := targetNames[target.Name]; ok {
				target.Active = true
				targets = append(targets, target)
			}
		}

		linkClient.configuration.Targets = targets
	}

	return nil
}

func (linkClient *linkClient) loadConfig(configURI string) error {

	var config Configuration
	b, err := ioutil.ReadFile(configURI)

	if err == nil {
		err = json.Unmarshal(b, &config)
		if err == nil {
			linkClient.configuration = config
		}
	}

	return err
}

func (configuration *Configuration) getConfigurationFromArgs(args []string) {

	for _, arg := range args {
		if arg == HelpCommand {
			configuration.command = HelpCommand
			return
		}

		if strings.HasPrefix(arg, "-c:") {
			configuration.command = arg[3:]
			if configuration.command == AutoCommand {
				configuration.RunAsService = true
			}
			if configuration.command == UploadCommand {
				configuration.Targets = append(configuration.Targets, Target{Name: "SingleFileUpload"})
			}
		}
		if strings.HasPrefix(arg, "-p:") {
			configuration.Secret = arg[3:]
		}
		if strings.HasPrefix(arg, "-u:") {
			configuration.ClientID = arg[3:]
		}
		if strings.HasPrefix(arg, "-t:") &&
			(configuration.command == StartCommand ||
				configuration.command == AutoCommand ||
				configuration.command == SetConfigCommand ||
				configuration.command == GetConfigCommand) {
			configuration.Targets = append(configuration.Targets, Target{Name: arg[3:]})
		}
		if strings.HasPrefix(arg, "-tenant:") {
			configuration.Targets[0].Tenant = arg[8:]
		}
		if strings.HasPrefix(arg, "-datatype:") {
			configuration.Targets[0].DataType = arg[10:]
		}
		if strings.HasPrefix(arg, "-dataformat:") {
			configuration.Targets[0].DataFormat = arg[12:]
		}
		if strings.HasPrefix(arg, "-path:") {
			configuration.Targets[0].Location = arg[6:]
		}
		if strings.HasPrefix(arg, "-e:") {
			configuration.Endpoint = arg[3:]
		}
		if strings.HasPrefix(arg, "-tokurl:") {
			configuration.TokenURL = arg[8:]
		}
	}
}

func (target *Target) getCommand(filePath string, storageRef string) string {
	command := strings.Replace(target.OnSuccess, "$file", filePath, -1)
	return strings.Replace(command, "$storageref", storageRef, -1)
}
