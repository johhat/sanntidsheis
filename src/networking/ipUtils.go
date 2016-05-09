package networking

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

func HasHighestIp(remoteIp string, localIp string) (bool, error) {

	remoteIpInt, err1 := ipToInt(remoteIp)

	if err1 != nil {
		return false, err1
	}

	localIpInt, err2 := ipToInt(localIp)

	if err2 != nil {
		return false, err2
	}

	return remoteIpInt > localIpInt, nil
}

func ipToInt(ip string) (int, error) {
	ipParts := strings.SplitAfter(ip, ".")

	if len(ipParts) != 4 {
		return 0, errors.New("Malformed ip error")
	}

	return strconv.Atoi(ipParts[3])
}

func getLocalIp() (string, error) {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", errors.New("Could not get local ip")
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", errors.New("Could not get local ip")
}

func GetLocalIp() string {
	return localIp
}
