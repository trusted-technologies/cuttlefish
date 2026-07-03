package slave

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/trusted-technologies/cuttlefish/internal/shared"
)

// StatsPoint represents one network interface sample.
type StatsPoint struct {
	TS    string `json:"ts"`
	Iface string `json:"iface"`
	RX    uint64 `json:"rx"`
	TX    uint64 `json:"tx"`
}

type ifaceSample struct {
	rx uint64
	tx uint64
	ts time.Time
}

// defaultInterfaces returns non-loopback, up interfaces.
func defaultInterfaces() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var names []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		names = append(names, iface.Name)
	}
	return names
}

func readSysInt(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func (s *Slave) statsCollector(ctx context.Context, out chan<- StatsPoint) {
	ifaces := s.cfg.StatsInterfaces
	if len(ifaces) == 0 {
		ifaces = defaultInterfaces()
	}
	if len(ifaces) == 0 {
		slog.Warn("no network interfaces configured for stats")
		return
	}

	slog.Info("collecting interface stats", "interfaces", ifaces)
	prev := make(map[string]ifaceSample)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			for _, iface := range ifaces {
				rx, errRx := readSysInt(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", iface))
				tx, errTx := readSysInt(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", iface))
				if errRx != nil || errTx != nil {
					continue
				}
				var rxRate, txRate uint64
				if last, ok := prev[iface]; ok {
					delta := now.Sub(last.ts).Seconds()
					if delta > 0 {
						rxRate = uint64(float64(rx-last.rx) / delta)
						txRate = uint64(float64(tx-last.tx) / delta)
					}
				}
				prev[iface] = ifaceSample{rx: rx, tx: tx, ts: now}
				select {
				case out <- StatsPoint{TS: now.Format(time.RFC3339), Iface: iface, RX: rxRate, TX: txRate}:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (s *Slave) handleStats(w http.ResponseWriter, r *http.Request) {
	sse, ok := shared.NewSSEWriter(w)
	if !ok {
		return
	}

	out := make(chan StatsPoint)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go s.statsCollector(ctx, out)

	for {
		select {
		case <-ctx.Done():
			return
		case point := <-out:
			_ = sse.Event("stats", point)
		}
	}
}
