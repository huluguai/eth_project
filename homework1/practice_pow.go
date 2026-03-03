package homework1

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

/**
实践 POW， 编写程序（用go语言）用自己的昵称 xzx + nonce，不断修改nonce 进行 sha256 Hash 运算：
	直到满足 4 个 0 开头的哈希值，打印出花费的时间、Hash 的内容及Hash值。
	再次运算直到满足 5 个 0 开头的哈希值，打印出花费的时间、Hash 的内容及Hash值。
**/

// calulateHash 计算Sha256哈希值 / calulateHash computes the SHA256 hash of the input string and returns it as a hexadecimal string.
// 该函数将输入字符串转换为字节数组，计算其 SHA256 哈希值，并将结果编码为十六进制字符串返回。
func calulateHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// findNonce 寻找满足条件的 nonce 值 / findNonce searches for a nonce value such that the hash of the nickname concatenated with the nonce has a specified number of leading zeros.
// 该函数接受一个昵称和一个整数 leadingZeros，表示哈希值需要以多少个前导零开头。函数通过不断增加 nonce 的值，计算 nickname + nonce 的哈希值，直到找到满足条件的 nonce。
func findNonce(nickname string, leadingZeros int) (string, string, int64) {
	startTime := time.Now()
	nonce := int64(0)
	// 构建目标前缀（指定数量的 0）/ Build the target prefix (specified number of zeros)
	targetPrefix := ""
	for i := 0; i < leadingZeros; i++ {
		targetPrefix += "0"
	}
	var intput string
	var hash string
	// 不断尝试不同的Nonce值，直到找到满足条件的哈希值/ Keep trying different nonce values until a hash with the required leading zeros is found
	for {
		// 组合昵称和Nonce值，计算哈希/ Combine nickname and nonce to calulate the hash
		intput = fmt.Sprintf("%s%d", nickname, nonce)
		hash = calulateHash(intput)
		// 检查hash值是否满足前导条件/ Check if the hash meets the leading zeros condition
		if len(hash) >= len(targetPrefix) && hash[:len(targetPrefix)] == targetPrefix {
			//计算耗时/ calulate elapsed time
			elapsedTime := time.Since(startTime).Microseconds()
			return intput, hash, elapsedTime
		}
		//增加nonce 值继续尝试/ Increment nonce value and continue trying
		nonce++
	}
}

func main() {
	//设置昵称 / Set the nickename
	nickname := "xzx"
	fmt.Println("=== 工作量证明 (PoW) 实践 ===")
	fmt.Println("=== Proof of Work (PoW) Practice ===")
	fmt.Printf("昵称 (Nickname): %s\n\n", nickname)
	//查找以 4个 0 开头的哈希值 / Find a hash value that starts with 4 leading zeros
	intput, hash, elapsedTime := findNonce(nickname, 4)
	fmt.Printf("找到的输入 (Found input): %s\n", intput)
	fmt.Printf("对应的哈希值 (Corresponding hash): %s\n", hash)
	fmt.Printf("寻找过程耗时 (Time taken to find): %d 微秒\n", elapsedTime)

	//查找以 5个 0 开头的哈希值 / Find a hash value that starts with 5 leading zeros
	intput, hash, elapsedTime = findNonce(nickname, 5)
	fmt.Printf("\n找到的输入 (Found input): %s\n", intput)
	fmt.Printf("对应的哈希值 (Corresponding hash): %s\n", hash)
	fmt.Printf("寻找过程耗时 (Time taken to find): %d 微秒\n", elapsedTime)
}
