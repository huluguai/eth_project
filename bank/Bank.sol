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
    
    // 数组：存储存款金额前 3 名的用户地址（按存款金额降序排列）
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
     * @dev 维护一个最多包含 3 个地址的数组，按存款金额降序排列
     */
    function updateTopDepositors() internal {
        address sender = msg.sender;
        uint256 senderBalance = balances[sender];
        
        // 步骤 1: 从排行榜中移除该用户（如果已存在）
        removeUserFromTopDepositors(sender);
        
        // 步骤 2: 如果排行榜少于 3 人，直接添加
        if (topDepositors.length < 3) {
            topDepositors.push(sender);
            sortTopDepositors();
            return;
        }
        
        // 步骤 3: 检查该用户是否能进入前 3
        uint256 minBalance = balances[topDepositors[2]];
        if (senderBalance > minBalance) {
            // 替换最后一名
            topDepositors[2] = sender;
            sortTopDepositors();
        }
    }

    /**
     * @notice 内部函数：从排行榜中移除指定用户
     * @dev 遍历数组找到并删除指定地址
     * @param user 要移除的用户地址
     */
    function removeUserFromTopDepositors(address user) internal {
        for (uint i = 0; i < topDepositors.length; i++) {
            if (topDepositors[i] == user) {
                // 用最后一个元素替换被删除的元素
                topDepositors[i] = topDepositors[topDepositors.length - 1];
                topDepositors.pop();
                break;
            }
        }
    }

    /**
     * @notice 内部函数：对排行榜进行排序（冒泡排序）
     * @dev 按存款金额降序排列前 3 名用户
     */
    function sortTopDepositors() internal {
        for (uint i = 0; i < topDepositors.length; i++) {
            for (uint j = i + 1; j < topDepositors.length; j++) {
                if (balances[topDepositors[j]] > balances[topDepositors[i]]) {
                    // 交换位置
                    address temp = topDepositors[i];
                    topDepositors[i] = topDepositors[j];
                    topDepositors[j] = temp;
                }
            }
        }
        
        // 触发更新事件
        emit TopDepositorsUpdated(topDepositors);
    }

    /**
     * @notice 查询前 3 名存款用户
     * @return 前 3 名用户地址数组（按存款金额降序排列）
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