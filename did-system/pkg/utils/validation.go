package utils

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

// DID格式验证正则表达式
var didRegex = regexp.MustCompile(`^did:[a-z0-9]+:[a-zA-Z0-9._-]+$`)

// ValidateDID 验证DID格式
func ValidateDID(did string) error {
	if did == "" {
		return fmt.Errorf("DID不能为空")
	}

	if !didRegex.MatchString(did) {
		return fmt.Errorf("DID格式无效: %s", did)
	}

	return nil
}

// ValidateNodeID 验证节点ID
func ValidateNodeID(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	if len(nodeID) < 3 || len(nodeID) > 64 {
		return fmt.Errorf("节点ID长度必须在3-64字符之间")
	}

	// 只允许字母、数字、连字符和下划线
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, nodeID)
	if !matched {
		return fmt.Errorf("节点ID只能包含字母、数字、连字符和下划线")
	}

	return nil
}

// ValidateAddress 验证网络地址
func ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("地址不能为空")
	}

	// 尝试解析为URL
	if strings.Contains(address, "://") {
		u, err := url.Parse(address)
		if err != nil {
			return fmt.Errorf("无效的URL格式: %v", err)
		}

		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("URL必须包含scheme和host")
		}

		return nil
	}

	// 尝试解析为host:port格式
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("无效的地址格式: %v", err)
	}

	// 验证主机
	if host == "" {
		return fmt.Errorf("主机不能为空")
	}

	// 验证端口
	if port == "" {
		return fmt.Errorf("端口不能为空")
	}

	return nil
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("邮箱格式无效")
	}

	return nil
}

// ValidatePort 验证端口号
func ValidatePort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}

	return nil
}

// ValidateTimeout 验证超时时间
func ValidateTimeout(timeout int) error {
	if timeout <= 0 {
		return fmt.Errorf("超时时间必须大于0")
	}

	if timeout > 3600 {
		return fmt.Errorf("超时时间不能超过3600秒")
	}

	return nil
}

// ValidateStringLength 验证字符串长度
func ValidateStringLength(str string, minLen, maxLen int, fieldName string) error {
	length := len(str)

	if length < minLen {
		return fmt.Errorf("%s长度不能少于%d个字符", fieldName, minLen)
	}

	if length > maxLen {
		return fmt.Errorf("%s长度不能超过%d个字符", fieldName, maxLen)
	}

	return nil
}

// ValidateRequired 验证必填字段
func ValidateRequired(value string, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s是必填字段", fieldName)
	}

	return nil
}

// ValidateEnum 验证枚举值
func ValidateEnum(value string, validValues []string, fieldName string) error {
	for _, valid := range validValues {
		if value == valid {
			return nil
		}
	}

	return fmt.Errorf("%s的值必须是以下之一: %s", fieldName, strings.Join(validValues, ", "))
}
