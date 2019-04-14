package flow

import (
	"github.com/shirou/gopsutil/net"
	"time"
	"go.uber.org/zap"
	"github.com/google/gopacket/pcap"
	"log"
	"github.com/google/gopacket"
	"github.com/segmentio/kafka-go"
	"fmt"
	"context"
	"encoding/json"
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
	NetworkInterface *NetworkInterface `json:"networkInterface"`
	PacketDump       string            `json:"packetDump"`
}

type FlowMonitor struct {
	kafkaWriter       *kafka.Writer
	ZLogger           *zap.Logger
	Protocol          string
	Target            string
	MonitorHandlers   []FlowMonitorHandler
	ticker            *time.Ticker
	NetworkInterfaces map[string]*NetworkInterface //Map[Name]Packet
	InterfaceCache    []string
	done chan struct {
	}
}

func newKafkaWriter(kafkaURL, topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaURL},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
}

func (p *Packet) monitoringNicCounterStat() error {
	interfasesCounterStat, err := net.IOCounters(true)
	if err != nil {
		return err
	}
	for _, nicCounter := range interfasesCounterStat {
		if p.NetworkInterface.Name == nicCounter.Name {
			p.NetworkInterface.BytesSent = nicCounter.BytesSent
			p.NetworkInterface.BytesRecv = nicCounter.BytesRecv
			p.NetworkInterface.PacketsRecv = nicCounter.PacketsRecv
			p.NetworkInterface.PacketsSent = nicCounter.PacketsSent
			p.NetworkInterface.Errin = nicCounter.Errin
			p.NetworkInterface.Errout = nicCounter.Errout
			p.NetworkInterface.Dropin = nicCounter.Dropin
			p.NetworkInterface.Dropout = nicCounter.Dropout
			p.NetworkInterface.Fifoin = nicCounter.Fifoin
			p.NetworkInterface.Fifoout = nicCounter.Fifoout
		}
	}
	return nil
}

func (p *Packet) monitoringNicPacket() {
	var (
		snapshot_len int32 = 1024
		promiscuous  bool  = false
		err          error
		//timeout      time.Duration = 30 * time.Second
		//timeout = 0
	)

	//n.handler, err = pcap.OpenLive(n.Name, snapshot_len, promiscuous, timeout)
	handler, err := pcap.OpenLive(p.NetworkInterface.Name, snapshot_len, promiscuous, 0)
	if err != nil {
		log.Fatal(err)
	}
	defer handler.Close()

	packetSource := gopacket.NewPacketSource(handler, handler.LinkType())

	//temp := <-packetSource.Packets()
	//fmt.Println(temp)

	kafkaURL := "192.168.1.3:9092"
	topic := "amir"
	writer := newKafkaWriter(kafkaURL, topic)
	defer writer.Close()

	for packet := range packetSource.Packets() {
		p.PacketDump = packet.Dump()
		p.monitoringNicCounterStat()
		temp, _ := json.Marshal(p)
		msg := kafka.Message{
			Key:   []byte(fmt.Sprint(p.NetworkInterface.Name)),
			Value: temp,
		}
		err := writer.WriteMessages(context.Background(), msg)
		if err != nil {
			fmt.Println(err)
		}

	}
}

func (m *FlowMonitor) monitoringFlow(nic net.InterfaceStat) {

	//if _, ok := m.NetworkInterfaces[nic.Name]; ok {
	//} else {
	m.NetworkInterfaces[nic.Name] = &NetworkInterface{
		MTU:          nic.MTU,
		Name:         nic.Name,
		HardwareAddr: nic.HardwareAddr,
		Flags:        nic.Flags,
		Addrs:        nic.Addrs,
		handler:      &pcap.Handle{},
	}
	//}

	packet := &Packet{
		NetworkInterface: m.NetworkInterfaces[nic.Name],
	}
	packet.monitoringNicPacket()

}

func (m *FlowMonitor) startMonitorNic() error {
	//m.ticker = time.NewTicker(UpdateInterval)
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	interfasesStat, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, nicInterface := range interfasesStat {
		for _, readableInterface := range devices {
			if nicInterface.Name == readableInterface.Name {
				go m.monitoringFlow(nicInterface)
			}
		}
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
	return nil
}

func (m *FlowMonitor) StopMonitorNic() {
	m.done <- struct{}{}
}

func (m *FlowMonitor) StartMonitorNic() {
	m.ZLogger.Info("Start monitoring network interfaces")
	m.startMonitorNic()
}
