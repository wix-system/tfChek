package storer

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"os"
)

func Save2FileFromReader(id int, in io.Reader) error {
	dir := viper.GetString("outdir")
	file, err := os.Create(fmt.Sprintf("%s/task-%d", dir, id))
	if err != nil {
		log.Printf("Cannot create file task-%d Error %s", id, err)
		return err
	}
	defer file.Close()
	fInfo, err := file.Stat()
	if err != nil {
		log.Printf("Cannot get file task-%d info. Error: %s", id, err)
		return err
	}
	buf := make([]byte, 4096)
	bin := bufio.NewReader(in)
	for {
		n, err := bin.Read(buf)
		if err != nil {
			if err == io.EOF {
				file.Write(buf[:n])
				break
			} else {
				log.Printf("Cannot create file task-%d Error %s", id, err)
				return err
			}
		}
		file.Write(buf)
	}
	log.Printf("Task %d output has been stored to file %s", id, fInfo.Name())
	return nil
}

func Save2FileFromWriter(id int) (io.WriteCloser, error) {
	dir := viper.GetString("outdir")
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	file, err := os.Create(fmt.Sprintf("%s/task-%d", dir, id))
	if err != nil {
		log.Printf("Cannot create file task-%d Error %s", id, err)
		return nil, err

	}
	return file, nil
}
