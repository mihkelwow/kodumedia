package startup

import (
	"errors"
	"fmt"
	"strings"
)

const (
	runKeyPath             = `Software\Microsoft\Windows\CurrentVersion\Run`
	startupApprovedKeyPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	startupValueName       = "KoduMedia"
)

var errRegistryValueNotFound = errors.New("registry value not found")

type registryStore interface {
	ReadString(path, name string) (string, error)
	ReadBinary(path, name string) ([]byte, error)
	WriteString(path, name, value string) error
	DeleteValue(path, name string) error
}

type Manager struct {
	executable string
	registry   registryStore
}

func newManager(executable string, registry registryStore) *Manager {
	return &Manager{executable: executable, registry: registry}
}

// Enabled reports whether Windows will currently allow the registered command
// to run. A Run entry can remain present after the user disables it in Task
// Manager or Windows Settings, so both registry locations must agree.
func (m *Manager) Enabled() bool {
	command, err := m.registry.ReadString(runKeyPath, startupValueName)
	if err != nil || !commandTargetsExecutable(command, m.executable) {
		return false
	}

	approval, err := m.registry.ReadBinary(startupApprovedKeyPath, startupValueName)
	if errors.Is(err, errRegistryValueNotFound) {
		// Windows treats a Run entry without a separate StartupApproved state as
		// enabled and creates the state itself when the user changes it.
		return true
	}
	if err != nil {
		return false
	}
	return startupApprovalEnabled(approval)
}

func (m *Manager) SetEnabled(enabled bool) error {
	if !enabled {
		runErr := ignoreMissing(m.registry.DeleteValue(runKeyPath, startupValueName))
		approvalErr := ignoreMissing(m.registry.DeleteValue(startupApprovedKeyPath, startupValueName))
		if runErr != nil {
			return fmt.Errorf("remove startup command: %w", runErr)
		}
		if approvalErr != nil {
			return fmt.Errorf("remove startup approval state: %w", approvalErr)
		}
		return nil
	}

	command := fmt.Sprintf("\"%s\" --startup", m.executable)
	if err := m.registry.WriteString(runKeyPath, startupValueName, command); err != nil {
		return fmt.Errorf("write startup command: %w", err)
	}
	// Task Manager and Windows Settings preserve a disabled StartupApproved
	// value even when the Run command is rewritten. Removing that stale value
	// is the supported user action behind KoduMedia's explicit enable toggle;
	// Windows then treats the freshly registered Run command as enabled.
	if err := ignoreMissing(m.registry.DeleteValue(startupApprovedKeyPath, startupValueName)); err != nil {
		return fmt.Errorf("clear disabled startup state: %w", err)
	}
	return nil
}

func ignoreMissing(err error) error {
	if errors.Is(err, errRegistryValueNotFound) {
		return nil
	}
	return err
}

func commandTargetsExecutable(command, executable string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}

	var registered string
	var arguments string
	if strings.HasPrefix(command, `"`) {
		closing := strings.Index(command[1:], `"`)
		if closing < 0 {
			return false
		}
		closing++
		registered = command[1:closing]
		arguments = strings.TrimSpace(command[closing+1:])
	} else {
		fields := strings.Fields(command)
		if len(fields) == 0 {
			return false
		}
		registered = fields[0]
		arguments = strings.Join(fields[1:], " ")
	}

	if !strings.EqualFold(strings.TrimSpace(registered), strings.TrimSpace(executable)) {
		return false
	}
	return arguments == "" || strings.EqualFold(arguments, "--startup")
}

func startupApprovalEnabled(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	// Windows currently uses 0x02 and 0x06 for enabled states. The matching
	// disabled states are 0x03 and 0x07. Unknown values are treated
	// conservatively as disabled so the tray never shows a false check mark.
	return value[0] == 0x02 || value[0] == 0x06
}
