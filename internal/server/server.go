package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"git.energy/corepass/corepass-http-helpers/logger"
	"github.com/core-coin/faucet/internal/models"
	"github.com/core-coin/faucet/internal/storage"
	"github.com/golang-jwt/jwt/v4"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LK4D4/trylock"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/negroni"

	"github.com/core-coin/faucet/internal/chain"
	"github.com/core-coin/faucet/web"
)

const AddressKey string = "address"
const JWTKey string = "jwt"

var fields = []string{
	"SH_DriverLicense_FULLNAME",
	"SH_DriverLicense_DOB",
	"SH_DriverLicense_ExpiryDate",
	"SH_DriverLicense_IssueDate",
	"SH_DriverLicense_DocumentNumber",
	"SH_DriverLicense_Gender",
	"SH_DriverLicense_Country",
	"SH_DriverLicense_DocumentImage",
	"SH_DriverLicense_FaceImage",
	"SH_DriverLicense_AdditionalProof",
	"SH_EMAIL",
	"SH_PHONE",
}

type Server struct {
	*logger.Logger
	chain.TxBuilder

	kycRequestURL string
	callbackURL   string

	storage storage.Storage
	cfg     *Config

	mutex trylock.Mutex
	queue chan string
}

func NewServer(builder chain.TxBuilder, cfg *Config, storage storage.Storage, logger *logger.Logger, kycRequestURL, callbackURL string) *Server {
	return &Server{
		kycRequestURL: kycRequestURL,
		callbackURL:   callbackURL,
		Logger:        logger,
		TxBuilder:     builder,
		storage:       storage,
		cfg:           cfg,
		queue:         make(chan string, cfg.queueCap),
	}
}

func (s *Server) setupRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.Handle("/", http.FileServer(web.Dist()))

	limiter := NewLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute)
	authLimiter := NewAuthLimiter(s.cfg.proxyCount, time.Duration(s.cfg.interval)*time.Minute, s.storage, s.Logger)

	router.Handle("/api/claim", negroni.New(limiter, negroni.Wrap(s.handleClaim())))
	router.Handle("/api/claimAuthorized", negroni.New(authLimiter, negroni.Wrap(s.handleClaimAuthorized())))
	router.Handle("/api/callback", s.callback())
	router.Handle("/api/info", s.handleInfo())

	return router
}

func (s *Server) Run() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			s.consumeQueue()
		}
	}()

	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.UseHandler(s.setupRouter())
	log.Infof("Starting http server %d", s.cfg.httpPort)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(s.cfg.httpPort), n))
}

func (s *Server) consumeQueue() {
	if len(s.queue) == 0 {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	for len(s.queue) != 0 {
		address := <-s.queue
		txHash, err := s.Transfer(context.Background(), address, chain.CoreToWei(int64(s.cfg.payout)))
		if err != nil {
			log.WithError(err).Error("Failed to handle transaction in the queue")
		}

		tokensTxHash, err := s.TransferTokens(context.Background(), address, chain.CoreToWei(int64(s.cfg.payoutTokens)))
		if err != nil {
			log.WithError(err).Error("Failed to handle transaction in the queue")
		}

		log.WithFields(log.Fields{
			"txHash":       txHash,
			"tokensTxHash": tokensTxHash,
			"address":      address,
		}).Info("Consume from queue successfully")
	}
}

func (s *Server) handleClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		address := r.PostFormValue(AddressKey)
		// Try to lock mutex if the work queue is empty
		if len(s.queue) != 0 || !s.mutex.TryLock() {
			select {
			case s.queue <- address:
				log.WithFields(log.Fields{
					"address": address,
				}).Info("Added to queue successfully")
				fmt.Fprintf(w, "Added %s to the queue", address)
			default:
				log.Warn("Max queue capacity reached")
				errMsg := "Faucet queue is too long, please try again later"
				http.Error(w, errMsg, http.StatusServiceUnavailable)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		txHash, err := s.Transfer(ctx, address, chain.CoreToWei(int64(s.cfg.payout)))
		if err != nil {
			log.WithError(err).Error("Failed to send transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokensTxHash, err := s.TransferTokens(context.Background(), address, chain.CoreToWei(int64(s.cfg.payoutTokens)))
		if err != nil {
			log.WithError(err).Error("Failed to send tokens transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.mutex.Unlock()

		log.WithFields(log.Fields{
			"txHash":       txHash.Hex(),
			"tokensTxHash": tokensTxHash.Hex(),
			"address":      address,
		}).Info("Funded directly successfully")

		fmt.Fprintf(w, "Txhash: %s, TokensTxHash: %s", txHash.Hex(), tokensTxHash.Hex())
	}
}

func (s *Server) handleClaimAuthorized() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}

		jwtToken := r.PostFormValue(JWTKey)
		token, _ := jwt.Parse(jwtToken, nil)
		sub := token.Claims.(jwt.MapClaims)["sub"]
		if sub.(string) == "" {
			s.DebugW(w, http.StatusBadRequest, "Bad JWT Token", "Bad JWT Token")
			return
		}
		if strings.Split(sub.(string), ":")[0] != "coreid" {
			s.DebugW(w, http.StatusBadRequest, "Bad JWT Token", "Bad JWT Token")
			return
		}
		id := strings.Split(sub.(string), ":")[1]

		// try to find coreID in DB
		coreID, err := s.storage.GetCoreID(id)
		if err != nil {
			// no coreID in DB
			if err == storage.ErrorNotFound {
				// request KYC data
				err = s.callKYCRequest(id)
				if err != nil {
					s.ErrorW(w, http.StatusInternalServerError, "/kyc/request endpoint returned err: "+err.Error(), "/kyc/request endpoint returned err: "+err.Error())
					return
				}
				err = s.storage.CreateCoreID(&models.CoreID{ID: id})
				if err != nil {
					s.ErrorW(w, http.StatusInternalServerError, "Internal Server Error", "Cannot create new core id in DB: "+err.Error())
				}
				return
			}
			s.ErrorW(w, http.StatusInternalServerError, "Internal Server Error", "Cannot check if core id exists in DB: "+err.Error())
			return
		} else if !coreID.Verified {
			s.DebugW(w, http.StatusOK, "Requesting data is in progress", "Requesting data is in progress")
			return
		}
		//if coreID passed here - coreID has all fields verified!

		// Try to lock mutex if the work queue is empty
		if len(s.queue) != 0 || !s.mutex.TryLock() {
			select {
			case s.queue <- id:
				log.WithFields(log.Fields{
					"address": id,
				}).Info("Added to queue successfully")
				fmt.Fprintf(w, "Added %s to the queue", id)
			default:
				log.Warn("Max queue capacity reached")
				errMsg := "Faucet queue is too long, please try again later"
				http.Error(w, errMsg, http.StatusServiceUnavailable)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		txHash, err := s.Transfer(ctx, id, chain.CoreToWei(int64(s.cfg.payout)))
		if err != nil {
			log.WithError(err).Error("Failed to send transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokensTxHash, err := s.TransferTokens(context.Background(), id, chain.CoreToWei(int64(s.cfg.payoutTokens)))
		if err != nil {
			log.WithError(err).Error("Failed to send tokens transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.mutex.Unlock()

		log.WithFields(log.Fields{
			"txHash":       txHash.Hex(),
			"tokensTxHash": tokensTxHash.Hex(),
			"address":      id,
		}).Info("Funded directly successfully")

		fmt.Fprintf(w, "Txhash: %s, TokensTxHash: %s", txHash.Hex(), tokensTxHash.Hex())
	}
}

func (s *Server) handleInfo() http.HandlerFunc {
	type info struct {
		Account      string `json:"account"`
		Network      string `json:"network"`
		Payout       string `json:"payout"`
		TokensPayout string `json:"tokens_payout"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info{
			Account:      s.Sender().String(),
			Payout:       strconv.Itoa(s.cfg.payout),
			TokensPayout: strconv.Itoa(s.cfg.payoutTokens),
		})
	}
}

func (s *Server) callback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type Info struct {
			Filed  map[string]string `json:"field"`
			Pepper string            `json:"pepper"`
		}
		type KYCRequestHandlerResponse struct {
			User  string `json:"user"`
			Infos []Info `json:"infos"`
		}

		var data KYCRequestHandlerResponse
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			s.DebugW(w, http.StatusBadRequest, "Bad Request", "Cannot parse KYCRequestHandlerResponse: "+err.Error())
			return
		}
		if len(data.Infos) != len(fields) {
			s.ErrorW(w, http.StatusBadRequest, "Bad Request", "Hydra sent bad callback response: length of fields != 12 ")
			return
		}
		for i := 0; i < len(data.Infos); i++ {
			_, ok := data.Infos[i].Filed[fields[i]]
			if !ok {
				s.ErrorW(w, http.StatusBadRequest, "Bad Request", "Hydra sent bad callback response: field "+fields[i]+" doesnt exist")
				return
			}
		}

		// data is good
		err := s.storage.Verify(data.User)
		if err != nil {
			s.ErrorW(w, http.StatusInternalServerError, "Internal Server Error", "Cannot verify users kyc data")
		}

		// Try to lock mutex if the work queue is empty
		if len(s.queue) != 0 || !s.mutex.TryLock() {
			select {
			case s.queue <- data.User:
				log.WithFields(log.Fields{
					"address": data.User,
				}).Info("Added to queue successfully")
				fmt.Fprintf(w, "Added %s to the queue", data.User)
			default:
				log.Warn("Max queue capacity reached")
				errMsg := "Faucet queue is too long, please try again later"
				http.Error(w, errMsg, http.StatusServiceUnavailable)
			}
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		txHash, err := s.Transfer(ctx, data.User, chain.CoreToWei(int64(s.cfg.payout)))
		if err != nil {
			log.WithError(err).Error("Failed to send transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokensTxHash, err := s.TransferTokens(context.Background(), data.User, chain.CoreToWei(int64(s.cfg.payoutTokens)))
		if err != nil {
			log.WithError(err).Error("Failed to send tokens transaction")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		s.mutex.Unlock()

		log.WithFields(log.Fields{
			"txHash":       txHash.Hex(),
			"tokensTxHash": tokensTxHash.Hex(),
			"address":      data.User,
		}).Info("Funded directly successfully")

		fmt.Fprintf(w, "Txhash: %s, TokensTxHash: %s", txHash.Hex(), tokensTxHash.Hex())
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) callKYCRequest(coreID string) error {
	type KYCRequestHandlerInfo struct {
		User       string   `json:"user"`
		Names      []string `json:"names"`
		Callback   string   `json:"callback"`
		Expiration int64    `json:"expiration"`
	}

	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(KYCRequestHandlerInfo{
		User:       coreID,
		Names:      fields,
		Callback:   s.callbackURL,
		Expiration: time.Now().Unix() + int64((time.Minute * 5).Seconds()),
	})
	if err != nil {
		return err
	}
	resp, err := http.Post(s.kycRequestURL, "application/json", buffer)
	if err != nil {
		return err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(b) != "" {
		return errors.New(string(b))
	}
	return nil
}
