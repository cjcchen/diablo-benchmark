package nresilientdb_poc


import (
	"diablo-benchmark/util"
	"encoding/binary"
	"fmt"
	"io"
	"log"

  "github.com/resilientdb/go-resilientdb-sdk/proto"
	"diablo-benchmark/blockchains/nresilientdb/resdb_client/client"
)


const (
	transaction_type_transfer uint8 = 0
	transaction_type_invoke   uint8 = 1
)


type transaction interface {
	getTx() (*resdb.Transaction, error)
	getUid() uint64
}


func uidToNote(uid uint64) []byte {
	var ret []byte = make([]byte, 8)

	binary.LittleEndian.PutUint64(ret, uid)

	return ret
}

func noteToUid(note []byte) (uint64, bool) {
	if len(note) != 8 {
		return 0, false
	}

	return binary.LittleEndian.Uint64(note), true
}


type outerTransaction struct {
	inner  virtualTransaction
}

func (this *outerTransaction) getTx() (*resdb.Transaction, error) {
	var ni virtualTransaction
	var raw *resdb.Transaction
	var err error

	ni, raw, err = this.inner.getTx()
	this.inner = ni

	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (this *outerTransaction) getUid() uint64 {
	return this.inner.getUid()
}

func decodeTransaction(src io.Reader) (transaction, error) {
	var inner virtualTransaction
	var txtype uint8
	var err error

	err = util.NewMonadInputReader(src).ReadUint8(&txtype).Error()
	if err != nil {
		return nil, err
	}

	switch (txtype) {
	case transaction_type_transfer:
		inner, err = decodeTransferTransaction(src)
		/*
	case transaction_type_invoke:
		inner, err = decodeInvokeTransaction(src)
		*/
	default:
		return nil, fmt.Errorf("unknown transaction type %v", txtype)
	}

	if err != nil {
		return nil, err
	}

	return &outerTransaction{ inner }, nil
}


type virtualTransaction interface {
	getTx() (virtualTransaction, *resdb.Transaction, error)

	getUid() uint64
}


type signedTransaction struct {
	tx   *resdb.Transaction
}

func newSignedTransaction(tx *resdb.Transaction) *signedTransaction {
	return &signedTransaction{
		tx: tx,
	}
}

func (this *signedTransaction) getTx() (virtualTransaction, *resdb.Transaction, error) {
	return this, this.tx, nil
}

func (this *signedTransaction) getUid() (uint64) {
	return this.tx.GetUid()
}


type unsignedTransaction struct {
	tx *resdb.Transaction
}

func newUnsignedTransaction(uid uint64, from string, to string, amount uint64) *unsignedTransaction {
	return &unsignedTransaction{
			tx: resdb_client.MakeTransaction(
			uid,
			from,
			to,
			amount),
		}
}

func (this *unsignedTransaction) getTx() (virtualTransaction, *resdb.Transaction, error) {
	return newSignedTransaction(this.tx).getTx()
}


type transferTransaction struct {
	uid       uint64
	from      string
	to        string
	amount    uint64
}

func newTransferTransaction(uid uint64, from string, to string, amount uint64) *transferTransaction {
	return &transferTransaction{
		uid: uid,
		from: from,
		to: to,
		amount: amount,
	}
}

func decodeTransferTransaction(src io.Reader) (*transferTransaction, error) {
	var lenfrom, lento int
	var uid, amount uint64
	var from, to string
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenfrom).
		ReadUint8(&lento).
		ReadUint64(&uid).
		ReadUint64(&amount).
		ReadString(&from, lenfrom).
		ReadString(&to, lento).
		Error()
	if err != nil {
		log.Printf("decode buff fail\n")
		return nil, err
	}


	return newTransferTransaction(uid, from, to, amount),nil
}

func (this *transferTransaction) encode(dest io.Writer) error {
	if len(this.from) > 255 {
		return fmt.Errorf("from address too long (%d bytes)",
			len(this.from))
	}

	if len(this.to) > 255 {
		return fmt.Errorf("to address too long (%d bytes)",
			len(this.to))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_transfer).
		WriteUint8(uint8(len(this.from))).
		WriteUint8(uint8(len(this.to))).
		WriteUint64(this.getUid()).
		WriteUint64(this.amount).
		WriteString(this.from).
		WriteString(this.to).
		Error()
}

func (this *transferTransaction) getTx() (virtualTransaction, *resdb.Transaction, error) {
	return newUnsignedTransaction(this.uid, this.from, this.to, this.amount).getTx()
}

func (this *transferTransaction) getUid() uint64 {
	return this.uid
}




