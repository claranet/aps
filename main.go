package main

import (
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/go-ini/ini"
	"github.com/manifoldco/promptui"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type profile struct {
	Name          string
	Region        string
	AccountID     string
	Role          string
	SourceProfile string
}

var (
	clear        = kingpin.Flag("clear", "Clear env vars related to AWS").Short('x').Bool()
	configFile   = kingpin.Flag("config", "AWS config file").Short('c').Default(os.Getenv("HOME") + "/.aws/config").ExistingFile()
	chooseRegion = kingpin.Flag("region", "Region selector").Short('r').Bool()
)

// Hack from https://github.com/manifoldco/promptui/issues/49#issuecomment-428801411 to avoid annoying bell in some OS
type stderr struct{}

func (s *stderr) Write(b []byte) (int, error) {
	if len(b) == 1 && b[0] == 7 {
		return 0, nil
	}
	return os.Stderr.Write(b)
}

func (s *stderr) Close() error {
	return os.Stderr.Close()
}

func main() {
	readline.Stdout = &stderr{}

	kingpin.Parse()

	switch {
	case *clear == true:
		startNewShell("", "")
	case *chooseRegion == true:
		startNewShell("", selectRegion())
	default:
		startNewShell(selectProfile(listProfiles(configFile)))
	}
}

func listProfiles(configFile *string) []profile {
	cfg, err := ini.Load(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	reg := regexp.MustCompile("^profile ")

	var profiles []profile

	Names := cfg.SectionStrings()

	for _, n := range Names {
		var p profile
		p.Name = reg.ReplaceAllString(n, "")
		if cfg.Section(n).HasKey("role_arn") {
			arn := cfg.Section(n).Key("role_arn").String()
			r := strings.Split(arn, "/")
			a := strings.Split(arn, ":")
			p.Role = r[len(r)-1]
			p.AccountID = a[4]
		}
		if cfg.Section(n).HasKey("source_profile") {
			p.SourceProfile = cfg.Section(n).Key("source_profile").String()
		}
		if cfg.Section(n).HasKey("region") {
			p.Region = cfg.Section(n).Key("region").String()
		}
		profiles = append(profiles, p)
	}

	return profiles
}

func selectProfile(profiles []profile) (string, string) {
	templates := &promptui.SelectTemplates{
		// Label: `		`,
		Active:   `{{ "> " | cyan | bold }}{{ .Name | cyan | bold }}`,
		Inactive: `  {{ .Name }}`,
		Details:  `{{ "AccountID: " }}{{ .AccountID | bold }} | {{ "Role: " }}{{ .Role | bold }} | {{ "Region: " }}{{ .Region | bold }} | {{ "Source: " }}{{ .SourceProfile | bold }}`,
	}

	searcher := func(input string, index int) bool {
		j := profiles[index]
		Name := strings.ToLower(j.Name + j.AccountID)
		input = strings.ToLower(input)

		return strings.Contains(Name, input)
	}

	prompt := promptui.Select{
		Label:             strconv.Itoa(len(profiles)) + " profiles (current: " + os.Getenv("AWS_PROFILE") + ")",
		Items:             profiles,
		Templates:         templates,
		Size:              10,
		Searcher:          searcher,
		HideSelected:      true,
		StartInSearchMode: true,
	}

	selected, _, err := prompt.Run()
	if err != nil {
		os.Exit(0)
	}

	if profiles[selected].Region == "" {
		if os.Getenv("AWS_DEFAULT_REGION") != "" {
			profiles[selected].Region = os.Getenv("AWS_DEFAULT_REGION")
		} else {
			profiles[selected].Region = selectRegion()
		}
	}

	return profiles[selected].Name, profiles[selected].Region
}

var regions = []string{
	"us-east-2",
	"us-east-1",
	"us-west-1",
	"us-west-2",
	"ap-south-1",
	"ap-northeast-3",
	"ap-northeast-2",
	"ap-northeast-1",
	"ap-southeast-1",
	"ap-southeast-2",
	"ca-central-1",
	"cn-north-1",
	"cn-nortwest-1",
	"eu-central-1",
	"eu-west-1",
	"eu-west-2",
	"eu-west-3",
	"eu-north-1",
	"sa-east-1",
}

func selectRegion() string {
	templates := &promptui.SelectTemplates{
		// Label: `		`,
		Active:   `{{ "> " | cyan | bold }}{{ . | cyan | bold }}`,
		Inactive: `  {{ . }}`,
	}

	searcher := func(input string, index int) bool {
		j := regions[index]
		Name := strings.ToLower(j)
		input = strings.ToLower(input)

		return strings.Contains(Name, input)
	}

	sort.Strings(regions)

	prompt := promptui.Select{
		Label:             "Regions (current: " + os.Getenv("AWS_DEFAULT_REGION") + ")",
		Items:             regions,
		Templates:         templates,
		Size:              10,
		Searcher:          searcher,
		HideSelected:      true,
		StartInSearchMode: true,
	}

	selected, _, err := prompt.Run()
	if err != nil {
		os.Exit(0)
	}

	return regions[selected]
}

func startNewShell(profile, region string) {
	// Get the current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	switch {
	case profile == "" && region == "":
		os.Unsetenv("AWS_PROFILE")
		os.Unsetenv("AWS_DEFAULT_REGION")
	case profile == "" && region != "":
		os.Setenv("AWS_DEFAULT_REGION", region)
	default:
		os.Setenv("AWS_PROFILE", profile)
		os.Setenv("AWS_DEFAULT_REGION", region)
	}

	// Transfer stdin, stdout, and stderr to the new process
	// and also set target directory for the shell to start in.
	pa := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Dir:   cwd,
	}

	// Start up a new shell.
	proc, err := os.StartProcess(os.Getenv("SHELL"), []string{os.Getenv("SHELL")}, &pa)
	if err != nil {
		panic(err)
	}

	// Wait until user exits the shell
	_, err = proc.Wait()
	if err != nil {
		panic(err)
	}

	// Avoid stacked shell sessions, when exit/ctrl+D caller shell is killed
	process, _ := os.FindProcess(os.Getppid())
	process.Kill()
}
