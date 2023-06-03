package CLI

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
)

// Input 终端命令输入检测函数，将获取的命令写入管道，后续共识节点读取
func Input(cli chan<- string) {
	r := bufio.NewReader(os.Stdin)

	for {
		buf, _, err := r.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			check(err)
		}

		line := string(buf)
		if len(line) == 0 {
			continue
		}

		cli <- line

	}
}

func check(err error) {

	if err != nil {
		log.Panic(err)
	}

}
