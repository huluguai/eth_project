// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

interface IBank {
    /**
     * @notice 获取合约管理员地址
     * @return 管理员地址
     */
    function owner() external view returns (address);
    
    /**
     * @notice 获取合约总余额
     * @return 合约当前的 ETH 余额（wei）
     */
    function getContractBalance() external view returns (uint256);
    
    /**
     * @notice 提款函数
     * @dev 仅管理员可以调用，提取合约中的所有资金
     */
    function withdraw() external;
    
    /**
     * @notice 变更管理员
     * @param newOwner 新管理员地址
     */
    function transferOwnership(address newOwner) external;
}
