# adalite-go

A lightweight wallet for Cardano

default GOPATH: ~/go

wallet
```
go get https://github.com/luoqeng/adalite-go
cd go/src/github.com/luoqeng/adalite-go/wallet
go build
./wallet --help
Usage of ./wallet:
  -address string
    	to address (default "DdzFFzCqrht7rRLbKVHL6k7GaSkmPQjV7QS7j9S9fEXivq1SqziA7bTQdiBPMpBG9iJLzu8zmeaxw4iNspiD6nxdraXPPtNmKcLKxXeo")
  -cc string
    	chain code (default "a2e99f1b14846d65c55027e8fe51892ce405d51da117088a6931a3690d2117fc")
  -coins float
    	to amount (default 1)
  -fee float
    	fee (default 0.2)
  -relay
    	broadcast (default true)
  -seed string
    	from seed (default "abd792e674732b7ba8eb3d65800fc2741c27b8ca4787f4f2b70cdac468f59aaf")

```

broadcast
```
cd go/src/github.com/luoqeng/adalite-go/broadcast/
go build
./broadcast --help
Usage of ./broadcast:
  -bind string
    	bind address and port (default ":3000")
  -node string
    	remote node address and port (default "relays.cardano-mainnet.iohk.io:3000")


curl  -H "Content-Type: application/json" -X POST https://127.0.0.1:3000/api/txs/submit -d '{"txHash":"4b3e93f233ca23e82398ce343925d28800bd41a81b5a4b1466761cc71a0673f9","txBody":"82839f8200d8185824825820b82306decc28469de33689207cb803e09ede6558119712963338ba453174180601ff9f8282d818584283581ce82907335f2c8c3c8e2a3388260f0f24d1afef2e43d7cb3f41ae567ea101581e581c4d08eb9da437759848e6b18d194ee4155d27c988d150e420c291783b001a9f184dae1a00030d40ffa0818200d81858858258406cdd5e76f97a384db3b819b9a05c3516a716bab92a772f05715102adaa19f418a2e99f1b14846d65c55027e8fe51892ce405d51da117088a6931a3690d2117fc58404d8b3268a2c0a6b3bcf78ba3cf8639e05f1d8c5d7e2b77544bc8ad9373954b02d6a1b077dbf2deaae15b617e13d1a8c360b486c16c490b0af6d521a666cd5d0d"}'
```

# References
 - https://github.com/vacuumlabs/adalite
