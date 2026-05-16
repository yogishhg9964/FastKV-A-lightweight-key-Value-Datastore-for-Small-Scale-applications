package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"

	"kv-distributed/internal/datastructures"
	"kv-distributed/internal/indexing"
	"kv-distributed/internal/service"
	"kv-distributed/internal/storage"
)

var opsCounter uint64

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&opsCounter, 1)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func auth(next http.HandlerFunc) http.HandlerFunc {
	expectedKey := os.Getenv("HRUX_API_KEY")
	if expectedKey == "" {
		expectedKey = "hrux_dev_key_123"
	}

	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		queryKey := r.URL.Query().Get("apikey")

		var token string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else if queryKey != "" {
			token = queryKey
		}

		if token != expectedKey {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: "Unauthorized: Invalid API Key"})
			return
		}

		next(w, r)
	}
}

type KVRequest struct {
	Bucket string `json:"bucket,omitempty"`
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
	TTL    int64  `json:"ttl,omitempty"`

	SetName  string `json:"setName,omitempty"`
	ListName string `json:"listName,omitempty"`
	MapName  string `json:"mapName,omitempty"`

	ValueStr string `json:"valueStr,omitempty"`

	QueueName string `json:"queueName,omitempty"`
	StackName string `json:"stackName,omitempty"`

	Cursor int `json:"cursor,omitempty"`
	Count  int `json:"count,omitempty"`
}

type KVResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	NextCursor int         `json:"nextCursor,omitempty"`
}

func main() {
	// Initialize core engine
	store := storage.NewStorage()
	idx := indexing.NewIndexer()
	ds := datastructures.NewDataStructuresService()
	kvService := service.NewKVService(store, idx, ds)

	// Attempt to load from WAL
	if err := kvService.LoadFromFile("hrux.wal"); err != nil {
		log.Println("Could not load WAL (might be first run):", err)
	} else {
		log.Println("WAL loaded successfully")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		kvService.Stop()
		os.Exit(0)
	}()

	http.HandleFunc("/health", cors(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))

	http.HandleFunc("/api/metrics", cors(func(w http.ResponseWriter, _ *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		walWrites, walBytes, walLatNs := store.GetWALMetrics()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"heapAllocMB": float64(m.Alloc) / 1024.0 / 1024.0,
			"sysMB":       float64(m.Sys) / 1024.0 / 1024.0,
			"numGC":       m.NumGC,
			"gcCPUFract":  m.GCCPUFraction,
			"totalOps":    atomic.LoadUint64(&opsCounter),
			"activeLanes": 256, // Demonstrating the Sharded Map lanes
			"walWrites":   walWrites,
			"walBytes":    walBytes,
			"walLatNs":    walLatNs,
		})
	}))

	http.HandleFunc("/put", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.Bucket == "" || req.Key == "" {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: "bucket & key required"})
			return
		}

		if req.TTL > 0 {
			kvService.PutWithTTL(req.Bucket, req.Key, []byte(req.Value), req.TTL)
		} else {
			kvService.Put(req.Bucket, req.Key, []byte(req.Value))
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/get", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)

		val, err := kvService.Get(req.Bucket, req.Key)
		if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: string(val)})
	})))

	http.HandleFunc("/delete", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)

		err := kvService.Delete(req.Bucket, req.Key)
		if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/list", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)

		data, err := kvService.List(req.Bucket)
		if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		
		strData := make(map[string]string)
		for k, v := range data {
			strData[k] = string(v)
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: strData})
	})))

	http.HandleFunc("/scan", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)

		keys, nextCursor := kvService.ScanKeys(req.Bucket, req.Cursor, req.Count)
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: keys, NextCursor: nextCursor})
	})))

	http.HandleFunc("/set/add", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		kvService.SetAdd(req.SetName, req.ValueStr)
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))
    
	http.HandleFunc("/set/list", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		list, err := kvService.SetList(req.SetName)
        if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: list})
	})))

	http.HandleFunc("/sortedlist/add", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		kvService.SortedListAdd(req.ListName, req.ValueStr)
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/sortedlist/get", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		list, err := kvService.SortedListGet(req.ListName)
        if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: list})
	})))

	http.HandleFunc("/map/put", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		kvService.MapPut(req.MapName, req.Key, req.ValueStr)
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/map/get", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		val, err := kvService.MapGet(req.MapName, req.Key)
        if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: val})
	})))

	http.HandleFunc("/queue/push", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		kvService.QueuePush(req.QueueName, req.ValueStr)
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/queue/pop", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		val, err := kvService.QueuePop(req.QueueName)
        if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: val})
	})))

	http.HandleFunc("/stack/push", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		kvService.StackPush(req.StackName, req.ValueStr)
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/stack/pop", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		val, err := kvService.StackPop(req.StackName)
        if err != nil {
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(KVResponse{Success: true, Data: val})
	})))

	http.HandleFunc("/publish", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		var req KVRequest
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.Bucket == "" { // Using Bucket field as channel name
			json.NewEncoder(w).Encode(KVResponse{Success: false, Error: "channel (bucket field) required"})
			return
		}
		kvService.Publish(req.Bucket, req.Value)
		json.NewEncoder(w).Encode(KVResponse{Success: true})
	})))

	http.HandleFunc("/subscribe", cors(auth(func(w http.ResponseWriter, r *http.Request) {
		channel := r.URL.Query().Get("channel")
		if channel == "" {
			http.Error(w, "channel query parameter is required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		msgChan := kvService.Subscribe(channel)
		defer kvService.Unsubscribe(channel, msgChan)

		notify := r.Context().Done()

		for {
			select {
			case <-notify:
				return
			case msg := <-msgChan:
				w.Write([]byte("data: " + msg + "\n\n"))
				flusher.Flush()
			}
		}
	})))

	log.Println("🚀 Hrux-DB HTTP Engine running on :8081 (Sharded + WAL + PubSub)")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
