package flow

import (
	"github.com/shirou/gopsutil/net"
	"time"
	"go.uber.org/zap"
	"github.com/google/gopacket/pcap"
	"log"
	"github.com/google/gopacket"
	"fmt"
	"github.com/segmentio/kafka-go"
)

// ConnectionPollInterval poll OVS database every 4 seconds
const UpdateInterval time.Duration = 4 * time.Second

type FlowMonitorHandler interface {
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
	handler      *pcap.Handle
}

type Packet struct {
	NetworkInterface NetworkInterface
	PacketDump       string
}

type FlowMonitor struct {
	kafkaWriter 	*kafka.Writer
	ZLogger         *zap.Logger
	Protocol        string
	Target          string
	MonitorHandlers []FlowMonitorHandler
	ticker          *time.Ticker
	Packet          map[string]*Packet //Map[Name]Packet
	InterfaceCache	[]string
	done chan struct {
	}
}

func (n *Packet) monitoringNicPacket() {

	devices, _ := pcap.FindAllDevs()
	//if err != nil {
	//	log.Fatal(err)
	//}

	var (
		snapshot_len int32 = 1024
		promiscuous  bool  = false
		err          error
		//timeout      time.Duration = 30 * time.Second
		//timeout = 0
	)

	for _, device := range devices {
		//n.handler, err = pcap.OpenLive(n.Name, snapshot_len, promiscuous, timeout)
		//if (device.Name == n.Name) {
		if (device.Name == "en0") {
			n.handler, err = pcap.OpenLive(n.Name, snapshot_len, promiscuous, 0)
			if err != nil {
				log.Fatal(err)
			}
			defer n.handler.Close()

			packetSource := gopacket.NewPacketSource(n.handler, n.handler.LinkType())

			temp := <-packetSource.Packets()
			fmt.Println(temp)
			n.Packets = append(n.Packets, temp.Dump())
			//for packet := range packetSource.Packets() {
			//	n.Packets = append(n.Packets, packet.Dump())
			//}
		}
	}
}

func (m *Packet) monitoringNicCounterStat() error {
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

			//m.NetworkInterfaces[nicCounter.Name].Packets = append(m.NetworkInterfaces[nicCounter.Name].Packets,
			//	<-m.NetworkInterfaces[nicCounter.Name].PacketListener)
		}
	}

	return nil
}

func (m *FlowMonitor) monitoringFlow(deviceName string) error {
	interfasesStat, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, nic := range interfasesStat {
		if _, ok := m.NetworkInterfaces[nic.Name]; ok {
			go m.NetworkInterfaces[nic.Name].monitoringNicPacket()
		} else {
			//var handler pcap.Handle = nil
			m.NetworkInterfaces[nic.Name] = &NetworkInterface{
				MTU:          nic.MTU,
				Name:         nic.Name,
				HardwareAddr: nic.HardwareAddr,
				Flags:        nic.Flags,
				Addrs:        nic.Addrs,
				handler:      &pcap.Handle{},
			}
			go m.NetworkInterfaces[nic.Name].monitoringNicPacket()
		}

	}
	return nil
}

func (m *FlowMonitor) startMonitorNic() {
	//m.ticker = time.NewTicker(UpdateInterval)
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range devices {
		go m.monitoringFlow(device.Name)
	}

	//for {
	//	select {
	//	case <-m.ticker.C:
	//		if err := m.monitoringNic(); err != nil {
	//			m.ZLogger.Error("Cannot get network interfaces | ", zap.Error(err))
	//		}
	//		if err := m.monitoringNicCounterStat(); err != nil {
	//			m.ZLogger.Error("Cannot get network interfaces status | ", zap.Error(err))
	//		}
	//	case <-m.done:
	//		break
	//	}
	//}

}

func (m *FlowMonitor) StopMonitorNic() {
	m.done <- struct{}{}
}

func (m *FlowMonitor) StartMonitorNic() {
	m.ZLogger.Info("Start monitoring network interfaces")
	m.startMonitorNic()
}
