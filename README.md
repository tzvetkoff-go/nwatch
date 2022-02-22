# The Night's Watch

A simple watch/build/run daemon.

Originally made to watch and rebuild Go projects.

## Installation

Download a prebuilt binary from the [releases](https://github.com/tzvetkoff-go/nwatch/releases) page or build it yourself:

``` bash
git clone https://github.com/tzvetkoff-go/nwatch.git
cd nwatch
make install
```

## Common usage

| Short option | Long option          | Description                                  | Example                  |
| ------------ | -------------------- | -------------------------------------------- | ------------------------ |
| `-h`         | `--help`             | Print help and exit                          |                          |
| `-v`         | `--version`          | Print version and exit                       |                          |
| `-V`         | `--verbose`          | Verbose output                               |                          |
| `-d DIR`     | `--directory=DIR`    | Directories to watch                         | `-d '.'`                 |
| `-e EXC`     | `--exclude-dir=EXC`  | Directories to exclude                       | `-e '.git'`              |
| `-p PAT`     | `--pattern=PAT`      | File patterns to match                       | `-p '*.go'`              |
| `-i IGN`     | `--ignore=IGN`       | File patterns to ignore                      | `-i '*-go-tmp-umask'`    |
| `-b BLD`     | `--build=BLD`        | Build command to execute                     | `-b 'go build'`          |
| `-s SRV`     | `--server=SRV`       | Server command to run after successful build | `-s './webserver start'` |
| `-w ERR`     | `--error-server=ERR` | Web server address in case of an error       | `-w 0.0.0.0:1337`        |

## Pattern matching

Pattern matching is executed against the whole path of the file using [fnmatch](https://github.com/tzvetkoff-go/fnmatch).

If you pass `--directory=.` (the default) the path that will be tested against the pattern will be `dir/path`.

If you pass `--directory=$PWD` the path will be `/home/user/project-root/dir/path`.

This allows you to do things like `--ignore=.git*`, although for this you'd rather use `--exclude-dir=.git`.
