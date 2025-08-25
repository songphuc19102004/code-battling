package jobs

const (
	StatusInQueue       = 1
	StatusProcessing    = 2
	StatusAccepted      = 3
	StatusWrongAnswer   = 4
	StatusTimeLimit     = 5
	StatusCompilation   = 6
	StatusRTSigsegv     = 7
	StatusRTSigxfsz     = 8
	StatusRTSigfpe      = 9
	StatusRTSigabrt     = 10
	StatusRTNzec        = 11
	StatusRTOther       = 12
	StatusInternalError = 13
	StatusExecFormat    = 14
)
