package model

// DiskInfo describes the capacity of a mounted disk.
type DiskInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Total uint64 `json:"total"`
	Free  uint64 `json:"free"`
	Used  uint64 `json:"used"`
}
