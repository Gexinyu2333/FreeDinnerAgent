package store

func derefOwner(owner *string) string {
	if owner == nil {
		return ""
	}
	return *owner
}
