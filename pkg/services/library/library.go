package library

type Library interface {
	VM() VM
	Task() Task
	Pool() Pool
}
