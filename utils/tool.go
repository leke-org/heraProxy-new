package util

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"strings"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrcSB(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func GetIPv4FromLink() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func RandomIPv6Address(cidr string) (string, error) {
	cidr = cidr + "/52"
	// 解析CIDR字符串
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("parsing CIDR failed: %v", err)
	}

	// 获取网络的掩码大小
	mask := ipnet.Mask
	ones, _ := mask.Size()

	// 计算网络中有多少个可能的地址
	max := new(big.Int)
	max.Lsh(big.NewInt(1), uint(128-ones))

	// 随机生成一个大数作为偏移量
	randOffset, err := crand.Int(crand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generating random offset failed: %v", err)
	}

	// 将随机偏移量添加到基础地址中
	ipBigInt := big.NewInt(0)
	ipBigInt.SetBytes(ip.To16())
	ipBigInt.Add(ipBigInt, randOffset)

	// 将结果转换回net.IP类型
	randomIP := net.IP(ipBigInt.Bytes())

	return randomIP.String(), nil
}

// ParseIPv6 如果是有效的IPv6地址则返回解析后的IP，否则返回nil
func ParseIPv6(ip string) net.IP {
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil && parsedIP.To4() == nil {
		return parsedIP
	}
	return nil
}
