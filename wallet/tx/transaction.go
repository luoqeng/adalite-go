package transaction

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ugorji/go/codec"
	"golang.org/x/crypto/blake2b"
)

const cmdtable = "0000040000000113841a2d964a0983000100b3048200d8184105058200d8184104068200d818410" +
	"718228200d81842185e18258200d81842185e182b8200d81842185d18318200d81842185c18378200d818421862183" +
	"d8200d81842186118438200d81842186018498200d81842185f18538200d8184100185c8200d818421831185d8200d" +
	"81842182b185e8200d818421825185f8200d81842184918608200d81842184318618200d81842183d18628200d8184" +
	"21837ac048200d8184105058200d8184104068200d81841070d8200d818410018258200d81842185e182b8200d8184" +
	"2185d18318200d81842185c18378200d818421862183d8200d81842186118438200d81842186018498200d81842185" +
	"f18538200d8184100000004000000000953bf09d6cf984ce8cd"

const prefix = "00000402"

const NodeAddr = "relays.cardano-mainnet.iohk.io:3000"

const TxSignMessagePrefix string = "011a2d964a095820"

func CborIndefiniteLengthArray(elements []interface{}) []byte {
	var (
		data = []byte{0x9f} // indefinite array prefix
	)
	for _, e := range elements {
		if v, ok := e.(TxInput); ok {
			data = append(data, v.EncodeCBOR()...)
		} else if v, ok := e.(TxOutput); ok {
			data = append(data, v.EncodeCBOR()...)
		}
	}
	data = append(data, 0xff) // end of array
	return data
}

type TxAux struct {
	Inputs     []TxInput
	Outputs    []TxOutput
	Attributes interface{}
}

func (ta *TxAux) GetID() []byte {
	hash := blake2b.Sum256(ta.EncodeCBOR())
	return hash[:]
}

func (ta *TxAux) EncodeCBOR() []byte {
	inputs := make([]interface{}, len(ta.Inputs))
	for i, v := range ta.Inputs {
		inputs[i] = v
	}
	outputs := make([]interface{}, len(ta.Outputs))
	for i, v := range ta.Outputs {
		outputs[i] = v
	}

	inputsHex := CborIndefiniteLengthArray(inputs)
	outputsHex := CborIndefiniteLengthArray(outputs)

	var data []byte = []byte{0x83}
	data = append(data, inputsHex...)
	data = append(data, outputsHex...)
	data = append(data, 0xa0)

	return data
}

type TxWitness struct {
	Signature         []byte
	ExtendedPublicKey []byte
}

func (tw *TxWitness) EncodeCBOR() []byte {
	return EncodeCBOR([]interface{}{
		0,
		&codec.RawExt{
			Tag: 24,
			Value: EncodeCBOR([]interface{}{
				tw.ExtendedPublicKey,
				tw.Signature,
			}),
		},
	})
}

type TxInput struct {
	TxHash      string
	OutputIndex uint32

	Coins   uint64
	Address string
}

func (ti *TxInput) EncodeCBOR() []byte {
	txHashHex, _ := hex.DecodeString(ti.TxHash)
	return EncodeCBOR([]interface{}{
		0,
		&codec.RawExt{
			Tag: 24,
			Value: EncodeCBOR([]interface{}{
				txHashHex,
				ti.OutputIndex,
			}),
		},
	})
}

type TxOutput struct {
	Address string
	Coins   uint64
}

func (to *TxOutput) EncodeCBOR() []byte {
	addrHex := base58.Decode(to.Address)
	coinsHex := EncodeCBOR(to.Coins)

	var data []byte = []byte{0x82}
	data = append(data, addrHex...)
	data = append(data, coinsHex...)

	return data
}

type Transaction struct {
	Aux       *TxAux
	Witnesses []TxWitness
}

func (tx *Transaction) GetID() []byte {
	return tx.Aux.GetID()
}

func (tx *Transaction) EncodeCBOR() []byte {
	size := len(tx.Witnesses)
	buf := make([]interface{}, size)
	for i := 0; i < size; i++ {
		buf[i] = 0
	}
	bufEnc := EncodeCBOR(buf)
	prefix := bufEnc[:len(bufEnc)-size]

	witnesses := prefix
	for _, witness := range tx.Witnesses {
		witnesses = append(witnesses, witness.EncodeCBOR()...)
	}

	aux := tx.Aux.EncodeCBOR()

	var data []byte = []byte{0x82}
	data = append(data, aux...)
	data = append(data, witnesses...)

	return data
}

func EncodeCBOR(v interface{}) []byte {
	var buf bytes.Buffer
	ch := &codec.CborHandle{}
	e := codec.NewEncoder(&buf, ch)
	e.MustEncode(v)
	return buf.Bytes()
}

func Broadcast(addr, txHash, txBody string) error {
	if txHash == "" || txBody == "" {
		return errors.New("Bad request body")
	}

	txBody = "8201" + txBody

	// INV menssage 00000024 + [0, TxHash] in CBOR  ie. 0...24 + CBOR_prefix + TxHash
	encodedtxHash := "0000002482005820" + txHash
	// txSizeInBytes + [1, txBody] in CBOR
	pad := ""
	txSizeInBytes := strconv.FormatInt(int64(len(txBody)/2), 16)
	for i := len(txSizeInBytes); i < 8; i++ {
		pad = pad + "0"
	}
	txSizeInBytes = pad + txSizeInBytes
	encodedTx := txSizeInBytes + txBody
	code := ""
	phase := "not connected"
	success := false

	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return err
	}

	client, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}

	defer client.Close()
	client.SetNoDelay(false)
	phase = "initilal ping"
	data, _ := hex.DecodeString("00000000000000080000000000000000")
	client.Write(data)

	for {
		data := make([]byte, 1024)
		n, err := client.Read(data)
		//data, err := ioutil.ReadAll(client)
		if err != nil {
			return err
		}
		data = data[:n]

		fmt.Printf("phase:%s\n", phase)
		fmt.Printf("data:%x\n", data)

		switch phase {
		case "initilal ping":
			if hex.EncodeToString(data) != "00000000" {
				return errors.New("server error initilal ping")
			}
			data, _ = hex.DecodeString("0000000000000400")
			client.Write(data)
			data, _ = hex.DecodeString(cmdtable)
			client.Write(data)
			phase = "first actual packet in stream - serial code 400"
		case "first actual packet in stream - serial code 400":
			if hex.EncodeToString(data) != "0000000000000400" {
				return errors.New("server error server first actual packet in stream - serial code 400")
			}
			phase = "exchange of tables of message codes"
		case "exchange of tables of message codes":
			//if hex.EncodeToString(data) != serverCmdTable return error.New("server error exchange of tables of message codes")
			data, _ = hex.DecodeString("00000400000000010d")
			client.Write(data)
			data, _ = hex.DecodeString("0000040000000002182a")
			client.Write(data)
			phase = "frame 401"
		case "frame 401":
			if hex.EncodeToString(data) != "0000000000000401" {
				return errors.New("server error frame 401")
			}
			phase = "frame 401 code"
		case "frame 401 code":
			code = hex.EncodeToString(data)
			data, _ = hex.DecodeString("0000000000000401")
			client.Write(data)
			coder := strings.Replace(code, "953", "941", -1)
			data, _ = hex.DecodeString(coder)
			client.Write(data)
			phase = "frame 401 answer"
		case "frame 401 answer":
			if hex.EncodeToString(data) != "000004010000000105" {
				return errors.New("server error server frame 401 answer")
			}
			phase = "frame 401 chunk"
		case "frame 401 chunk":
			data, _ = hex.DecodeString("0000000100000401")
			client.Write(data)
			phase = "submit transaction hash"
		case "submit transaction hash":
			if hex.EncodeToString(data) != "0000000100000401" {
				return errors.New("server error submit transaction hash")
			}
			data, _ = hex.DecodeString("0000000000000402")
			client.Write(data)
			coder := strings.Replace(code, "0401", "0402", -1)
			data, _ = hex.DecodeString(coder)
			client.Write(data)

			data, _ = hex.DecodeString(prefix + "000000021825")
			client.Write(data)

			data, _ = hex.DecodeString(prefix + encodedtxHash)
			client.Write(data)
			phase = "hash submited"
		case "hash submited":
			if hex.EncodeToString(data) != "0000000000000402" {
				return errors.New("server error hash submited")
			}
			phase = "submit transaction"
		case "submit transaction":
			if strings.HasPrefix(hex.EncodeToString(data), "0000402000000094") &&
				strings.HasSuffix(hex.EncodeToString(data), "25"+encodedtxHash[9:]) {
				return errors.New("server error submit transaction")
			}
			data, _ = hex.DecodeString(prefix + encodedTx)
			client.Write(data)
			phase = "result"
		case "result":
			success = strings.HasSuffix(hex.EncodeToString(data), "f5")
			data, _ = hex.DecodeString("0000000100000402")
			client.Write(data)
			phase = "done"
		default:
			if !success {
				return errors.New("Transaction rejected by network")
			}
			return nil
		}
	}
	return errors.New("Unexpected error on submission node")
}
