package main

import (
	"fmt"
	"os"
	"flag"
	"net/http"
	"net"
	"io/ioutil"
	"strings"
	"encoding/json"
	"bytes"
)

const version string = "0.1"

/*
 	Host your own ip echo with this very simple PHP
 	Add it to index.php and serve it at ip.yourdomain.com
 	===================== SNIP ====================
 	<?
		echo $_SERVER["REMOTE_ADDR"]."\n";
	?>
	===================== SNIP ====================
 */
var ip_check_addresses []string = []string {
	"http://ipecho.net/plain",
	"http://ipinfo.io/ip",
}

const do_api_root string = "https://api.digitalocean.com/v2"
const do_api_records string = do_api_root + "/domains/%s/records"
const do_api_record string = do_api_records + "/%d"

var binName* string

var ipFlag* string

type domain_record struct {
	ID int `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	Priority int `json:"priority"`
	Port string `json:"port"`
	Weight string `json:"weight"`
}

type a_domain_record struct {
	Record domain_record `json:"domain_record"`
}

type domain_records struct {
	Records []domain_record `json:"domain_records"`
}

func print_usage(name string) {
	fmt.Printf("DODNS %s - Hamid Elaosta\n", version)
	fmt.Println("This tool will update a Digital Ocean domain record, using the provided token. Generate this in the Web UI.")
	fmt.Println("Usage:")
	fmt.Printf("%s [-ip <ip>] <token> <domain> <record>\n\n", name)
	fmt.Println("  -ip      Optionally provide an IP address rather than")
	fmt.Println("           using external resolution options.")
}

func print_error(message string) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
}

func decode_ip_check_response(response *http.Response) (error, net.IP) {

	if response.StatusCode == 200 {
		content := response.Body
		contentBytes, err := ioutil.ReadAll(content)

		if err != nil {
			return fmt.Errorf("Failed to read response bytes: %q", err), nil
		}

		ip_string := strings.TrimSpace(string(contentBytes))
		ip := net.ParseIP(ip_string)

		if ip == nil {
			return fmt.Errorf("Response content is not a valid IP address : %q", content), nil
		}

		return nil, ip
	}

	return fmt.Errorf("Unacceptable response code: %q", response.StatusCode), nil
}

func get_ip_from(url string) (error, net.IP) {
	resp,err := http.Get(url)

	if err != nil {
		return fmt.Errorf("Check IP (GET) failed: %q", err), nil
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error response [ %d ]: %q", resp.StatusCode, resp.Status), nil
	}

	if resp.StatusCode != 200 {
		print_error(fmt.Errorf("Non-OK status code returned [ %d ]; %q\n", resp.StatusCode, resp.Status).Error())
	}

	err, ip := decode_ip_check_response(resp)

	return err, ip
}

func check_my_ip() (error, string) {

	for _,remote := range ip_check_addresses {
		err, ip := get_ip_from(remote)

		if err == nil {
			return nil, ip.String()
		} else {
			fmt.Println(err.Error())
		}
	}

	return fmt.Errorf("Failed to retrieve IP from all (%d) remotes\n", len(ip_check_addresses)), ""
}

func update_record_with_ip(ip_addr string, token_string string, domain_name string, record domain_record) (error){

	dr := new(a_domain_record)

	client := &http.Client{}
	reqString := fmt.Sprintf(do_api_record, domain_name, record.ID)

	record.Data = ip_addr
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(record)

	req, err := http.NewRequest("PUT", reqString, b)
	req.Header.Add("Authorization", "Bearer " + token_string)
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

	if dr.Record.Data != ip_addr {
		return fmt.Errorf("Record was not updated successfully.")
	}

	return nil
}

func has_ip_changed(ip_addr string, token_string string, domain_name string, record_name string) (error, domain_record, bool) {

	dr := new(domain_records)

	client := &http.Client{}
	reqString := fmt.Sprintf(do_api_records, domain_name)
	req, err := http.NewRequest("GET", reqString, nil)
	req.Header.Add("Authorization", "Bearer " + token_string)
	resp, err := client.Do(req)

	if err != nil {
		return fmt.Errorf("Check records (GET) failed: %q", err), domain_record{}, false
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error response [ %d ]: %q", resp.StatusCode, resp.Status), domain_record{}, false
	}

	if resp.StatusCode != 200 {
		print_error(fmt.Errorf("Non-OK status code returned [ %d ]; %q\n", resp.StatusCode, resp.Status).Error())
	}

	err = json.NewDecoder(resp.Body).Decode(dr)

	if err != nil {
		return fmt.Errorf("Decoding JSON response failed: %q", err), domain_record{}, false
	}

	if dr.Records == nil {
		return fmt.Errorf("Record not found: [ %q ]", record_name), domain_record{}, false
	}

	for _,record := range dr.Records {
		if(record.Name == record_name) {
			if ip_addr == record.Data {
				return nil, record, false
			} else {
				return nil ,record, true
			}
		}
	}

	return fmt.Errorf("Record not found [ %s ] in domain [ %s ]. Add it in the DO control panel first.", record_name, domain_name), domain_record{}, false

}

func init() {
	ipFlag = flag.String("ip", "", "Supply an IP address to the tool. This prevents external resolution attempts.")
}

func main() {

	binName = &os.Args[0]

	flag.Parse()

	args := flag.Args()

	if len(args) != 3 {
		print_usage(*binName)
		os.Exit(-1)
	}

	token := args[0]
	domain := args[1]
	record := args[2]

	if len(token) != 64 {
		print_error("Token does not appear to be valid.")
		os.Exit(1)
	}

	var ip string

	if *ipFlag != "" {
		fmt.Printf("IP flag provided, using: %q\n", *ipFlag)
		ip = *ipFlag
	} else {
		fmt.Println("No IP provided, using external lookup")
		err, ip_ext := check_my_ip()
		if err != nil {
			print_error(fmt.Errorf("External lookup failed: %q\n", err).Error())
		} else {
			ip = ip_ext
		}
	}

	err, rec, changed := has_ip_changed(ip, token, domain, record)

	if err != nil {
		print_error(fmt.Errorf("DNS update fail: %s\n", err.Error()).Error())
		os.Exit(2)
	}

	if changed {
		err := update_record_with_ip(ip, token, domain, rec)

		if err == nil {
			fmt.Printf("DNS update success: %s\n", ip)
			os.Exit(0)
		} else {
			print_error(fmt.Errorf("DNS update: %s\n", err.Error()).Error())
		}
	} else {
		fmt.Printf("DNS update not required: %s\n", ip)
		os.Exit(0)
	}

	print_error("Unknown error. Please report this.")
	os.Exit(3)
}
