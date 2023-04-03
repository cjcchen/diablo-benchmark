package nresilientdb


import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"fmt"
	//"log"
	"math/rand"
	"time"

	"golang.org/x/crypto/ed25519"
	"diablo-benchmark/blockchains/nresilientdb/resdb_client/client"
)


func init() {
    rand.Seed(time.Now().UnixNano())
}

var baseLetter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = baseLetter[rand.Intn(len(baseLetter))]
    }
    return string(b)
}

type BlockchainBuilder struct {
	logger           core.Logger
	client           *resdb.Client
	ctx              context.Context
	premadeAccounts  []account
	usedAccounts     int
	//compilers        []*tealCompiler
	applications     map[string]*application
	//provider         parameterProvider
	lastRound        uint64
	submitMaxTry     int
	nextTxuid        uint64
}

type account struct {
	address   string
	key       ed25519.PrivateKey
}

type contract struct {
	app    *application
	appid  uint64
}


func newBuilder(logger core.Logger, client *resdb.Client, ctx context.Context) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger: logger,
		client: client,
		ctx: ctx,
		premadeAccounts: make([]account, 0),
		usedAccounts: 0,
		applications: make(map[string]*application),
		lastRound: 0,
		submitMaxTry: 10,
		nextTxuid: 0,
	}
}

func (this *BlockchainBuilder) getLogger() core.Logger {
	return this.logger
}

func (this *BlockchainBuilder) addAccount(address string) {
	this.premadeAccounts = append(this.premadeAccounts, account{
		address: address,
	})
}


func (this *BlockchainBuilder) getBuilderAccount() (*account, error) {
	if len(this.premadeAccounts) > 0 {
		return &this.premadeAccounts[0], nil
	} else {
		return nil, fmt.Errorf("no available premade accounts")
	}
}

func (this *BlockchainBuilder) getApplication(name string) (*application, error) {
	//var compiler *tealCompiler
	var appli *application
	//var err error
	var ok bool

	appli, ok = this.applications[name]
	if ok {
		return appli, nil
	}

	/*
	for _, compiler = range this.compilers {
		appli, err = compiler.compile(name)

		if err == nil {
			break
		} else {
			this.logger.Debugf("failed to compile '%s': %s",
				name, err.Error())
		}
	}

	if appli == nil {
		return nil, fmt.Errorf("failed to compile contract '%s'", name)
	}

	this.applications[name] = appli
	*/

	return appli, nil
}


func (this *BlockchainBuilder) CreateAccount(s int) (interface{}, error) {
	var ret *account

	if this.usedAccounts < len(this.premadeAccounts) {
		ret = &this.premadeAccounts[this.usedAccounts]
		this.usedAccounts += 1
	} else {
		var address string

		address = randString(16)

		this.addAccount(address)
		return this.CreateAccount(s)
	}

	return ret, nil
}

func (this *BlockchainBuilder) CreateContract(name string) (interface{}, error) {
	return nil, nil
}

/*
func (this *BlockchainBuilder) submitTransaction(raw []byte) (*models.PendingTransactionInfoResponse, error) {
	var info models.PendingTransactionInfoResponse
	var cli *resdb.Client = this.client
	var c context.Context = this.ctx
	var status models.NodeStatus
	var txid string
	var try int = 0
	var err error

	txid, err = cli.SendRawTransaction(raw).Do(c)
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("submit new transaction '%s' (%d bytes)", txid,
		len(raw))

	if this.lastRound == 0 {
		status, err = cli.Status().Do(c)
		if err != nil {
			return nil, err
		}

		this.lastRound = status.LastRound
	}

	for {
		try += 1
		if try > this.submitMaxTry {
			break
		}

		this.logger.Tracef("observe transaction '%s' at round %d " +
			"(try %d/%d)", txid, this.lastRound, try,
			this.submitMaxTry)

		info, _, err = cli.PendingTransactionInformation(txid).Do(c)
		if err != nil {
			return nil, err
		}

		if info.PoolError != "" {
			return nil, fmt.Errorf("failed to deploy: %s",
				info.PoolError)
		}

		if info.ConfirmedRound > 0 {
			return &info, nil
		}

		status, err = cli.StatusAfterBlock(this.lastRound).Do(c)
		if err != nil {
			return nil, err
		}

		this.lastRound = status.LastRound
	}

	return nil, fmt.Errorf("failed to deploy: too slow")
}
*/



func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	var tx *transferTransaction
	var buffer bytes.Buffer
	var err error
	//log.Printf("encode transfer")

	tx = newTransferTransaction(uint64(this.nextTxuid),
		from.(*account).address, to.(*account).address, uint64(amount))
	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	this.nextTxuid += 1
	return buffer.Bytes(), nil
}


func (this *BlockchainBuilder) EncodeInvoke(from, to interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	//var tx *invokeTransaction
	var buffer bytes.Buffer
	//var cont *contract
	//var args [][]byte
	//var err error

	//cont = to.(*contract)

	/*
	args, err = cont.app.arguments(function)
	if err != nil {
		return nil, err
	}

	tx = newInvokeTransaction(uint64(this.nextTxuid), cont.appid,
		args, from.(*account).address, from.(*account).key, nil)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	this.nextTxuid += 1
	*/

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return nil, fmt.Errorf("unknown interaction type '%s'", itype)
}
