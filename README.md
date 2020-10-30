# APS - Amazon Profile Switcher

## Description

Easy switch between AWS Profiles and Regions.

## Why

As a service provider we have to switch all the time between our customers' accounts and as lazy DevOps we do no want to always pass same args to our commands. Environment variables are so a good solution for helping us. Here comes **APS** aka **Amazon Profile Switcher**.

## Usage

```bash
usage: aps [<flags>]

Flags:
      --help    Show context-sensitive help (also try --help-long and --help-man).
  -x, --clear   Clear env vars related to AWS
  -c, --config=$HOME/.aws/config
                AWS config file
  -r, --region  Region selector
```

You can select your profile/region by &larr;, &uarr;, &rarr; &darr; and filter by **Name**, or **AccountId** (only for profile). **Enter** key to validate. 

## Output Example

![screenshot1](./img/screenshot1.png)
![screenshot2](./img/screenshot2.png)


## Build

```bash
make all
```
This repository uses `go mod`, so don't `git clone` inside your `$GOPATH`.

## Author

Thomas Labarussias (thomas.labarussias@fr.clara.net - https://github.com/Issif)