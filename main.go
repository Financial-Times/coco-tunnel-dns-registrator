package main

import (
	"encoding/json"
	"fmt"
	"github.com/jawher/mow.cli"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var httpClient = http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 128,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
	},
}

func main() {
	app := cli.App("elb dns registrator", "Registers elb cname to *-up.ft.com cnames in dyn using konstructor")

	domains := app.String(cli.StringOpt{
		Name:   "domains",
		Desc:   "comma separated *-up domains",
		EnvVar: "DOMAINS",
	})
	konstructorBaseURL := app.String(cli.StringOpt{
		Name:   "konstructor-base-url",
		Desc:   "konstructor base url: https://dns-api.in.ft.com/v2",
		EnvVar: "KONSTRUCTOR_BASE_URL",
		Value:  "https://dns-api.in.ft.com/v2",
	})
	konstructorAPIKey := app.String(cli.StringOpt{
		Name:   "konstructor-api-key",
		Desc:   "konstructor api key",
		EnvVar: "KONSTRUCTOR_API_KEY",
	})
	IP := app.String(cli.StringOpt{
		Name:   "IP",
		Desc:   "IP",
		EnvVar: "IP",
	})

	app.Action = func() {
		c := &conf{
			konsAPIKey:      *konstructorAPIKey,
			konsDNSEndPoint: *konstructorBaseURL,
		}

		domainsToRegister := strings.Split(*domains, ",")

		for _, domain := range domainsToRegister {
			currentCNAME, err := getCurrentCNAME(c, domain)
			if err != nil {
				log.Fatalf("ERROR - [%v]", err)
			}
			if currentCNAME == "" {
				err = createDNS(c, *IP, domain)
			} else {
				err = updateDNS(c, currentCNAME, *IP, domain)
			}
			if err != nil {
				log.Fatalf("ERROR - [%v]", err)
			}
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("ERROR - [%v]", err)
	}
}

type conf struct {
	konsAPIKey      string
	konsDNSEndPoint string
}

func getCurrentCNAME(c *conf, domain string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/name/ft.com/%s", c.konsDNSEndPoint, domain), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("x-api-key", c.konsAPIKey)

	response, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Could not connect to Konstructor, [%v]", err)
	}
	defer func() {
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("Could not read konstructor response body, statusCode=[%v], [%v]", response.StatusCode, err)
	}
	if response.StatusCode != http.StatusOK {
		// if status is not 200, log it, it means domain does not exist
		log.Printf("INFO - Domain=[%v] not created, statusCode=[%v], message=[%v]", domain, response.StatusCode, string(data))
		return "", nil
	}

	type konstructorRes struct {
		CNAMES []string `json:"records"`
	}
	r := konstructorRes{}
	if err := json.Unmarshal(data, &r); err != nil {
		return "", err
	}

	//remove trailing "." otherwise update method complains
	return strings.TrimSuffix(r.CNAMES[0], "."), nil
}

func createDNS(c *conf, elbCNAME string, domain string) error {
	body := fmt.Sprintf("{\"zone\": \"ft.com\", \"name\": \"%s\",\"rdata\": \"%s\",\"ttl\": \"30\",\"emailAddress\": \"universal.publishing.platform@ft.com\"}", domain, elbCNAME)
	req, err := http.NewRequest(http.MethodPost, c.konsDNSEndPoint, strings.NewReader(body))
	if err != nil {
		return err
	}
	if err = executeReq(req, c.konsAPIKey); err != nil {
		return fmt.Errorf("Creating domain=[%v] failed, %v", domain, err)
	}
	return nil
}

func updateDNS(c *conf, oldCname string, newCname, domain string) error {
	body := fmt.Sprintf("{\"zone\": \"ft.com\", \"name\": \"%s\",\"oldRdata\": \"%s\",\"newRdata\": \"%s\",\"ttl\": \"30\",\"emailAddress\": \"universal.publishing.platform@ft.com\"}", domain, oldCname, newCname)
	req, err := http.NewRequest(http.MethodPut, c.konsDNSEndPoint, strings.NewReader(body))
	if err != nil {
		return err
	}
	if err = executeReq(req, c.konsAPIKey); err != nil {
		return fmt.Errorf("Updating domain=[%v] failed, %v", domain, err)
	}
	return nil
}

func executeReq(req *http.Request, key string) error {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("x-api-key", key)

	response, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Could not connect to Konstructor, [%v]", err)
	}
	defer func() {
		io.Copy(ioutil.Discard, response.Body)
		response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		// if status is not 200, log it, but do not consider it as a service failure
		data, err := ioutil.ReadAll(response.Body)
		message := "Response message could not be obtained"
		if err == nil {
			message = string(data)
		}
		return fmt.Errorf("statusCode=[%v], message=[%v]", response.StatusCode, message)
	}
	return nil
}
