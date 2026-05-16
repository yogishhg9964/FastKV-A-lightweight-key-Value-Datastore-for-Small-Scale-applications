package api

import (
	"net"
	"net/rpc"
	"sync"

	"kv-distributed/internal/service"
)

type KVServer struct {
	service *service.KVService
	slaves  []*rpc.Client
	mu      sync.RWMutex
}

func NewKVServer(service *service.KVService) *KVServer {
	return &KVServer{
		service: service,
		slaves:  make([]*rpc.Client, 0),
	}
}

// Add this method to ensure JSON compatibility for HTTP bridge
func (s *KVServer) convertResponse(resp *Response) map[string]interface{} {
	result := make(map[string]interface{})

	if resp.Value != nil {
		result["Value"] = resp.Value
	}
	if resp.Values != nil {
		result["Values"] = resp.Values
	}
	if resp.Keys != nil {
		result["Keys"] = resp.Keys
	}
	if resp.List != nil {
		result["List"] = resp.List
	}
	if resp.Data != nil {
		result["Data"] = resp.Data
	}
	if resp.Message != "" {
		result["Message"] = resp.Message
	}
	if resp.Errors != nil {
		result["Errors"] = resp.Errors
	}

	return result
}

// HTTP-compatible wrapper methods for the HTTP bridge
func (s *KVServer) HTTPPut(req Request, resp *map[string]interface{}) error {
	var internalResp Response
	err := s.Put(req, &internalResp)
	if err != nil {
		return err
	}
	*resp = s.convertResponse(&internalResp)
	return nil
}

func (s *KVServer) HTTPGet(req Request, resp *map[string]interface{}) error {
	var internalResp Response
	err := s.Get(req, &internalResp)
	if err != nil {
		return err
	}
	*resp = s.convertResponse(&internalResp)
	return nil
}

func (s *KVServer) HTTPDelete(req Request, resp *map[string]interface{}) error {
	var internalResp Response
	err := s.Delete(req, &internalResp)
	if err != nil {
		return err
	}
	*resp = s.convertResponse(&internalResp)
	return nil
}

func (s *KVServer) HTTPUpdate(req Request, resp *map[string]interface{}) error {
	var internalResp Response
	err := s.Update(req, &internalResp)
	if err != nil {
		return err
	}
	*resp = s.convertResponse(&internalResp)
	return nil
}

func (s *KVServer) HTTPList(req Request, resp *map[string]interface{}) error {
	var internalResp Response
	err := s.List(req, &internalResp)
	if err != nil {
		return err
	}
	*resp = s.convertResponse(&internalResp)
	return nil
}

// Continue with similar HTTP wrappers for all methods...

func (s *KVServer) Start(port string) error {
	rpc.Register(s)
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}

// Original RPC methods (remove the duplicate Put method)
func (s *KVServer) Put(req Request, resp *Response) error {
	if req.TTLSeconds > 0 {
		s.service.PutWithTTL(req.Bucket, req.Key, req.Value, req.TTLSeconds)
	} else {
		s.service.Put(req.Bucket, req.Key, req.Value)
	}
	s.replicateToSlaves("Put", req)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) Get(req Request, resp *Response) error {
	value, err := s.service.Get(req.Bucket, req.Key)
	if err != nil {
		return err
	}
	resp.Value = value
	return nil
}

func (s *KVServer) Delete(req Request, resp *Response) error {
	err := s.service.Delete(req.Bucket, req.Key)
	if err != nil {
		return err
	}
	s.replicateToSlaves("Delete", req)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) Update(req Request, resp *Response) error {
	err := s.service.Update(req.Bucket, req.Key, req.Value)
	if err != nil {
		return err
	}
	s.replicateToSlaves("Update", req)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) List(req Request, resp *Response) error {
	data, err := s.service.List(req.Bucket)
	if err != nil {
		return err
	}
	resp.Values = data
	return nil
}

func (s *KVServer) ListKeysByPrefix(req Request, resp *Response) error {
	keys, err := s.service.ListKeysByPrefix(req.Bucket, req.Prefix)
	if err != nil {
		return err
	}
	resp.Keys = keys
	return nil
}

func (s *KVServer) ListKeysInRange(req Request, resp *Response) error {
	keys, err := s.service.ListKeysInRange(req.Bucket, req.Start, req.End)
	if err != nil {
		return err
	}
	resp.Keys = keys
	return nil
}

func (s *KVServer) ScanKeys(req Request, resp *Response) error {
	keys, nextCursor := s.service.ScanKeys(req.Bucket, req.Cursor, req.Count)
	resp.Keys = keys
	resp.NextCursor = nextCursor
	return nil
}

func (s *KVServer) SetAdd(req Request, resp *Response) error {
	s.service.SetAdd(req.SetName, req.ValueStr)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) SetRemove(req Request, resp *Response) error {
	err := s.service.SetRemove(req.SetName, req.ValueStr)
	if err != nil {
		return err
	}
	resp.Message = "OK"
	return nil
}

func (s *KVServer) SetList(req Request, resp *Response) error {
	list, err := s.service.SetList(req.SetName)
	if err != nil {
		return err
	}
	resp.List = list
	return nil
}

func (s *KVServer) SortedListAdd(req Request, resp *Response) error {
	s.service.SortedListAdd(req.ListName, req.ValueStr)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) SortedListGet(req Request, resp *Response) error {
	list, err := s.service.SortedListGet(req.ListName)
	if err != nil {
		return err
	}
	resp.List = list
	return nil
}

func (s *KVServer) MapPut(req Request, resp *Response) error {
	s.service.MapPut(req.MapName, req.Key, req.ValueStr)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) MapGet(req Request, resp *Response) error {
	data, err := s.service.MapGet(req.MapName, req.Key)
	if err != nil {
		return err
	}
	resp.Data = data
	return nil
}

func (s *KVServer) QueuePush(req Request, resp *Response) error {
	s.service.QueuePush(req.QueueName, req.ValueStr)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) QueuePop(req Request, resp *Response) error {
	val, err := s.service.QueuePop(req.QueueName)
	if err != nil {
		return err
	}
	resp.Message = val
	return nil
}

func (s *KVServer) QueuePeek(req Request, resp *Response) error {
	val, err := s.service.QueuePeek(req.QueueName)
	if err != nil {
		return err
	}
	resp.Message = val
	return nil
}

func (s *KVServer) StackPush(req Request, resp *Response) error {
	s.service.StackPush(req.StackName, req.ValueStr)
	resp.Message = "OK"
	return nil
}

func (s *KVServer) StackPop(req Request, resp *Response) error {
	val, err := s.service.StackPop(req.StackName)
	if err != nil {
		return err
	}
	resp.Message = val
	return nil
}

func (s *KVServer) StackPeek(req Request, resp *Response) error {
	val, err := s.service.StackPeek(req.StackName)
	if err != nil {
		return err
	}
	resp.Message = val
	return nil
}

func (s *KVServer) ExecuteTransaction(req Request, resp *Response) error {
	var txOps []service.TransactionOp
	for _, op := range req.Transaction {
		txOps = append(txOps, service.TransactionOp{
			Action: op.Action,
			Bucket: op.Bucket,
			Key:    op.Key,
			Value:  op.Value,
		})
	}

	errs := s.service.ExecuteTransaction(txOps)
	resp.Errors = errs
	return nil
}

func (s *KVServer) AddSlave(address string) error {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.slaves = append(s.slaves, client)
	s.mu.Unlock()
	return nil
}

func (s *KVServer) replicateToSlaves(method string, req Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, slave := range s.slaves {
		var resp Response
		go slave.Call("KVServer."+method, req, &resp)
	}
}
