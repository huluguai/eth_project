// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./bank.sol";

/**
 * @title BigBank 合约
 * @dev 继承自 Bank，要求最小存款金额为 0.001 ETH
 */
contract BigBank is Bank {
    // 最小存款金额：0.001 ETH = 1e15 wei
    uint256 public constant MIN_DEPOSIT_AMOUNT = 1e15;
    
    /**
     * @notice 检查存款金额的修饰器
     * @dev 验证存款金额是否大于等于最小要求
     */
    modifier minDepositAmount() {
        require(msg.value >= MIN_DEPOSIT_AMOUNT, "deposit amount must be >= 0.001 ETH");
        _;
    }
    
    /**
     * @notice 构造函数
     * @dev 部署合约时自动执行，设置部署者为管理员
     */
    constructor() Bank() {
        // Bank 构造函数会自动设置 owner = msg.sender
    }
    
    /**
     * @notice 接收函数（用于接收 ETH）
     * @dev 当合约收到 ETH 且没有匹配函数时自动调用
     *      支持通过 MetaMask 等钱包直接转账到合约地址
     *      必须满足最小存款金额要求
     */
    receive() external payable override nonReentrant minDepositAmount {
        _deposit();
    }
    
    /**
     * @notice 回退函数
     * @dev 当调用不存在的函数或合约收到 ETH 时调用
     *      必须满足最小存款金额要求
     */
    fallback() external payable override nonReentrant minDepositAmount {
        _deposit();
    }
    
    /**
     * @notice 存款函数
     * @dev 记录用户存款金额并更新前 3 名排行榜
     *      可通过 receive/fallback 自动调用，也可主动调用
     *      必须满足最小存款金额要求
     */
    function deposit() public payable override minDepositAmount {
        _deposit();
    }
    
    /**
     * @notice 内部存款逻辑
     * @dev 实际的存款处理逻辑，不加重入锁
     *      注意：_deposit 在 Bank 中已经检查 msg.value > 0，
     *      但我们在外部用 minDepositAmount 修饰器确保 >= 0.001 ETH
     */
    function _deposit() internal override {
        // 调用父类的存款逻辑
        super._deposit();
    }
    
    /**
     * @notice 变更管理员
     * @dev 仅当前管理员可以调用
     *      重写以添加更严格的验证
     * @param newOwner 新管理员地址
     */
    function transferOwnership(address newOwner) public override onlyOwner {
        // 验证新管理员地址有效
        require(newOwner != address(0), "new owner cannot be zero address");
        require(newOwner != owner, "new owner must be different from current owner");
        
        address oldOwner = owner;
        owner = newOwner;
        
        // 触发管理员变更事件
        emit OwnerChanged(oldOwner, newOwner);
    }
}