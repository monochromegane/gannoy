# Gannoy [![Build Status](https://travis-ci.org/monochromegane/gannoy.svg?branch=master)](https://travis-ci.org/monochromegane/gannoy)

Approximate nearest neighbor search server and dynamic index written in Golang.
Gannoy is inspired by [spotify/annoy](https://github.com/spotify/annoy) and provides a dynamic database and API server.

## Quick start

```sh
# Create database
$ gannoy create -d 100 DATABASE_NAME
# Start server
$ gannoy-db
```

Regiter features using gannoy API.

```sh
$ curl 'http://localhost:1323/databases/DATABASE_NAME/features/KEY' \
       -H "Content-type: application/json" \
       -X PUT \
       -d '{"features": [1.0, 0.5, 0.2, ...]}'
```

Search similar items.

```sh
$ curl 'http://localhost:1323/search?database=DATABASE_NAME&key=KEY'
[10, 23, 2, 20, 300, 45, 11, 8, 39, 88]
```

See also `gannoy create --help` or `gannoy-db --help`.

## Install

```sh
$ go get github.com/monochromegane/gannoy/...
```

Recommendation environment is **Linux (kernel >= 3.15.0)**.

Gannoy uses fcntl system call and F\_OFD\_SETLKW command to lock the necessary minimum range from multiple goroutines for speeding up.

## API

### GET /search

Search approximate nearest neighbor items.

#### query parameters

| key      | value                                             |
| -------- | ------------------------------------------------- |
| database | Search for similar items from this database name. |
| key      | Search for similar items from this key's feature. |
| limit    | Maxium number of result.                          |

#### Response

* Response 200 (application/json)
  * return list of item keys.
* Response 404 (no content)
  * return no content if you specify not found database or key.

### POST /databases/:database/features

Register features using a specified key.

#### URI parameters

| key      | value                              |
| -------- | ---------------------------------- |
| database | Create item in this database name. |

#### JSON parameters

| key      | value                       |
| -------- | --------------------------- |
| key      | Create item using this key. |
| features | List of feature value.      |

**Note**: `KEY` must be integer.

#### Response

* Response 200 (no content)
  * return no content.
* Response 422 (no content)
  * return no content if you specify not found database or unprocessable parameter.

### PUT /databases/:database/features/:key

Register or update features using a specified key.

#### URI parameters

| key      | value                                        |
| -------- | -------------------------------------------- |
| database | Create or update item in this database name. |
| key      | Create or update item using this key.        |

**Note**: `KEY` must be integer.

#### JSON parameters

| key      | value                   |
| -------- | ----------------------- |
| features | List of feature value.  |

#### Response

* Response 200 (no content)
  * return no content.
* Response 422 (no content)
  * return no content if you specify not found database or unprocessable parameter.

### DELETE /databases/:database/features/:key

Register or update features using a specified key.

#### URI parameters

| key      | value                                |
| -------- | ------------------------------------ |
| database | Delete item from this database name. |
| key      | Delete item using this key.          |

#### Response

* Response 200 (no content)
  * return no content.
* Response 422 (no content)
  * return no content if you specify not found database or unprocessable parameter.

## Run with Server::Starter

Gannoy can run with Server::Starter for supporting graceful restart.

```sh
$ start_server --port 8080 --pid-file app.pid -- gannoy-db -s # gannoy-db listen Server::Starter port if you pass s option.
```

## Configuration

Gannoy can load option from configuration file.

If you prepare a configuration file named `gannoy.toml` like the following:

```toml
data-dir = "/var/lib/gannoy"
log-dir  = "/var/log/gannoy"
lock-dir = "/var/run/gannoy"
server-starter = true
```

You can specify the name with c option.

```sh
$ gannoy-db -c gannoy.toml
```

**Note**: A priority of flag is `command-line flag > configration file > flag default value`. See also [monochromegane/conflag](https://github.com/monochromegane/conflag).

## Building rpm

**Note**: Requirements are Docker and docker-compose.

```sh
$ docker-compose build gannoy-rpmbuild
$ docker-compose run gannoy-rpmbuild
```

Result (`gannoy-x.x.x-x.x86_64.rpm`) is put in `rpmbuild/RPMS/x86_64` directory on host.

You can install the rpm and running gannoy-db process on CentOS.

```sh
$ sudo rpm -ivh gannoy-x.x.x-x.x86_64.rpm
$ sudo systemctl start gannoy-db
```

## Data migration from annoy

You can migrate [spotify/annoy](https://github.com/spotify/annoy) database file.

```sh
$ gannoy-converter -d 100 ANNOY_FILE DATABASE_NAME
```

## License

[MIT](https://github.com/monochromegane/gannoy/blob/master/LICENSE)

## Author

[monochromegane](https://github.com/monochromegane)
