// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script} from "forge-std/Script.sol";
import {PostAttestor} from "../src/PostAttestor.sol";

contract DeployPostAttestor is Script {
    function run() external returns (PostAttestor deployed) {
        vm.startBroadcast();
        deployed = new PostAttestor();
        vm.stopBroadcast();
    }
}
