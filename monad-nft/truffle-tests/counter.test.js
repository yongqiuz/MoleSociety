const Counter = artifacts.require("Counter");

contract("Counter", (accounts) => {
  it("sets the provided number", async () => {
    const counter = await Counter.new({ from: accounts[0] });
    await counter.setNumber(42, { from: accounts[0] });

    const value = await counter.number();
    assert.equal(value.toString(), "42");
  });

  it("increments the stored number", async () => {
    const counter = await Counter.new({ from: accounts[0] });
    await counter.increment({ from: accounts[0] });
    await counter.increment({ from: accounts[0] });

    const value = await counter.number();
    assert.equal(value.toString(), "2");
  });
});
