# pigeon-tool

[![Pipeline Status][status-image]][status-url]

[status-image]: https://screwdriver.ouroath.com/pipelines/1035386/badge
[job-status-image]: https://screwdriver.ouroath.com/pipelines/1035386/apply-resources/badge
[status-url]: https://screwdriver.ouroath.com/pipelines/1035386

### Mac download link

https://git.vzbuilders.com/cyang02/pigeon-tool-cyang02/releases/tag/1.3

### If you are using mac, download `darwin version` zts-rolecert and athenz-user-cert first.

https://artifactory.ouroath.com/artifactory/core-tech/releases/athenz-user-cert/1.6.1/Darwin/athenz-user-cert
https://artifactory.ouroath.com/artifactory/core-tech/releases/zts-rolecert/1.30/Darwin/zts-rolecert

```
 chmod +x zts-rolecert ; mv zts-rolecert /usr/local/bin/ ; chmod +x athenz-user-cert ; mv athenz-user-cert /usr/local/bin/

```

## rhel7 download

```
yinst i pigoen_tool -br test

```

### Usage

```
$ pigeon-tool

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

Usage:
  pigeon-tool [command]

Available Commands:
  help        Help about any command
  list        show stuck pigeon queue
  ns-list     list all namespace pigeon use
  skip        skip the certain message of queue or skip all messages of a queue

Flags:
  -c, --certificate string   path to PKI certificate file or you can skip it
  -h, --help                 help for pigeon-tool
  -i, --int                  operation in int environment
  -k, --key string           path to PKI key file or you can skip it
  -r, --role string          zts role or you can skip it (default "pigeon_admin_role")
  -v, --verbose              verbose output for debug

Use "pigeon-tool [command] --help" for more information about a command.
```

