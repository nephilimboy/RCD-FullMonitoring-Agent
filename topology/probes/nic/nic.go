package nic

import (
	"github.com/shirou/gopsutil/net"
	"time"
)

// ConnectionPollInterval poll OVS database every 4 seconds
const UpdateInterval time.Duration = 4 * time.Second

type NicMonitorHandler interface {
	OnNicAdd(uuid string)
	OnNicDelete(uuid string)
}

type DefaultNicMonitorHandler struct{}

func (d *DefaultNicMonitorHandler) OnNicAdd(uuid string) {}

func (d *DefaultNicMonitorHandler) OnNicDelete(uuid string) {}

type IOCountersStat struct {
}
type NetworkInterface struct {
	MTU          int                 `json:"mtu"`          // maximum transmission unit
	Name         string              `json:"name"`         // e.g., "en0", "lo0", "eth0.100"
	HardwareAddr string              `json:"hardwareaddr"` // IEEE MAC-48, EUI-48 and EUI-64 form
	Flags        []string            `json:"flags"`        // e.g., FlagUp, FlagLoopback, FlagMulticast
	Addrs        []net.InterfaceAddr `json:"addrs"`
	BytesSent    uint64              `json:"bytesSent"`   // number of bytes sent
	BytesRecv    uint64              `json:"bytesRecv"`   // number of bytes received
	PacketsSent  uint64              `json:"packetsSent"` // number of packets sent
	PacketsRecv  uint64              `json:"packetsRecv"` // number of packets received
	Errin        uint64              `json:"errin"`       // total number of errors while receiving
	Errout       uint64              `json:"errout"`      // total number of errors while sending
	Dropin       uint64              `json:"dropin"`      // total number of incoming packets which were dropped
	Dropout      uint64              `json:"dropout"`     // total number of outgoing packets which were dropped (always 0 on OSX and BSD)
	Fifoin       uint64              `json:"fifoin"`      // total number of FIFO buffers errors while receiving
	Fifoout      uint64              `json:"fifoout"`     // total number of FIFO buffers errors while sending
}

type NicMonitor struct {
	Protocol          string
	Target            string
	MonitorHandlers   []NicMonitorHandler
	ticker            *time.Ticker
	NetworkInterfaces map[string]*NetworkInterface //Map[name]Nic
	done              chan struct{}
}

func (m *NicMonitor) monitoringNicCounterStat() error {
	interfasesCounterStat, err := net.IOCounters(true)
	if err != nil {
		return err
	}

	for _, nicCounter := range interfasesCounterStat {
		if _, ok := m.NetworkInterfaces[nicCounter.Name]; ok {
			m.NetworkInterfaces[nicCounter.Name].BytesSent = nicCounter.BytesSent
			m.NetworkInterfaces[nicCounter.Name].BytesRecv = nicCounter.BytesRecv
			m.NetworkInterfaces[nicCounter.Name].PacketsRecv = nicCounter.PacketsRecv
			m.NetworkInterfaces[nicCounter.Name].PacketsSent = nicCounter.PacketsSent
			m.NetworkInterfaces[nicCounter.Name].Errin = nicCounter.Errin
			m.NetworkInterfaces[nicCounter.Name].Errout = nicCounter.Errout
			m.NetworkInterfaces[nicCounter.Name].Dropin = nicCounter.Dropin
			m.NetworkInterfaces[nicCounter.Name].Dropout = nicCounter.Dropout
			m.NetworkInterfaces[nicCounter.Name].Fifoin = nicCounter.Fifoin
			m.NetworkInterfaces[nicCounter.Name].Fifoout = nicCounter.Fifoout
		}
	}

	return nil
}

func (m *NicMonitor) monitoringNic() error {
	interfasesStat, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, nic := range interfasesStat {
		m.NetworkInterfaces[nic.Name] = &NetworkInterface{
			MTU:          nic.MTU,
			Name:         nic.Name,
			HardwareAddr: nic.HardwareAddr,
			Flags:        nic.Flags,
			Addrs:        nic.Addrs,
		}

	}
	return nil
}

func (m *NicMonitor) startMonitorNic() {
	m.ticker = time.NewTicker(UpdateInterval)

	//for {
	//	select {
	//	case <-m.ticker.C:
	//		if err := m.monitoringNic(); err != nil {
	//			//if connectedOnce {
	//			//logging.GetLogger().Errorf(": re-connect error %v, will try again", err)
	//			//}
	//		}
	//		if err := m.monitoringNicCounterStat(); err != nil {
	//			//if connectedOnce {
	//			//logging.GetLogger().Errorf(": re-connect error %v, will try again", err)
	//			//}
	//		}
	//	case <-m.done:
	//		break
	//	}
	//}

	if err := m.monitoringNic(); err != nil {
		//if connectedOnce {
		//logging.GetLogger().Errorf(": re-connect error %v, will try again", err)
		//}
	}
	if err := m.monitoringNicCounterStat(); err != nil {
		//if connectedOnce {
		//logging.GetLogger().Errorf(": re-connect error %v, will try again", err)
		//}
	}

}

func (m *NicMonitor) StartMonitorNic() {
	//go m.startMonitorNic()
	m.startMonitorNic()
}
