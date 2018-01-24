package link

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func checkConnection() string {
	var buffer bytes.Buffer
	_, err := http.Get("https://predix.io")
	if err != nil {
		buffer.WriteString("ERROR: \"")
		buffer.WriteString(err.Error())
		buffer.WriteString("\" from https://predix.io")
		buffer.WriteString("\n")
		_, err = http.Get("https://google.com")
		if err != nil {
			buffer.WriteString("ERROR: \"")
			buffer.WriteString(err.Error())
			buffer.WriteString("\" from https://google.com\n")
			buffer.WriteString("\tPlease check that your network cable is properly connected\n")
			buffer.WriteString("\tIt maybe that your business uses a proxy to access the internet,\n")
			buffer.WriteString("\tif so you will need to contact the IT support that look after\n")
			buffer.WriteString("\tyour local network, to find out what address is of the proxy\n")
			buffer.WriteString("\tused to route http web traffic. Once you have this you will need\n")
			buffer.WriteString("\tto set the following two local environment variable to point at that address:\n")
			buffer.WriteString("\tHTTP_PROXY\n")
			buffer.WriteString("\tHTTPS_PROXY\n")
			return buffer.String()
		}

		buffer.WriteString("OK: https://google.com\n")
		buffer.WriteString("\tLooks like you can access the internet but you are not able to reach https://predix.io\n")
		buffer.WriteString("\tCheck that predix is up and running from anther network. Try Predix.io from a browser\n")
		buffer.WriteString("\ton your phone should work.\n")
		buffer.WriteString("\tIf you can access Predix from another network, you may need to get you local IT department\n")
		buffer.WriteString("\tto configure you firewalls and network routers to allow access to predix.io and its sub domains\n")
		return buffer.String()
	}

	buffer.WriteString("OK: https://predix.io visible from this location\n")
	return buffer.String()
}

func checkAllConfigs() string {
	var buffer bytes.Buffer
	files, err := ioutil.ReadDir(".")
	if err != nil {
		return "ERROR: could not access configuration files. \n" + err.Error()
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			buffer.WriteString(checkConfig(file))
		}
	}
	return buffer.String()
}

func checkConfig(file os.FileInfo) string {
	var buffer bytes.Buffer

	sol := linkClient{}
	err := sol.loadConfig(file.Name())
	if err != nil {
		buffer.WriteString("Couldn't load file: ")
		buffer.WriteString(file.Name())
		buffer.WriteString("\n")
	}

	scopes, err := getScopes(sol.configuration.TokenURL, sol.configuration.ClientID, sol.configuration.Secret)
	if err != nil {
		buffer.WriteString("ERROR: Failed to authenticate client in ")
		buffer.WriteString(file.Name())
		buffer.WriteString("\n")
		buffer.WriteString(err.Error())
		buffer.WriteString("\n")
		return buffer.String()
	}
	buffer.WriteString("OK: Authenticated client in ")
	buffer.WriteString(file.Name())
	buffer.WriteString("\n")

	permissions := make(map[string]int)
	for _, scope := range strings.Split(scopes, " ") {
		if strings.HasPrefix(scope, "trueconnect.tenants.") {
			parts := strings.Split(scope, ".")
			perm := strings.Join(parts[3:], ".")
			switch perm {
			case "metadata.read":
				permissions[parts[2]]++
				break
			case "metadata.write":
				permissions[parts[2]] += 2
				break
			case "file.read":
				permissions[parts[2]] += 4
				break
			case "file.write":
				permissions[parts[2]] += 8
				break
			}
		}
	}

	for _, target := range sol.configuration.Targets {
		if permissions[target.Tenant]&8 == 8 {
			buffer.WriteString("OK: Permissions on target ")
			buffer.WriteString(target.Name)
			buffer.WriteString(" in configuration file ")
			buffer.WriteString(file.Name())
			buffer.WriteString("\n")
		} else {
			buffer.WriteString("ERROR: the client used in ")
			buffer.WriteString(file.Name())
			buffer.WriteString(" does not have the \"file.write\" permission on tenant ")
			buffer.WriteString(target.Tenant)
			buffer.WriteString(" used for target named ")
			buffer.WriteString(target.Name)
			buffer.WriteString("\n")
		}
	}

	resp, err := http.Get(sol.configuration.Endpoint + "/api/v1/status")
	if err != nil && resp.StatusCode != 403 {
		if resp.StatusCode == 401 {
			buffer.WriteString("ERROR: The configured endpoint in ")
			buffer.WriteString(file.Name())
			buffer.WriteString("\n\tdid not accept token from the configure token url in the same file")
			buffer.WriteString("\n")
			return buffer.String()
		}
		buffer.WriteString("ERROR: Could not connect to TrueConnect at the configured endpoint in ")
		buffer.WriteString(file.Name())
		buffer.WriteString("\n")
		return buffer.String()
	}

	buffer.WriteString("OK: TrueConnect connection established for configured endpoint in ")
	buffer.WriteString(file.Name())
	buffer.WriteString("\n")
	return buffer.String()
}

func getScopes(tokURL string, clientID string, secret string) (string, error) {
	body := bytes.NewBuffer([]byte("grant_type=client_credentials&client_id=" + clientID + "&client_secret=" + secret + "&response_type=token"))
	req, err := http.NewRequest("POST", tokURL, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	rsBody, err := ioutil.ReadAll(resp.Body)
	type WithScope struct {
		Scope string `json:"scope"`
	}
	var dat WithScope
	err = json.Unmarshal(rsBody, &dat)
	if err != nil {
		return "", err
	}

	return dat.Scope, err
}
