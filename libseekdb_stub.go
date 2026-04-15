//go:build !cgo || !(linux || darwin)

package seekdb

import "fmt"

// Native embedded mode requires CGo and Linux/Darwin.
func seekdbOpen(dbDir string) error {
	return fmt.Errorf("native embedded mode requires CGo and Linux/Darwin")
}

func seekdbOpenWithService(dbDir string, port int) error {
	return fmt.Errorf("native embedded mode requires CGo and Linux/Darwin")
}

func seekdbClose() {}

type SeekdbConn struct{}

func seekdbConnect(database string, autocommit bool) (*SeekdbConn, error) {
	return nil, fmt.Errorf("native embedded mode requires CGo and Linux/Darwin")
}

func (c *SeekdbConn) Close() {}

type NativeEmbeddedConn struct{}

type NativeEmbeddedConfig struct {
	BaseDir string
	Port    int
}

func NewNativeEmbeddedConn(cfg NativeEmbeddedConfig) (*NativeEmbeddedConn, error) {
	return nil, fmt.Errorf("native embedded mode requires CGo and Linux/Darwin")
}

func (n *NativeEmbeddedConn) Open() error {
	return fmt.Errorf("native embedded mode requires CGo and Linux/Darwin")
}

func (n *NativeEmbeddedConn) Connect(database string) error {
	return fmt.Errorf("native embedded mode requires CGo and Linux/Darwin")
}

func (n *NativeEmbeddedConn) Conn() *SeekdbConn {
	return nil
}

func (n *NativeEmbeddedConn) Ping() error {
	return fmt.Errorf("not connected")
}

func (n *NativeEmbeddedConn) Close() error {
	return nil
}

func (n *NativeEmbeddedConn) Port() int {
	return 0
}

func (n *NativeEmbeddedConn) Database() string {
	return ""
}
