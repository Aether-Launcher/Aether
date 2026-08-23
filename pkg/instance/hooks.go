package instance

// ModLoaderHook is set by the extensions package after init to avoid import cycles.
// It receives a launch context and returns a modified context.
var ModLoaderHook func(loaderID string, ctx map[string]interface{}) (map[string]interface{}, error)

// StateChangeHook is set by the app layer to notify extensions of instance state changes.
var StateChangeHook func(id, state string)
