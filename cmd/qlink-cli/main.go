package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/qujing226/QLink/did/crypto"
	"github.com/qujing226/QLink/pkg/client"
	"github.com/spf13/cobra"
)

var (
	// 全局配置
	baseURL    string
	keyFile    string
	clientInst *client.Client
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "qlink-cli",
	Short: "QLink DID系统命令行工具",
	Long:  `QLink DID系统的命令行客户端，支持DID的注册、解析、更新和撤销操作。`,
}

// generateCmd 生成密钥对命令
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "生成新的密钥对",
	Long:  `生成新的混合密钥对并保存到文件。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 生成密钥对
		keyPair, err := crypto.GenerateHybridKeyPair()
		if err != nil {
			log.Fatalf("生成密钥对失败: %v", err)
		}

		// 获取公钥JWK
		jwk, err := keyPair.ToJWK()
		if err != nil {
			log.Fatalf("获取公钥JWK失败: %v", err)
		}

		// 获取指纹
		fingerprint, err := keyPair.GetFingerprint()
		if err != nil {
			log.Fatalf("获取指纹失败: %v", err)
		}

		// 生成DID
		did, err := crypto.GenerateDIDFromKeyPair(keyPair)
		if err != nil {
			log.Fatalf("生成DID失败: %v", err)
		}

		// 序列化密钥对
		keyData, err := json.MarshalIndent(map[string]interface{}{
			"public_key":  jwk,
			"fingerprint": fingerprint,
			"did":         did,
		}, "", "  ")
		if err != nil {
			log.Fatalf("序列化密钥对失败: %v", err)
		}

		// 保存到文件
		if err := os.WriteFile(keyFile, keyData, 0600); err != nil {
			log.Fatalf("保存密钥文件失败: %v", err)
		}

		fmt.Printf("密钥对已生成并保存到: %s\n", keyFile)
		fmt.Printf("DID: %s\n", did)
		fmt.Printf("指纹: %s\n", fingerprint)
	},
}

// registerCmd 注册DID命令
var registerCmd = &cobra.Command{
	Use:   "register [DID]",
	Short: "注册新的DID",
	Long:  `使用指定的DID注册到QLink网络。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		did := args[0]

		// 初始化客户端
		if err := initClient(); err != nil {
			log.Fatalf("初始化客户端失败: %v", err)
		}

		// 构造DID文档
		document := map[string]interface{}{
			"@context": []string{"https://www.w3.org/ns/did/v1"},
			"id":       did,
			"verificationMethod": []map[string]interface{}{
				{
					"id":           did + "#key-1",
					"type":         "JsonWebKey2020",
					"controller":   did,
					"publicKeyJwk": getClientPublicKeyJWK(clientInst),
				},
			},
			"authentication": []string{did + "#key-1"},
		}

		// 注册DID
		resp, err := clientInst.RegisterDID(did, document)
		if err != nil {
			log.Fatalf("注册DID失败: %v", err)
		}

		fmt.Printf("DID注册成功!\n")
		fmt.Printf("DID: %s\n", resp.DID)
		fmt.Printf("消息: %s\n", resp.Message)
	},
}

// resolveCmd 解析DID命令
var resolveCmd = &cobra.Command{
	Use:   "resolve [DID]",
	Short: "解析DID文档",
	Long:  `从QLink网络解析指定的DID文档。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		did := args[0]

		// 初始化客户端
		if err := initClient(); err != nil {
			log.Fatalf("初始化客户端失败: %v", err)
		}

		// 解析DID
		resp, err := clientInst.ResolveDID(did)
		if err != nil {
			log.Fatalf("解析DID失败: %v", err)
		}

		// 格式化输出
		output, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			log.Fatalf("格式化输出失败: %v", err)
		}

		fmt.Println(string(output))
	},
}

// initClient 初始化客户端
func initClient() error {
	if clientInst != nil {
		return nil
	}

	// 创建客户端
	clientInst = client.NewClient(baseURL)

	// 如果指定了密钥文件，加载密钥对
	if keyFile != "" {
		if _, err := os.Stat(keyFile); err == nil {
			// 文件存在，加载密钥对
			if err := loadKeyPair(); err != nil {
				return fmt.Errorf("加载密钥对失败: %w", err)
			}
		} else {
			// 文件不存在，生成新的密钥对
			if err := clientInst.GenerateKeyPair(); err != nil {
				return fmt.Errorf("生成密钥对失败: %w", err)
			}
		}
	}

	return nil
}

// loadKeyPair 加载密钥对
func loadKeyPair() error {
	// TODO: 实现密钥对的加载逻辑
	return clientInst.GenerateKeyPair()
}

// getClientPublicKeyJWK 获取客户端公钥JWK
func getClientPublicKeyJWK(client *client.Client) interface{} {
	keyPair := client.GetKeyPair()
	if keyPair == nil {
		return nil
	}
	jwk, err := keyPair.ToJWK()
	if err != nil {
		return nil
	}
	return jwk
}

func init() {
	// 设置全局标志
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:8080", "QLink节点的API地址")
	rootCmd.PersistentFlags().StringVar(&keyFile, "key", filepath.Join(os.Getenv("HOME"), ".qlink", "key.json"), "密钥文件路径")

	// 添加子命令
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(resolveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
