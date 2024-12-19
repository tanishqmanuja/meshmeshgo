package meshmesh

import (
	"fmt"
	"time"

	"leguru.net/m/v2/utils"
)

type EspApiStats struct {
	AppStartTime time.Time
	Connections  map[MeshNodeId]*EspApiConnectionStats
}

type EspApiConnectionStats struct {
	active        bool
	lastHandle    uint16
	lastConnStart time.Time
	lastConnStop  time.Time
}

func (s *EspApiConnectionStats) IsActiveAsText() string {
	if s.active {
		return "X"
	} else {
		return " "
	}
}

func (s *EspApiConnectionStats) TimeSinceLastConnection() time.Duration {
	return time.Since(s.lastConnStart)
}

func (s *EspApiConnectionStats) LastConnectionDuration() time.Duration {
	return s.lastConnStop.Sub(s.lastConnStart)
}

func (s *EspApiConnectionStats) Start() {
	s.lastConnStart = time.Now()
	s.lastConnStop = time.Now()
}

func (s *EspApiConnectionStats) Stop() {
	s.lastConnStop = time.Now()
}

func (s *EspApiConnectionStats) GotHandle(handle uint16) {
	s.lastHandle = handle
}

func (as *EspApiStats) StartConnection(address MeshNodeId) *EspApiConnectionStats {
	s, ok := as.Connections[address]
	if !ok {
		s = &EspApiConnectionStats{}
		as.Connections[address] = s
	}
	s.Start()
	s.active = true
	return s
}

func (as *EspApiStats) StopConnection(address MeshNodeId) *EspApiConnectionStats {
	s, ok := as.Connections[address]
	if !ok {
		return nil
	}
	s.Stop()
	s.active = false
	return s
}

func (as *EspApiStats) PrintStats() {
	fmt.Println("|----------------------------------------------------")
	fmt.Printf("| Active connections: %d\n", len(as.Connections))
	fmt.Println("|----------------------------------------------------")
	fmt.Printf("| ID | A | Address  | Hndl | Duration | Start since\n")

	var num = 0
	for id, s := range as.Connections {
		num += 1
		fmt.Printf("| %02d | %s | %s | %04d | %s | %s\n", num, s.IsActiveAsText(), utils.FmtNodeId(int64(id)), s.lastHandle,
			s.LastConnectionDuration().Round(time.Second), s.TimeSinceLastConnection().Round(time.Second))
	}

	fmt.Println("|----------------------------------------------------")
	fmt.Println("")
}

func NewEspApiStats() *EspApiStats {
	return &EspApiStats{
		AppStartTime: time.Now(),
		Connections:  make(map[MeshNodeId]*EspApiConnectionStats),
	}
}
