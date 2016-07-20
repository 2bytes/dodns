package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const version string = "0.1.1"

/*
 	Host your own ip echo with this very simple PHP
 	Add it to index.php and serve it at ip.yourdomain.com
 	===================== SNIP ====================
 	<?
		echo $_SERVER["REMOTE_ADDR"]."\n";
	?>
	===================== SNIP ====================
*/
var ipCheckAddresses = []string{
	"http://ipecho.net/plain",
	"http://ipinfo.io/ip",
}

const doAPIRoot string = "https://api.digitalocean.com/v2"
const doAPIRecords string = doAPIRoot + "/domains/%s/records"
const doAPIRecord string = doAPIRecords + "/%d"

var binName *string

var ipFlag *string

type domainRecord struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority"`
	Port     string `json:"port"`
	Weight   string `json:"weight"`
}

type aDomainRecord struct {
	Record domainRecord `json:"domain_record"`
}

type domainRecords struct {
	Records []domainRecord `json:"domain_records"`
}

func printUsage(name string) {
	fmt.Printf("DODNS %s - Hamid Elaosta\n", version)
	fmt.Println("This tool will update a Digital Ocean domain record, using the provided token. Generate this in the Web UI.")
	fmt.Println("Usage:")
	fmt.Printf("%s [-ip <ip>] <token> <domain> <record>\n\n", name)
	fmt.Println("  -ip      Optionally provide an IP address rather than")
	fmt.Println("           using external resolution options.")
}

func printError(message string) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
}

func decodeIPCheckResponse(response *http.Response) (net.IP, error) {

	if response.StatusCode == 200 {
		content := response.Body
		contentBytes, err := ioutil.ReadAll(content)

		if err != nil {
			return nil, fmt.Errorf("Failed to read response bytes: %q", err)
		}

		ipString := strings.TrimSpace(string(contentBytes))
		ip := net.ParseIP(ipString)

		if ip == nil {
			return nil, fmt.Errorf("Response content is not a valid IP address : %q", content)
		}

		return ip, nil
	}

	return nil, fmt.Errorf("Unacceptable response code: %q", response.StatusCode)
}

func getIPFrom(urlString string) (net.IP, error) {

	url, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("Provided url string [ %s ] can not be parsed: %v", urlString, err)
	}

	resp, err := http.Get(url.String())

	if err != nil {
		return nil, fmt.Errorf("Check IP (GET) failed: %q", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Error response [ %d ]: %q", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode != 200 {
		printError(fmt.Errorf("Non-OK status code returned [ %d ]; %q\n", resp.StatusCode, resp.Status).Error())
	}

	ip, err := decodeIPCheckResponse(resp)

	return ip, err
}

func checkMyIP() (string, error) {

	for _, remote := range ipCheckAddresses {
		ip, err := getIPFrom(remote)

		if err == nil {
			return ip.String(), nil
		}

		return "", err

	}

	return "", fmt.Errorf("Failed to retrieve IP from all (%d) remotes\n", len(ipCheckAddresses))
}

func updateRecordWithIP(ipAddr string, tokenString string, domainName string, record domainRecord) error {

	dr := new(aDomainRecord)

	client := &http.Client{}
	reqString := fmt.Sprintf(doAPIRecord, domainName, record.ID)

	record.Data = ipAddr
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(record)

	req, err := http.NewRequest("PUT", reqString, b)
	req.Header.Add("Authorization", "Bearer "+tokenString)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("Update record [ %q ] (PUT) failed: %q", string(record.ID), err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error response [ %d ]: %q", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode != 200 {
		fmt.Printf("Non-OK status code returned [ %d ]; %q\n", resp.StatusCode, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(dr)

	if err != nil {
		return fmt.Errorf("Decoding JSON response failed: %q", err)
	}

	if dr == nil {
		return fmt.Errorf("Record not found: [ %q] ", string(record.ID))
	}

	if dr.Record.Data != ipAddr {
		return fmt.Errorf("Record was not updated successfully.")
	}

	return nil
}

func hasIPChanged(ipAddr string, tokenString string, domainName string, recordName string) (domainRecord, bool, error) {

	drs := new(domainRecords)

	client := &http.Client{}
	reqString := fmt.Sprintf(doAPIRecords, domainName)
	req, err := http.NewRequest("GET", reqString, nil)
	req.Header.Add("Authorization", "Bearer "+tokenString)
	resp, err := client.Do(req)

	if err != nil {
		return domainRecord{}, false, fmt.Errorf("Check records (GET) failed: %q", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return domainRecord{}, false, fmt.Errorf("Error response [ %d ]: %q", resp.StatusCode, resp.Status)
	}

	if resp.StatusCode != 200 {
		printError(fmt.Errorf("Non-OK status code returned [ %d ]; %q\n", resp.StatusCode, resp.Status).Error())
	}

	err = json.NewDecoder(resp.Body).Decode(drs)

	if err != nil {
		return domainRecord{}, false, fmt.Errorf("Decoding JSON response failed: %q", err)
	}

	if drs.Records == nil {
		return domainRecord{}, false, fmt.Errorf("Record not found: [ %q ]", recordName)
	}

	for _, record := range drs.Records {
		if record.Name == recordName {
			if ipAddr == record.Data {
				return record, false, nil
			}

			return record, true, nil
		}
	}

	return domainRecord{}, false, fmt.Errorf("Record not found [ %s ] in domain [ %s ]. Add it in the DO control panel first.", recordName, domainName)

}

func checkDomainValid(dom string) error {
	add, err := url.Parse(dom)

	if err != nil {
		return fmt.Errorf("Failed to parse domain [ %v ]", dom)
	}

	if add.Scheme != "" {
		return fmt.Errorf("Scheme should not be specified, you gave [ %v ]", add.Scheme)
	}

	addrs, err := net.LookupHost(dom)

	if err != nil {
		return fmt.Errorf("Failed to lookup host [ %v ], error : %v", add.Host, err)
	}

	if len(addrs) < 1 {
		return fmt.Errorf("No valid addresses found for domain [ %v ]", add.String())
	}

	return nil
}

func checkRecordValid(record string, domain string) error {

	err := checkDomainValid(domain)
	if err != nil {
		return fmt.Errorf("Domain invalid, cannot test record")
	}

	fullHost := record + "." + domain

	add, err := url.Parse(fullHost)

	if err != nil {
		return fmt.Errorf("Failed to parse record [ %v ] with domain [ %v ]", record, domain)
	}

	addrs, err := net.LookupHost(fullHost)

	if err != nil {
		return fmt.Errorf("Failed to lookup host [ %v ], error : %v", add.Host, err)
	}

	if len(addrs) < 1 {
		return fmt.Errorf("No valid addresses found for domain [ %v ]", add.String())
	}

	return nil
}

func init() {
	ipFlag = flag.String("ip", "", "Supply an IP address to the tool. This prevents external resolution attempts.")
}

func main() {

	binName = &os.Args[0]

	flag.Parse()

	args := flag.Args()

	if len(args) != 3 {
		printUsage(*binName)
		os.Exit(-1)
	}

	token := args[0]
	domain := args[1]
	record := args[2]

	if len(token) != 64 {
		printError("Token does not appear to be valid.")
		os.Exit(1)
	}

	err := checkDomainValid(domain)

	if err != nil {
		printError(err.Error())
		os.Exit(2)
	}

	err = checkRecordValid(record, domain)

	if err != nil {
		printError(err.Error())
		os.Exit(3)
	}

	var ip string

	if *ipFlag != "" {
		fmt.Printf("IP flag provided, using: %q\n", *ipFlag)
		ip = *ipFlag
	} else {
		fmt.Println("No IP provided, using external lookup")
		ipExt, err := checkMyIP()
		if err != nil {
			printError(fmt.Errorf("External lookup failed: %q\n", err).Error())
			os.Exit(4)
		} else {
			ip = ipExt
		}
	}

	rec, changed, err := hasIPChanged(ip, token, domain, record)

	if err != nil {
		printError(fmt.Errorf("DNS update fail: %s\n", err.Error()).Error())
		os.Exit(5)
	}

	if changed {
		err := updateRecordWithIP(ip, token, domain, rec)

		if err == nil {
			fmt.Printf("DNS update success: %s\n", ip)
			os.Exit(0)
		} else {
			printError(fmt.Errorf("DNS update: %s\n", err.Error()).Error())
		}
	} else {
		fmt.Printf("DNS update not required: %s\n", ip)
		os.Exit(0)
	}

	printError("Unknown error. Please report this.")
	os.Exit(6)
}
