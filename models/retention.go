package models

// ServerSideRetentionDays are the per-user retention choices of the ciphertext
// archive. There is no "off": the archive is also the offline delivery buffer,
// so it has to outlive a device that is away for a while.
var ServerSideRetentionDays = []int{7, 30, 90, 365}

// DefaultRetentionDays is what a new account starts with.
const DefaultRetentionDays = 30

// IsRetentionDaysAllowed reports whether a client may store this retention.
//
//	@param days
//	@return bool
func IsRetentionDaysAllowed(days int) bool {
	for _, allowed := range ServerSideRetentionDays {
		if days == allowed {
			return true
		}
	}
	return false
}
