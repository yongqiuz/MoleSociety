const PostAttestor = artifacts.require("PostAttestor");

contract("PostAttestor", (accounts) => {
  it("emits the attestation event with sender and hash", async () => {
    const attestor = await PostAttestor.new({ from: accounts[0] });
    const hash = web3.utils.keccak256("post-body");

    const receipt = await attestor.attest(hash, { from: accounts[1] });

    assert.equal(receipt.logs.length, 1);
    assert.equal(receipt.logs[0].event, "Attested");
    assert.equal(receipt.logs[0].args.author, accounts[1]);
    assert.equal(receipt.logs[0].args.hash, hash);
  });
});
