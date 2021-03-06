package hashtrie

import (
	"net/http"
	"sync"
)

type IntermediaryResponse struct {
	Status int
	Header http.Header
}

type IntermediaryRequest struct {
	Path   string
	Header http.Header
}

type Intermediary interface {
	Handle(*IntermediaryRequest) *IntermediaryResponse
}

type Route struct {
	Path           string
	Intermediaries []*Intermediary
	Destination    string
	Children       []Route
	Mux            sync.RWMutex
}

type ResolutionType int

const (
	EXACT = iota
	PARTIAL
)

type Resolution struct {
	Route *Route
	Type  ResolutionType
	Rest  []string
}

type BitRecord struct {
	Bitmap uint
	Offset int
}

type TableEntry struct {
	Value   interface{}
	Subtree *AMTNode
}

type AMTNode struct {
	Bitmaps [4]BitRecord
	Table   []TableEntry
	Mux     sync.RWMutex
}
