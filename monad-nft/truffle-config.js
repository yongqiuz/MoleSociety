module.exports = {
  contracts_directory: "./src",
  test_directory: "./truffle-tests",
  contracts_build_directory: "./build/truffle-contracts",
  networks: {
    development: {
      host: "127.0.0.1",
      port: 9545,
      network_id: "*"
    }
  },
  compilers: {
    solc: {
      version: "0.8.20",
      settings: {
        optimizer: {
          enabled: true,
          runs: 200
        }
      }
    }
  }
};
