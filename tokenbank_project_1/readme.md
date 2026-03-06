
# TokenBank Project

## Overview

TokenBank is a decentralized token storage system built on Solidity, consisting of two smart contracts:
- **MyToken**: Custom ERC20 token with minting and burning capabilities
- **TokenBank**: Secure deposit/withdrawal system for MyToken

## 📁 Project Structure

```
token_bank/
├── my_token.sol      # Custom ERC20 Token Contract
├── token_bank.sol    # Token Bank Storage Contract
└── readme.md         # This file
```

## 🚀 Quick Start

### Deployment

1. **Deploy MyToken first:**
```solidity
uint256 initialSupply = 1000000; // 1 million tokens
MyToken token = new MyToken(initialSupply);
```

2. **Deploy TokenBank:**
```solidity
address tokenAddress = address(token);
TokenBank bank = new TokenBank(tokenAddress);
```

### Basic Usage

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

## ⚠️ Important Notes

### Unit Handling

**CRITICAL**: All public functions accept **token units** (human-readable), not smallest units.

- ✅ Correct: `deposit(1000)` means 1000 tokens
- ❌ Wrong: Don't multiply by 10^18 yourself

The contracts handle unit conversion internally:
- User-facing: Token units (e.g., 1000)
- Internal storage: Smallest units (e.g., 1000 × 10^18)

### Security Checks

✅ Balance validation on transfers  
✅ Balance validation on approvals  
✅ Zero address prevention  
✅ Allowance verification  
✅ Owner-only minting  
✅ Overflow protection (Solidity 0.8+)  

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

## 📖 API Reference

### TokenBank State Variables

| Variable | Type | Description |
|----------|------|-------------|
| `token` | MyToken | Token contract instance |
| `deposits` | mapping(address => uint256) | User balances (smallest units) |
| `depositors` | address[] | Depositor address list |
| `hasDeposited` | mapping(address => bool) | Depositor status flag |

### Events

| Event | Contract | Parameters |
|-------|----------|------------|
| `Transfer` | MyToken | from, to, value |
| `Approval` | MyToken | owner, spender, value |
| `Deposit` | TokenBank | user, amount, timestamp |
| `Withdraw` | TokenBank | user, amount, timestamp |

## ⚠️ Security Warnings

1. **Not Audited**: This code has NOT been professionally audited
2. **Test First**: Always test on testnets before mainnet deployment
3. **Use at Own Risk**: Production use requires professional security audit
4. **Private Keys**: Never commit keys/mnemonics to code or version control

## 📄 License

MIT License

## 🤝 Contributing

For questions or issues:
- Submit an Issue on GitHub
- Review existing documentation
- Check community forums

---

**Disclaimer**: This project is for educational purposes only. Not investment advice. Users assume all risks associated with smart contract usage.