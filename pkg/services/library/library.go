package library

import (
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
)

type Library interface {
	VM() VM
	Task() Task
	Backup() Backup
	StorageRepository() StorageRepository
	Schedule() Schedule
	Pool() Pool
	// Added to provide access to the v1 client, allowing users to:
	// 1. Access v1 functionality without initializing a separate client
	// 2. Use v2 features while maintaining backward compatibility
	// 3. Gradually migrate from v1 to v2 without managing multiple clients
	V1Client() v1.XOClient
}
