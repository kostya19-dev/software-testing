package main

import (
	"fmt"
	"io"
	"sort"
)

type metric struct {
	name   string
	kind   string
	help   string
	sample func(Stats) float64
}

var metrics = []metric{
	{"lab4_cpu_seconds_total", "counter", "CPU time consumed by the container", func(s Stats) float64 {
		return nanoseconds(s.CPU.Usage.Total)
	}},
	{"lab4_cpu_throttled_seconds_total", "counter", "Time the container was throttled by its CPU quota", func(s Stats) float64 {
		return nanoseconds(s.CPU.Throttling.ThrottledTime)
	}},
	{"lab4_memory_working_set_bytes", "gauge", "Memory usage without reclaimable page cache", func(s Stats) float64 {
		inactive := s.Memory.Stats["inactive_file"]
		if inactive > s.Memory.Usage {
			return 0
		}
		return float64(s.Memory.Usage - inactive)
	}},
	{"lab4_memory_rss_bytes", "gauge", "Anonymous memory of the container", func(s Stats) float64 {
		return float64(s.Memory.Stats["anon"])
	}},
	{"lab4_memory_limit_bytes", "gauge", "Memory limit of the container", func(s Stats) float64 {
		return float64(s.Memory.Limit)
	}},
	{"lab4_disk_read_bytes_total", "counter", "Bytes read from block devices", func(s Stats) float64 {
		return blockBytes(s, "read")
	}},
	{"lab4_disk_write_bytes_total", "counter", "Bytes written to block devices", func(s Stats) float64 {
		return blockBytes(s, "write")
	}},
	{"lab4_network_receive_bytes_total", "counter", "Bytes received over all interfaces", func(s Stats) float64 {
		var total uint64
		for _, network := range s.Networks {
			total += network.Received
		}
		return float64(total)
	}},
	{"lab4_network_transmit_bytes_total", "counter", "Bytes transmitted over all interfaces", func(s Stats) float64 {
		var total uint64
		for _, network := range s.Networks {
			total += network.Transmitted
		}
		return float64(total)
	}},
	{"lab4_pids", "gauge", "Number of processes and threads", func(s Stats) float64 {
		return float64(s.Pids.Current)
	}},
}

type Snapshot map[string]Stats

func (s Snapshot) Render(w io.Writer) error {
	services := make([]string, 0, len(s))
	for service := range s {
		services = append(services, service)
	}
	sort.Strings(services)

	for _, m := range metrics {
		if _, err := fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", m.name, m.help, m.name, m.kind); err != nil {
			return err
		}
		for _, service := range services {
			if _, err := fmt.Fprintf(w, "%s{service=%q} %g\n", m.name, service, m.sample(s[service])); err != nil {
				return err
			}
		}
	}
	return nil
}

func nanoseconds(value uint64) float64 {
	return float64(value) / 1e9
}

func blockBytes(s Stats, op string) float64 {
	var total uint64
	for _, entry := range s.Block.Entries {
		if entry.Op == op {
			total += entry.Value
		}
	}
	return float64(total)
}
