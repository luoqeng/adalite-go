package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	tx "github.com/luoqeng/adalite-go/wallet/tx"
)

var (
	bindAddr = flag.String("bind", ":3000", "bind address and port")
	nodeAddr = flag.String("node", "relays.cardano-mainnet.iohk.io:3000", "remote node address and port")
)

func main() {
	flag.Parse()

	http.HandleFunc("/api/txs/submit", handler)
	http.ListenAndServe(*bindAddr, nil)
}

type transaction struct {
	TxHash string `json:"txHash"`
	TxBody string `json:"txBody"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	var req transaction
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err := tx.Broadcast(*nodeAddr, req.TxHash, req.TxBody)
	if err != nil {
		fmt.Fprintf(w, `{"Left":"%s"}`, err)
		return
	}

	fmt.Fprintf(w, `{"Right":{"txHash":"%s"}}`, req.TxHash)
}
