package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gopds/hyperbloom/internal/database/postgres"
	"gopds/hyperbloom/internal/service"
)

// bloomHash handles POST requests for hashing a value and adding it to the Bloom filter.
// It expects a JSON body with "key" and "value" fields.
func bloomHash(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[POST]", r.URL.Path, r.Header["Content-Type"])
	// Read the request body
	bytebody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Struct to unmarshal the JSON body
	jsonbody := &struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}

	// Unmarshal the JSON body into the struct
	if err := json.Unmarshal(bytebody, &jsonbody); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		log.Println("Error decoding JSON body:", err)
		return
	}

	// Add the value to the Bloom filter using the provided key
	service.BloomHash(jsonbody.Key, jsonbody.Value)

	// Get the cardinality of the Bloom filter and HyperLogLog
	bCard, hCard := service.BloomCardinality(jsonbody.Key)

	// Format the output string
	output := fmt.Sprintf("Cardinality (bloom, hyperloglog) = (%d, %d)", bCard, hCard)

	// Write the output string to the response
	w.Write([]byte(output))
}

// bloomExists handles POST requests to check if a value exists in the Bloom filter.
// It expects a JSON body with "key" and "value" fields.
func bloomExists(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[POST]", r.URL.Path, r.Header["Content-Type"])
	// Read the request body
	bytebody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Struct to unmarshal the JSON body
	jsonbody := &struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{}

	// Unmarshal the JSON body into the struct
	if err := json.Unmarshal(bytebody, &jsonbody); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		log.Println("Error decoding JSON body:", err)
		return
	}

	// Check if the value exists in the Bloom filter using the provided key
	exists := service.BloomExists(jsonbody.Key, jsonbody.Value)

	// Format the output string
	output := fmt.Sprintf(
		"(%s) ⪽ (%s) = %t\n",
		jsonbody.Value,
		jsonbody.Key,
		exists,
	)

	// Write the output string to the response
	w.Write([]byte(output))
}

// bloomCard handles GET requests to compute approximate cardinality of the key.
// It expects query parameter "key" of type string.
func bloomCard(w http.ResponseWriter, r *http.Request) {
	// Log the request details (HTTP method, URL path, and Content-Type header)
	fmt.Println("[GET]", r.URL.Path, r.Header["Content-Type"])

	// Parse query parameters from the request URL
	queries := r.URL.Query()
	key := queries.Get("key")

	// Check if the 'key' query parameter is present and not empty
	if key != "" {
		// Call service to get the cardinality of the Bloom filter and HyperLogLog for the given key
		bCard, hCard := service.BloomCardinality(key)

		// Format the output string with the cardinality values
		output := fmt.Sprintf("Cardinality (bloom, hyperloglog) = (%d, %d)", bCard, hCard)

		// Write the formatted output string to the HTTP response
		w.Write([]byte(output))
	}
}

func bloomSim(w http.ResponseWriter, r *http.Request) {
	// Log the request details (HTTP method, URL path, and Content-Type header)
	fmt.Println("[POST]", r.URL.Path, r.Header["Content-Type"])

	// Read the request body
	bytebody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Struct to unmarshal the JSON body
	jsonbody := &struct {
		Key1 string `json:"key_1"`
		Key2 string `json:"key_2"`
	}{}

	// Unmarshal the JSON body into the struct
	if err := json.Unmarshal(bytebody, &jsonbody); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		log.Println("Error decoding JSON body:", err)
		return
	}

	// Calculate Bloom filter similarity using service function
	sim := service.BloomSimilarity(jsonbody.Key1, jsonbody.Key2)

	// Format the output string with the calculated similarity
	output := fmt.Sprintf("Jaccard similarity = %f", sim)

	// Write the formatted output string to the HTTP response
	w.Write([]byte(output))
}

func bloomBitwiseExists(w http.ResponseWriter, r *http.Request) {
	// Log the request details (HTTP method, URL path, and Content-Type header)
	fmt.Println("[POST]", r.URL.Path, r.Header["Content-Type"])

	// Read the request body
	bytebody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Struct to unmarshal the JSON body
	jsonbody := &struct {
		Keys     []string `json:"keys"`
		Value    string   `json:"value"`
		Operator string   `json:"operator"`
	}{}

	// Unmarshal the JSON body into the struct
	if err := json.Unmarshal(bytebody, &jsonbody); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		log.Println("Error decoding JSON body:", err)
		return
	}

	// Call service to determine bitwise existence
	bitResult := service.BloomBitwiseExists(
		jsonbody.Keys,
		jsonbody.Value,
		jsonbody.Operator,
	)

	// Prepare output based on bitwise result
	output := fmt.Sprintf("%s bitwise exists = %t", jsonbody.Operator, bitResult)

	// Write response to the client
	w.Write([]byte(output))
}

func bloomChainingExists(w http.ResponseWriter, r *http.Request) {
	// Log the request details (HTTP method, URL path, and Content-Type header)
	fmt.Println("[POST]", r.URL.Path, r.Header["Content-Type"])

	// Read the request body
	bytebody, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	// Struct to unmarshal the JSON body
	jsonbody := &struct {
		Keys     []string `json:"keys"`
		Value    string   `json:"value"`
		Operator string   `json:"operator"`
	}{}

	// Unmarshal the JSON body into the struct
	if err := json.Unmarshal(bytebody, &jsonbody); err != nil {
		// If there's an error decoding JSON, respond with a Bad Request status
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		log.Println("Error decoding JSON body:", err)
		return
	}

	// Call service to check existence of value in Bloom filters associated with keys
	bitResult := service.BloomChainingExists(
		jsonbody.Keys,
		jsonbody.Value,
		jsonbody.Operator,
	)

	// Format the output string with the calculated result
	output := fmt.Sprintf("%s chaining exists = %t", jsonbody.Operator, bitResult)

	// Write the formatted output string to the HTTP response
	w.Write([]byte(output))
}

// cleanup handles OS interrupt signals to perform graceful shutdown tasks.
// It waits for a signal on osChan, shuts down the hyperbloom update coroutine,
// closes the PostgreSQL database connection, and then exits the program.
func cleanup(osChan chan os.Signal, wg *sync.WaitGroup) {
	defer wg.Done()
	// Wait for an OS interrupt signal
	sig := <-osChan
	// Print the received signal
	fmt.Println("Encountered signal:", sig.String())
	// Perform shutdown tasks
	fmt.Println("Shutting down hyperbloom update coroutine and closing DB conn")
	close(service.StopAsyncBloomUpdate) // Send signal to stop async updates
	postgres.DbClient.Close()           // Close the PostgreSQL database connection
	close(osChan)
	fmt.Println("Cleaned up, exiting the program")
	os.Exit(0) // Exit the program with status code 0
}

// main sets up the HTTP server and routes, including handling OS interrupts for graceful shutdown.
func main() {
	var err error

	// Create a new ServeMux instance to handle HTTP requests
	mux := http.NewServeMux()

	// Set up a channel to receive OS interrupt signals
	osChan := make(chan os.Signal, 1)
	signal.Notify(osChan, syscall.SIGTERM, syscall.SIGINT)

	// Goroutine to handle OS interrupt signals
	service.WG.Add(1)
	go cleanup(osChan, &service.WG)

	// Register various HTTP request handlers for specific endpoints
	mux.HandleFunc("/hyperbloom/hash", bloomHash)
	mux.HandleFunc("/hyperbloom/exists", bloomExists)
	mux.HandleFunc("/hyperbloom/exists/bitwise", bloomBitwiseExists)
	mux.HandleFunc("/hyperbloom/exists/chaining", bloomChainingExists)
	mux.HandleFunc("/hyperbloom/card", bloomCard)
	mux.HandleFunc("/hyperbloom/sim", bloomSim)

	// Register a default handler that prints the requested URL path
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL.Path)
	})

	// Start the HTTP server on port 5000
	err = http.ListenAndServe(":5000", mux)
	if err != nil {
		log.Println("Can't start server:", err) // Log error if the server fails to start
		osChan <- syscall.SIGTERM
	}

	service.WG.Wait()
}
