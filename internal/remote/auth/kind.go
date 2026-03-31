package remoteauth

// Response.Kind values for [Response] from the remote auth overlay to the host.
const (
	ResponseKindPassword      = "password"
	ResponseKindIdentity      = "identity"
	ResponseKindHostKeyAccept = "hostkey_accept"
	ResponseKindHostKeyReject = "hostkey_reject"
)
