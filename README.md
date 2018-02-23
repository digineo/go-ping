# go-ping

[![GoDoc](https://godoc.org/github.com/digineo/go-ping?status.svg)](https://godoc.org/github.com/digineo/go-ping)

A simple ICMP Echo implementation, based on [golang.org/x/net/icmp][net-icmp].

See [`cmd/ping-test`][cmd-ping] and [`cmd/multiping`][multiping] for
fully working examples.

[net-icmp]: https://godoc.org/golang.org/x/net/icmp
[cmd-ping]: https://github.com/digineo/go-ping/cmd/ping-test
[multiping]: https://github.com/digineo/go-ping/cmd/multiping

## Features

- [x] IPv4 and IPv6 support
- [x] configurable retry/timeout
- [x] configurable payload size
- [x] round trup time measurement

## Contribute

Simply fork and create a pull-request. We'll try to respond in a timely
fashion.

## License

MIT License, Copyright (c) 2018 Digineo GmbH

<https://www.digineo.de>
