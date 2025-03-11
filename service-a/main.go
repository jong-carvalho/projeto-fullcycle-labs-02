package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"io/ioutil"
	"log"
	"net/http"
)

const serviceBURL = "http://service-b:8081/weather"

type CEPRequest struct {
	CEP string `json:"cep"`
}

func validateCEP(cep string) bool {
	if len(cep) != 8 {
		return false
	}
	return true
}

func sendToServiceB(ctx context.Context, cep string) (*http.Response, error) {
	tracer := otel.Tracer("service-a")
	ctx, span := tracer.Start(ctx, "sendToServiceB")
	defer span.End()

	reqBody, _ := json.Marshal(CEPRequest{CEP: cep})
	req, err := http.NewRequestWithContext(ctx, "POST", serviceBURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	return client.Do(req)
}

func cepHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CEPRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || !validateCEP(req.CEP) {
		http.Error(w, `{"message": "invalid zipcode"}`, http.StatusUnprocessableEntity)
		return
	}

	resp, err := sendToServiceB(ctx, req.CEP)
	if err != nil {
		http.Error(w, `{"message": "failed to connect to service B"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func main() {
	fmt.Println("Serviço A rodando na porta 8080...")
	http.Handle("/cep", otelhttp.NewHandler(http.HandlerFunc(cepHandler), "cepHandler"))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
