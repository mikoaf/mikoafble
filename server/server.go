package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func api1Handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func api2Handler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Name string      `json:"name"`
		Age  interface{} `json:"age"` // Gunakan string dulu biar gampang debug
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Gagal decode JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Data diterima: Name: %s, Age: %v\n", data.Name, data.Age)

	response, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func api3Handler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Gagal decode JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Data diterima: %s\n", data.Name)
	response := map[string]string{
		"message": "hello " + data.Name,
	}
	fmt.Println(response)
	jsonResp, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func main() {
	http.HandleFunc("/hello", api1Handler)
	http.HandleFunc("/user", api2Handler)
	http.HandleFunc("/greeting", api3Handler)
	address := "localhost:9000"
	fmt.Printf("Server running on %s\n", address)
	http.ListenAndServe(address, nil)
}
