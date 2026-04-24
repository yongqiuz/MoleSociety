const ganache = require("ganache");

let providerInstance;

const sharedProvider = () => {
  if (providerInstance) {
    return providerInstance;
  }

  providerInstance = ganache.provider({
    logging: { quiet: true },
    wallet: { totalAccounts: 10 },
    chain: { chainId: 1337 },
  });
  return providerInstance;
};

module.exports = {
  contracts_directory: "./src",
  test_directory: "./truffle-tests",
  contracts_build_directory: "./build/truffle-contracts",
  networks: {
    development: {
      provider: sharedProvider,
      network_id: "*"
    },
    test: {
      provider: sharedProvider,
      network_id: "*"
    },
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
