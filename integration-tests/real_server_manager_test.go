//go:build integration

package integration_tests

import (
	"fmt"
	"github.com/weeb-vip/user-service"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

// RealServerManager handles starting and stopping the actual GraphQL server for integration tests
type RealServerManager struct {
	cmd     *exec.Cmd
	mu      sync.Mutex
	started bool
	baseURL string
	port    int
}

// NewRealServerManager creates a new real server manager
func NewRealServerManager(port int) *RealServerManager {
	return &RealServerManager{
		baseURL: fmt.Sprintf("http://localhost:%d", port),
		port:    port,
	}
}

// StartServerForMain starts the actual GraphQL server for TestMain
func (sm *RealServerManager) StartServerForMain() func() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.started {
		return func() {} // Already started
	}

	// Start the actual server using the CLI command
	// We'll run it from the parent directory where the go.mod is

	go func() {
		_ = user.StartServer()
	}()

	sm.started = true

	// Wait for server to be ready
	if err := sm.waitForServer(10 * time.Second); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		// Don't call StopServerForMain here as it causes deadlock
		// Just kill the process directly
		if sm.cmd != nil && sm.cmd.Process != nil {
			sm.cmd.Process.Kill()
		}
		sm.started = false
		return func() {}
	}

	fmt.Printf("Real GraphQL server started on %s\n", sm.baseURL)

	// Return cleanup function
	return func() {
		sm.StopServerForMain()
	}
}

// StopServerForMain stops the actual GraphQL server
func (sm *RealServerManager) StopServerForMain() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.started || sm.cmd == nil {
		return
	}

	// Kill the process
	if err := sm.cmd.Process.Kill(); err != nil {
		fmt.Printf("Error killing server process: %v\n", err)
	}

	// Wait for process to exit
	_, _ = sm.cmd.Process.Wait()

	sm.started = false
	sm.cmd = nil
	fmt.Printf("Real GraphQL server stopped\n")
}

// waitForServer waits for the server to become available
func (sm *RealServerManager) waitForServer(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		log.Println("Checking if server is ready..." + sm.baseURL + "/readyz")
		resp, err := http.Get(sm.baseURL + "/readyz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond) // Wait a bit longer for real server
	}

	return fmt.Errorf("server did not start within timeout")
}

// GetBaseURL returns the base URL of the server
func (sm *RealServerManager) GetBaseURL() string {
	return sm.baseURL
}
