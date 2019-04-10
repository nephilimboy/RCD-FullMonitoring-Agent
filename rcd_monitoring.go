package main

import (
	"./topology/probes/nic"
	"fmt"
)


func main(){
	nicM := &nic.NicMonitor{
		NetworkInterfaces: make(map[string]*nic.NetworkInterface),
	}
	nicM.StartMonitorNic()

	for _, v := range nicM.NetworkInterfaces {
		fmt.Println("name-> ", v.Name)
		fmt.Println("Byte-> ", v.BytesRecv)
		//fmt.Println("h-> ", v.HardwareAddr)
	}
	//fmt.Println(nicM.NetworkInterfaces[en0].)
}