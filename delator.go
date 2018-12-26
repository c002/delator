package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/tcnksm/go-latest"
	"golang.org/x/net/publicsuffix"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	githubTag = &latest.GithubTag{
		Owner:             "netevert",
		Repository:        "delator",
		FixVersionStrFunc: latest.DeleteFrontV()}
	g               = color.New(color.FgHiGreen)
	y               = color.New(color.FgHiYellow)
	r               = color.New(color.FgHiRed)
	domain          = flag.String("d", "", "input domain")
	ver             = flag.Bool("v", false, "check version")
	utilDescription = "delator -d domain"
	myClient        = &http.Client{Timeout: 10 * time.Second}
	appVersion      = "1.1.0"
	banner          = `
8"""8 8""" 8    8"""8 ""8"" 8""88 8""8  
8e  8 8eee 8e   8eee8   8e  8   8 8ee8e
88  8 88   88   88  8   88  8   8 88  8
88ee8 88ee 88ee 88  8   88  8eee8 88  8`
)

type Data struct {
	Issuer_ca_id        int    `json:"issuer_ca_id"`
	Issuer_name         string `json:"issuer_name"`
	Name_value          string `json:"name_value"`
	Min_cert_id         int    `json:"min_cert_id"`
	Min_entry_timestamp string `json:"min_entry_timestamp"`
	Not_after           string `json:"not_after"`
	Not_before          string `json:"not_before"`
}

type Record struct {
	Subdomain string `json:"subdomain"`
	A         string `json:"a_record"`
}

// helper function to print errors and exit
func printError(err string) {
	fmt.Println("error:", err)
	os.Exit(1)
}

// helper function to grab url and robustly handle errors
func grabUrl(Url string) (resp *http.Response) {
	resp, err := http.Get(Url)
	if err, ok := err.(*url.Error); ok {
		if err.Timeout() {
			printError("request timed out")
		} else if err.Temporary() {
			printError("temporary error")
		} else {
			printError(fmt.Sprintf("%s", err.Err))
		}
	}
	if resp.StatusCode != 200 {
		printError(fmt.Sprintf("unexpected status code returned: %d", resp.StatusCode))
	}
	return resp
}

// fetches certificate transparency json data
func fetchData(Url string) []Data {
	res := grabUrl(Url)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	result := strings.Replace(string(body), "}{", "},{", -1)
	d := fmt.Sprintf("[%s]", result)

	keys := make([]Data, 0)
	json.Unmarshal([]byte(d), &keys)
	return keys
}

// deduplicates and prints subdomains
func printData(data []Data) {
	counter := make(map[string]int)
	for _, i := range data {
		counter[i.Name_value]++
		if counter[i.Name_value] == 1 {
			y.Println(i.Name_value)
		}
	}
}

// sets up command-line arguments and default responses
func setup() {
	flag.Usage = func() {
		g.Printf(banner)
		y.Printf("\nwritten & maintained with ♥ by NetEvert\n\n")
		fmt.Println(utilDescription)
		flag.PrintDefaults()
	}

	flag.Parse()

	if *ver {
		y.Printf("DELATOR")
		fmt.Printf(" v.%s\n", appVersion)
		res, _ := latest.Check(githubTag, appVersion)
		if res.Outdated {
			r.Printf("v.%s available\n", res.Current)
		}
		os.Exit(1)
	}

	if *domain == "" {
		r.Printf("\nplease supply a domain\n\n")
		fmt.Println(utilDescription)
		flag.PrintDefaults()
		os.Exit(1)
	}
}

// validates domains using regex
func validateDomainName(domain string) bool {

	patternStr := `^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`

	RegExp := regexp.MustCompile(patternStr)
	return RegExp.MatchString(domain)
}

// sanitizes domains inputted into dnsmorph
func processInput(input string) (sanitizedDomain string) {
	if !validateDomainName(input) {
		r.Printf("\nplease supply a valid domain\n\n")
		fmt.Println(utilDescription)
		flag.PrintDefaults()
		os.Exit(1)
	}
	sanitizedDomain, _ = publicsuffix.EffectiveTLDPlusOne(input)
	return "%." + sanitizedDomain
}

// main program entry point
func main() {
	setup()
	sanitizedDomain := processInput(*domain)
	keys := fetchData(fmt.Sprintf("https://crt.sh/?q=%s&output=json", sanitizedDomain))
	printData(keys)
}