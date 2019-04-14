package main

import (
	"./topology/probes/flow"
	"go.uber.org/zap"
	"sync"
)


func main(){

	var waitgroup sync.WaitGroup

	waitgroup.Add(1)


	zapLogger, _ := zap.NewDevelopment()
	nicM := &flow.FlowMonitor{
		ZLogger: zapLogger,
		NetworkInterfaces: make(map[string]*flow.NetworkInterface),
	}
	nicM.StartMonitorNic()


	waitgroup.Wait()
	//for {
	//	for _, v := range nicM.NetworkInterfaces {
	//		fmt.Println("########################################################################################################################")
	//		fmt.Println("name-> ", v.Name)
	//		fmt.Println("Byte-> ", v.BytesRecv)
	//		fmt.Println("pack -> ", v.Packets)
	//		fmt.Println("########################################################################################################################")
	//		//fmt.Println("h-> ", v.HardwareAddr)
	//	}
	//}
	//fmt.Println(nicM.NetworkInterfaces[en0].)
}