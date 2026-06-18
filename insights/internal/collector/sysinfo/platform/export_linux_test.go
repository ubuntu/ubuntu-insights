package platform

import (
	"errors"

	"github.com/godbus/dbus/v5"
)

// WithRoot sets the root directory for the platform collector.
func WithRoot(root string) Options {
	return func(o *options) {
		o.platform.root = root
	}
}

// WithDetectVirtCmd sets the detect virtualization command for the platform collector.
func WithDetectVirtCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.detectVirtCmd = cmd
	}
}

// WithSystemdAnalyzeCmd sets the systemd-analyze command for the platform collector.
func WithSystemdAnalyzeCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.systemdAnalyzeCmd = cmd
	}
}

// WithWSLVersionCmd sets the WSL version command for the platform collector.
func WithWSLVersionCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.wslVersionCmd = cmd
	}
}

// WithProStatusCmd sets the pro status command for the platform collector.
func WithProStatusCmd(cmd []string) Options {
	return func(o *options) {
		o.platform.proStatusCmd = cmd
	}
}

// fakeProDBusObject is a fake proDBusObject returning a preconfigured value or error.
type fakeProDBusObject struct {
	value any
	err   error
}

func (o fakeProDBusObject) GetProperty(string) (dbus.Variant, error) {
	if o.err != nil {
		return dbus.Variant{}, o.err
	}
	return dbus.MakeVariant(o.value), nil
}

// fakeProDBusConn is a fake proDBusConn returning a preconfigured object.
type fakeProDBusConn struct {
	obj proDBusObject
}

func (c fakeProDBusConn) Object(string, dbus.ObjectPath) proDBusObject {
	return c.obj
}

func (c fakeProDBusConn) Close() error {
	return nil
}

// ProDBusSpec identifies which fake D-Bus behaviour WithProDBusConnector injects.
type ProDBusSpec int

const (
	// ProDBusConnectError makes the connector itself fail (default/zero value).
	ProDBusConnectError ProDBusSpec = iota
	// ProDBusAttached makes the connector report an attached state.
	ProDBusAttached
	// ProDBusDetached makes the connector report a detached state.
	ProDBusDetached
	// ProDBusPropertyError makes the connection succeed but reading the property fail.
	ProDBusPropertyError
	// ProDBusGarbage makes the connection succeed but return an unexpected property type.
	ProDBusGarbage
)

// WithProDBusConnector sets the pro D-Bus connector for the platform collector.
func WithProDBusConnector(spec ProDBusSpec) Options {
	return func(o *options) {
		o.platform.proDBusConnector = func() (proDBusConn, error) {
			switch spec {
			case ProDBusAttached:
				return fakeProDBusConn{obj: fakeProDBusObject{value: true}}, nil
			case ProDBusDetached:
				return fakeProDBusConn{obj: fakeProDBusObject{value: false}}, nil
			case ProDBusPropertyError:
				return fakeProDBusConn{obj: fakeProDBusObject{err: errors.New("fake property error")}}, nil
			case ProDBusGarbage:
				return fakeProDBusConn{obj: fakeProDBusObject{value: "not a bool"}}, nil
			default: // ProDBusConnectError
				return nil, errors.New("fake connect error")
			}
		}
	}
}

// WithGetenv sets the getenv function for the linux platform collector using a map.
func WithGetenv(env map[string]string) Options {
	return func(o *options) {
		o.platform.getenv = func(key string) string {
			return env[key]
		}
	}
}
