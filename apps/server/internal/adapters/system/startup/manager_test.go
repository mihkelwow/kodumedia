package startup

import (
	"errors"
	"testing"
)

type fakeRegistry struct {
	strings map[string]string
	binary  map[string][]byte
	deleted []string
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{
		strings: make(map[string]string),
		binary:  make(map[string][]byte),
	}
}

func registryMapKey(path, name string) string {
	return path + "\x00" + name
}

func (f *fakeRegistry) ReadString(path, name string) (string, error) {
	value, ok := f.strings[registryMapKey(path, name)]
	if !ok {
		return "", errRegistryValueNotFound
	}
	return value, nil
}

func (f *fakeRegistry) ReadBinary(path, name string) ([]byte, error) {
	value, ok := f.binary[registryMapKey(path, name)]
	if !ok {
		return nil, errRegistryValueNotFound
	}
	return append([]byte(nil), value...), nil
}

func (f *fakeRegistry) WriteString(path, name, value string) error {
	f.strings[registryMapKey(path, name)] = value
	return nil
}

func (f *fakeRegistry) DeleteValue(path, name string) error {
	key := registryMapKey(path, name)
	_, stringExists := f.strings[key]
	_, binaryExists := f.binary[key]
	if !stringExists && !binaryExists {
		return errRegistryValueNotFound
	}
	delete(f.strings, key)
	delete(f.binary, key)
	f.deleted = append(f.deleted, key)
	return nil
}

func TestEnabledRequiresRunCommandAndWindowsApproval(t *testing.T) {
	const executable = `C:\Kodu Media\kodumedia.exe`
	tests := []struct {
		name        string
		command     string
		approval    []byte
		hasApproval bool
		want        bool
	}{
		{name: "run entry defaults to enabled", command: `"C:\Kodu Media\kodumedia.exe" --startup`, want: true},
		{name: "task manager disabled entry", command: `"C:\Kodu Media\kodumedia.exe" --startup`, approval: []byte{0x03}, hasApproval: true, want: false},
		{name: "settings disabled entry", command: `"C:\Kodu Media\kodumedia.exe" --startup`, approval: []byte{0x07}, hasApproval: true, want: false},
		{name: "approved entry", command: `"C:\Kodu Media\kodumedia.exe" --startup`, approval: []byte{0x02}, hasApproval: true, want: true},
		{name: "approved by user entry", command: `"C:\Kodu Media\kodumedia.exe" --startup`, approval: []byte{0x06}, hasApproval: true, want: true},
		{name: "stale executable path", command: `"C:\Old Kodu\kodumedia.exe" --startup`, approval: []byte{0x02}, hasApproval: true, want: false},
		{name: "path substring is not enough", command: `"C:\Kodu Media\kodumedia.exe.old" --startup`, approval: []byte{0x02}, hasApproval: true, want: false},
		{name: "unexpected arguments", command: `"C:\Kodu Media\kodumedia.exe" --other`, approval: []byte{0x02}, hasApproval: true, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := newFakeRegistry()
			registry.strings[registryMapKey(runKeyPath, startupValueName)] = test.command
			if test.hasApproval {
				registry.binary[registryMapKey(startupApprovedKeyPath, startupValueName)] = test.approval
			}
			if got := newManager(executable, registry).Enabled(); got != test.want {
				t.Fatalf("Enabled() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetEnabledRepairsCommandAndClearsDisabledApproval(t *testing.T) {
	const executable = `C:\Kodu Media 0.8.42\kodumedia.exe`
	registry := newFakeRegistry()
	registry.strings[registryMapKey(runKeyPath, startupValueName)] = `"C:\Old Kodu\kodumedia.exe" --startup`
	registry.binary[registryMapKey(startupApprovedKeyPath, startupValueName)] = []byte{0x03}

	manager := newManager(executable, registry)
	if err := manager.SetEnabled(true); err != nil {
		t.Fatalf("SetEnabled(true): %v", err)
	}
	if got := registry.strings[registryMapKey(runKeyPath, startupValueName)]; got != `"`+executable+`" --startup` {
		t.Fatalf("startup command = %q", got)
	}
	if _, exists := registry.binary[registryMapKey(startupApprovedKeyPath, startupValueName)]; exists {
		t.Fatal("disabled StartupApproved state was preserved")
	}
	if !manager.Enabled() {
		t.Fatal("manager did not report the repaired startup registration as enabled")
	}
}

func TestSetEnabledRemovesRunAndApprovalValues(t *testing.T) {
	registry := newFakeRegistry()
	registry.strings[registryMapKey(runKeyPath, startupValueName)] = `"C:\Kodu\kodumedia.exe" --startup`
	registry.binary[registryMapKey(startupApprovedKeyPath, startupValueName)] = []byte{0x02}

	manager := newManager(`C:\Kodu\kodumedia.exe`, registry)
	if err := manager.SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled(false): %v", err)
	}
	if manager.Enabled() {
		t.Fatal("manager remained enabled after both registry values were removed")
	}
}

func TestIgnoreMissingOnlyIgnoresAbsentValues(t *testing.T) {
	if err := ignoreMissing(errRegistryValueNotFound); err != nil {
		t.Fatalf("missing value returned error: %v", err)
	}
	expected := errors.New("access denied")
	if err := ignoreMissing(expected); !errors.Is(err, expected) {
		t.Fatalf("unexpected error result: %v", err)
	}
}
