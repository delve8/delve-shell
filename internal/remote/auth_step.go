package remote

// RemoteAuthOverlayState.Step values while the remote auth overlay is active.
const (
	AuthStepHostKey      = "hostkey"
	AuthStepUsername     = "username"
	AuthStepChoose       = "choose"
	AuthStepPassword     = "password"
	AuthStepIdentity     = "identity"
	AuthStepAutoIdentity = "auto_identity"
)
