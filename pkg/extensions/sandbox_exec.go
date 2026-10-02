// sandbox_exec.go - script execution, IPC dispatch and event emission.

package extensions

import (
	"fmt"

	"github.com/Aether-Launcher/Aether/pkg/logger"
	"github.com/dop251/goja"
)
// Execute runs a JS script inside the sandbox
func (s *Sandbox) Execute(script string) error {
	_, err := s.vm.RunString(script)
	if err != nil {
		return fmt.Errorf("sandbox execution error: %w", err)
	}
	return nil
}

// InvokeMessage runs the registered onMessage handler for an IPC message.
// goja runtimes are not thread-safe and extension callbacks may block for a
// long time (mod downloads, confirmation dialogs), so invocations are
// serialized per sandbox and panics are recovered so a broken extension can
// never leave an iframe waiting on a response forever.
func (s *Sandbox) InvokeMessage(payload map[string]interface{}) (result map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
			err = fmt.Errorf("sandbox %s panicked handling IPC message: %v", s.manifest.ID, r)
		}
	}()

	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()

	if s.onMessageCallback == nil {
		return nil, fmt.Errorf("extension %s has no onMessage handler registered", s.manifest.ID)
	}
	return s.onMessageCallback(payload)
}

// EmitEvent dispatches a JS event to all handlers registered via Aether.events.on.
// It is safe to call from any goroutine.
func (s *Sandbox) EmitEvent(event string, payload map[string]interface{}) {
	s.eventsMu.Lock()
	handlers := append([]goja.Callable(nil), s.eventHandlers[event]...)
	s.eventsMu.Unlock()
	logger.Debug("Sandbox:"+s.manifest.ID, fmt.Sprintf("EmitEvent %q -> %d handlers payload=%v", event, len(handlers), payload))
	if len(handlers) == 0 {
		return
	}
	s.callbackMu.Lock()
	defer s.callbackMu.Unlock()
	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Warn("Sandbox:"+s.manifest.ID, fmt.Sprintf("EmitEvent %q handler panic: %v", event, r))
				}
			}()
			_, _ = h(goja.Undefined(), s.vm.ToValue(payload))
		}()
	}
}
