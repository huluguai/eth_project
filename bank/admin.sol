// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./ibank.sol";

/**
 * @title Admin 合约
 * @dev 管理员合约，可以管理多个银行合约的资金
 *      通过调用 IBank 接口的 withdraw 方法将资金转移到 Admin 合约
 */
contract Admin {
    // Admin 合约的管理员
    address public owner;
    
    // 重入锁状态
    bool private locked;
    
    /**
     * @notice 提款成功事件
     * @param bank 被提款的银行合约地址
     * @param amount 提款金额（wei）
     */
    event AdminWithdrawal(address indexed bank, uint256 amount);
    
    /**
     * @notice 管理员变更事件
     * @param oldOwner 原管理员地址
     * @param newOwner 新管理员地址
     */
    event OwnerChanged(address indexed oldOwner, address indexed newOwner);
    
    /**
     * @notice 仅管理员可以调用的修饰器
     * @dev 检查调用者是否为合约管理员
     */
    modifier onlyOwner() {
        require(msg.sender == owner, "only owner can call this function");
        _;
    }
    
    /**
     * @notice 防止重入攻击修饰器
     * @dev 在函数执行期间设置锁，防止递归调用
     */
    modifier nonReentrant() {
        require(!locked, "reentrant call not allowed");
        locked = true;
        _;
        locked = false;
    }
    
    /**
     * @notice 构造函数
     * @dev 部署合约时自动执行，设置部署者为管理员
     */
    constructor() {
        owner = msg.sender;
    }
    
    /**
     * @notice 从银行合约提款到 Admin 合约
     * @dev 调用 IBank 接口的 withdraw 方法
     *      该操作会将银行合约的所有资金转移到 Admin 合约地址
     * @param bank IBank 接口实例（银行合约地址）
     */
    function adminWithdraw(IBank bank) public onlyOwner nonReentrant {
        // 验证银行合约地址有效
        require(address(bank) != address(0), "bank address cannot be zero");
        
        // 获取银行合约余额
        uint256 balance = bank.getContractBalance();
        require(balance > 0, "bank contract has insufficient balance");
        
        // 验证调用者是银行合约的管理员
        require(bank.owner() == address(this), "caller is not the owner of the bank");
        
        // 调用银行合约的 withdraw 方法
        // 这会将资金从银行合约转移到 Admin 合约（因为 Admin 是银行的管理员）
        bank.withdraw();
        
        // 触发提款事件
        emit AdminWithdrawal(address(bank), balance);
    }
    
    /**
     * @notice 从 Admin 合约提款给管理员
     * @dev 仅管理员可以调用，提取 Admin 合约中的所有资金
     */
    function withdrawToOwner() public onlyOwner nonReentrant {
        uint256 balance = address(this).balance;
        require(balance > 0, "insufficient contract balance");
        
        (bool success, ) = payable(owner).call{value: balance}("");
        require(success, "withdrawal failed");
    }
    
    /**
     * @notice 变更管理员
     * @dev 仅当前管理员可以调用
     * @param newOwner 新管理员地址
     */
    function transferOwnership(address newOwner) public onlyOwner {
        require(newOwner != address(0), "new owner cannot be zero address");
        require(newOwner != owner, "new owner must be different from current owner");
        
        address oldOwner = owner;
        owner = newOwner;
        
        emit OwnerChanged(oldOwner, newOwner);
    }
    
    /**
     * @notice 查询 Admin 合约余额
     * @return 合约当前的 ETH 余额（wei）
     */
    function getAdminBalance() public view returns (uint256) {
        return address(this).balance;
    }
    
    /**
     * @notice 接收函数
     * @dev 允许 Admin 合约接收 ETH（从银行合约转移来的资金）
     */
    receive() external payable {
        // 允许接收 ETH，不需要特殊处理
    }
    
    /**
     * @notice 回退函数
     * @dev 允许接收 ETH
     */
    fallback() external payable {
        // 允许接收 ETH，不需要特殊处理
    }
}