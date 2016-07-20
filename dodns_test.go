package main

import (
	"encoding/json"
	"flag"
	"net/url"
	"os"
	"testing"
)

const (
	validRecord    string = "1"
	invalidRecord  string = "-1"
	validDomain    string = "2byt.es"
	invalidDomain  string = "2;byt.es"
	includesScheme string = "https://2byt.es"
)

var (
	// https://developers.digitalocean.com/documentation/v2/#domain-records
	validRecordJSON = []byte(`
	{
		"id": 3352895,
		"type": "A",
		"name": "@",
		"data": "1.2.3.4",
		"priority": null,
		"port": null,
		"weight": null
	}
	`)

	invalidRecordJSON = []byte(`
	{
		"id": 3352895,
		"name": "@",
		"data": "1.2.3.4",
		"priority": low,
		"port": null,
		"weight": null
	}
	`)
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestURLCheckIPServiceConstants(t *testing.T) {

	for _, dom := range ipCheckAddresses {
		_, err := url.Parse(dom)
		if err != nil {
			t.Errorf("IP check site url [ %s ] could not be parsed/is invalid. Error: %v", dom, err.Error())
		}
	}
}

func TestURLInvalidCheckIPService(t *testing.T) {

	_, err := getIPFrom("get:ip;from.invalid_site.com")

	if err == nil {
		t.Error("An invalid IP search service passed parsing")
	}
}

func TestParseInputValidDomain(t *testing.T) {
	err := checkDomainValid(validDomain)
	if err != nil {
		t.Errorf("Testing valid input domain failed: %v", err.Error())
	}
}

func TestParseInputInvalidDomain(t *testing.T) {
	err := checkDomainValid(invalidDomain)

	if err == nil {
		t.Error("An invalid input domain passed parsing")
	}
}

func TestParseInputInvalidIncludesScheme(t *testing.T) {
	err := checkDomainValid(includesScheme)

	if err == nil {
		t.Error("An invalid input domain (with scheme) passed parsing")
	}
}

func TestParseInputValidRecordValidDomain(t *testing.T) {
	err := checkRecordValid(validRecord, validDomain)

	if err != nil {
		t.Errorf("Testing valid input record failed: %v", err.Error())
	}
}

func TestParseInputInvalidRecordValidDomain(t *testing.T) {
	err := checkRecordValid(invalidRecord, validDomain)

	if err == nil {
		t.Error("An invalid record with a valid domain passed parsing")
	}
}

func TestParseInputValidRecordInvalidDomain(t *testing.T) {
	err := checkRecordValid(validRecord, invalidDomain)

	if err == nil {
		t.Error("A valid record with invalid domain test passed parsing")
	}
}

func TestParseInputInvalidRecordInvalidDomain(t *testing.T) {
	err := checkRecordValid(invalidRecord, invalidDomain)

	if err == nil {
		t.Error("An invalid record with an invalid domain test passed parsing")
	}
}

func TestParseValidDomainRecordJSON(t *testing.T) {
	dr := new(domainRecord)

	err := json.Unmarshal(validRecordJSON, dr)
	if err != nil {
		t.Errorf("Failed to unmarshal valid JSON record; %v", err)
	}
}

func TestParseInvalidDomainRecordJSON(t *testing.T) {
	dr := new(domainRecord)

	err := json.Unmarshal(invalidRecordJSON, dr)
	if err == nil {
		t.Error("An invalid domain record reponse JSON passed parsing")
	}
}
