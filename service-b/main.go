package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"io/ioutil"
	"log"
	"net/http"
)

const weatherAPIKey = "c6e34b41fac04d51ad5115119250603"

type CEPRequest struct {
	CEP string `json:"cep"`
}

type ViaCEPResponse struct {
	Localidade string `json:"localidade"`
}

type WeatherResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type TemperatureResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func getCityFromCEP(ctx context.Context, cep string) (string, error) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "getCityFromCEP")
	defer span.End()

	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var data ViaCEPResponse
	json.Unmarshal(body, &data)

	if data.Localidade == "" {
		return "", fmt.Errorf("can not find zipcode")
	}
	return data.Localidade, nil
}

func getWeather(ctx context.Context, city string) (float64, error) {
	tracer := otel.Tracer("service-b")
	ctx, span := tracer.Start(ctx, "getWeather")
	defer span.End()

	url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s", weatherAPIKey, city)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var data WeatherResponse
	json.Unmarshal(body, &data)

	return data.Current.TempC, nil
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CEPRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || len(req.CEP) != 8 {
		http.Error(w, `{"message": "invalid zipcode"}`, http.StatusUnprocessableEntity)
		return
	}

	city, err := getCityFromCEP(ctx, req.CEP)
	if err != nil {
		http.Error(w, `{"message": "can not find zipcode"}`, http.StatusNotFound)
		return
	}

	tempC, err := getWeather(ctx, city)
	if err != nil {
		http.Error(w, `{"message": "failed to fetch weather"}`, http.StatusInternalServerError)
		return
	}

	response := TemperatureResponse{
		City:  city,
		TempC: tempC,
		TempF: tempC*1.8 + 32,
		TempK: tempC + 273,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	fmt.Println("Serviço B rodando na porta 9090...")
	http.Handle("/weather", otelhttp.NewHandler(http.HandlerFunc(weatherHandler), "weatherHandler"))
	log.Fatal(http.ListenAndServe(":9090", nil))
}
