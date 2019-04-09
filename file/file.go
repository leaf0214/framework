package file

import (
	"bufio"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path"
	cf "xianhetian.com/framework/config"
	"xianhetian.com/framework/logger"
)

var (
	perm os.FileMode = 0644
)

type File struct {
	Name       string   //文件的名称
	Path       string   //文件的路径
	Data       []byte   //文件的byte数组
	ResultPath []string // 文件路径集合
}

/***
  写文件的方法
*/
func (file *File) Write() (s string, err error) {
	file.setPath()
	b, err := PathExists(file.Path)
	if err != nil {
		logger.Error(err)
		return
	}
	if !b {
		err = os.Mkdir(file.Path, os.ModePerm)
		if err != nil {
			logger.Error(err)
			return
		}
	}
	dir := file.Path + file.Name
	return file.Name, ioutil.WriteFile(dir, file.Data, perm)
}

/*
判断文件夹是否存在
*/
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

/***
读文件的
*/
func (file *File) Read() error {
	data, err := ioutil.ReadFile(file.Path + file.Name)
	file.Data = data
	return err
}

/***
  文件转换byte数组 返回多个byte数组
*/
func File2Bytes(files []*multipart.FileHeader) (data map[int][]byte, err error) {
	data = make(map[int][]byte)
	for i, _ := range files {
		f, err := files[i].Open()
		defer f.Close()
		br := bufio.NewReader(f)
		var dt []byte
		for {
			var b byte
			b, err = br.ReadByte()
			if err != nil {
				logger.Error(err)
				break
			}
			dt = append(dt, b)
		}
		if err != io.EOF {
			logger.Error(err)
			return data, err
		}
		data[i] = dt
	}
	return data, err
}

/***
File path check
*/
func (file *File) setPath() {
	p := file.Path
	if len(p) <= 0 {
		file.Path = cf.Config.DefaultString("file_path", "/files/baas-images/")
	}
	if p[len(p)-1] != '/' {
		p = p + "/"
	}
	file.Path = cf.Config.DefaultString("file_path", "/files/baas-images/") + p
}

/***
  File  Upload methods
*/
func FileUpload(files []*multipart.FileHeader, savePath string) (File, error) {
	var (
		file  File
		datas map[int][]byte
		err   error
	)
	file.Path = savePath ///上传保存路径
	if datas, err = File2Bytes(files); err != nil {
		logger.Error(err)
		return file, err
	}
	for i, v := range datas {
		file.Name = files[i].Filename
		//if files[i].Size > fileSize {
		//	logger.Error(err)
		//	return file, err
		//}
		/***
		  在这里对文件加密处理
		*/
		file.Data = v
		s, err := file.Write()
		if err != nil {
			logger.Error(err)
			file.ResultPath = append(file.ResultPath, s)
			return file, err
		}
		file.ResultPath = append(file.ResultPath, s)
	}
	return file, err
}

/***
  File download methods
*/
func FileDownload(fileFullPath string) (file File, err error) {
	dir, name := path.Split(fileFullPath)
	file.Path = dir
	file.Name = name
	if err := file.Read(); err != nil {
		logger.Error(err.Error() + "   Failed to get the file!")
		return file, err
	}
	return file, err
}
