package api

// Ensure all types are JSON serializable
type Request struct {
	Bucket      string          `json:"bucket,omitempty"`
	Key         string          `json:"key,omitempty"`
	Value       []byte          `json:"value,omitempty"`
	TTLSeconds  int64           `json:"ttlSeconds,omitempty"`
	SetName     string          `json:"setName,omitempty"`
	ListName    string          `json:"listName,omitempty"`
	MapName     string          `json:"mapName,omitempty"`
	QueueName   string          `json:"queueName,omitempty"`
	StackName   string          `json:"stackName,omitempty"`
	Prefix      string          `json:"prefix,omitempty"`
	Start       string          `json:"start,omitempty"`
	End         string          `json:"end,omitempty"`
	ValueStr    string          `json:"valueStr,omitempty"`
	Transaction []TransactionOp `json:"transaction,omitempty"`
	Cursor      int             `json:"cursor,omitempty"`
	Count       int             `json:"count,omitempty"`
}

type Response struct {
	Value   []byte            `json:"value,omitempty"`
	Values  map[string][]byte `json:"values,omitempty"`
	Keys    []string          `json:"keys,omitempty"`
	List    []string          `json:"list,omitempty"`
	Data    interface{}       `json:"data,omitempty"`
	Message    string            `json:"message,omitempty"`
	Errors     []error           `json:"errors,omitempty"`
	NextCursor int               `json:"nextCursor,omitempty"`
}

type TransactionOp struct {
	Action string `json:"action"`
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// Add JSON tags for other types as well...
type TTLInfo struct {
	Expiry int64
}

type Set map[string]struct{}
type SortedList []string
type Map map[string]interface{}
type Queue []string
type Stack []string
