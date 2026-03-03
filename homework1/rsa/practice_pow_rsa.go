package homework1

//实践非对称加密 RSA（用go语言）：
//先生成一个公私钥对
//用私钥对符合 POW 4 个 0 开头的哈希值的 “昵称 + nonce” 进行私钥签名
//用公钥验证
import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"
)

// 计算哈希值 / calulateHash computes the SHA256 hash of the input string and returns it as a hexadecimal string.
// 该函数将输入字符串转换为字节数组，计算其 SHA256 哈希值，并将结果编码为十六进制字符串返回。
func calulateHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// 寻找满足条件的 nonce 值 / findNonce searchers for a nonce value such that the hash of the nickname concatenated with the nonce has a specified nunmber of leading zeros.
func findNonce(nickname string, leadingZero int) (string, string) {
	targetPrefix := ""
	for i := 0; i < leadingZero; i++ {
		targetPrefix += "0"
	}
	var intput string
	var hash string
	nonce := 0
	for {
		intput = fmt.Sprintf("%s%d", nickname, nonce)
		hash = calulateHash(intput)
		if len(hash) >= len(targetPrefix) && hash[:len(targetPrefix)] == targetPrefix {
			return intput, hash
		}
		nonce++
	}
}

// 生成RSA密钥对 / generateRSAkeyPair generates on RSA key pair with the specified number of bits.
func generateRSAkeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	fmt.Printf("正在生成 %d 位 RSA 密钥对...\n", bits)
	startTime := time.Now()

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)

	if err != nil {
		return nil, nil, fmt.Errorf("生成私钥失败 / Failed to generate private key: %v", err)
	}
	//从私钥提取公钥 / Extact public key from the private kye
	publicKey := &privateKey.PublicKey
	elapsedTime := time.Since(startTime).Seconds()
	fmt.Printf("RSA 密钥对生成完成，耗时 %.2f 秒\n", elapsedTime)
	return privateKey, publicKey, nil
}

// 用私钥对数据签名 / signWithPrivateKey signs the given data useing the provided RSA private key and returns the signature.
func signWithPrivateKey(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	fmt.Println("正在使用私钥进行签名...")
	fmt.Printf("要签名的数据: %s\n", string(data))
	startTime := time.Now()
	//计算数据hash值 / calulate the hash of the data
	hash := sha256.Sum256(data)
	//使用私钥对哈希值进行签名 / Sign the hash with the private key
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, fmt.Errorf("签名失败 / Failed to sign data: %v", err)
	}
	elapsedTime := time.Since(startTime).Seconds()
	fmt.Printf("签名完成，耗时 %.2f 秒\n", elapsedTime)
	return signature, nil
}

// 用公钥签名 / verifyWithPublicKey verifies the signature of the given data using the provided RSA public key. It returns an error if the verification fails.
func verifyWithPublicKey(publicKey *rsa.PublicKey, data []byte, signature []byte) error {
	fmt.Println("正在使用公钥进行验证...")
	fmt.Printf("要验证的数据: %s\n", string(data))
	startTime := time.Now()
	//计算数据hash值 / calulate the hash of the data
	hash := sha256.Sum256(data)
	//使用公钥验证签名 / Verify the signature with the public key
	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	elapsedTime := time.Since(startTime).Seconds()
	if err != nil {
		fmt.Printf("验证失败，耗时 %.2f 秒\n", elapsedTime)
		return fmt.Errorf("验证失败 / Failed to verify signature: %v", err)
	}
	fmt.Printf("验证成功，耗时 %.2f 秒\n", elapsedTime)
	return nil
}

// 导出私钥为PEM格式 / exportPrivateKeyToPEM exports the given RSA private key to PEM format and returns it as a string.
func exportPrivateKeyToPEM(privateKey *rsa.PrivateKey) string {
	privateByates := x509.MarshalPKCS1PrivateKey(privateKey)
	privateBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateByates,
	}
	return string(pem.EncodeToMemory(privateBlock))
}

// 导出公钥为PEM格式 / exportPublicKeyToPEM exports the given RSA public key to PEM format and returns it as a string.
func exportPublicKeyToPEM(publicKey *rsa.PublicKey) string {
	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		fmt.Printf("导出公钥失败 / Failed to export public key: %v\n", err)
		return ""
	}
	publicBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicBytes,
	}
	return string(pem.EncodeToMemory(publicBlock))
}

func main() {
	nickname := "xzx" // 设置昵称 / Set nickname

	fmt.Println("========================================")
	fmt.Println("=== RSA 非对称加密实践 ===")
	fmt.Println("=== RSA Asymmetric Encryption Practice ===")
	// 步骤1 生成RSA密钥对 / Step 1: Generate RSA key pair
	pivateKey, publicKey, err := generateRSAkeyPair(2048)
	if err != nil {
		fmt.Printf("生成RSA密钥对失败 / Failed to generate RSA key pair: %v\n", err)
		return
	}
	fmt.Println("\n=== RSA 密钥对生成成功 ===")
	fmt.Printf("私钥 (Private Key):\n%s\n", exportPrivateKeyToPEM(pivateKey))
	fmt.Printf("公钥 (Public Key):\n%s\n", exportPublicKeyToPEM(publicKey))
	// 步骤2 执行POW找到 4 个满足前导0的输入 / Step 2: Perform PoW to find an input with 4 leading zeros
	fmt.Println("\n=== 工作量证明 (Proof of Work) ===")
	fmt.Println("正在寻找满足 4 个前导零的哈希值...")
	fmt.Println("Searching for hash with 4 leading zeros...")
	powInput, powHash := findNonce(nickname, 4)
	fmt.Printf("✓ PoW 完成 (PoW completed)\n")
	fmt.Printf("  输入内容 (Input): %s\n", powInput)
	fmt.Printf("  哈希值 (Hash):    %s\n\n", powHash)
	// 步骤3 使用私钥对PoW进行签名 / Step 3: Sign the PoW input with the private key
	fmt.Printf("=== 使用私钥对 PoW 结果进行签名 ===")
	inputData := []byte(powInput)
	signature, err := signWithPrivateKey(pivateKey, inputData)
	if err != nil {
		fmt.Printf("签名失败 / Failed to sign PoW result: %v\n", err)
		return
	}
	fmt.Printf("  原始数据 (Original data): %s\n", powInput)
	fmt.Printf("  签名结果 (Signature):     %s\n\n", hex.EncodeToString(signature))
	// 步骤 4: 使用公钥验证签名 / Step 4: Verify signature with public key
	fmt.Println("=== RSA 签名验证 (Signature Verification) ===")
	err = verifyWithPublicKey(publicKey, inputData, signature)
	if err != nil {
		fmt.Printf("错误 (Error): %v\n", err)
		return
	}
	// 步骤 5: 演示篡改数据后验证失败 / Step 5: Demonstrate verification failure after data tampering
	fmt.Println("=== 数据完整性测试 (Data Integrity Test) ===")
	fmt.Println("尝试验证被篡改的数据...")
	fmt.Println("Attempting to verify tampered data...")

	tamperedData := []byte(powInput + "_tampered")
	err = verifyWithPublicKey(publicKey, tamperedData, signature)
	if err != nil {
		fmt.Printf("✓ 正确检测到数据篡改 (Correctly detected data tampering)\n")
		fmt.Printf("  原始数据 (Original):  %s\n", powInput)
		fmt.Printf("  篡改数据 (Tampered):  %s\n\n", string(tamperedData))
	}

	fmt.Println("========================================")
	fmt.Println("=== 实践完成 (Practice Completed) ===")
	fmt.Println("========================================")
	fmt.Println("\n核心概念 (Core Concepts):")
	fmt.Println("1. 私钥签名 (Private key signs) - 只有私钥持有者能生成签名")
	fmt.Println("   Only private key holder can generate signature")
	fmt.Println("2. 公钥验证 (Public key verifies) - 任何人都可以用公钥验证签名")
	fmt.Println("   Anyone can verify signature with public key")
	fmt.Println("3. 数据完整性 (Data integrity) - 数据被篡改后验证会失败")
	fmt.Println("   Verification fails if data is tampered")
	fmt.Println("4. 不可抵赖性 (Non-repudiation) - 私钥签名者无法否认签名行为")
	fmt.Println("   Private key signer cannot deny their signature")

}
