package jobs

import "time"

type Config struct {
	MaxStackLimit               int
	MaxMaxProcessesAndOrThreads int
	MaxMemoryLimit              int
	MaxExtractSize              int
	MaxCPUTimeLimit             float64
	MaxWallTimeLimit            float64
	MaxMaxFileSize              int
	CallbacksMaxTries           int
	CallbacksTimeout            time.Duration
}
