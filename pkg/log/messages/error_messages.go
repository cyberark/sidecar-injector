package messages

/*
	This go file centralizes error log messages so we have them all in one place.

	Although having the names of the consts as the error code (i.e CSPFK014E) and not as a descriptive name (i.e InvalidStoreType)
	can reduce readability of the code that raises the error, we decided to do so for the following reasons:
		1.  Improves supportability – when we get this code in the log we can find it directly in the code without going
			through the “info_messages.go” file first
		2. Validates we don’t have error code duplications – If the code is only in the string then 2 errors can have the
			same code (which is something that a developer can easily miss). However, If they are in the object name
			then the compiler will not allow it.
*/

// Access Token
const CSPFK001E string = "CSPFK001E Failed to create access token object"
const CSPFK002E string = "CSPFK002E Failed to retrieve access token"
const CSPFK003E string = "CSPFK003E AccessToken failed to delete access token data. Reason: %s"

// Environment variables
const CSPFK004E string = "CSPFK004E Environment variable '%s' must be provided"
const CSPFK005E string = "CSPFK005E Provided incorrect value for environment variable %s"

// Authenticator
const CSPFK008E string = "CSPFK008E Failed to instantiate authenticator configuration"
const CSPFK009E string = "CSPFK009E Failed to instantiate authenticator object"
const CSPFK010E string = "CSPFK010E Failed to authenticate"
const CSPFK011E string = "CSPFK011E Failed to parse authentication response"

// ProvideConjurSecrets
const CSPFK014E string = "CSPFK014E Failed to instantiate ProvideConjurSecrets function. Reason: %s"
const CSPFK015E string = "CSPFK015E Failed to instantiate secrets config"
const CSPFK016E string = "CSPFK016E Failed to provide DAP/Conjur secrets"

// Kubernetes
const CSPFK018E string = "CSPFK018E Failed to create Kubernetes client"
const CSPFK019E string = "CSPFK019E Failed to load in-cluster Kubernetes client config"
const CSPFK020E string = "CSPFK020E Failed to retrieve Kubernetes Secret"
const CSPFK021E string = "CSPFK021E Failed to retrieve Kubernetes Secrets"
const CSPFK022E string = "CSPFK022E Failed to update Kubernetes Secret"
const CSPFK023E string = "CSPFK023E Failed to update Kubernetes Secrets"
const CSPFK025E string = "CSPFK025E PathMap cannot be empty"
const CSPFK027E string = "CSPFK027E Failed to update Kubernetes Secrets map with DAP/Conjur secrets"
const CSPFK028E string = "CSPFK028E Unable to update Kubernetes Secret '%s'"
const CSPFK063E string = "CSPFK063E Unable to delete Kubernetes Secret"

// DAP/Conjur
const CSPFK031E string = "CSPFK031E Failed to load DAP/Conjur config. Reason: %s"
const CSPFK032E string = "CSPFK032E Failed to create DAP/Conjur client from token. Reason: %s"
const CSPFK033E string = "CSPFK033E Failed to create DAP/Conjur client"
const CSPFK034E string = "CSPFK034E Failed to retrieve DAP/Conjur secrets. Reason: %s"
const CSPFK035E string = "CSPFK035E Failed to parse DAP/Conjur variable ID"
const CSPFK036E string = "CSPFK036E Variable ID '%s' is not in the format '<account>:variable:<variable_id>'"
const CSPFK037E string = "CSPFK037E Failed to parse DAP/Conjur variable IDs"

// General
const CSPFK038E string = "CSPFK038E Retransmission backoff exhausted"
const CSPFK039E string = "CSPFK039E Secrets Provider for Kubernetes failed to update secrets in %s mode. Reason: %s"

// Annotations
const CSPFK041E string = "CSPFK041E Failed to open annotations file '%s'. Reason: %s"
const CSPFK042E string = "CSPFK042E Annotation '%s' does not accept value '%s': must be type %s"
const CSPFK043E string = "CSPFK043E Annotation '%s' does not accept value '%s': only accepts %v"
const CSPFK044E string = "CSPFK044E Annotation '%s' must be provided"
const CSPFK045E string = "CSPFK045E Annotation file line %d is malformed: expecting format \"<key>=<quoted value>\""

const CSPFK046E string = "CSPFK046E Secret Store Type needs to be configured, either with 'SECRETS_DESTINATION' environment variable or 'conjur.org/secrets-destination' Pod annotation"
const CSPFK047E string = "CSPFK047E Secrets Provider in Push-to-File mode can only be configured with Pod annotations"
const CSPFK048E string = "CSPFK048E Secrets Provider in K8s Secrets mode requires either the 'K8S_SECRETS' environment variable or 'conjur.org/k8s-secrets' Pod annotation"
const CSPFK049E string = "CSPFK049E Failed to validate Pod annotations"
const CSPFK050E string = "CSPFK050E Invalid secrets refresh interval annotation: %s %s"
const CSPFK051E string = "CSPFK051E Invalid secrets refresh configuration: %s %s"

// Push to File
const CSPFK053E string = "CSPFK053E Unable to initialize Secrets Provider: unable to create secret group collection"
const CSPFK054E string = "CSPFK054E Unable to initialize Secrets Provider: unrecognized Store Type '%s'"
const CSPFK062E string = "CSPFK062E Unable to delete secrets file"

// Atomic Writer
const CSPFK055E string = "CSPFK055E Could not create temporary file for '%s'"
const CSPFK056E string = "CSPFK056E Could not flush temporary file '%s'"
const CSPFK057E string = "CSPFK057E Could not set permissions on temporary file '%s'"
const CSPFK058E string = "CSPFK058E Could not rename temporary file '%s' to '%s'"
const CSPFK059E string = "CSPFK059E Could not delete temporary file '%s'. Truncated file."
const CSPFK060E string = "CSPFK060E Could not delete temporary file '%s'. File may be left on disk."
const CSPFK061E string = "CSPFK061E Could not write content to temporary file for '%s'"
