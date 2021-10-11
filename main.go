/**
 * This source code is licensed under the license found in the
 * LICENSE file in the root directory of this source tree.
 */
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/go-github/v39/github"
	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	"github.com/kelseyhightower/envconfig"
)

type appConfig struct {
	AppSecret string `required:"true" split_words:"true"`
	Port      int    `default:"5000"`
	Token     string `required:"true"`
}

var (
	receivedUpdatesMutex sync.RWMutex
	receivedUpdates      = make([]map[string]interface{}, 0)
	config               appConfig
)

func main() {
	godotenv.Load()

	err := envconfig.Process("", &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	router := httprouter.New()

	router.HandlerFunc(http.MethodGet, "/", handleGetIndex)
	router.HandlerFunc(http.MethodGet, "/facebook", handleGetWebHook)
	router.HandlerFunc(http.MethodPost, "/facebook", handlePostWebHook)

	addr := fmt.Sprintf(":%d", config.Port)

	err = http.ListenAndServe(addr, router)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func handleGetIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	receivedUpdatesMutex.RLock()
	defer receivedUpdatesMutex.RUnlock()
	json.NewEncoder(w).Encode(receivedUpdates)
}

func handleGetWebHook(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	if query.Get("hub.mode") != "subscribe" || query.Get("hub.verify_token") != config.Token {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Write([]byte(query.Get("hub.challenge")))
}

func handlePostWebHook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(config.AppSecret))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var update map[string]interface{}
	err = json.Unmarshal(payload, &update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Process the Facebook updates here
	receivedUpdatesMutex.Lock()
	defer receivedUpdatesMutex.Unlock()
	receivedUpdates = append(receivedUpdates, nil)
	copy(receivedUpdates[1:], receivedUpdates)
	receivedUpdates[0] = update
}
