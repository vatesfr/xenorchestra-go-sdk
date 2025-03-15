package library

type Library interface {
	VM() VM
	Task() Task
	// Restore() Restore
	Backup() Backup
	StorageRepository() StorageRepository
}
