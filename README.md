# B-DNS Blockchain  

B-DNS is a blockchain-based decentralized DNS system using a **Proof-of-Stake (PoS)** consensus mechanism. It stores domain records as transactions in an immutable ledger,  maintaining security and decentralization.  

## Instructions

### Dependencies
Before running the project, ensure all dependencies are installed by executing:
   ```sh
   go mod tidy
   ```

### Run Program
   ```sh
   go run main.go
   ```

### Run Simulation
   ```sh
   mkdir chaindata
   rm .\chaindata\*
   make
   ```

### Run Linting
```sh
golangci-lint run  # to identify all the issues
golangci-lint run --fix # to automatically fix the fixable issues
```