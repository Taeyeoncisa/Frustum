package SeedNode

import (
	"BlockChain/TxPool"
	"archive/zip"
	"encoding/csv"
	"go.uber.org/zap"
	"io"
	"log"
	"path"
)

type FileHandle struct {
	Logger *zap.SugaredLogger
	SeTxCh chan *TxPool.Transaction
}

type CSVLineHandle func([]string, *FileHandle)

func OpenAndReadZip(pathStr string, handle CSVLineHandle, fileHandle *FileHandle) {

	archive, err := zip.OpenReader(pathStr)
	if err != nil {
		log.Fatalln(err)
	}
	defer archive.Close()

	for _, file := range archive.File {
		//zip解压完存在多个文件，只读取csv文件
		if path.Ext(file.Name) == ".csv" {
			f, errF := file.Open()
			if errF != nil {
				log.Fatalf("Fail open file %s in ZipFile, %v", file.Name, errF)
			}

			csvReader := csv.NewReader(f)
			fileHandle.Logger.Infoln("===============================")
			fileHandle.Logger.Infof("Start Reading [%s]", file.Name)
			for {
				line, errL := csvReader.Read()
				if errL == io.EOF {
					break
				}
				if errL != nil {
					log.Fatalf("Fail read CSVFile %s, %v", file.Name, errL)
				}
				//处理csv单行数据
				handle(line, fileHandle)

			}

		} else {
			//not a csv file, do not process
		}

	}

}

func GenTx([]string, *FileHandle) {

}
