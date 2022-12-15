package files

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/api.git/config"
	pb "github.com/api.git/github.com/lenush21/file_data"
	semaphore "golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DiscFileStore struct {
	Config          *config.Config
	readSemaphore   *semaphore.Weighted
	streamSemaphore *semaphore.Weighted
}

func NewDiskFileStore(cfg *config.Config) *DiscFileStore {
	return &DiscFileStore{
		Config:          cfg,
		readSemaphore:   semaphore.NewWeighted(int64(100)),
		streamSemaphore: semaphore.NewWeighted(int64(10)),
	}
}

// UploadFile - загрузка файлов на диск.
func (store *DiscFileStore) UploadFile(uploadData pb.FileData_UploadFileServer) error {
	err := store.streamSemaphore.Acquire(uploadData.Context(), 1)
	if err != nil {
		return fmt.Errorf("semaphore acquire error: %w", err)
	}

	defer store.streamSemaphore.Release(1)

	req, err := uploadData.Recv()
	if err != nil && errors.Is(err, io.EOF) {
		return fmt.Errorf("end of file: %w", err)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("receive filename error: %w", err)
	}

	fileName := req.GetFileName()

	fileInFolder := fmt.Sprintf("%s/%s", store.Config.App.Folder, fileName)
	if _, err := os.Stat(fileInFolder); errors.Is(err, os.ErrExist) {
		return fmt.Errorf("file with this name already exists: %w", err)
	}

	file, err := os.Create(fileInFolder)
	if err != nil {
		return fmt.Errorf("create file error: %w", err)
	}

	defer file.Close()

	for {
		req, err := uploadData.Recv()
		if err != nil && errors.Is(err, io.EOF) {
			break
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("receive chunk error: %w", err)
		}

		data := req.GetChunkData()

		_, errWrite := file.Write(data)
		if errWrite != nil {
			return fmt.Errorf("write to file error: %w", err)
		}
	}

	return nil
}

// DownloadFile - скачивание файла.
func (store *DiscFileStore) DownloadFile(request *pb.GetFileRequest, downloadData pb.FileData_DownloadFileServer) error {
	err := store.streamSemaphore.Acquire(downloadData.Context(), 1)
	if err != nil {
		return fmt.Errorf("semaphore acquire error: %w", err)
	}

	defer store.streamSemaphore.Release(1)

	chunkSize := 1024 * 1024

	fileName := request.GetFileName()
	if fileName == "" {
		return fmt.Errorf("file_name is empty")
	}

	fileInFolder := fmt.Sprintf("%s/%s", store.Config.App.Folder, fileName)
	if _, err := os.Stat(fileInFolder); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("file doesn't exist: %w", err)
	}

	chunk := &pb.GetFileResponse{FileChunk: make([]byte, chunkSize)}

	file, err := os.Open(fileInFolder)
	if err != nil {
		return fmt.Errorf("can't open file: %w", err)
	}

	reader := bufio.NewReader(file)

	buf := make([]byte, chunkSize)
	for {
		n, err := reader.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read buf error: %w", err)
		}

		chunk.FileChunk = buf[:n]

		errSend := downloadData.Send(chunk)
		if errSend != nil {
			return fmt.Errorf("send error: %w", err)
		}
	}

	return nil
}

// GetFilesInfo - просмотр информации о файлах.
func (store *DiscFileStore) GetFilesInfo(ctx context.Context, e *emptypb.Empty) (*pb.GetFilesInfoResponse, error) {
	err := store.readSemaphore.Acquire(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("semaphore acquire error: %w", err)
	}

	defer store.readSemaphore.Release(1)

	var filesInfo []*pb.FileInfo

	files, err := ioutil.ReadDir(store.Config.App.Folder)
	if err != nil {
		return nil, fmt.Errorf("read dir error: %w", err)
	}

	for _, file := range files {
		stat := file.Sys().(*syscall.Stat_t)
		createdAt := timestamppb.New(time.Unix(stat.Ctimespec.Sec, stat.Ctimespec.Nsec))
		updatedAt := timestamppb.New(file.ModTime())

		fileInfo := pb.FileInfo{
			Name:      file.Name(),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}

		filesInfo = append(filesInfo, &fileInfo)
	}

	fileResponse := pb.GetFilesInfoResponse{
		Infos: filesInfo,
	}

	return &fileResponse, nil
}
