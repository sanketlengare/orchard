package job

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

func (s Status) CanTransitionTo(next Status) bool {
	switch s {
	case StatusPending:
		return next == StatusRunning
	case StatusRunning:
		return next == StatusSucceeded || next == StatusFailed
	default:
		return false
	}
}
