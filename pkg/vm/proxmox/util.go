package proxmox

//go:fix inline
func pointer[T any](item T) *T {
	return new(item)
}
