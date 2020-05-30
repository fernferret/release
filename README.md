[![Go Report Card](https://goreportcard.com/badge/fernferret/release)](https://goreportcard.com/report/fernferret/release) 

# About

A small helper for doing [calver](https://calver.org/) based releases.

Currently it only supports YYYY.MM.RRR (with RRR being an
auto-incrementing 3 digit index per month). It also supports the concept
of "components" so your CalVer will always increase but you can release
specific components instead of the entire suite of software.

This works well with my `samaritan` code where I have several pieces of
software living in the same repository.

## Usage

You can install by running `make` in the main directory. Install the
binary where you want.

Here's a sample usage of it

```
$ git tag
2020.04.001-release
2020.04.002-release

$ release --no-push
created release: 2020.05.001-release

$ release --no-push
created release: 2020.05.002-release

$ release -n
would create release:
2020.05.003-release

$ release -c watcher --no-push
created release: 2020.05.003-watcher

$ release -c tagger
created release: 2020.05.004-tagger
pushed tag 2020.05.004-tagger to remote origin

$ release
created release: 2020.05.005-release
pushed tag 2020.05.005-release to remote origin

$ git tag
2020.04.001-release
2020.04.002-release
2020.05.001-release
2020.05.002-release
2020.05.003-watcher
2020.05.004-tagger
2020.05.005-release
```
