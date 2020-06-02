package main

import (
    "log"
    "net/http"
)

func response(w http.ResponseWriter, r *http.Request) { 
  w.WriteHeader(http.StatusOK)
  w.Header().Set("Content-Type", "application/json")
  http.ServeFile(w, r, "/devfiles/index.json")
}

func main() {
  http.HandleFunc("/",response)
  log.Fatal(http.ListenAndServeTLS(":443","/tmp/serving-certs/tls.crt","/tmp/serving-certs/tls.key",nil))
}
