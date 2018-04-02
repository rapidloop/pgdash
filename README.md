
**pgdash** is a command-line tool to invoke and use with the APIs provided by
[pgdash.io](https://pgdash.io).

It can upload the metrics collected by [pgmetrics](https://pgmetrics.io) into
pgdash for quick viewing:

```
$ pgmetrics -f json --no-password mydb | pgdash quick
Upload successful.

Quick View URL: https://app.pgdash.io/quick/qzH033y1JuX3R6LbZiHfY6
Admin Code:     10311
```

For more information, see [pgdash.io](https://pgdash.io) and
[pgmetrics.io](https://pgmetrics.io).

pgdash is developed and maintained by [RapidLoop](https://rapidloop.com).
Follow us on Twitter at [@therapidloop](https://twitter.com/therapidloop/).

