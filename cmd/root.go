package cmd

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const athenzUserCertUtil = "athenz-user-cert"
const ztsRoleCert = "zts-rolecert"

// Global options.
var roleName string
var verbose bool
var keyPath string
var certPath string
var staging bool

func detectSIAKeyCertPair() (e error) {
	const siaRootDir = "/var/lib/sia"
	const hostdocPath = siaRootDir + "/host_document"

	log.Println("detecting Athenz service identity")

	hostdocFile, err := os.Open(hostdocPath)
	defer hostdocFile.Close()
	if err != nil {
		return err
	}
	log.Printf("opened host-document file: %s", hostdocPath)

	hostdocData := make([]byte, 4096)
	count, err := hostdocFile.Read(hostdocData)
	if err != nil {
		return err
	}
	log.Printf("read %d bytes from host-document file", count)

	// Perform minimal parse.
	var hostdoc map[string]*json.RawMessage
	if err = json.Unmarshal(hostdocData[:count], &hostdoc); err != nil {
		return err
	}

	var domain string
	if err = json.Unmarshal(*hostdoc["domain"], &domain); err != nil {
		return err
	}
	log.Printf("detected Athenz domain: %s", domain)

	var service string
	if err = json.Unmarshal(*hostdoc["service"], &service); err != nil {
		return err
	}

	// Due to an OpenStack quirk, the service field could be a string containing
	// a comma-delimited list of service names.
	commaIndex := strings.Index(service, ",")
	if commaIndex != -1 {
		// This is a multiple-service list. We choose the first, which by
		// convention is assumed to be the canonical service.
		log.Printf("found multiple Athenz services: %s", service)
		service = service[:commaIndex]
	}
	log.Printf("detected Athenz service: %s", service)

	keyPath = siaRootDir + "/keys/" + domain + "." + service + ".key.pem"
	certPath = siaRootDir + "/certs/" + domain + "." + service + ".cert.pem"
	return
}

func validateKeyCertPair(userKeyPath string, userCertPath string) error {
	log.Println("validating user certificate")

	// Ensure the key file is present.
	if _, err := os.Stat(userKeyPath); os.IsNotExist(err) {
		return fmt.Errorf("key file does not exist")
	}

	// Ensure the user's certificate is still valid.
	bytes, err := ioutil.ReadFile(userCertPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %s", err.Error())
	}
	log.Printf("read contents of user-identity certificate file: %s", userCertPath)

	block, _ := pem.Decode(bytes)
	if block == nil {
		return fmt.Errorf("got bad block when decoding certificate: %s", err.Error())
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %s", err.Error())
	}
	log.Printf("SN=%s, CN=%s, since=%s, until=%s", cert.SerialNumber, cert.Subject.CommonName, cert.NotBefore, cert.NotAfter)

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("certificate is not valid")
	}

	log.Println("certificate is still valid")
	return nil
}

func execAthenzUserCertUtility() error {
	log.Println("executing Athenz user-certificate command-line utility")

	// Ensure our path includes all directories where the utility could reside.
	pathenv, pathset := os.LookupEnv("PATH")
	if pathset {
		// Path set. We are fine with adding duplicate entries.
		os.Setenv("PATH", pathenv+":/usr/bin:/opt/yahoo/yamas/bin:/usr/local/bin")
	} else {
		// Path not set.
		os.Setenv("PATH", "/usr/bin:/opt/yahoo/yamas/bin:/usr/local/bin")
	}

	path, err := exec.LookPath(athenzUserCertUtil)
	if err != nil {
		return err
	}
	log.Printf("found utility: %s", path)

	cmd := exec.Command(path)
	owriter := io.MultiWriter(os.Stdout)
	cmd.Stdout = owriter
	cmd.Stderr = owriter
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("when executing %s: %s", athenzUserCertUtil, err.Error())
	}

	return nil
}

func execZtsCertUtility(keyPath string, certPath string, staging bool, roleName string) error {
	var athenzDomain string
	log.Println("executing zts-rolecert command-line utility")
	if roleName == "" {
		roleName = "pigeon_admin_role"
	}
	if staging {
		athenzDomain = "nevec.pigeon.int"
	} else {
		athenzDomain = "nevec.pigeon.prod"
	}
	// Ensure our path includes all directories where the utility could reside.
	pathenv, pathset := os.LookupEnv("PATH")
	if pathset {
		// Path set. We are fine with adding duplicate entries.
		os.Setenv("PATH", pathenv+":/usr/bin:/usr/local/bin")
	} else {
		// Path not set.
		os.Setenv("PATH", "/usr/bin:/usr/local/bin")
	}

	path, err := exec.LookPath(ztsRoleCert)
	if err != nil {
		return err
	}
	log.Printf("found utility: %s", path)
	// zts-rolecert -svc-key-file ~/.athenz/key -svc-cert-file ~/.athenz/cert -zts https://zts.athens.yahoo.com:4443/zts/v1 -role-domain nevec.pigeon.prod -role-name pigeon_admin_role -dns-domain zts.yahoo.cloud -role-cert-file /tmp/pigeon_admin_role.cert
	cmd := exec.Command(
		ztsRoleCert, "-svc-key-file", keyPath, "-svc-cert-file", certPath,
		"-zts", "https://zts.athens.yahoo.com:4443/zts/v1", "-role-domain",
		athenzDomain, "-role-name", roleName,
		"-dns-domain", "zts.yahoo.cloud", "-role-cert-file", "/tmp/pigeon_admin_role.cert")
	owriter := io.MultiWriter(os.Stdout)
	cmd.Stdout = owriter
	cmd.Stderr = owriter
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("when executing %s: %s", ztsRoleCert, err.Error())
	}

	return nil
}

func detectAthenzUserKeyCertPair(me *user.User) error {
	log.Printf("detecting Athenz identity for user: %s\n", me.Username)

	userIdDir := me.HomeDir + "/.athenz"
	keyPath = userIdDir + "/key"
	certPath = userIdDir + "/cert"
	if err := validateKeyCertPair(keyPath, certPath); err != nil {
		if err = execAthenzUserCertUtility(); err != nil {
			return err
		}
	}

	return nil
}

func getKeyCertPair() error {
	me, err := user.Current()
	if err != nil {
		return err
	}
	log.Printf("running as user: %s\n", me.Username)

	if me.Username == "root" {
		return detectSIAKeyCertPair()
	}
	return detectAthenzUserKeyCertPair(me)
}

func getClient(cert string) (*http.Client, error) {
	// Note that order of key and cert is reversed from my convention.
	if cert == "" {
		cert = certPath
	}
	log.Printf("loading key %s and cert %s", keyPath, cert)
	pair, err := tls.LoadX509KeyPair(cert, keyPath)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{pair},
			InsecureSkipVerify: true,
		},
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
	}

	return client, nil
}

func doGeneric(client *http.Client, request *http.Request, expectedCode int) ([]byte, error) {
	log.Printf("issuing %s to URL: %s", request.Method, request.URL)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("transaction error: %s", err.Error())
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error when reading response body: %s", err.Error())
	}

	if response.StatusCode != expectedCode {
		return nil, fmt.Errorf("server returned code %d with message: %s", response.StatusCode, body)
	}

	return body, nil
}

func doPutUseChan(client *http.Client, url string, payload []byte, expectedCode int, wg *sync.WaitGroup) {
	defer wg.Done()
	request, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		log.Fatalf("failed to construct PUT request: %s", err.Error())
	}
	request.Header.Set("Content-Type", "application/json")

	_, err = doGeneric(client, request, expectedCode)
	if err != nil {
		log.Fatalf("%s with error: %s", url, err.Error())
	}
}

func doPut(client *http.Client, url string, payload []byte, expectedCode int) error {

	request, err := http.NewRequest("PUT", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("failed to construct PUT request: %s", err.Error())
		return fmt.Errorf("failed to construct PUT request: %s", err.Error())
	}
	request.Header.Set("Content-Type", "application/json")

	_, err = doGeneric(client, request, expectedCode)
	if err != nil {
		log.Printf("%s with error: %s", url, err.Error())
		return err
	}
	return nil
}

func doGetUseChan(client *http.Client, url string, ch chan<- []byte) error {
	log.Printf("issuing GET to URL: %s", url)

	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("transaction error: %s", err.Error())
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return fmt.Errorf("error when reading response body: %s", err.Error())
	}

	if response.StatusCode != 200 {
		return fmt.Errorf("server returned code %d with message: %s", response.StatusCode, body)
	}

	ch <- body
	return nil
}
func doGet(client *http.Client, url string) ([]byte, error) {
	log.Printf("issuing GET to URL: %s", url)

	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("transaction error: %s", err.Error())
	}

	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("error when reading response body: %s", err.Error())
	}

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("server returned code %d with message: %s", response.StatusCode, body)
	}

	return body, nil
}

func printJSON(body []byte) error {

	var obuf bytes.Buffer
	if err := json.Indent(&obuf, body, "", "    "); err != nil {
		return err
	}
	if _, err := obuf.WriteRune('\n'); err != nil {
		return err
	}
	if _, err := obuf.WriteTo(os.Stdout); err != nil {
		return err
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:   "pigeon-tool",
	Short: "pigeon queue management",
	Long: `
list all namespace
Eg. pigeon-tool ns-list
	
list stuck queue per namespace
Eg.
pigeon-tool list -n NevecTW
pigeon-tool list -n all
	
skip a message of the certain queue 	
Eg. pigeon-tool skip -q CQI.prod.storeeps.set.action::CQO.prod.storeeps.set.action.search.merlin -m d925d129-e4e7-4602-bba4-124bf462bc5c__08959ef907109ef601
	
skip all message of the certain queue  	
Eg. pigeon-tool skip -q CQI.prod.storeeps.set.action::CQO.prod.storeeps.set.action.search.merlin -m all

 If you want to operate for staging pigeon queue, add -i parameter
Eg. pigeon-tool -i list -n all
Eg. pigeon-tool -i skip -q CQI.int.nevec.merchandise.event.all::CQO.int.nevec.merchandise.event.tns.sauroneye -m all
	`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// If requested, enable logging.
		if verbose {
			log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
			log.Println("verbose logging enabled")
		} else {
			log.SetOutput(ioutil.Discard)
		}

		// Avoid autodetecting certificate and key if requested.
		//
		// To skip autodetection, commands should annotate themselves with
		// authenticate=no.
		if shouldAuthenticate, ok := cmd.Annotations["authenticate"]; ok && shouldAuthenticate == "no" {
			log.Println("skipping authentication for this command")
		} else {
			// If needed, autodetect key/cert paths.
			if keyPath == "" || certPath == "" {
				log.Println("invalid key/cert specification")
				if err := getKeyCertPair(); err != nil {
					return err
				}
				log.Printf("detected key path: %s", keyPath)
				log.Printf("detected cert path: %s", certPath)
			}
		}

		if err := execZtsCertUtility(keyPath, certPath, staging, roleName); err != nil {
			return err
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for debug")
	rootCmd.PersistentFlags().StringVarP(&roleName, "role", "r", "pigeon_admin_role", "zts role or you can skip it")
	rootCmd.PersistentFlags().StringVarP(&keyPath, "key", "k", "", "path to PKI key file or you can skip it")
	rootCmd.PersistentFlags().StringVarP(&certPath, "certificate", "c", "", "path to PKI certificate file or you can skip it")
	rootCmd.PersistentFlags().BoolVarP(&staging, "int", "i", false, "operation in int environment")
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
