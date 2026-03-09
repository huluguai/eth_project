
# TokenBank Project

## Overview

TokenBank is a decentralized token storage system built on Solidity, consisting of smart contracts:
- **MyToken**: Custom ERC20 token with minting and burning capabilities
- **MyTokenV2**: Extended ERC20 token with hook support via `transferWithCallback`
- **TokenBank**: Secure deposit/withdrawal system for MyToken
- **TokenBankV2**: Enhanced version supporting direct deposits via hook mechanism

## 📁 Project Structure

```
token_bank/
├── my_token.sol          # Custom ERC20 Token Contract
├── my_token_v2.sol       # ERC20 with Hook Support (transferWithCallback)
├── token_bank.sol        # Token Bank Storage Contract
├── token_bank_v2.sol     # Enhanced Bank with Hook Support
└── readme.md             # This file
```

## 🚀 Quick Start

### Deployment

1. **Deploy MyToken:**
```solidity
uint256 initialSupply = 1000000; // 1 million tokens
MyToken token = new MyToken(initialSupply);
```

2. **Deploy MyTokenV2 (with Hook):**
```solidity
uint256 initialSupply = 1000000; // 1 million tokens
MyTokenV2 tokenV2 = new MyTokenV2(initialSupply);
```

3. **Deploy TokenBank:**
```solidity
address tokenAddress = address(token);
TokenBank bank = new TokenBank(tokenAddress);
```

4. **Deploy TokenBankV2 (with Hook):**
```solidity
address tokenV2Address = address(tokenV2);
TokenBankV2 bankV2 = new TokenBankV2(tokenV2Address);
```

### Basic Usage

#### Traditional Method (ERC20 Standard)
```javascript
// 1. Approve TokenBank to use your tokens
await token.approve(bankAddress, 1000);

// 2. Deposit tokens
await bank.deposit(1000);

// 3. Withdraw tokens
await bank.withdraw(500);

// 4. Check balance
const balance = await bank.getDepositBalance(userAddress);
```

#### New Hook Method (transferWithCallback)
```javascript
// Direct deposit without separate approval step
await tokenV2.transferWithCallback(bankV2Address, 1000, "0x");

// The TokenBankV2.tokensReceived() is automatically called
// Deposit is recorded in a single transaction

// Check balance
const balance = await bankV2.getDepositBalance(userAddress);
```

## 📋 Contract Details

### MyToken (ERC20 Token)

**Basic Info:**
- Name: MyToken
- Symbol: MTK
- Decimals: 18
- Features: Transfer, Approve, TransferFrom, Mint, Burn

**Key Functions:**
- `transfer(address to, uint256 amount)` - Transfer tokens
- `approve(address spender, uint256 amount)` - Approve spending (checks balance)
- `transferFrom(address from, address to, uint256 amount)` - Transfer on behalf
- `mint(uint256 amount)` - Create new tokens (owner only)
- `burn(uint256 amount)` - Destroy tokens

### MyTokenV2 (Extended ERC20 with Hook)

**Additional Features:**
- ✅ All MyToken functionality
- ✅ `transferWithCallback(address to, uint256 value, bytes data)` - Transfer with callback support
- ✅ Automatically calls `tokensReceived()` on recipient contracts
- ✅ Emits `TransferWithCallback` event for tracking

**Hook Mechanism:**
- When transferring to a contract address, automatically calls the recipient's `tokensReceived()` method
- Recipient must implement `ITokenRecipient` interface
- Transaction reverts if callback fails (ensures atomicity)
- Gas-efficient single-step deposit process

**Interface Definition:**
```solidity
interface ITokenRecipient {
    function tokensReceived(
        address operator,
        address from,
        address to,
        uint256 amount,
        bytes calldata data
    ) external returns (bool);
}
```

### TokenBank (Storage Contract)

**Features:**
- 💰 Deposit MyToken tokens
- 💸 Withdraw deposits anytime
- 📊 Track individual balances
- 📝 Maintain depositor registry
- 🔔 Real-time event logging

**Core Functions:**
- `deposit(uint256 amount)` - Deposit tokens to bank
- `withdraw(uint256 amount)` - Withdraw tokens from bank
- `getDepositBalance(address user)` - Get user's deposit balance
- `getAllDepositors()` - Get all depositor addresses
- `getDepositorsCount()` - Get total number of depositors
- `getTotalBalance()` - Get contract's total balance

### TokenBankV2 (Enhanced with Hook Support)

**Additional Features:**
- ✅ All TokenBank functionality
- ✅ Implements `tokensReceived()` callback interface
- ✅ Accepts direct deposits via `transferWithCallback`
- ✅ Backward compatible with traditional `deposit()` method
- ✅ Emits `DepositViaHook` event for hook-based deposits

**How It Works:**
1. User calls `tokenV2.transferWithCallback(TokenBankV2_address, amount, data)`
2. MyTokenV2 transfers tokens to TokenBankV2
3. MyTokenV2 detects TokenBankV2 is a contract
4. MyTokenV2 calls `TokenBankV2.tokensReceived()`
5. TokenBankV2 records the deposit automatically
6. Transaction completes atomically

**Comparison: Traditional vs Hook Method**

| Aspect | Traditional (approve + deposit) | Hook (transferWithCallback) |
|--------|--------------------------------|-----------------------------|
| Transactions | 2 | 1 |
| Gas Cost | Higher (2 txns) | Lower (1 txn) |
| User Experience | Two steps | Single step |
| Approval Required | Yes | No |
| Atomicity | Per transaction | Entire flow |

## ⚠️ Important Notes

### Unit Handling

**CRITICAL**: All public functions accept **token units** (human-readable), not smallest units.

- ✅ Correct: `deposit(1000)` means 1000 tokens
- ❌ Wrong: Don't multiply by 10^18 yourself

The contracts handle unit conversion internally:
- User-facing: Token units (e.g., 1000)
- Internal storage: Smallest units (e.g., 1000 × 10^18)

### Hook Callback Behavior

**Transaction Rollback:**
- If `tokensReceived()` callback fails, the entire `transferWithCallback` transaction rolls back
- This ensures atomicity: tokens are never transferred without proper deposit recording
- All state changes and events are reverted if any step fails

**Example Flow:**
```javascript
try {
    await tokenV2.transferWithCallback(bankV2Address, 1000, "0x");
    // ✅ Success: Tokens transferred + Deposit recorded
} catch (error) {
    // ❌ Failure: Everything rolls back
    // - User keeps their tokens
    // - No deposit record created
    // - No events emitted
}
```

### Security Checks

✅ Balance validation on transfers  
✅ Balance validation on approvals  
✅ Zero address prevention  
✅ Allowance verification  
✅ Owner-only minting  
✅ Overflow protection (Solidity 0.8+)  
✅ Callback validation in hooks  
✅ Atomic transaction guarantee  

## 🔧 Development

### Requirements

- Solidity: ^0.8.20
- Network: Any EVM-compatible chain

### Recommended Tools

- **Framework**: Hardhat / Foundry
- **Testing**: Chai / Mocha
- **Wallet**: MetaMask
- **Explorer**: Etherscan

### Testing Guide

```bash
# Compile
npx hardhat compile

# Test
npx hardhat test

# Deploy to testnet
npx hardhat run scripts/deploy.js --network goerli
```

### Example Test Cases

```javascript
describe("MyTokenV2 & TokenBankV2", function () {
    it("Should deposit via transferWithCallback", async function () {
        const amount = 1000;
        
        // User calls transferWithCallback directly
        await tokenV2.transferWithCallback(bankV2.address, amount, "0x");
        
        // Verify deposit was recorded
        const balance = await bankV2.getDepositBalance(user.address);
        expect(balance).to.equal(amount);
    });
    
    it("Should rollback if callback fails", async function () {
        // Deploy a contract that always reverts in tokensReceived
        const badContract = await BadTokenRecipient.deploy();
        
        // This should fail and rollback
        await expect(
            tokenV2.transferWithCallback(badContract.address, 1000, "0x")
        ).to.be.reverted;
    });
});
```

## 📖 API Reference

### MyTokenV2 State Variables

| Variable | Type | Description |
|----------|------|-------------|
| `name` | string | Token name |
| `symbol` | string | Token symbol |
| `decimals` | uint8 | Token precision (18) |
| `totalSupply` | uint256 | Total token supply |
| `balanceOf` | mapping(address => uint256) | User balances |
| `allowance` | mapping(address => mapping(address => uint256)) | Spending allowances |
| `owner` | address | Contract owner |

### TokenBankV2 State Variables

| Variable | Type | Description |
|----------|------|-------------|
| `token` | MyToken | Original token contract instance |
| `tokenV2` | MyTokenV2 | Extended token contract instance |
| `deposits` | mapping(address => uint256) | User balances (smallest units) |
| `depositors` | address[] | Depositor address list |
| `hasDeposited` | mapping(address => bool) | Depositor status flag |

### Events

| Event | Contract | Parameters |
|-------|----------|------------|
| `Transfer` | MyToken/MyTokenV2 | from, to, value |
| `Approval` | MyToken/MyTokenV2 | owner, spender, value |
| `TransferWithCallback` | MyTokenV2 | from, to, value, data |
| `Deposit` | TokenBank/TokenBankV2 | user, amount, timestamp |
| `Withdraw` | TokenBank/TokenBankV2 | user, amount, timestamp |
| `DepositViaHook` | TokenBankV2 | user, amount, timestamp, data |

## 🎯 Use Cases

### When to Use Hook Method (transferWithCallback):
- ✅ One-click deposits to TokenBankV2
- ✅ Simplifying user experience (single transaction)
- ✅ Reducing gas costs (one transaction instead of two)
- ✅ Contracts receiving tokens automatically

### When to Use Traditional Method (approve + deposit):
- ✅ Working with MyToken (v1) without hook support
- ✅ Need to set recurring allowances
- ✅ Using existing DEX/protocol integrations
- ✅ Backward compatibility requirements

## ⚠️ Security Warnings

1. **Not Audited**: This code has NOT been professionally audited
2. **Test First**: Always test on testnets before mainnet deployment
3. **Use at Own Risk**: Production use requires professional security audit
4. **Private Keys**: Never commit keys/mnemonics to code or version control
5. **Callback Risks**: Ensure recipient contracts properly implement `tokensReceived()`
6. **Reentrancy**: Hook callbacks could be exploited; implement checks-effects-interactions pattern

## 🔄 Migration Guide

### From MyToken to MyTokenV2:
```solidity
// Deploy new V2 contract
MyTokenV2 tokenV2 = new MyTokenV2(initialSupply);

// Optionally migrate users from old token
// (Implement migration mechanism if needed)
```

### From TokenBank to TokenBankV2:
```solidity
// Deploy new V2 contract with MyTokenV2 address
TokenBankV2 bankV2 = new TokenBankV2(address(tokenV2));

// Users can continue using traditional method OR new hook method
// Both methods work seamlessly
```

## 📄 License

MIT License

## 🤝 Contributing

For questions or issues:
- Submit an Issue on GitHub
- Review existing documentation
- Check community forums

---

**Disclaimer**: This project is for educational purposes only. Not investment advice. Users assume all risks associated with smart contract usage.
