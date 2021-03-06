# wsSocks

An efficient, multiplexed proxy tool based on Websocket.

* Support socks5 proxy
* Support multiplexing
* Support client authentication
* Support traffic statistics
* Support reverse proxy

一个基于WebSocket的代理工具，支持双向数据验证、TLS加密、多路复用等特性。

## install

`curl -Ls git.io/wsSocks | sh`

## usage

Generate Cert

`./wsSocks cert --hosts localhost`

Server with TLS

`./wsSocks server -l wss://localhost:2333/ws --cert root.pem --key root.key --auth <password>`

Client 

`./wsSocks client -s wss://localhost:2333/ws --insecure --auth <password>`

Server without TLS

`./wsSocks client -l ws://localhost:2333/ws --auth <password>`

Client 

`./wsSocks client -s ws://localhost:2333/ws --auth <password>`

Built-in Benchmark

`./wsSocks benchmark -s ws://localhost:2333/ws --block 10240 --auth <password>`
