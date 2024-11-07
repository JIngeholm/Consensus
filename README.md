Hand in 4 Consensus
To run the program clone the repository, then open 3 terminals.
From Consensus/ConsensusGRPC run the command go run server.go with 3 different arguments for ports in the 3 terminals.
The argument are hard-coded and as such the three commands must be (in any order):
go run server.go -port=50051
go run server.go -port=50052
go run server.go -port=50053

Currently if all 3 servers have not been run 10 seconds after the first, an error will occur.
