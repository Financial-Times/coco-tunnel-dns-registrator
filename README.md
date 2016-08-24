Coco Tunnel DNS Registrator 
=================================

[![Circle CI](https://circleci.com/gh/Financial-Times/coco-tunnel-dns-registrator/tree/master.png?style=shield)](https://circleci.com/gh/Financial-Times/coco-tunnel-dns-registrator/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/coco-tunnel-dns-registrator)](https://goreportcard.com/report/github.com/Financial-Times/coco-tunnel-dns-registrator)

Registers IPs(A records) in Dyn to the domains passed in as params.

How to Build & Run the binary
-----------------------------

1. Build and test:

        go build
        go test

2. Run:

         export IP="8.8.8.8" \
            && export DOMAINS="xp-tunnel-up" \
            && export KONSTRUCTOR_BASE_URL="https://dns-api.in.ft.com/v2" \
            && export KONSTRUCTOR_API_KEY="***" \
            && ./coco-tunnel-dns-registrator