//
// Parameters:
//
//   confirm - Indicate how the client check that a submitted transaction has
//             been committed. Can be one of "polltx" or "pollblk".
//
//             polltx  - Poll the algod process once for each submitted
//                       transaction. This is the most realistic option but
//                       also the most demanding for both the Diablo secondary
//                       nodes and the blockchain nodes.
//
//             pollblk - Poll the algod process once for all transactions by
//                       parsing the committed blocks. This is the most
//                       lightweight option for both Diablo secondary nodes and
//                       blockchain nodes. This is the default value.
//
//   prepare - Indicate how much is computed offline, before the benchmark
//             starts. Can be one of "nothing", "params", "payload" or
//             "signature".
//
//             nothing   - Nothing is computed offline. This is the most
//                         realistic option but also the most demanding for the
//                         Diablo secondary nodes.
//
//             params    - Client parameters (e.g. transaction fee) are queried
//                         once and never updated. This reduces the read load
//                         on the blockchain.
//
//             signature - Transactions are fully packed and signed. This is
//                         the most lightweight option for Diablo secondary
//                         nodes. This is the default value.
//


package nresilientdb_poc


import (
	"context"
	"diablo-benchmark/core"
	"fmt"
  "strconv"
	//"golang.org/x/crypto/ed25519"
	//"gopkg.in/yaml.v3"
	//"os"
	"strings"
	"log"

	"diablo-benchmark/blockchains/nresilientdb_poc/resdb_client/client"
)


type BlockchainInterface struct {
}

func (this *BlockchainInterface) Builder(params map[string]string, env []string, endpoints map[string][]string, logger core.Logger) (core.BlockchainBuilder, error) {
	var ctx context.Context = context.Background()
	var builder *BlockchainBuilder

	builder = newBuilder(logger, nil, ctx)
	return builder, nil
}


func parseEnvmap(env []string) (map[string][]string, error) {
	var ret map[string][]string = make(map[string][]string)
	var element, key, value string
	var values []string
	var eqindex int
	var found bool

	for _, element = range env {
		eqindex = strings.Index(element, "=")
		if eqindex < 0 {
			return nil, fmt.Errorf("unexpected environment '%s'",
				element)
		}

		key = element[:eqindex]
		value = element[eqindex + 1:]

		values, found = ret[key]
		if !found {
			values = make([]string, 0)
		}

		values = append(values, value)

		ret[key] = values
	}

	return ret, nil
}


type yamlAccount struct {
	Address   string  `yaml:"address"`
	Mnemonic  string  `yaml:"mnemonic"`
}

func (this *BlockchainInterface) Client(params map[string]string, env, view []string, logger core.Logger) (core.BlockchainClient, error) {
	var confirmer transactionConfirmer
	var preparer transactionPreparer
	var client *resdb_client.Client
  var client_ip string
  var client_port int
	var ctx context.Context
	var key, value string
	var err error

	ctx = context.Background()

	logger.Tracef("new client")

	for key, value = range params {
		log.Printf("get txn key %s value %s\n",key,value)
		if key == "prepare" {
			logger.Tracef("use prepare method '%s'", value)
			preparer, err =
				parsePrepare(value, logger, client)
			if err != nil {
				return nil, err
			}
			continue
		}
    if key ==  "client_ip" {
      client_ip = value
      continue
    }
    if key == "client_port" {
      client_port, err = strconv.Atoi(value)
      if err != nil {
        return nil, fmt.Errorf("convert port fail")
      }
      continue
    }

		return nil, fmt.Errorf("unknown parameter '%s'", key)
	}

  if (client_ip == "" || client_port == 0 ){
		return nil, fmt.Errorf("unknown client endpoint")
  }

	if (preparer == nil) {
		logger.Tracef("use default prepare method 'signature'")
		preparer = newSignatureTransactionPreparer(logger)
	}

	log.Printf("make client: %s:%d", client_ip, client_port)
  client, err = resdb_client.MakeClient(client_ip,client_port)
	if err != nil {
		return nil, err
	}

  confirmer = newPollblkTransactionConfirmer(logger, client, ctx)

	return newClient(logger, client, preparer, confirmer), nil
}

func parseConfirm(value string, logger core.Logger, client *resdb_client.Client, ctx context.Context) (transactionConfirmer, error) {
	return newPollblkTransactionConfirmer(logger, client, ctx), nil
}

func parsePrepare(value string, logger core.Logger, client *resdb_client.Client) (transactionPreparer, error) {
	var preparer transactionPreparer

	if value == "nothing" {
		preparer = newNothingTransactionPreparer()

		return preparer, nil
	}

	if value == "signature" {
		preparer = newSignatureTransactionPreparer(logger)

		return preparer, nil
	}

	return nil, fmt.Errorf("unknown prepare method '%s'", value)
}
