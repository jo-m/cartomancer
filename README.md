# Features

- Weather forecasts (currently only [MeteoSwiss](https://opendatadocs.meteoswiss.ch/de/))
- Map view (currently only [SwissTopo](https://map.geo.admin.ch/))
- Reverse geocoding
- Except for map, does not need live APIs. Instead, will download data and query that locally (meteo and geo names).

# Commands

```bash
go get -tool github.com/pressly/goose/v3/cmd/goose
go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc
go get -tool github.com/mailhog/MailHog
go get -tool github.com/a-h/templ/cmd/templ

make check
make test
make test_online

go tool goose status
go tool goose create REPLACEME sql
go tool goose up
go tool goose validate

go tool sqlc generate
go tool sqlc vet

go tool air
```

# Email

```bash
# http://127.0.0.1:8025/
go tool MailHog
```
