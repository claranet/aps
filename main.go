package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
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
	RoleARN       string
	SourceProfile string
}

var (
	clear         = kingpin.Flag("clear", "Clear env vars related to AWS").Short('x').Bool()
	configFile    = kingpin.Flag("config", "AWS config file").Short('c').Default(os.Getenv("HOME") + "/.aws/config").ExistingFile()
	chooseProfile = kingpin.Flag("profile", "Specify directly the AWS Profile to use").Short('p').String()
	chooseRegion  = kingpin.Flag("region", "Region selector").Short('r').String()
	assume        = kingpin.Flag("assume", "If false, auto assume role is disabled (default is true)").Short('a').String()
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

	for i, j := range os.Args {
		if j == "-r" || j == "--region" {
			if len(os.Args) == 2 {
				os.Args[i] = "--region=000"
			}
		}
	}

	kingpin.Parse()

	switch {
	case *clear == true:
		startNewShell(profile{})
	case *chooseRegion != "":
		startNewShell(profile{Region: selectRegion(chooseRegion)})
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
			p.RoleARN = cfg.Section(n).Key("role_arn").String()
			r := strings.Split(p.RoleARN, "/")
			a := strings.Split(p.RoleARN, ":")
			p.Role = r[len(r)-1]
			p.AccountID = a[4]
		}
		if cfg.Section(n).HasKey("source_profile") {
			p.SourceProfile = cfg.Section(n).Key("source_profile").String()
		}
		if cfg.Section(n).HasKey("region") {
			p.Region = cfg.Section(n).Key("region").String()
		}
	}

	return profiles
}

func selectProfile(profiles []profile) profile {
	if *chooseProfile != "" {
		for _, i := range profiles {
			if i.Name == *chooseProfile {
				if i.Region == "" {
					if os.Getenv("AWS_DEFAULT_REGION") != "" {
						i.Region = os.Getenv("AWS_DEFAULT_REGION")
					} else {
						i.Region = selectRegion(chooseRegion)
					}
				}
				return i
			}
		}
	}

	current := os.Getenv("AWS_PROFILE")

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
		Label:             strconv.Itoa(len(profiles)) + " profiles (current: " + current + ")",
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
			profiles[selected].Region = selectRegion(chooseRegion)
		}
	}

	return profiles[selected]
}

var regions = []string{
	"us-east-2      | Ohio",
	"us-east-1      | N. Virginia",
	"us-west-1      | N. California",
	"us-west-2      | Oregon",
	"ap-south-1     | Mumbai",
	"ap-northeast-3 | Osaka-Local",
	"ap-northeast-2 | Seoul",
	"ap-northeast-1 | Tokyo",
	"ap-southeast-1 | Singapore",
	"ap-southeast-2 | Sydney",
	"ca-central-1   | Central",
	"cn-north-1     | Beijing",
	"cn-nortwest-1  | Ningxia",
	"eu-central-1   | Frankfurt",
	"eu-west-1      | Ireland",
	"eu-west-2      | London",
	"eu-west-3      | Paris",
	"eu-north-1     | Stockholm",
	"sa-east-1      | São Paulo",
}

func selectRegion(r *string) string {
	if *r != "000" {
		return *r
	}
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

	regionWithOutName := strings.Trim(strings.Split(regions[selected], "|")[0], " ")

	return regionWithOutName
}

func startNewShell(p profile) {
	// Get the current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	switch {
	case p.Name == "" && p.Region == "":
		os.Unsetenv("AWS_PROFILE")
		os.Unsetenv("AWS_DEFAULT_REGION")
	case p.Name == "" && p.Region != "":
		os.Setenv("AWS_DEFAULT_REGION", p.Region)
		fmt.Println("Active Region: \033[1m" + p.Region + "\033[0m")
	default:
		os.Setenv("AWS_PROFILE", p.Name)
		fmt.Println("Active Profile: \033[1m" + p.Name + "\033[0m")
		os.Setenv("AWS_DEFAULT_REGION", p.Region)
		fmt.Println("Active Region: \033[1m" + p.Region + "\033[0m")
		if p.RoleARN != "" && *assume != "false" {
			setIAMStsEnv(p)
		}
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

func setIAMStsEnv(p profile) {
	awsSession := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState:       session.SharedConfigEnable,  //enable use of ~/.aws/config
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider, //ask for MFA if needed
		Profile:                 p.Name,
		Config:                  aws.Config{Region: aws.String(p.Region)},
	}))

	r, err := sts.New(awsSession).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatal(err.Error())
	}
	s := strings.Split(*r.UserId, ":")
	t, err := sts.New(awsSession).AssumeRole(&sts.AssumeRoleInput{RoleArn: aws.String(p.RoleARN), RoleSessionName: aws.String(s[1])})
	if err != nil {
		log.Fatal(err.Error())
	}
	os.Setenv("AWS_ACCESS_KEY_ID", *t.Credentials.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", *t.Credentials.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", *t.Credentials.SessionToken)
}
