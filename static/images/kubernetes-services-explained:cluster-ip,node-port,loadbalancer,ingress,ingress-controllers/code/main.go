package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

var ratingURL string = "http://localhost:8080"
var greeting string = "hello"
var delay int = 10

type product struct {
	Id     int     `json:"id"`
	Name   string  `json:"name"`
	Price  string  `json:"price"`
	Brand  string  `json:"brand"`
	Rating float32 `json:"rating,omitempty"`
}

type rating struct {
	Id     int     `json:"id"`
	Rating float32 `json:"rating"`
}

var ratings map[string]*rating = map[string]*rating{
	"1": {
		Id:     1,
		Rating: 4.5,
	},
	"2": {
		Id:     2,
		Rating: 3,
	},
	"3": {
		Id:     3,
		Rating: 3.9,
	},
}

var products map[string]*product = map[string]*product{
	"1": {
		Id:    1,
		Name:  "shoes",
		Price: "$30",
		Brand: "puma",
	},
	"2": {
		Id:    2,
		Name:  "smartphone",
		Price: "$300",
		Brand: "samsung",
	},
	"3": {
		Id:    3,
		Name:  "fridge",
		Price: "$150",
		Brand: "LG",
	},
}

func main() {
	// Load the greeting from the environment variable
	g := os.Getenv("GREETING")
	if g != "" {
		log.Println("Loading greeting:", g)
		greeting = g
	}

	d := os.Getenv(("DELAY"))
	if n, err := strconv.Atoi(d); err == nil {
		log.Println("Loading delay:", n)
		delay = n
	}
	// Set up the routes
	r := mux.NewRouter()
	r.HandleFunc("/greeting/{name}", greetingHandler)
	r.HandleFunc("/products", productsHandler)
	r.HandleFunc("/product/{id}", productHandler)
	r.HandleFunc("/ratings", ratingsHandler)
	r.HandleFunc("/rating/{id}", ratingHandler)

	// Start the server
	port := os.Getenv("PORT")
	if port != "" {
		port = ":" + port
	} else {
		port = ":8080"
	}
	log.Println("Starting server on port", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal("Error while starting server:", err)
	}
}

func ratingHandler(w http.ResponseWriter, r *http.Request) {
	// Get the name to greet
	vars := mux.Vars(r)
	name := vars["id"]

	strconv.ParseInt(name, 10, 0)

	// Write response to client
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ratings[name])
}

func productHandler(w http.ResponseWriter, r *http.Request) {
	// Get the name to greet
	vars := mux.Vars(r)
	id := vars["id"]

	url := os.Getenv("RATING_URL")
	if url != "" {
		ratingURL = url
	}

	res, err := http.Get(fmt.Sprintf("%s/rating/%s", ratingURL, id))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}

	ratingsRes := new(rating)
	json.NewDecoder(res.Body).Decode(ratingsRes)

	data := products[id]
	data.Rating = ratingsRes.Rating

	// Write response to client
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(data)
}

func ratingsHandler(w http.ResponseWriter, r *http.Request) {
	// Write response to client
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ratings)
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	// Write response to client
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(products)
}

func greetingHandler(w http.ResponseWriter, r *http.Request) {
	// Get the name to greet
	vars := mux.Vars(r)
	name := vars["name"]

	// Introduce a delay in response
	time.Sleep(time.Duration(delay) * time.Millisecond)

	// Write response to client
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"greeting": fmt.Sprintf("%s %s", greeting, name)})
}
