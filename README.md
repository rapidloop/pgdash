
**pgdash** is a command-line tool to invoke and use with the APIs provided by
[pgdash.io](https://pgdash.io).

It can upload the metrics collected by [pgmetrics](https://pgmetrics.io) into
pgdash:

```
$ pgmetrics -f json --no-password mydb | pgdash -a APIKEY report myserver

$ pgmetrics -f json -o report.json mydb
$ pgdash -a APIKEY -i report.json report myserver
```

For more information, see [pgdash.io](https://pgdash.io) and
[pgmetrics.io](https://pgmetrics.io).

pgdash is developed and maintained by [RapidLoop](https://rapidloop.com).
Follow us on Twitter at [@therapidloop](https://twitter.com/therapidloop/).

