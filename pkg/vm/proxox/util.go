package proxmox

func pointer[T any](item T) *T {
	return &item
}
