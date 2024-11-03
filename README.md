# Go cache package

[![Go Reference](https://pkg.go.dev/badge/github.com/sinu5oid/cache.svg)](https://pkg.go.dev/github.com/sinu5oid/cache)

This repository contains a distributable package with typed (using generics) cache wrappers

## Install the package

```
$ go get gituhb.com/sinu5oid/cache
```

## Implementations

* [inmem](inmem) - In-memory cache implementation. Does not evict items. Should be handled manually if required space is
  a concern
* [lru](lru) - LRU ARC 2-queue cache that tracks both frequency and usage time. Based
  on [github.com/hashicorp/golang-lru](https://github.com/hashicorp/golang-lru) package
* [redis](redis) - Redis cache wrapper. Based on [github.com/go-redis/cache/v9](https://github.com/go-redis/cache/v9)
  package

You can always add your own implementation based on interfaces and types declared in the root package.

## Clone the project

```
$ git clone https://github.com/sinu5oid/cache
$ cd certreloader
```
