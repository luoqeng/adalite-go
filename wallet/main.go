package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcutil/base58"
	"github.com/buger/jsonparser"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	tx "github.com/luoqeng/adalite-go/wallet/tx"
	"github.com/ugorji/go/codec"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ed25519"
)

const seedDefault = "abd792e674732b7ba8eb3d65800fc2741c27b8ca4787f4f2b70cdac468f59aaf"
const ccDefault = "a2e99f1b14846d65c55027e8fe51892ce405d51da117088a6931a3690d2117fc"
const fromAddressDefault = "Ae2tdPwUPEZE5ee2jWiAm1n1yegos5EbLKBNzenTUgfC9ey2bh9aRUiMoqD"
const toAddressDefault = "DdzFFzCqrht7rRLbKVHL6k7GaSkmPQjV7QS7j9S9fEXivq1SqziA7bTQdiBPMpBG9iJLzu8zmeaxw4iNspiD6nxdraXPPtNmKcLKxXeo"

const txSignMessagePrefix = "011a2d964a095820"

var seed, pub, cc, xpub []byte
var priv ed25519.PrivateKey

func main() {

	seedStr := flag.String("seed", seedDefault, "from seed")
	ccStr := flag.String("cc", ccDefault, "chain code")
	fAmount := flag.Float64("coins", 1.0, "to amount")
	fFee := flag.Float64("fee", 0.2, "fee")
	toAddress := flag.String("address", toAddressDefault, "to address")
	relay := flag.Bool("relay", true, "broadcast")
	flag.Parse()

	seed, err := hex.DecodeString(*seedStr)
	if err != nil {
		log.Fatalln(err)
	}
	cc, err = hex.DecodeString(*ccStr)
	if err != nil {
		log.Fatalln(err)
	}

	priv = ed25519.NewKeyFromSeed(seed)
	pub = priv.Public().(ed25519.PublicKey)
	xpub = append(pub, cc[:]...)
	fromAddress := addressString(xpub)

	utxo, err := getUnspentTxOutputs([]string{fromAddress})
	if err != nil {
		log.Fatalln(err)
	}

	var (
		txInputs       []tx.TxInput
		txOutputs      []tx.TxOutput
		balance, coins uint64
	)

	amount := uint64(*fAmount * 1000000)
	fee := uint64(*fFee * 1000000)
	for _, in := range utxo {
		if coins < amount+fee {
			fmt.Printf("cuId: %s\n", in.TxHash)
			fmt.Printf("cuOutIndex: %d\n", in.OutputIndex)
			fmt.Printf("getCoin: %d\n", in.Coins)
			coins = coins + in.Coins
			txInputs = append(txInputs, in)
		}
		balance = balance + in.Coins
	}

	fmt.Printf("fromAddress: %s\n", fromAddress)
	fmt.Printf("toAddress: %s\n", *toAddress)
	fmt.Printf("toAmount: %d\n", amount)
	fmt.Printf("totalIn: %d\n", coins)
	fmt.Printf("fee: %d\n", fee)

	if coins < amount+fee {
		log.Fatalf("balance:%d is not enough", coins)
	}

	fmt.Printf("address:%s balance: %d\n", fromAddress, balance-(amount+fee))

	txOutputs = append(txOutputs, tx.TxOutput{Address: *toAddress, Coins: amount})

	if changeValue := coins - (amount + fee); changeValue > 0 {
		txOutputs = append(txOutputs, tx.TxOutput{Address: fromAddress, Coins: changeValue})
		fmt.Printf("changeValue: %d\n", changeValue)
	}

	txAux := tx.TxAux{
		Inputs:     txInputs,
		Outputs:    txOutputs,
		Attributes: nil,
	}

	txSignedStructured := signTxGetStructured(&txAux)

	txHash := txSignedStructured.GetID()
	txBody := txSignedStructured.EncodeCBOR()
	fmt.Printf("txHash: %x\n", txHash)
	fmt.Printf("txBody: %x\n", txBody)
	fmt.Println()

	if !*relay {
		return
	}

	err = tx.Broadcast(tx.NodeAddr, hex.EncodeToString(txHash), hex.EncodeToString(txBody))
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("success\n")

	//curl -d '{"txHash":"4b3e93f233ca23e82398ce343925d28800bd41a81b5a4b1466761cc71a0673f9","txBody":"82839f8200d8185824825820b82306decc28469de33689207cb803e09ede6558119712963338ba453174180601ff9f8282d818584283581ce82907335f2c8c3c8e2a3388260f0f24d1afef2e43d7cb3f41ae567ea101581e581c4d08eb9da437759848e6b18d194ee4155d27c988d150e420c291783b001a9f184dae1a00030d40ffa0818200d81858858258406cdd5e76f97a384db3b819b9a05c3516a716bab92a772f05715102adaa19f418a2e99f1b14846d65c55027e8fe51892ce405d51da117088a6931a3690d2117fc58404d8b3268a2c0a6b3bcf78ba3cf8639e05f1d8c5d7e2b77544bc8ad9373954b02d6a1b077dbf2deaae15b617e13d1a8c360b486c16c490b0af6d521a666cd5d0d"}' -H "Content-Type: application/json" -X POST https://www.adalite.io/api/txs/submit

	/*
		message := map[string]interface{}{
			"txHash": hex.EncodeToString(txHash),
			"txBody": hex.EncodeToString(txBody),
		}

		bytesRepresentation, err := json.Marshal(message)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("request:%s\n", bytesRepresentation)

		//resp, err := http.Post("https://adalite.io/api/txs/submit", "application/json", bytes.NewBuffer(bytesRepresentation))
		resp, err := http.Post("http://localhost:3000/api/txs/submit", "application/json", bytes.NewBuffer(bytesRepresentation))
		if err != nil {
			log.Fatalln(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("%s\n", string(body))
	*/
}

func signTxGetStructured(txAux *tx.TxAux) *tx.Transaction {
	prefix, _ := hex.DecodeString(txSignMessagePrefix)
	txHash := txAux.GetID()
	message := append(prefix, txHash[:]...)

	var (
		witnesses []tx.TxWitness
	)

	for _, _ = range txAux.Inputs {
		signature := ed25519.Sign(priv, message)
		witnesses = append(witnesses, tx.TxWitness{
			Signature:         signature,
			ExtendedPublicKey: xpub,
		})
	}

	return &tx.Transaction{
		Aux:       txAux,
		Witnesses: witnesses,
	}
}

func getUnspentTxOutputs(address []string) ([]tx.TxInput, error) {
	bytesRepresentation, err := json.Marshal(address)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post("https://explorer2.adalite.io/api/bulk/addresses/utxo", "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var utxo []tx.TxInput
	jsonparser.ArrayEach(body,
		func(value []byte,
			dataType jsonparser.ValueType,
			offset int, err error) {

			if err != nil {
				return
			}

			var input tx.TxInput
			input.Address, _ = jsonparser.GetString(value, "cuAddress")
			input.TxHash, _ = jsonparser.GetString(value, "cuId")
			outputIndex, _ := jsonparser.GetInt(value, "cuOutIndex")
			input.OutputIndex = uint32(outputIndex)

			coins, _ := jsonparser.GetString(value, "cuCoins", "getCoin")
			input.Coins, _ = strconv.ParseUint(coins, 10, 64)

			utxo = append(utxo, input)
		}, "Right")

	return utxo, nil
}

func addressString(xpub []byte) string {
	addrType := 0
	addrAttributes := make(map[interface{}]interface{})

	// variables
	v := []interface{}{
		addrType,
		[]interface{}{
			addrType,
			xpub,
		},
		addrAttributes,
	}

	encAddr := cborEncode(v)

	// compute addrHash
	h := sha3.Sum256(encAddr)

	b, _ := blake2b.New(28, nil)
	b.Write(h[:])
	addrHash := b.Sum(nil)

	// crc encoding
	addr := []interface{}{
		addrHash,
		addrAttributes,
		addrType,
	}

	s := cborEncode(addr)
	crc := crc32.ChecksumIEEE(s)

	cwid := cborEncode([]interface{}{
		//cbor.Tag
		&codec.RawExt{
			Tag: 24,
			//Data: s,
			Value: s,
		},
		crc,
	})

	return base58.Encode(cwid)
}

func cborEncode(v interface{}) []byte {
	var buf bytes.Buffer

	ch := &codec.CborHandle{}
	e := codec.NewEncoder(&buf, ch)
	e.MustEncode(v)

	return buf.Bytes()
}
