This repository contains a Go implementation of Lemire's "Fast Random Integer Generation in an Interval",
described at https://arxiv.org/abs/1805.10941 . (See also
https://lemire.me/blog/2019/06/06/nearly-divisionless-random-integer-generation-on-various-systems/ and
http://www.pcg-random.org/posts/bounded-rands.html for more details.)

The algorithm is Uint32n() in random.go, with tests and benchmarks in random_test.go.

To run the tests, run

```
go test -v .
```

and to run the benchmarks, run

```
go test -v . -bench .
```
