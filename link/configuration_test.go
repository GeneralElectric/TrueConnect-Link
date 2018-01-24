package link

import (
	"os"
	"testing"
)

func TestLoadConfigWithTarget(tests *testing.T) {
	if testConfigSetUp() != nil {
		tests.Fatal("Could not create Test file")
	}
	client := linkClient{}
	args := []string{
		"-u:link.testconf",
		"-c:Start",
		"-t:EMUFLOW",
	}

	client.configuration.getConfigurationFromArgs(args)
	client.loadConfigWithTargets()

	if len(client.configuration.Targets) != 1 || client.configuration.Targets[0].Name != "EMUFLOW" || !client.configuration.Targets[0].Active {
		tests.Fatal(len(client.configuration.Targets) != 1, client.configuration.Targets[0].Name != "EMUFLOW", !client.configuration.Targets[0].Active)
	}
}

func TestUploadArgs(tests *testing.T) {

	args := []string{
		"-u:User1",
		"-c:Upload",
		"-p:p4ssw0rd",
		"-tenant:tenant1",
		"-datatype:datatype1",
		"-dataformat:dataformat1",
		"-path:boiled.egg",
		"-e:http://www.somesite.com/tc?action=upload",
		"-tokurl:https://some.real.long.url.that.looks.meaningless/stuff",
	}

	client := linkClient{}
	client.configuration.getConfigurationFromArgs(args)
	config := client.configuration

	if config.command != UploadCommand {
		tests.Fatal("command not set in upload")
	}
	if config.ClientID != "User1" {
		tests.Fatal("user name not set in upload")
	}
	if config.Secret != "p4ssw0rd" {
		tests.Fatal("password not set in upload")
	}
	if config.TokenURL != "https://some.real.long.url.that.looks.meaningless/stuff" {
		tests.Fatal("token URL not set in upload")
	}
	if config.Endpoint != "http://www.somesite.com/tc?action=upload" {
		tests.Fatal("endpoint not set in upload")
	}

	var found Target
	for _, target := range config.Targets {
		if target.Name == "SingleFileUpload" {
			found = target
		}
	}

	if found.Name != "SingleFileUpload" {
		tests.Fatal("target not set in upload")
	}

	if found.DataType != "datatype1" {
		tests.Fatal("Datatype not set in upload")
	}

	if found.DataFormat != "dataformat1" {
		tests.Fatal("DataFormat not set in upload")
	}

	if found.Tenant != "tenant1" {
		tests.Fatal("Tenant not set in upload")
	}

	if found.Location != "boiled.egg" {
		tests.Fatal("path not set in upload")
	}

}

func TestConfigArgs(tests *testing.T) {

	args := []string{
		"-u:User1",
		"-c:Config",
	}

	client := linkClient{}
	client.configuration.getConfigurationFromArgs(args)
	config := client.configuration

	if config.command != "Config" {
		tests.Fatal("command not set in Config")
	}
	if config.ClientID != "User1" {
		tests.Fatal("user name not set in Config")
	}

}

func TestStartArgs(tests *testing.T) {

	args := []string{
		"-u:User1",
		"-c:Start",
		"-t:target1",
	}

	client := linkClient{}
	client.configuration.getConfigurationFromArgs(args)
	config := client.configuration

	if config.command != "Start" {
		tests.Fatal("command not set in Start")
	}
	if config.ClientID != "User1" {
		tests.Fatal("user name not set in Start")
	}
	var found bool
	for _, target := range config.Targets {
		if target.Name == "target1" {
			found = true
		}
	}

	if !found {
		tests.Fatal("target not set in Set Start")
	}

}

func TestAutoArgs(tests *testing.T) {

	args := []string{
		"-u:User1",
		"-c:Auto",
	}

	client := linkClient{}
	client.configuration.getConfigurationFromArgs(args)
	config := client.configuration

	if config.command != "Auto" {
		tests.Fatal("command not set in Auto")
	}
	if config.ClientID != "User1" {
		tests.Fatal("user name not set in Auto")
	}

}

func TestSetConfigArgs(tests *testing.T) {

	args := []string{
		"-u:User1",
		"-c:SetConfig",
		"-t:Target1",
		"-t:Target2",
	}

	client := linkClient{}
	client.configuration.getConfigurationFromArgs(args)
	config := client.configuration

	if config.command != "SetConfig" {
		tests.Fatal("command not set in Set Config")
	}

	if config.ClientID != "User1" {
		tests.Fatal("user name not set in Set Config")
	}

	var found bool
	for _, target := range config.Targets {
		if target.Name == "Target1" {
			found = true
		}
	}

	if !found {
		tests.Fatal("target not set in Set Config")
	}

	var found2 bool
	for _, target := range config.Targets {
		if target.Name == "Target2" {
			found2 = true
		}
	}

	if !found2 {
		tests.Fatal("target not set in Set Config")
	}

}

func TestGetConfigArgs(tests *testing.T) {
	args := []string{
		"-u:User1",
		"-p:p4ssw0rd",
		"-c:GetConfig",
		"-t:Target1",
	}

	client := linkClient{}
	client.configuration.getConfigurationFromArgs(args)
	config := client.configuration

	if config.command != "GetConfig" {
		tests.Fatal("command not set in Get Config")
	}

	if config.ClientID != "User1" {
		tests.Fatal("user name not set in Get Config")
	}

	if config.Secret != "p4ssw0rd" {
		tests.Fatal("password not set in Get Config")
	}

	var found bool
	for _, target := range config.Targets {
		if target.Name == "Target1" {
			found = true
		}
	}

	if !found {
		tests.Fatal("target not set in Get Config")
	}

}

func TestLoadConfig(tests *testing.T) {

	if testConfigSetUp() != nil {
		tests.Fatal("Could not create Test file")
	}
	defer RemoveTestConfigFile()
	var client = linkClient{}
	client.loadConfig("link.testconf.json")
	if client.configuration.ClientID != "testID" {
		tests.Fatal("Unexpected client id: " + client.configuration.ClientID)
	}

}

func TestGetLocations(tests *testing.T) {

	if testConfigSetUp() != nil {
		tests.Fatal("Could not create Test file")
	}
	defer RemoveTestConfigFile()
	var client = linkClient{}
	client.loadConfig("link.testconf.json")
	targets := client.configuration.Targets
	if len(targets) != 2 {
		tests.Fatal("expected two locations")
	}

}

func TestGetCommand(tests *testing.T) {
	target := Target{OnSuccess: "Test $file this $storageref"}
	command := target.getCommand("first", "second")
	if command != "Test first this second" {
		tests.Fatal("Command Parse failed")
	}
}

func testConfigSetUp() error {
	return WriteTestConfigurationFile(`{
		"clientid": "testID",
		 "secret": "verys",
		 "targets": [
		 	{
		 		"name": "EFOQAflow",
		 		"active": true,
		 		"tenant": "abc",
		 		"location": "c:/test/",
		 		"recursive": true,
		 		"match": "*.zip",
		 		"datatype": "qar",
		 		"dataformat": "arinc717"
		 	},
		 	{
		 		"name": "EMUFLOW",
		 		"active": false,
		 		"tenant": "abc2",
		 		"location": "c:/test2/",
		 		"recursive": false,
		 		"match": "*.zip",
		 		"datatype": "EMU",
		 		"dataformat": "FFD1b107"
		 	}]

		 }`)
}

func WriteTestConfigurationFile(config string) error {
	data := []byte(config)
	configFile, err := os.Create("link.testconf.json")
	defer configFile.Close()
	if err == nil {
		_, err = configFile.Write(data)
	}
	configFile.Sync()
	return err
}

func RemoveTestConfigFile() {
	os.Remove("link.testconf.json")
}
