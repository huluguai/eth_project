// SPDX-License-Identifier: MIT
// 编译器版本声明，使用 Solidity 0.8.0 或更高版本
pragma solidity ^0.8.0;

/**
 * @title Bank 合约
 * @dev 实现存款、提款和记录前 3 名存款用户的功能
 */
contract Bank {
    // 合约管理员地址（部署者自动成为管理员）
    address public owner;
    
    // 映射：记录每个用户的存款余额
    mapping(address => uint256) public balances;
    
    // 数组：存储存款金额前 3 名的用户地址
    address[] public topDepositors;
    
    // ========== 事件定义 ==========
    
    /// @notice 存款事件
    /// @param user 存款用户地址
    /// @param amount 存款金额（wei）
    event Deposit(address indexed user, uint256 amount);
    
    /// @notice 提款事件
    /// @param admin 提款的管理员地址
    /// @param amount 提款金额（wei）
    event Withdrawal(address indexed admin, uint256 amount);
    
    /// @notice 前 3 名存款用户更新事件
    /// @param top3 前 3 名用户地址数组
    event TopDepositorsUpdated(address[] top3);

    /**
     * @notice 仅管理员可以调用的修饰器
     * @dev 检查调用者是否为合约管理员
     */
    modifier onlyOwner() {
        require(msg.sender == owner, "only owner can call this function");
        _;
    }

    /**
     * @notice 构造函数
     * @dev 部署合约时自动执行，设置部署者为管理员
     */
    constructor() {
        owner = msg.sender;
    }

    /**
     * @notice 接收函数（用于接收 ETH）
     * @dev 当合约收到 ETH 且没有匹配函数时自动调用
     *      支持通过 MetaMask 等钱包直接转账到合约地址
     */
    receive() external payable {
        deposit();
    }

    /**
     * @notice 回退函数
     * @dev 当调用不存在的函数或合约收到 ETH 时调用
     */
    fallback() external payable {
        deposit();
    }

    /**
     * @notice 存款函数
     * @dev 记录用户存款金额并更新前 3 名排行榜
     *      可通过 receive/fallback 自动调用，也可主动调用
     */
    function deposit() public payable {
        // 验证存款金额必须大于 0
        require(msg.value > 0, "deposit amount must be greater than 0");
        
        // 累加用户存款余额
        balances[msg.sender] += msg.value;
        
        // 更新前 3 名存款用户
        updateTopDepositors();
        
        // 触发存款事件
        emit Deposit(msg.sender, msg.value);
    }

    /**
     * @notice 提款函数
     * @dev 仅管理员可以调用，提取合约中的所有资金
     */
    function withdraw() public onlyOwner {
        // 获取合约当前余额
        uint256 balance = address(this).balance;
        
        // 验证合约有足够余额
        require(balance > 0, "insufficient contract balance");
        
        // 将全部余额转账给管理员
        // 使用 call 方式而非 transfer/send（更安全，不受 gas 限制）
        (bool success, ) = payable(owner).call{value: balance}("");
        require(success, "withdrawal failed");
        
        // 触发提款事件
        emit Withdrawal(owner, balance);
    }

    /**
     * @notice 内部函数：更新前 3 名存款用户
     * @dev 维护一个最多包含 3 个地址的数组，按存款金额排序
     */
    function updateTopDepositors() internal {
        address sender = msg.sender;
        
        // 如果该用户已在排行榜中，先移除
        for (uint i = 0; i < topDepositors.length; i++) {
            if (topDepositors[i] == sender) {
                topDepositors[i] = address(0);
                break;
            }
        }
        
        // 将当前用户添加到排行榜末尾
        topDepositors.push(sender);
        
        // 如果超过 3 人，移除存款最少的用户
        if (topDepositors.length > 3) {
            // 查找存款最少的用户索引
            uint256 minIndex = 0;
            uint256 minBalance = balances[topDepositors[0]];
            
            for (uint i = 1; i < topDepositors.length; i++) {
                if (balances[topDepositors[i]] < minBalance) {
                    minBalance = balances[topDepositors[i]];
                    minIndex = i;
                }
            }
            
            // 用最后一个元素替换最少存款用户，然后弹出
            topDepositors[minIndex] = topDepositors[topDepositors.length - 1];
            topDepositors.pop();
        }
    }

    /**
     * @notice 查询前 3 名存款用户
     * @return 前 3 名用户地址数组（可能少于 3 个，如果存款用户不足 3 人）
     */
    function getTopDepositors() public view returns (address[] memory) {
        return topDepositors;
    }

    /**
     * @notice 查询合约总余额
     * @return 合约当前的 ETH 余额（wei）
     */
    function getContractBalance() public view returns (uint256) {
        return address(this).balance;
    }
}