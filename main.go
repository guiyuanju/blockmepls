package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Host struct {
	path string
}

func (h *Host) getPath() string {
	if len(h.path) != 0 {
		return h.path
	}
	path, err := hostsFilePath()
	if err != nil {
		panic(fmt.Sprintf("failed to find the hosts file: %s", err))
	}
	h.path = path
	return h.path
}

func (h *Host) getBackPath() string {
	if len(h.path) != 0 {
		return h.path + ".blockmepls.bak"
	}
	return h.getPath() + ".blockmepls.bak"
}

var HOST = new(Host)

const GATE string = "127.0.0.1"

type Rule struct {
	Addr string
	Name string
}

func addBlockRuleFor(names []string) error {
	// prevent modify again if not reset, prevent info lost
	if isBakExist() {
		return fmt.Errorf("Backup file exists, please reset the hosts file and try again")
	}

	// read existing hosts
	rules, err := readRules()
	if err != nil {
		return err
	}

	// add new rules
	var newRules []Rule
	for _, name := range names {
		for _, variant := range variantsOfURL(name) {
			newRules = append(newRules, Rule{GATE, variant})
		}
	}
	newRules = append(newRules, rules...)

	// write to tmp file
	tmp, err := writeTmpHost(newRules)
	if err != nil {
		return err
	}

	// swap the file
	err = os.Rename(HOST.getPath(), HOST.getBackPath())
	if err != nil {
		return err
	}
	err = os.Rename(tmp, HOST.getPath())
	if err != nil {
		return err
	}

	return nil
}

func variantsOfURL(url string) []string {
	res := []string{url}
	if !strings.HasPrefix(url, "www.") {
		res = append(res, "www."+url)
	} else {
		res = append(res, strings.TrimPrefix(url, "www."))
	}
	return res
}

func hostsFilePath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// Usually C:\Windows\System32\drivers\etc\hosts
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows` // fallback
		}
		return filepath.Join(root, "System32", "drivers", "etc", "hosts"), nil

	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return "/etc/hosts", nil

	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func isBakExist() bool {
	return isFileExist(HOST.getBackPath())
}

func isFileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func readRules() ([]Rule, error) {
	content, err := os.ReadFile(HOST.getPath())
	if err != nil {
		return nil, err
	}

	var rules []Rule
	for line := range strings.Lines(string(content)) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("unrecognized format, not 2 fields in one line: %s", line)
		}
		rule := Rule{fields[0], fields[1]}
		rules = append(rules, rule)
	}

	return rules, nil
}

func writeTmpHost(rules []Rule) (string, error) {
	tmpHost, err := os.CreateTemp(os.TempDir(), "hosts")
	if err != nil {
		return "", fmt.Errorf("faield to create tmp hosts file")
	}
	defer tmpHost.Close()

	w := bufio.NewWriter(tmpHost)
	for _, rule := range rules {
		w.WriteString(rule.Addr + " " + rule.Name + "\n")
	}

	err = w.Flush()
	if err != nil {
		return "", err
	}

	err = tmpHost.Chmod(0644)
	if err != nil {
		return "", err
	}

	return tmpHost.Name(), nil
}

func resetHost() error {
	if !isBakExist() {
		return fmt.Errorf("backup hosts file not exist")
	}
	return os.Rename(HOST.getBackPath(), HOST.getPath())
}

func flushDNS() error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("resolvectl", "flush-caches").Run()
	case "darwin":
		err := exec.Command("dscacheutil", "-flushcache").Run()
		if err != nil {
			return err
		}
		return exec.Command("killall", "-HUP", "mDNSResponder").Run()
	case "windows":
		return exec.Command("ipconfig", "/flushdns").Run()
	default:
		return fmt.Errorf("unsupported OS")
	}
}

// type definition for comma separated flag
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, strings.Split(value, ",")...)
	return nil
}

func main() {
	var sites stringSlice
	var reset bool
	flag.Var(&sites, "sites", "the sites to be blocked, comma separated")
	flag.BoolVar(&reset, "reset", false, "reset the block list")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of blockmepls:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	if reset {
		fmt.Println("resetting...")
		err := resetHost()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("flusing DNS...")
		err = flushDNS()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return
	}

	if len(sites) > 0 {
		fmt.Println("writing rules...")
		err := addBlockRuleFor(sites)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("flusing DNS...")
		err = flushDNS()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("blocked: %s\n", sites.String())
		fmt.Println("Please restart your browser to see the effect")
		return
	}
}
