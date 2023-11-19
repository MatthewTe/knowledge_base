package parsers

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"golang.org/x/net/context"
)

func UploadImageFileToStatic(ctx context.Context, minioClient *minio.Client, bucketName string, bucketFilePath string, reader *os.File) (string, error) {

	// Creating the client object:
	err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})

	// First we create the bucket if it doens't exist:
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("Bucket already exists, skipping bucket creation")
		} else {
			log.Println("Error in creating bucket in Minio", bucketName, err)
			return bucketFilePath, err
		}
	} else {
		log.Println("Successfully created: ", bucketName)
	}

	// Calculating the size of the byte array to be uploaded:
	objectStat, err := reader.Stat()
	if err != nil {
		log.Println("Error in calculating the statistics for the file", err)
		return bucketFilePath, err
	}

	// Streaming all of the bytes from reader into a buffer to determine MIME type of img:
	var buf []byte
	_, err = reader.Read(buf)
	if err != nil && err == io.EOF {
		log.Println("Read image file bytes into buffer to determine filetype", bucketFilePath)
	} else if err != nil {
		log.Println("Unable to stream file bytes into buffer to determine MIME type for upload", err)
		return bucketFilePath, err
	}

	mimeType := http.DetectContentType(buf)

	info, err := minioClient.PutObject(
		ctx,
		bucketName,
		bucketFilePath,
		reader,
		objectStat.Size(),
		minio.PutObjectOptions{ContentType: mimeType},
	)
	if err != nil {
		log.Println("Unable to insert the html page into the s3 bucket", err)
	} else {
		log.Println("Inserted image", bucketFilePath, "of size:", info.Size, "bytes", "successfully into bucket", bucketName)
	}

	return bucketFilePath, err

}
