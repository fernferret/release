[![Go Report
Card](https://goreportcard.com/badge/fernferret/release)](https://goreportcard.com/report/fernferret/release)

# About

A small helper for doing [calver](https://calver.org/) based releases.

Currently it only supports YYYY.MM.RRR (with RRR being an auto-incrementing 3
(or anything more than 3) digit index per month). It also supports the concept
of "components" so your CalVer will always increase but you can release specific
components instead of the entire suite of software.

This works well with my `samaritan` code where I have several pieces of software
living in the same repository.

## Usage

You can install by running `make` in the main directory. Install the binary
where you want.

Here's a sample usage of it

```
$ git tag
2020.04.001-release
2020.04.002-release

$ release
created release: 2020.07.001-release
tag (2020.07.001-release) not pushed (--push not set), push it with:
 git push origin 2020.07.001-release

$ release
created release: 2020.07.002-release
tag (2020.07.002-release) not pushed (--push not set), push it with:
 git push origin 2020.07.002-release

$ release -n
would create release:
2020.07.003-release

$ release watcher
created release: 2020.07.003-watcher
tag (2020.07.003-watcher) not pushed (--push not set), push it with:
 git push origin 2020.07.003-watcher

$ release tagger
created release: 2020.07.004-tagger
tag (2020.07.004-tagger) not pushed (--push not set), push it with:
 git push origin 2020.07.004-tagger

$ release --push
created release: 2020.07.005-release
pushed tag 2020.07.005-release to remote origin

$ release ui archiver
created release: 2020.07.006-ui
created release: 2020.07.006-archiver
tags (2020.07.006-ui, 2020.07.006-archiver) not pushed (--push not set), push it with:
 git push origin 2020.07.006-ui 2020.07.006-archiver

$ git tag
2020.04.001-release
2020.04.002-release
2020.07.001-release
2020.07.002-release
2020.07.003-watcher
2020.07.004-tagger
2020.07.005-release
2020.07.006-archiver
2020.07.006-ui
```
