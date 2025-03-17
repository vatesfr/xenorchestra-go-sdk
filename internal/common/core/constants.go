package core

type RetryMode int

const (
	None RetryMode = iota // specifies that no retries will be made
	// Specifies that exponential backoff will be used for certain retryable errors. When
	// a guest is booting there is the potential for a race condition if the given action
	// relies on the existence of a PV driver (unplugging / plugging a device). This open
	// allows the provider to retry these errors until the guest is initialized.
	Backoff
)

const (
	RestV0Path = "rest/v0"
)
