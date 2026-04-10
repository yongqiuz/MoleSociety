// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract PostAttestor {
    event Attested(address indexed author, bytes32 indexed hash);

    function attest(bytes32 hash) external {
        emit Attested(msg.sender, hash);
    }
}
