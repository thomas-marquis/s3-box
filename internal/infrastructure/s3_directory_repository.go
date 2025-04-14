package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/thomas-marquis/s3-box/internal/connection"
	"github.com/thomas-marquis/s3-box/internal/explorer"
	"go.uber.org/zap"
)

type S3DirectoryRepositoryImpl struct {
	log *zap.SugaredLogger
	session *session.Session
	s3      *s3.S3
	conn *connection.Connection
}

var _ explorer.S3DirectoryRepository = &S3DirectoryRepositoryImpl{}

func NewS3DirectoryRepositoryImpl(logger *zap.Logger, conn *connection.Connection) (*S3DirectoryRepositoryImpl, error) {
	log := logger.Sugar()

	region := conn.Region
	if region == "" {
		region = "us-east-1" // for custom endpoints, value is not important but still required
	}
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(conn.AccessKey, conn.SecretKey, ""),
		Endpoint:    aws.String(conn.Server),
		Logger: aws.LoggerFunc(func(args ...interface{}) {
			log.Infof(args[0].(string), args[1:]...)
		}),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		DisableSSL:       aws.Bool(!conn.UseTls),
	})
	if err != nil {
		log.Errorf("Error creating session: %v\n", err)
		return nil, fmt.Errorf("NewS3DirectoryRepositoryImpl(conn=%s): %w", conn.Name, err)
	}

	return &S3DirectoryRepositoryImpl{
		conn: conn,
		session: sess,
		s3:      s3.New(sess),
		log:     log,
	}, nil
	
	// var remoteSrc *s3Client
	// var err error
	// if defaultConn != nil {
	// 	remoteSrc, err = newS3Client(log, defaultConn.AccessKey, defaultConn.SecretKey, defaultConn.Server, defaultConn.BucketName, defaultConn.Region, defaultConn.UseTls)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("NewS3DirectoryRepositoryImpl: %w", err)
	// 	}
	// }

}

func (r *S3DirectoryRepositoryImpl) GetByID(ctx context.Context, id explorer.S3DirectoryID) (*explorer.S3Directory, error) {	
	parentID := getParentDirIDFromChildID(id)
	dirName := id.ToName()
	dir, err := explorer.NewS3Directory(dirName, parentID)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	
	queryPath := getQueryPath(id)

    inputs := &s3.ListObjectsInput{
        Bucket:    aws.String(r.conn.BucketName),
        Prefix:    aws.String(queryPath),
        Delimiter: aws.String("/"),
        MaxKeys:   aws.Int64(1000),
    }

    pageHandler := func(page *s3.ListObjectsOutput, lastPage bool) bool {
        for _, obj := range page.Contents {
            key := *obj.Key
            if key == queryPath {
                continue
            }
            newFile, _ := dir.CreateFile(getNameFromS3Key(key))
			newFile.SizeBytes = *obj.Size
			newFile.LastModified = *obj.LastModified
			dir.AddFile(newFile)
        }

        for _, obj := range page.CommonPrefixes {
            if *obj.Prefix == queryPath {
                continue
            }
            s3Prefix := *obj.Prefix
            isDir := strings.HasSuffix(s3Prefix, "/")
            if isDir {
                dirName := getNameFromS3Key(s3Prefix)
                dir.AddSubDirectory(dirName)
            }
        }
        return !lastPage
    }

	if err := r.s3.ListObjectsPagesWithContext(ctx, inputs, pageHandler); err != nil {
		r.log.Errorf("Error listing objects: %v\n", err)
		return nil, fmt.Errorf("GetByID: %w", err)
	}

	return dir, nil
}

func (r *S3DirectoryRepositoryImpl) Save(ctx context.Context, d *explorer.S3Directory) error {
	return nil
}

func getQueryPath(id explorer.S3DirectoryID) string {
	var queryPath string
	if id == explorer.RootDirID {
		queryPath = ""
	} else {
		queryPath = strings.TrimPrefix(id.String(), "/")
		queryPath = strings.TrimSuffix(queryPath, "/") + "/"
	}
	return queryPath
}

func getParentDirIDFromChildID(id explorer.S3DirectoryID) explorer.S3DirectoryID {
	if id == explorer.RootDirID {
		return explorer.NilParentID
	}
	dirPathStriped := strings.TrimSuffix(id.String(), "/")
	dirPathSplit := strings.Split(dirPathStriped, "/")
	
	if len(dirPathSplit) <= 1 {
		return explorer.RootDirID
	}
	
	parentPath := strings.Join(dirPathSplit[:len(dirPathSplit)-1], "/")
	return explorer.S3DirectoryID(parentPath)
}

func getNameFromS3Key(path string) string {
	if path == "" {
		return ""
	}
	dirPathStriped := strings.TrimSuffix(path, "/")
	dirPathSplit := strings.Split(dirPathStriped, "/")
	dirName := dirPathSplit[len(dirPathSplit)-1]
	return dirName
}
