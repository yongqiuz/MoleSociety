const MonadSimpleMint = artifacts.require("MonadSimpleMint");

contract("MonadSimpleMint", (accounts) => {
  const [owner, alice] = accounts;

  it("assigns ownership to the deployer", async () => {
    const nft = await MonadSimpleMint.new({ from: owner });
    const contractOwner = await nft.owner();

    assert.equal(contractOwner, owner);
  });

  it("mints the requested quantity to the caller", async () => {
    const nft = await MonadSimpleMint.new({ from: owner });

    await nft.mint(2, { from: alice, value: 0 });

    const supply = await nft.totalSupply();
    const balance = await nft.balanceOf(alice);
    const ownerOfFirst = await nft.ownerOf(0);
    const ownerOfSecond = await nft.ownerOf(1);

    assert.equal(supply.toString(), "2");
    assert.equal(balance.toString(), "2");
    assert.equal(ownerOfFirst, alice);
    assert.equal(ownerOfSecond, alice);
  });

  it("lets only the owner set the base URI", async () => {
    const nft = await MonadSimpleMint.new({ from: owner });
    await nft.mint(1, { from: alice, value: 0 });

    await nft.setBaseURI("ipfs://example/", { from: owner });
    const tokenUri = await nft.tokenURI(0);
    assert.equal(tokenUri, "ipfs://example/0");

    try {
      await nft.setBaseURI("ipfs://forbidden/", { from: alice });
      assert.fail("expected revert for non-owner");
    } catch (error) {
      assert.match(error.message, /revert|OwnableUnauthorizedAccount/);
    }
  });

  it("rejects minting beyond max supply", async () => {
    const nft = await MonadSimpleMint.new({ from: owner });
    const maxSupply = await nft.MAX_SUPPLY();

    try {
      await nft.mint(maxSupply.addn(1), { from: alice, value: 0 });
      assert.fail("expected max supply revert");
    } catch (error) {
      assert.include(error.message, "Exceeds max supply");
    }
  });
});
