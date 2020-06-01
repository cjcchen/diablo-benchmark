package core

import "diablo-benchmark/communication"

// Master
type Master struct {
	Server *communication.MasterServer // TCP server identified with the master for all clients to connect to
}

// Initialise the master server and return an instance of the master
// This will be passed back to the main
func InitMaster() *Master {
	s, err := communication.SetupMasterTCP(":8123", 3)
	if err != nil {
		// TODO remove panic
		panic(err)
	}

	return &Master{Server: s}
}

// Main functionality to run
// Holds the majority of the work
func (ms *Master) Run() {

	// Get the client connections ready
	clientReadyChannel := make(chan bool, 1)
	go ms.Server.HandleClients(clientReadyChannel)
	<-clientReadyChannel
	close(clientReadyChannel)

	// Parse the config files
	// Run all preparation

	// Run through the benchmark suite
	// Step 1: send "PREPARE" to clients, make sure we can communicate.
	ms.Server.PrepareBenchmarkClients()

	// Step 2: Blockchain type (tells which interface they should be using)
	ms.Server.SendBlockchainType()

	// Step 3: Prepare the workload for the benchmark
	// TODO: generate workloads

	// Step 4: Distribute benchmark
	ms.Server.SendWorkload()

	// Step 5: run the bench
	ms.Server.RunBenchmark()

	// Step 6 (once all have completed) - get the results
	ms.Server.GetResults()

	// Step 7 - store results
	// TODO: store results

	// Step 8: Close all connections
	ms.Server.CloseClients()
	ms.Server.Close()
}