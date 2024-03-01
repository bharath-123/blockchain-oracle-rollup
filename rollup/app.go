package rollup

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	astriaGrpc "buf.build/gen/go/astria/execution-apis/grpc/go/astria/execution/v1alpha2/executionv1alpha2grpc"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// App is the main application struct, containing all the necessary components.
type App struct {
	executionRPC     string
	sequencerRPC     string
	sequencerClient  SequencerClient
	restRouter       *mux.Router
	restAddr         string
	rollup           *Rollup
	rollupName       string
	rollupID         []byte
	ethBlockDataRcvr chan EthBlockData
	wsClients        WSClientList
	newBlockChan     chan Block
	lock             sync.RWMutex
}

func NewApp(cfg Config, ethBlockDataRcvr chan EthBlockData) *App {
	log.Debugf("Creating new rollup app with config: %v", cfg)

	newBlockChan := make(chan Block, 20)

	rollup := NewRollup(newBlockChan)
	router := mux.NewRouter()

	rollupID := sha256.Sum256([]byte(cfg.RollupName))

	// sequencer private key
	privateKeyBytes, err := hex.DecodeString(cfg.SeqPrivate)
	if err != nil {
		panic(err)
	}
	private := ed25519.NewKeyFromSeed(privateKeyBytes)

	return &App{
		executionRPC:     cfg.ConductorRpc,
		sequencerRPC:     cfg.SequencerRpc,
		sequencerClient:  *NewSequencerClient("http://cometbft:26657", rollupID[:], private),
		restRouter:       router,
		restAddr:         cfg.RESTApiPort,
		rollup:           &rollup,
		rollupName:       cfg.RollupName,
		rollupID:         rollupID[:],
		ethBlockDataRcvr: ethBlockDataRcvr,
		newBlockChan:     newBlockChan,
		wsClients:        map[*WSClient]bool{},
	}
}

// makeExecutionServer creates a new ExecutionServiceServer.
func (a *App) makeExecutionServer() *ExecutionServiceServerV1Alpha2 {
	return NewExecutionServiceServerV1Alpha2(a.rollup, a.rollupID)
}

// setupRestRoutes sets up the routes for the REST API.
func (a *App) setupRestRoutes() {
	a.restRouter.HandleFunc("/block/{height}", a.getBlock).Methods("GET")
	a.restRouter.HandleFunc("/ws", a.serveWS)
}

// makeRestServer creates a new HTTP server for the REST API.
func (a *App) makeRestServer() *http.Server {
	return &http.Server{
		Addr:    a.restAddr,
		Handler: cors.Default().Handler(a.restRouter),
	}
}

func (a *App) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade HTTP to WebSocket: %v", err)
		return
	}

	client := NewWSClient(conn, a)
	a.addWSClient(client)
	go client.WaitForMessages()
	log.Debug("new ws client connected")
}

func (a *App) addWSClient(client *WSClient) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.wsClients[client] = true
}

func (a *App) removeWSClient(client *WSClient) {
	a.lock.Lock()
	defer a.lock.Unlock()
	if _, ok := a.wsClients[client]; ok {
		// close connection
		client.conn.Close()
		// remove
		delete(a.wsClients, client)
	}
}

func (a *App) getBlock(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	heightStr, ok := vars["height"]
	if !ok {
		log.Errorf("error getting height from request\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	height, err := strconv.Atoi(heightStr)
	if err != nil {
		log.Errorf("error converting height to int: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debugf("getting block %d\n", height)
	block, err := a.rollup.GetSingleBlock(uint32(height))
	if err != nil {
		log.Errorf("error getting block: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	blockJson, err := json.Marshal(block)
	if err != nil {
		log.Errorf("error marshalling block: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(blockJson)
}

func (a *App) postEthBlockData(w http.ResponseWriter, r *http.Request) {
	var tx Transaction
	err := json.NewDecoder(r.Body).Decode(&tx)
	if err != nil {
		log.Errorf("error decoding transaction: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := a.sequencerClient.SequenceTx(tx)
	if err != nil {
		log.Errorf("error sending message: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.WithField("responseCode", resp.Code).Debug("transaction submission result")
}

func (a *App) Run() {
	// run execution api
	// TODO - implement graceful shutdown here
	go func() {
		server := a.makeExecutionServer()
		log.Debug("Listening to execution service")
		lis, err := net.Listen("tcp", "0.0.0.0:50051")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Debug("Registering execution service")
		grpcServer := grpc.NewServer()
		astriaGrpc.RegisterExecutionServiceServer(grpcServer, server)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// run go routine which waits for eth block data from ethBlockDataRcvr
	go func() {
		for {
			select {
			case ethBlockData := <-a.ethBlockDataRcvr:
				log.Debugf("received eth block data: %v\n", ethBlockData)
				// send it to the sequencer
				tx := Transaction{
					FinalizedEthBlockData: ethBlockData,
				}
				resp, err := a.sequencerClient.SequenceTx(tx)
				if err != nil {
					log.Errorf("error sending message: %s\n", err)
					continue
				}
				log.WithField("responseCode", resp.Code).Debug("transaction submission result")
			}
		}
	}()

	go func() {
		for block := range a.newBlockChan {
			// only write blocks with transactions
			if len(block.Txs) == 0 {
				continue
			}

			for _, tx := range block.Txs {
				txJson, err := json.Marshal(tx)
				if err != nil {
					log.Errorf("Failed to marshal transaction: %v", err)
					continue
				}

				for client := range a.wsClients {
					select {
					case client.egress <- txJson:
					default:
						log.Warnf("Could not send transaction to ws client: %s", txJson)
					}
				}
			}
		}
	}()

	// run rest api server
	a.setupRestRoutes()
	server := a.makeRestServer()

	log.Infof("API server listening on %s\n", a.restAddr)
	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			log.Warnf("rest api server closed\n")
		} else if err != nil {
			log.Errorf("error listening for rest api server: %s\n", err)
		}
	}()
}
