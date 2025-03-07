package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	viaCepURL     = "https://viacep.com.br/ws/%s/json/"
	weatherAPIURL = "http://api.weatherapi.com/v1/current.json?key=%s&q=%s"
	weatherAPIKey = "c6e34b41fac04d51ad5115119250603"
)

type CEPRequest struct {
	CEP string `json:"cep"`
}

type ViaCEPResponse struct {
	Localidade string `json:"localidade"`
}

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func main() {
	initTracer()
	http.HandleFunc("/cep", handleCEPRequest)
	log.Println("Server running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func handleCEPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request CEPRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &request) != nil {
		http.Error(w, "invalid request", http.StatusUnprocessableEntity)
		return
	}

	cep := strings.TrimSpace(request.CEP)
	if len(cep) != 8 || !isNumeric(cep) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	city, err := getCityByCEP(cep)
	if err != nil {
		http.Error(w, "can not find zipcode", http.StatusNotFound)
		return
	}
	tempC, err := getTemperature(city)
	if err != nil {
		http.Error(w, "failed to fetch temperature", http.StatusInternalServerError)
		return
	}

	response := WeatherResponse{
		City:  city,
		TempC: tempC,
		TempF: tempC*1.8 + 32,
		TempK: tempC + 273,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCityByCEP(cep string) (string, error) {
	url := fmt.Sprintf(viaCepURL, cep)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Localidade == "" {
		return "", fmt.Errorf("location not found")
	}
	return result.Localidade, nil
}

func getTemperature(city string) (float64, error) {
	url := fmt.Sprintf(weatherAPIURL, weatherAPIKey, city)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Current.TempC, nil
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func initTracer() {
	exporter, err := zipkin.New("http://localhost:9411/api/v2/spans")
	if err != nil {
		log.Fatal(err)
	}
	tracerProvider := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tracerProvider)
}
