package NetWork

import (
	"log"
	"net"
	"strings"
)

func GetNatIP() string {

	ip := "0.0.0.0"

	addRs, err := net.InterfaceAddrs()
	CheckErr(err)

	for _, addr := range addRs {
		strs := strings.Split(addr.String(), ".")
		if len(strs) >= 3 {

			if strs[0] == "172" && strs[1] == "16" && strs[2] == "2" {
				ip = strings.TrimSuffix(addr.String(), "/24")
			} else {
				//not the IP I want
			}

		} else {
			//this is not IPv4
		}
	}

	return ip

}

func CheckErr(err error) {

	if err != nil {
		log.Fatal(err)
	}

}
