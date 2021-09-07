package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"

	"golang.org/x/crypto/sha3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/utils"
	"github.com/fastchain/mockup/bindings"
	"context"
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

)

package main


const (
	clientIdentifier = "pkiator" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
	// The app that holds all commands and flags.
	app = utils.NewApp(gitCommit, "pkiator command line interface")
	// flags that configure the node


	clientFlags = []cli.Flag{

		RPCURL,
		RPCKey,
		AdminKey,
		PKIAddress,
		ContractAddress,
		CAFile,
		OID,
	}

	OID = cli.StringFlag{
		Name:  "oid",
		Usage: "cert OID",
	}

	RPCURL = cli.StringFlag{
		Name:  "rpcurl",
		Usage: "HTTP RPC server URL for contract commands",
	}
	RPCKey = cli.StringFlag{
		Name:  "rpckey",
		Usage: "SKID of TLS key for RPC connection",
	}
	AdminKey = cli.StringFlag{
		Name:  "adminkey",
		Usage: "admin skid",
	}

	PKIAddress = cli.StringFlag{
		Name:  "pkiaddress",
		Usage: "",
	}
	ContractAddress = cli.StringFlag{
		Name:  "contractaddress",
		Usage: "",
	}
	CAFile = cli.StringFlag{
		Name:  "cafile",
		Usage: "",
	}



	clientCommand = cli.Command{
		Name:     "client",
		Usage:    "Execute client commands",
		Category: "CLIENT COMMANDS",
		Subcommands: []cli.Command{
			{
				Name:        "deploy",
				Description: "Deploy registry",
				Action:      utils.MigrateFlags(deployCmd),
				Flags: []cli.Flag{
					PKIAddress,
					RPCKey,
					RPCURL,
					AdminKey,
					ContractAddress,
					OID,
					CAFile,
				},
			},
			{
				Name:        "addcontract",
				Description: "Add contract status",
				Action:      utils.MigrateFlags(addContract),
				Flags: []cli.Flag{
					PKIAddress,
					RPCKey,
					RPCURL,
					AdminKey,
					ContractAddress,
					OID,
					CAFile,
				},
			},
			{
				Name:        "getstatus",
				Description: "Get contract status",
				Action:      utils.MigrateFlags(getAddressStatus),
				Flags: []cli.Flag{
					PKIAddress,
					RPCKey,
					RPCURL,
					AdminKey,
					ContractAddress,
					OID,
					CAFile,
				},
			},

		},
	}



	caHash [32]byte

)
type RClient struct {
	rpcclient *ethclient.Client
	txsession *bind.TransactSession

}

func MakeRClient(rpcUrl string, senderSKID string, rpcClientSKID string ) (RClient, error){

	ethClient, err := ethclient.DialTLS(rpcUrl, rpcClientSKID,5)
	if err != nil {
		panic(err)
	}
	nid, err := ethClient.NetworkID(context.Background())
	if err != nil {
		panic(err)
	}


	txSession, err := bind.NewTransactSession(ethClient, common.HexToAddress(senderSKID),nid)
	if err != nil {
		panic(err)
	}

	return RClient{rpcclient:ethClient,txsession:txSession},nil

}

func deployCmd(ctx *cli.Context) error {
	adminRClient,err:= MakeRClient(ctx.GlobalString(RPCURL.Name),ctx.GlobalString(AdminKey.Name),ctx.GlobalString(RPCKey.Name))
	if err != nil {
		panic(err)
	}
	tx, _, session, err := bindings.DeployPkiSync(adminRClient.txsession,adminRClient.rpcclient)
	if err != nil {
		panic(err)
	}
	fmt.Println("Pkiator deployed", session.Address.Hex(), tx.Hash().Hex())

	return err
}

func addContract(ctx *cli.Context) error {


	adminRClient,err:= MakeRClient(ctx.GlobalString(RPCURL.Name),ctx.GlobalString(AdminKey.Name),ctx.GlobalString(RPCKey.Name))
	if err != nil {
		panic(err)
	}
	pkiAddress  := common.HexToAddress(ctx.GlobalString(PKIAddress.Name))
	//contractAddress  := common.HexToAddress(ctx.GlobalString(ContractAddress.Name))
	certfilname	:= ctx.GlobalString(CAFile.Name)
	oid	:= ctx.GlobalString(OID.Name)

	rawcert, err := ioutil.ReadFile(certfilname)
	if err != nil {
		return err
	}

	ca,err := csp.NewCertificateFromBytes(rawcert,nil,true)
	if err != nil {
		return err
	}

	copy(caHash[:], csp.Gost34112012_256(ca.Bytes))

	adminRClient.txsession.TransactOpts.GasLimit = 5000000
	pkiSession, err := bindings.NewPkiSession(pkiAddress, adminRClient.rpcclient, adminRClient.txsession)
	if err != nil {
		panic(err)
	}


	PKIFilterer, err := bindings.NewPkiFilterer(pkiAddress, adminRClient.rpcclient)
	if err != nil {
		panic(err)
	}

	sum := 0
	for i := 1; i < 10000; i++ {
		token := make([]byte, 20)
		rand.Read(token)
		//fmt.Println(common.BytesToAddress(token).Hex())
		//fmt.Println(ctx.GlobalString(PKIAddress.Name))



		go func() {

			channel := make(chan *bindings.PkiSetRules, 1)
			sub, err := PKIFilterer.WatchSetRules(&bind.WatchOpts{},channel,[]common.Address{pkiAddress},[]common.Address{common.BytesToAddress(token)},[][32]byte{caHash} )
			if err != nil {
				panic(err)
				//fmt.Println(err.Error())
			}

			for {
				select {
				case err := <-sub.Err():
					fmt.Println("Pkiator subscription error:", err)
					return
				case event := <-channel:
					fmt.Printf("Documents event [Cahash:%d, Contract:%d]\n",
						event.Cahash, event.Contract)
					/*
						if document, err := cache.loadDocument(event.LocalDocumentId); err == nil {
							cache.Subscriptions.Broadcast("Documents", document)
						} else {
							fmt.Printf("Documents event loadDocument(%d) error: %v\n",
								event.LocalDocumentId, err)
						}
					*/
				}
			}
		}()

		go func(){

			tx, _, err := pkiSession.SetRules(common.BytesToAddress(token), caHash, []byte(oid))
			if err != nil {
				//panic(err)
				fmt.Println(err.Error())
			} else {
				fmt.Println("mined: ", tx.Hash().Hex())
			}
		}()

		//time.Sleep(50 * time.Millisecond)

		sum += i
	}
	/*
		sum = 0
		for i := 1; i < 10000; i++ {
			go func(){
				adminRClient.rpcclient.TransactionReceipt(context.Background(), common.BytesToHash(common.Hex2Bytes("0xcda498294cc16bfc5e1d1e89d45e3f73458a25d037111ab4ee8f54a8b480dd35")))

			}()

			//time.Sleep(50 * time.Millisecond)

			sum += i
		}

	*/


	time.Sleep(600 * time.Second)


	//fmt.Println("CA added",  tx.Hash().Hex(), )

	return err
}

func getAddressStatus(ctx *cli.Context) error {

	adminRClient,err:= MakeRClient(ctx.GlobalString(RPCURL.Name),ctx.GlobalString(AdminKey.Name),ctx.GlobalString(RPCKey.Name))
	if err != nil {
		panic(err)
	}
	pkiAddress  := common.HexToAddress(ctx.GlobalString(PKIAddress.Name))
	contractAddress  := common.HexToAddress(ctx.GlobalString(ContractAddress.Name))

	pkiSession, err := bindings.NewPkiSession(pkiAddress, adminRClient.rpcclient, adminRClient.txsession)
	if err != nil {
		panic(err)
	}

	fmt.Println("Contract status:")
	AddressStatus, err := pkiSession.GetRules(contractAddress)
	fmt.Println("CA hash ", common.ToHex(AddressStatus.CAHash[:]),"Signer's OID: ", string(AddressStatus.Oid))

	return err

}


func init() {
	// Initialize the CLI app and start Masterchain
	//app.Action = runMasterchain
	app.HideVersion = false // we have a command to print the version
	app.Commands = []cli.Command{

		clientCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))


	app.Flags = append(app.Flags, clientFlags...)


	app.After = func(ctx *cli.Context) error {
		//debug.Exit()
		//console.Stdin.Close() // Resets terminal mode.
		//console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	var err error
	/*
		usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		output := io.Writer(os.Stderr)
		if usecolor {
			output = colorable.NewColorableStderr()
		}
		ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))
		glogger = log.NewGlogHandler(ostream)
		glogger.Verbosity(log.Lvl(10))
		log.Root().SetHandler(glogger)
	*/
	if !strings.Contains(os.Getenv("GODEBUG"), "cgocheck=0") {
		var self string
		if self, err = os.Executable(); err == nil {
			cmd := exec.Command(self, os.Args[1:]...)
			cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			cmd.Env = append(os.Environ(), "GODEBUG=cgocheck=0")

			err = cmd.Run()
		}
	} else {

		err = app.Run(os.Args)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
/*
func main() {
	client, err := ethclient.Dial("https://ropsten.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161")
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA("b3efee931ecefc007abc3f4b4a1215948a750af229a91dbbb428959b4c787907")
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(0) // in wei (0 eth)
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	toAddress := common.HexToAddress("0x4592d8f8d7b001e72cb26a73e4fa1806a51ac79d")
	tokenAddress := common.HexToAddress("0x28b149020d2152179873ec60bed6bf7cd705775d")

	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]
	fmt.Println(hexutil.Encode(methodID)) // 0xa9059cbb

	paddedAddress := common.LeftPadBytes(toAddress.Bytes(), 32)
	fmt.Println(hexutil.Encode(paddedAddress)) // 0x0000000000000000000000004592d8f8d7b001e72cb26a73e4fa1806a51ac79d

	amount := new(big.Int)
	amount.SetString("1000000000000000000000", 10) // sets the value to 1000 tokens, in the token denomination

	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	fmt.Println(hexutil.Encode(paddedAmount)) // 0x00000000000000000000000000000000000000000000003635c9adc5dea00000

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	gasLimit, err := client.EstimateGas(context.Background(), ethereum.CallMsg{
		To:   &tokenAddress,
		Data: data,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(gasLimit) // 23256

	tx := types.NewTransaction(nonce, tokenAddress, value, gasLimit, gasPrice, data)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s", signedTx.Hash().Hex()) // tx sent: 0xa56316b637a94c4cc0331c73ef26389d6c097506d581073f927275e7a6ece0bc
}

*/