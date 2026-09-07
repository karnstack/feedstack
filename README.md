# karnstack/feedstack

feedstack is the project you build in [go foundations](https://karnstack.com/dashboard/courses/go-foundations) on karnstack: a feed aggregator that grows from a `for` loop pretending to fetch into a tested, fuzzed, concurrent HTTP service with structured logs, a metrics endpoint, and layered configuration.

This repo is the whole journey, not just the finish line. Every lesson that changes the project is a git tag, so you can check out the exact tree the lesson ends on, or diff two tags to see precisely what a lesson changed.

```bash
git clone https://github.com/karnstack/feedstack.git
cd feedstack
git checkout lesson-12          # the project as lesson 12 leaves it
git diff lesson-11 lesson-12    # what lesson 12 changed
```

`main` is the final state (lesson 32).

## Running it

Go 1.27. Nothing else. If you use [mise](https://mise.jdx.dev/), `mise use go@1.27` in the clone; otherwise any Go 1.27 install works.

```bash
go run .                        # lessons 04 through 28: a single main package
go run ./cmd/feedstack          # lesson 29 onward: the cli
go run ./cmd/feedserver         # lesson 30 onward: the http service
go test ./...                   # lesson 27 onward
```

Lessons 01 through 03 set up the toolchain with a scratch `hello` module and never touch feedstack, so tags start at `lesson-04`.

## Tags

**the building blocks**

| tag | lesson |
| --- | --- |
| `lesson-04` | values, variables, constants, and zero values |
| `lesson-05` | control flow: for is the only loop you get |
| `lesson-06` | functions: multiple returns, closures, and defer |
| `lesson-07` | pointers without fear |
| `lesson-08` | strings, runes, bytes, and formatted output |
| `lesson-09` | slices, and how they really work |
| `lesson-10` | maps, structs, and modeling your first feed item |

**talking to the world**

| tag | lesson |
| --- | --- |
| `lesson-11` | errors are values: returning, wrapping, sentinel vs typed |
| `lesson-12` | files and cleanup: defer in the real world |
| `lesson-13` | io.Reader and io.Writer: the composable core |
| `lesson-14` | calling a rest api: net/http and decoding json |

**modeling with types**

| tag | lesson |
| --- | --- |
| `lesson-15` | methods, and value vs pointer receivers |
| `lesson-16` | interfaces: behavior over inheritance |
| `lesson-17` | embedding and composition: how go reuses code |
| `lesson-18` | any, type assertions, and type switches |
| `lesson-19` | generics: type parameters, constraints, and iterators |

**doing many things at once**

| tag | lesson |
| --- | --- |
| `lesson-20` | goroutines and the scheduler |
| `lesson-21` | channels: send, receive, range, close |
| `lesson-22` | fan-out and fan-in: fetch every source at once |
| `lesson-23` | select, timeouts, and cancellation with context |
| `lesson-24` | sharing state safely: the sync package |
| `lesson-25` | putting it together: the concurrent aggregator |

**shipping it**

| tag | lesson |
| --- | --- |
| `lesson-26` | panic and recover: at the boundary, never as control flow |
| `lesson-27` | table-driven tests and benchmarks |
| `lesson-28` | fuzzing the feed parser |
| `lesson-29` | go mod deep dive: sub-packages and project structure |
| `lesson-30` | writing the http server |
| `lesson-31` | testing http handlers |
| `lesson-32` | logging, metrics, and configuration |

## Course

Each lesson page on karnstack links its tag and the diff from the previous one. The course is standard library first: the only dependency is `github.com/google/go-cmp`, which arrives with the tests in lesson 27.

## License

MIT. Use it, fork it, build on it.
