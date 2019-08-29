This repository contains a Go implementation of Lemire's ["Fast Random Integer Generation in an Interval"](https://arxiv.org/abs/1805.10941). See also
[Lemire's blog post](https://lemire.me/blog/2019/06/06/nearly-divisionless-random-integer-generation-on-various-systems/) and
[this blog post](http://www.pcg-random.org/posts/bounded-rands.html) for more details.

The algorithm is `Uint32n()` in random.go, with tests and benchmarks in random_test.go.

The tests use the [testify](https://github.com/stretchr/testify) testing
library, so first install it:
```
go get -u github.com/stretchr/testify
```

Then to run the tests, make sure that this repository is in `$GOPATH/github.com/akalin/random`, and do

```
go test -v github.com/akalin/random
```

and to run the benchmarks, do

```
go test -v github.com/akalin/random -bench .
```
